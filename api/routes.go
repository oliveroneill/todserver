package api

import (
	"fmt"
	"googlemaps.github.io/maps"
	"strconv"
	"strings"
	"time"
)

// NanosecondsInAMillisecond is used for conversion
const NanosecondsInAMillisecond = 1e6

// UnixTime is a wrapper around time.Time for the purpose of
// encoding and decoding JSON into unix timestamp in milliseconds
type UnixTime struct {
	time.Time
}

// UnixTimestampToTime converts a unix timestamp in milliseconds to a
// time.Time value
func UnixTimestampToTime(ts int64) time.Time {
	return time.Unix(0, ts*NanosecondsInAMillisecond)
}

// TimeToUnixTimestamp converts a UnixTime value to a unix timestamp
// in milliseconds. UnixTime is used as input simply because it was
// needed more often than generic time
func TimeToUnixTimestamp(t UnixTime) int64 {
	return t.UnixNano() / NanosecondsInAMillisecond
}

// UnmarshalJSON will convert unix timestamp string to time.Time
func (t *UnixTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	i, err := strconv.ParseInt(s, 10, 64)
	t.Time = UnixTimestampToTime(i)
	return err
}

// MarshalJSON will convert time.Time to unix timestamp string
func (t *UnixTime) MarshalJSON() ([]byte, error) {
	ts := TimeToUnixTimestamp(*t)
	return []byte(fmt.Sprintf("%d", ts)), nil
}

// RouteOption is a search result found through RouteFinder
// This is information useful for the user to determine their trip
type RouteOption struct {
	DepartureTime UnixTime `json:"departure_time"`
	ArrivalTime   UnixTime `json:"arrival_time"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	// optional transit information
	// This will only be set by GoogleMapsFinder
	transitDetails *maps.TransitDetails
}

// RouteFinder - a generic interface for finding routes
type RouteFinder interface {
	FindRoutes(originLat, originLng, destLat, destLng float64,
		transportType string, arrivalTime time.Time,
		routeName string) []RouteOption
}

// NewRouteOption will create a new RouteOption object using the input data
// @param departureTime - the time that a user will leave
// @param arrivalTime - time the user will arrive
// @param routeName - the name of the route
// @param description - optional additional data about the route. If this is
// is an empty string, the description will also be set as the route name
func NewRouteOption(departureTime time.Time, arrivalTime time.Time, routeName string,
	description string) RouteOption {
	if description == "" {
		description = routeName
	}
	return RouteOption{
		DepartureTime: UnixTime{departureTime},
		ArrivalTime:   UnixTime{arrivalTime},
		Name:          routeName,
		Description:   description,
	}
}
