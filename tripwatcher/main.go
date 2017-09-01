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

const IOS = "ios"
const Android = "android"

const dbCheckFrequency = 1 * time.Minute

// allow an extra 30 seconds for safety
const waitingWindowThreshold = 30 * 1000

// gorush configuration file
const configFile = "config.yml"

type RouteGenerator interface {
	GenerateRoutes(trip *api.TripSchedule) <-chan *api.RouteOption
}

type DefaultRouteGenerator struct {
	finder api.RouteFinder
}

func NewDefaultRouteGenerator(finder api.RouteFinder) *DefaultRouteGenerator {
	return &DefaultRouteGenerator{finder: finder}
}

func main() {
	keyArg := kingpin.Arg("googlemapskey", "Google Maps API key for querying routes").Required().String()
	kingpin.Parse()
	apiKey := *keyArg
	if len(apiKey) == 0 {
		log.Fatal("No api key set.")
	}

	// set up push notification configuration
	gorush.PushConf = config.BuildDefaultPushConf()
	var err error
	gorush.PushConf, err = config.LoadConfYaml(configFile)
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
	finder := api.NewGoogleMapsFinder(apiKey)
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

func roundToNextInterval(milliseconds int64) int64 {
	if milliseconds > 60*60*1000 {
		// wait until an hour before starting checks
		return milliseconds - 60*60*1000
	}
	// quarter the minutes remaining each time
	minuteAsFloat := (float64(milliseconds) / (1000.0 * 60.0)) / 4.0
	// if it's very close then we return
	if minuteAsFloat < 0.25 {
		return milliseconds
	}
	// add an extra minute so that we don't return zero and continue
	// returning intervals when milliseconds is small
	rounded := int64(minuteAsFloat + 1)
	return rounded * 60 * 1000
}

// watchTrip will watch the trip using the specified generator to generate
// new routes and will return the latest route when it's reached notification
// time
func watchTrip(trip *api.TripSchedule, generator RouteGenerator) *api.RouteOption {
	now := getCurrentMillis()
	// add some extra time to the waiting window since push notification won't be instant
	safetyBuffer := trip.WaitingWindowMs + waitingWindowThreshold
	// get next departure time
	departureTime := api.GetDepartureTime(trip)
	notificationTime := departureTime - safetyBuffer
	// time until notification should be sent
	timeout := notificationTime - now
	// create a new route with current dates as opposed to the stored ones
	// from the original schedule
	// this variable will keep track of the last valid route for this trip
	prevRoute := updateRouteDates(trip.Route, departureTime)
	for {
		// We select on a timeout or until a route has been found so that
		// we will always send a notification instead of potentially failing
		// on slow or unresponsive route requests
		select {
		case route := <-generator.GenerateRoutes(trip):
			if route == nil {
				route = prevRoute
			}
			// store this route
			prevRoute = route
			now = getCurrentMillis()
			// calculate next notification time
			notificationTime = route.DepartureTime - safetyBuffer
			timeLeft := notificationTime - now
			if timeLeft <= 0 {
				return prevRoute
			}
			// get the next time that we should re-check the route
			nextCheck := roundToNextInterval(timeLeft)
			// ensure that we don't overshoot the notification time
			if now+nextCheck > notificationTime {
				nextCheck = now - notificationTime
			}
			// sleep until next check
			time.Sleep(time.Duration(nextCheck) * time.Millisecond)
			// if nextCheck matches timeLeft then we're done
			if nextCheck == timeLeft {
				return prevRoute
			}
			// update the timeout so that we don't miss the notification
			// time while waiting for a response
			now = getCurrentMillis()
			timeout = notificationTime - now
		case <-time.After(time.Duration(timeout) * time.Millisecond):
			// check whether we should have finished by now
			now = getCurrentMillis()
			if notificationTime-now <= 0 {
				return prevRoute
			}
		}
	}
}

func getCurrentMillis() int64 {
	return time.Now().UnixNano() / 1e6
}

func (g *DefaultRouteGenerator) GenerateRoutes(trip *api.TripSchedule) <-chan *api.RouteOption {
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
	now := getCurrentMillis()
	// if it's been two hours since the trip then it should be safe
	var twoHours int64 = 2 * 60 * 60 * 1000
	return now-trip.Route.ArrivalTime > twoHours
}

// updateRouteDates returns a new RouteOption that has updated dates based on
// the input timestamp, so that all dates share the same day
func updateRouteDates(route *api.RouteOption, departureTime int64) *api.RouteOption {
	routeDeparture := time.Unix(route.DepartureTime/1000, 0)
	routeArrival := time.Unix(route.ArrivalTime/1000, 0)
	newDeparture := time.Unix(departureTime/1000, 0)
	if routeDeparture == newDeparture {
		return route
	}
	// create new dates based on new departure time where the time of
	// day is left intact
	departure := time.Date(
		newDeparture.Year(), newDeparture.Month(), newDeparture.Day(),
		routeDeparture.Hour(),
		routeDeparture.Minute(), routeDeparture.Second(),
		routeDeparture.Nanosecond(), newDeparture.Location(),
	)
	arrival := time.Date(
		newDeparture.Year(), newDeparture.Month(), newDeparture.Day(),
		routeArrival.Hour(),
		routeArrival.Minute(), routeArrival.Second(),
		routeArrival.Nanosecond(), newDeparture.Location(),
	)
	newRoute := &api.RouteOption{
		DepartureTime: departure.Unix() * 1000,
		ArrivalTime:   arrival.Unix() * 1000,
		Name:          route.Name,
		Description:   route.Description,
	}
	return newRoute
}

func sendNotification(route *api.RouteOption, user *api.UserInfo) error {
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