package api

import (
	"errors"
	"math"
	"time"
)

// UserInfo stores information regarding the user
type UserInfo struct {
	ID                string `json:"user_id"`
	NotificationToken string `json:"notification_token"`
	DeviceOS          string `json:"device_os"`
}

// Point stores a lat and lng to indicate a location
type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Date stores the Unix timestamp in milliseconds as well as a local date
// string. This is useful for timezone calculations
type Date struct {
	String           string `json:"local_date_string"`
	Timestamp        int64  `json:"timestamp"`
	TimezoneLocation string `json:"timezone_location"`
}

// TripSchedule is all information regarding a scheduled trip
type TripSchedule struct {
	ID          string       `json:"trip_id"`
	User        *UserInfo    `json:"user"`
	Origin      Point        `json:"origin"`
	Destination Point        `json:"destination"`
	Route       *RouteOption `json:"route"`
	// the date the user entered when searching for routes
	InputArrivalTime *Date `json:"input_arrival_time"`
	// the alert should be sent this many milliseconds before departure time
	WaitingWindowMs int64  `json:"waiting_window_ms"`
	TransportType   string `json:"transport_type"`
	RepeatDays      []bool `json:"repeat_days"`
	Enabled         bool   `json:"enabled"`
	// timestamp the last notification for this trip was sent
	LastNotificationSent int64 `json:"last_notification"`
}

// UpsertUser will add this user if they aren't already added.
// If they are added then this will update the notification token and
// the device operating system to the new values
func UpsertUser(user *UserInfo) error {
	db := NewPostgresInterface()
	defer db.Close()
	return db.UpsertUser(user)
}

// GetScheduledTrips returns all trips scheduled for this user
func GetScheduledTrips(user string) ([]TripSchedule, error) {
	db := NewPostgresInterface()
	defer db.Close()
	return db.GetTrips(user)
}

// GetAllScheduledTrips returns every scheduled trip on this server. This
// is used by the tripwatcher
func GetAllScheduledTrips() ([]*TripSchedule, error) {
	db := NewPostgresInterface()
	defer db.Close()
	return db.GetAllScheduledTrips()
}

// EnableDisableTrip will turn on or off notifications so that you can
// temporarily disable a scheduled trip
// @param id - the id of the trip to remove
// @param userID - this is used to ensure that the user modifying this trip
// actually scheduled it
func EnableDisableTrip(id string, userID string) error {
	// toggle trip
	db := NewPostgresInterface()
	defer db.Close()
	return db.EnableDisableTrip(id, userID)
}

// DeleteTrip will delete the trip with the specified id
// @param id - the id of the trip to remove
// @param userID - this is used to ensure that the user modifying this trip
// actually scheduled it
func DeleteTrip(id string, userID string) error {
	db := NewPostgresInterface()
	defer db.Close()
	return db.DeleteTrip(id, userID)
}

// SetLastNotificationTime records the time that this trip was last watched
// This is useful to ensure that duplicate notifications aren't sent for
// the same day
func SetLastNotificationTime(trip *TripSchedule, timestamp int64) error {
	db := NewPostgresInterface()
	defer db.Close()
	return db.SetLastNotificationTime(trip, timestamp)
}

// IsEnabled returns whether the input trip is enabled
func IsEnabled(trip *TripSchedule) bool {
	db := NewPostgresInterface()
	defer db.Close()
	return db.IsEnabled(trip)
}

// ScheduleTrip will add this trip to the database
func ScheduleTrip(trip *TripSchedule) error {
	// store trip
	db := NewPostgresInterface()
	defer db.Close()
	err := db.ScheduleTrip(trip)
	return err
}

// GetRoute will find a route suitable for this scheduled trip
func GetRoute(finder RouteFinder, trip *TripSchedule) (RouteOption, error) {
	resp := finder.FindRoutes(trip.Origin.Lat, trip.Origin.Lng,
		trip.Destination.Lat, trip.Destination.Lng,
		trip.TransportType, GetInputArrivalTime(trip),
		trip.Route.Name)
	return getRouteFromDescription(trip, resp)
}

// getRouteFromDescription find a route with the same description as the
// scheduled trip. If this can't be found then we will fall back to
// a route with the closest arrival time to that recorded in the scheduled
// trip
// @param trip - scheduled trip
// @param routes - the routes to sort through
func getRouteFromDescription(trip *TripSchedule, routes []RouteOption) (RouteOption, error) {
	if len(routes) == 0 {
		return RouteOption{}, errors.New("No routes")
	}
	filtered := []RouteOption{}
	// Find all routes with the same descriptions as the trip
	for _, r := range routes {
		if r.Description == trip.Route.Description {
			filtered = append(filtered, r)
		}
	}
	// If there aren't any then find one close to arrival time
	if len(filtered) == 0 {
		return findRouteClosestToArrival(GetArrivalTime(trip), routes), nil
	}
	// If there are multiple with the same description then choose the
	// one closest to arrival of the filtered list
	if len(filtered) > 1 {
		return findRouteClosestToArrival(GetArrivalTime(trip), filtered), nil
	}
	return filtered[0], nil
}

func findRouteClosestToArrival(arrivalTime int64, routes []RouteOption) RouteOption {
	var closest int64 = -1
	var choice RouteOption
	for _, r := range routes {
		diff := int64(math.Abs(float64(r.ArrivalTime) - float64(arrivalTime)))
		if diff < closest || closest == -1 {
			choice = r
			closest = diff
		}
	}
	return choice
}

