package api

// RouteOption is a search result found through RouteFinder
// This is information useful for the user to determine their trip
type RouteOption struct {
	DepartureTime int64  `json:"departure_time"`
	ArrivalTime   int64  `json:"arrival_time"`
	Name          string `json:"name"`
	Description   string `json:"description"`
}

// RouteFinder - a generic interface for finding routes
type RouteFinder interface {
	FindRoutes(originLat, originLng, destLat, destLng float64,
		transportType string, arrivalTime int64,
		routeName string) []RouteOption
}

// NewRouteOption will create a new RouteOption object using the input data
// @param departureTime - the time that a user will leave
// @param arrivalTime - time the user will arrive
// @param routeName - the name of the route
// @param description - optional additional data about the route. If this is
// is an empty string, the description will also be set as the route name
func NewRouteOption(departureTime int64, arrivalTime int64, routeName string,
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
