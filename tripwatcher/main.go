package main

import (
	"fmt"
	"github.com/appleboy/gorush/config"
	"github.com/appleboy/gorush/gorush"
	"github.com/oliveroneill/todserver/api"
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
	"sync"
	"time"
)

// IOS is a string identifier for ios devices
const IOS = "ios"

// Android is a string identifier for android devices
const Android = "android"

const dbCheckFrequency = 1 * time.Minute

// allow an extra 30 seconds for safety
const waitingWindowThreshold = 30 * time.Millisecond

// gorush configuration file
const configFile = "config.yml"

// RouteGenerator is an interface that will send routes back over a channel
// This is useful for timing out route search requests
type RouteGenerator interface {
	GenerateRoute(trip *api.TripSchedule) <-chan *api.RouteOption
}

// DefaultRouteGenerator is an implementaation of RouteGenerator that
// wraps api.RouteFinder
type DefaultRouteGenerator struct {
	finder api.RouteFinder
}

// NewDefaultRouteGenerator will create an instance of DefaultRouteGenerator
// @param finder - the finder used to generate a route
func NewDefaultRouteGenerator(finder api.RouteFinder) *DefaultRouteGenerator {
	return &DefaultRouteGenerator{finder: finder}
}

func main() {
	mapsKeyArg := kingpin.Arg("googlemapskey", "Google Maps API key for querying routes").Required().String()
	nxtBusKeyArg := kingpin.Flag("nxtbuskey", "NXTBUS API key for real time data in Canberra").String()
	kingpin.Parse()
	mapsAPIKey := *mapsKeyArg
	if len(mapsAPIKey) == 0 {
		log.Fatal("No api key set.")
	}
	nxtBusAPIKey := *nxtBusKeyArg
	mapsFinder := api.NewGoogleMapsFinder(mapsAPIKey)
	var finder api.RouteFinder = mapsFinder
	if len(nxtBusAPIKey) > 0 {
		finder = api.NewNxtBusFinder(nxtBusAPIKey, mapsFinder)
	}

	// set up push notification configuration
	var err error
	// passing in an empty string will load the default
	gorush.PushConf, err = config.LoadConf("")
	if err != nil {
		panic(fmt.Sprintf("Failed to load default gorush config: '%v'", err))
	}
	gorush.PushConf, err = config.LoadConf(configFile)
	if err != nil {
		panic(fmt.Sprintf("Load yaml config file error: '%v'", err))
	}
	if err = gorush.InitLog(); err != nil {
		fmt.Println(err)
		return
	}

	// watchList will keep track of which trips are already running
	// so that we don't watch a trip twice
	watchList := make(map[string]bool)
	mux := &sync.Mutex{}
	trips, err := api.GetAllScheduledTrips()
	if err == nil {
		watchTrips(trips, finder, watchList, mux)
	}
	// check the database for new scheduled trips
	for _ = range time.Tick(dbCheckFrequency) {
		trips, err := api.GetAllScheduledTrips()
		if err != nil {
			fmt.Println(err)
			continue
		}
		watchTrips(trips, finder, watchList, mux)
	}
}

// watchTrips will keep track of the trips and ensure that notifications
// are sent when necessary
// @param trips - trips to watch
// @param finder - used to find routes for these trips
// @param watchList - this should be updated with the currently watched
//        trips so that we don't double up on a trip and send an alert twice
// @param mux - used so that we can safely delete trips from the watch list
// @return the number of new trips now being watched
func watchTrips(trips []*api.TripSchedule, finder api.RouteFinder, watchList map[string]bool, mux *sync.Mutex) {
	// create a generator that uses the input finder to get routes
	generator := NewDefaultRouteGenerator(finder)
	for _, t := range trips {
		// ensure that we're not already watching this trip
		mux.Lock()
		_, alreadyWatching := watchList[t.ID]
		mux.Unlock()
		if alreadyWatching {
			continue
		}
		// clear out disabled non-repeating trips
		if !t.Enabled && !api.IsRepeating(t) {
			// Delete a few hours after the arrival date
			if tripHasPast(t) {
				api.DeleteTrip(t.ID, t.User.ID)
				// delete the trip from the watch list
				mux.Lock()
				delete(watchList, t.ID)
				mux.Unlock()
			}
			continue
		}
		mux.Lock()
		// add trip to watch list
		watchList[t.ID] = true
		mux.Unlock()
		// in the background wait for the trip to reach notification
		// time
		go func(trip *api.TripSchedule) {
			watchTrip(trip, generator)
			// check that it's still enabled
			if api.IsEnabled(trip) {
				// send alert
				fmt.Println("Sending alert for", trip.Route.Description)
				sendNotification(trip.Route, trip.User)
			}
			// delete scheduled trip if it's not repeating
			if !api.IsRepeating(trip) {
				api.DeleteTrip(trip.ID, trip.User.ID)
			} else {
				api.SetLastNotificationTime(trip, time.Now().Unix()*1000)
			}
			// delete the trip from the watch list
			mux.Lock()
			delete(watchList, trip.ID)
			mux.Unlock()
		}(t)
	}
}

func roundToNextInterval(timeLeft time.Duration) time.Duration {
	if timeLeft > 1*time.Hour {
		// wait until an hour before starting checks
		return timeLeft - 1*time.Hour
	}
	// quarter the minutes remaining each time
	minuteAsFloat := float64(timeLeft/time.Minute) / 4.0
	// if it's very close then we return
	if minuteAsFloat < 0.25 {
		return timeLeft
	}
	// add an extra minute so that we don't return zero and continue
	// returning intervals when milliseconds is small
	rounded := int64(minuteAsFloat + 1)
	return time.Duration(rounded) * time.Minute
}