// IsRepeating will return true if this trip is set up to repeat
func IsRepeating(trip *TripSchedule) bool {
	for _, b := range trip.RepeatDays {
		if b {
			return true
		}
	}
	return false
}

// GetDepartureTime will return the next departure time for the trip
// If this is a repeating trip then the departure time will be updated to the
// next repeating day
func GetDepartureTime(trip *TripSchedule) int64 {
	return getNextTime(trip, trip.Route.DepartureTime)
}

// GetInputArrivalTime will return the arrival time that the user input for the
// trip. This is specifically useful for finding routes that are close to
// what the user originally searched for.
// If this is a repeating trip then this timestamp will be updated to the
// next repeating day
func GetInputArrivalTime(trip *TripSchedule) int64 {
	return getNextTime(trip, trip.InputArrivalTime.Timestamp)
}

// GetArrivalTime will return the next arrival time for the trip
// If this is a repeating trip then the arrival time will be updated to the
// next repeating day
func GetArrivalTime(trip *TripSchedule) int64 {
	return getNextTime(trip, trip.Route.ArrivalTime)
}

// getNextTime will return an updated timestamp using the repeated
// days of the trip. Where the input ts is updated to the next repeating
// day and the time of day is left intact.
// @param trip - the trip to use to find next repeating day
// @param ts - timestamp in milliseconds that will be returned with an
// updated day
// @returns a new timestamp with same time of day as input but the day
// is the next repeating day
func getNextTime(trip *TripSchedule, ts int64) int64 {
	if wasOriginalAlertSent(trip) && IsRepeating(trip) {
		localArrival := getLocalTime(
			trip.InputArrivalTime.String,
			trip.InputArrivalTime.TimezoneLocation,
			ts,
		)
		return getNextRepeatTime(trip.LastNotificationSent,
			localArrival,
			trip.RepeatDays)
	}
	return ts
}

func wasOriginalAlertSent(trip *TripSchedule) bool {
	// if LastNotificationSent is greater than zero then we've sent
	// the original response
	return trip.LastNotificationSent > 0
}

// getLocalTime will return a timestamp in the timezone as the date string
func getLocalTime(dateString string, timezoneName string, ts int64) time.Time {
	layout := "2006-01-02T15:04:05-07:00 MST"
	t, err := time.Parse(layout, dateString)
	// if there's an error then just use the default timezone
	if err != nil {
		return time.Unix(ts/1000, 0)
	}
	loc, err := time.LoadLocation(timezoneName)
	if err != nil {
		loc = t.Location()
	}
	return time.Unix(ts/1000, 0).In(loc)
}

// getNextRepeatTime will return the departure time as a unix timestamp
// based on when the last notification was sent
// @param lastNotification - when the last notification for this trip was sent
// as a timestamp in milliseconds
// @param departureTime - the departure time of the trip
// @param repeatDays - the days which this trip will repeat on where Monday is
// zero index
// @returns a timestamp with same time of day as departureTime but the day
// is the next repeating day
func getNextRepeatTime(lastNotification int64, departureTime time.Time, repeatDays []bool) int64 {
	// get the current time in the trip's timezone
	now := time.Now().In(departureTime.Location())
	// convert to correct timezone
	lastNotificationDate := time.Unix(lastNotification/1000, 0).In(departureTime.Location())
	return getNextRepeatTimeFromDate(now, lastNotificationDate, departureTime, repeatDays)
}

// getNextRepeatTimeFromDate will return the departure time as a unix timestamp
// based on the current time and when the last notification was sent
// @param now - the current time
// @param lastNotification - when the last notification for this trip was sent
// @param departureTime - the departure time of the trip
// @param repeatDays - the days which this trip will repeat on where Monday is
// zero index
// @returns a timestamp with same time of day as departureTime but the day
// is the next repeating day
func getNextRepeatTimeFromDate(now time.Time, lastNotification time.Time,
	departureTime time.Time, repeatDays []bool) int64 {
	prevDay := mondayAsZeroIndex(int(lastNotification.Weekday()), len(repeatDays))
	// find the next repeat
	nextDay := getNextDay(prevDay, repeatDays)
	currentDay := mondayAsZeroIndex(int(now.Weekday()), len(repeatDays))
	// find out how many days until the next repeat
	daysTillNextDay := daysTillNextDay(currentDay, nextDay, len(repeatDays))
	next := now.AddDate(0, 0, daysTillNextDay)
	// update departure time with this info
	dateWithMatchingTime := time.Date(
		next.Year(), next.Month(), next.Day(), departureTime.Hour(),
		departureTime.Minute(), departureTime.Second(),
		departureTime.Nanosecond(), departureTime.Location(),
	)
	return dateWithMatchingTime.Unix() * 1000
}

func mondayAsZeroIndex(day int, daysInAWeek int) int {
	// Monday is our zero index, so we handle this situation by subtracting one
	// and handling -1 case
	convertedDay := day - 1
	if convertedDay == -1 {
		convertedDay = daysInAWeek - 1
	}
	return convertedDay
}

func daysTillNextDay(day int, nextDay int, numberOfDays int) int {
	if nextDay < day {
		return nextDay + numberOfDays - day
	}
	return nextDay - day
}

func getNextDay(day int, days []bool) int {
	for i := (day + 1) % len(days); i != day; i = (i + 1) % len(days) {
		if days[i] {
			return i
		}
	}
	return day
}
