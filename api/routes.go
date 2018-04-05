package api

import (
	"fmt"
	"googlemaps.github.io/maps"
	"strconv"
	"strings"
	"time"
)

type UnixTime struct {
	time.Time
}

func (t *UnixTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	i, err := strconv.ParseInt(s, 10, 64)
	t.Time = time.Unix(0, i*1e6)
	return err
}

func (t *UnixTime) MarshalJSON() ([]byte, error) {
	ts := t.UnixNano() / 1e6
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
