package api

import (
	"errors"
	"math"
	"time"
)

type UserInfo struct {
	ID                string `json:"user_id"`
	NotificationToken string `json:"notification_token"`
	DeviceOS          string `json:"device_os"`
}

type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Date struct {
	String    string `json:"local_date_string"`
	Timestamp int64  `json:"timestamp"`
}

type TripSchedule struct {
	ID               string       `json:"trip_id"`
	User             *UserInfo    `json:"user"`
	Origin           Point        `json:"origin"`
	Destination      Point        `json:"destination"`
	Route            *RouteOption `json:"route"`
	InputArrivalTime *Date        `json:"input_arrival_time"`
	WaitingWindowMs  int64        `json:"waiting_window_ms"`
	TransportType    string       `json:"transport_type"`
	RepeatDays       []bool       `json:"repeat_days"`
	Enabled          bool         `json:"enabled"`
	// timestamp the last notification for this trip was sent
	LastNotificationSent int64 `json:"last_notification"`
}

func UpsertUser(user *UserInfo) error {
	db := NewPostgresInterface()
	defer db.Close()
	return db.UpsertUser(user)
}

func GetScheduledTrips(user string) ([]TripSchedule, error) {
	db := NewPostgresInterface()
	defer db.Close()
	return db.GetTrips(user)
}

func GetAllScheduledTrips() ([]*TripSchedule, error) {
	db := NewPostgresInterface()
	defer db.Close()
	return db.GetAllScheduledTrips()
}

func EnableDisableTrip(id string, userID string) error {
	// toggle trip
	db := NewPostgresInterface()
	defer db.Close()
	return db.EnableDisableTrip(id, userID)
}

func DeleteTrip(id string, userID string) error {
	db := NewPostgresInterface()
	defer db.Close()
	return db.DeleteTrip(id, userID)
}

func SetLastNotificationTime(trip *TripSchedule, timestamp int64) error {
	db := NewPostgresInterface()
	defer db.Close()
	return db.SetLastNotificationTime(trip, timestamp)
}

func IsEnabled(trip *TripSchedule) bool {
	db := NewPostgresInterface()
	defer db.Close()
	return db.IsEnabled(trip)
}

func ScheduleTrip(trip *TripSchedule) error {
	// store trip
	db := NewPostgresInterface()
	defer db.Close()
	err := db.ScheduleTrip(trip)
	return err
}

func GetRoute(finder RouteFinder, trip *TripSchedule) (RouteOption, error) {
	resp := finder.FindRoutes(trip.Origin.Lat, trip.Origin.Lng,
		trip.Destination.Lat, trip.Destination.Lng,
		trip.TransportType, GetInputArrivalTime(trip),
		trip.Route.Name)
	return getRouteFromDescription(trip, resp)
}

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

func IsRepeating(trip *TripSchedule) bool {
	for _, b := range trip.RepeatDays {
		if b {
			return true
		}
	}
	return false
}

func GetDepartureTime(trip *TripSchedule) int64 {
	return getNextTime(trip, trip.Route.DepartureTime)
}

func GetInputArrivalTime(trip *TripSchedule) int64 {
	return getNextTime(trip, trip.InputArrivalTime.Timestamp)
}

func GetArrivalTime(trip *TripSchedule) int64 {
	return getNextTime(trip, trip.Route.ArrivalTime)
}

// getNextTime returns the next time that this timestamp will occur
// in relation to the trip's repeating days and the last notification
// send time
func getNextTime(trip *TripSchedule, ts int64) int64 {
	if wasOriginalAlertSent(trip) && IsRepeating(trip) {
		localArrival := getLocalTime(trip.InputArrivalTime.String, ts)
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
func getLocalTime(dateString string, ts int64) time.Time {
	layout := "2006-01-02T15:04:05-07:00 MST"
	t, err := time.Parse(layout, dateString)
	// if there's an error then just use the default timezone
	if err != nil {
		return time.Unix(ts/1000, 0)
	}
	return time.Unix(ts/1000, 0).In(t.Location())
}

func getNextRepeatTime(lastNotification int64, departureTime time.Time, repeatDays []bool) int64 {
	// get the current time in the trip's timezone
	now := time.Now().In(departureTime.Location())
	lastNotificationDate := time.Unix(lastNotification/1000, 0).In(departureTime.Location())
	return getNextRepeatTimeFromDate(now, lastNotificationDate, departureTime, repeatDays)
}

func getNextRepeatTimeFromDate(now time.Time, lastNotification time.Time,
	departureTime time.Time, repeatDays []bool) int64 {
	prevDay := mondayAsZeroIndex(int(lastNotification.Weekday()), len(repeatDays))
	nextDay := getNextDay(prevDay, repeatDays)
	currentDay := mondayAsZeroIndex(int(now.Weekday()), len(repeatDays))
	daysTillNextDay := daysTillNextDay(currentDay, nextDay, len(repeatDays))
	next := now.AddDate(0, 0, daysTillNextDay)
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
