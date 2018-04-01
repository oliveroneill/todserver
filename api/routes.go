package api

import (
	"googlemaps.github.io/maps"
	"time"
)

// RouteOption is a search result found through RouteFinder
// This is information useful for the user to determine their trip
type RouteOption struct {
	DepartureTime time.Time `json:"departure_time"`
	ArrivalTime   time.Time `json:"arrival_time"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
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
		DepartureTime: departureTime,
		ArrivalTime:   arrivalTime,
		Name:          routeName,
		Description:   description,
	}
}
