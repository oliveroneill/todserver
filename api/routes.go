package api

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