// watchTrip will watch the trip using the specified generator to generate
// new routes and will return the latest route when it's reached notification
// time
func watchTrip(trip *api.TripSchedule, generator RouteGenerator) *api.RouteOption {
	now := time.Now()
	// add some extra time to the waiting window since push notification won't be instant
	waitingWindow := time.Duration(trip.WaitingWindowMs) * time.Millisecond
	safetyBuffer := waitingWindow + waitingWindowThreshold
	// get next departure time
	departureTime := api.GetDepartureTime(trip)
	notificationTime := departureTime.Add(-safetyBuffer)
	// time until notification should be sent
	timeout := notificationTime.Sub(now)
	// create a new route with current dates as opposed to the stored ones
	// from the original schedule
	// this variable will keep track of the last valid route for this trip
	prevRoute := updateRouteDates(trip.Route, departureTime)
	for {
		// We select on a timeout or until a route has been found so that
		// we will always send a notification instead of potentially failing
		// on slow or unresponsive route requests
		select {
		case route := <-generator.GenerateRoute(trip):
			if route == nil {
				route = prevRoute
			}
			// store this route
			prevRoute = route
			now = time.Now()
			// calculate next notification time
			notificationTime = route.DepartureTime.Add(-safetyBuffer)
			timeLeft := notificationTime.Sub(now)
			if timeLeft <= 0 {
				return prevRoute
			}
			// get the next time that we should re-check the route
			nextCheck := roundToNextInterval(timeLeft)
			// ensure that we don't overshoot the notification time
			if now.Add(nextCheck).After(notificationTime) {
				nextCheck = now.Sub(notificationTime)
			}
			// sleep until next check
			time.Sleep(nextCheck)
			// if we've reached or passed the notification time then we're done
			if time.Now().UnixNano() >= notificationTime.UnixNano() {
				return prevRoute
			}
			// update the timeout so that we don't miss the notification
			// time while waiting for a response
			now = time.Now()
			timeout = notificationTime.Sub(now)
		case <-time.After(timeout):
			// check whether we should have finished by now
			now = time.Now()
			if notificationTime.Sub(now) <= 0 {
				return prevRoute
			}
		}
	}
}

// GenerateRoute will send a route back over the returned channel.
// The returned route will be the most similar route available to that
// specified in the input trip.
// This is done asynchronously, if there is an error a nil value will be sent
// over the channel
// @param trip - a route will be searched for based on information specified
// in this trip
// @returns channel that will send a route or nil value if an error occurs
func (g *DefaultRouteGenerator) GenerateRoute(trip *api.TripSchedule) <-chan *api.RouteOption {
	channel := make(chan *api.RouteOption)
	// will send a route over the channel
	go func() {
		defer close(channel)
		route, err := api.GetRoute(g.finder, trip)
		if err != nil {
			fmt.Println(err)
			channel <- nil
			return
		}
		channel <- &route
	}()
	return channel
}

func tripHasPast(trip *api.TripSchedule) bool {
	// check whether it's safe to delete a disabled trip
	now := time.Now()
	// if it's been two hours since the trip then it should be safe
	twoHours := 2 * time.Hour
	return now.Sub(trip.Route.ArrivalTime) > twoHours
}

// updateRouteDates returns a new RouteOption that has updated dates based on
// the input timestamp, so that all dates share the same day
func updateRouteDates(route *api.RouteOption, departureTime time.Time) *api.RouteOption {
	if route.DepartureTime.Equal(departureTime) {
		return route
	}
	// create new dates based on new departure time where the time of
	// day is left intact
	departure := time.Date(
		departureTime.Year(), departureTime.Month(), departureTime.Day(),
		route.DepartureTime.Hour(),
		route.DepartureTime.Minute(), route.DepartureTime.Second(),
		route.DepartureTime.Nanosecond(), departureTime.Location(),
	)
	arrival := time.Date(
		departureTime.Year(), departureTime.Month(), departureTime.Day(),
		route.ArrivalTime.Hour(),
		route.ArrivalTime.Minute(), route.ArrivalTime.Second(),
		route.ArrivalTime.Nanosecond(), departureTime.Location(),
	)
	newRoute := &api.RouteOption{
		DepartureTime: departure,
		ArrivalTime:   arrival,
		Name:          route.Name,
		Description:   route.Description,
	}
	return newRoute
}

func sendNotification(route *api.RouteOption, user *api.UserInfo) error {
	// If using with production you must specify a Topic in this struct
	req := gorush.PushNotification{
		Tokens:  []string{user.NotificationToken},
		Message: fmt.Sprintf("Time to leave for route: %s", route.Description),
		Sound:   "default",
	}
	if user.DeviceOS == IOS {
		req.Platform = gorush.PlatFormIos
		err := gorush.CheckMessage(req)
		if err != nil {
			fmt.Println(err)
		}
		if err := gorush.InitAppStatus(); err != nil {
			fmt.Println(err)
			return err
		}
		if err := gorush.InitAPNSClient(); err != nil {
			fmt.Println(err)
			return err
		}
		gorush.PushToIOS(req)
	} else {
		req.Platform = gorush.PlatFormAndroid
		// You can specify the notification icon that the client will use here
		err := gorush.CheckMessage(req)
		if err != nil {
			fmt.Println(err)
		}
		if err := gorush.InitAppStatus(); err != nil {
			fmt.Println(err)
			return err
		}
		gorush.PushToAndroid(req)
	}
	return nil
}
