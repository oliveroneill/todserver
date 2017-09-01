package api

import (
	"fmt"
	"golang.org/x/net/context"
	"googlemaps.github.io/maps"
	"time"
)

// GoogleMapsFinder - an implementation of RouteFinder that searches GoogleMaps
// for options
type GoogleMapsFinder struct {
	RouteFinder
	apiKey string
}

// NewGoogleMapsFinder - create a GoogleMapsFinder with api key
func NewGoogleMapsFinder(apiKey string) *GoogleMapsFinder {
	finder := new(GoogleMapsFinder)
	finder.apiKey = apiKey
	return finder
}

func (finder *GoogleMapsFinder) FindRoutes(originLat, originLng, destLat, destLng float64,
	transportType string, arrivalTime int64,
	routeName string) []RouteOption {
	routes := getRoutes(finder.apiKey, originLat, originLng, destLat, destLng,
		transportType, arrivalTime)
	options := []RouteOption{}
	for _, route := range routes {
		depart := getDepartureTime(route, arrivalTime)
		arrive := getArrivalTime(route, arrivalTime)
		desc := getDescription(route)
		if len(routeName) > 0 {
			if getRouteName(route) == routeName {
				options = append(options, NewRouteOption(depart, arrive, routeName, desc))
			}
		} else {
			options = append(options, NewRouteOption(depart, arrive, getRouteName(route), desc))
		}
	}
	return options
}

func getRoutes(apiKey string, originLat float64, originLng float64, destLat float64,
	destLng float64, transportType string, arrivalTime int64) []maps.Route {
	c, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		fmt.Println("fatal error:", err)
	}
	r := &maps.DirectionsRequest{
		Alternatives: true,
		Origin:       fmt.Sprintf("%f, %f", originLat, originLng),
		Destination:  fmt.Sprintf("%f, %f", destLat, destLng),
		Mode:         maps.Mode(transportType),
		ArrivalTime:  fmt.Sprintf("%d", arrivalTime/1000),
	}
	resp, _, err := c.Directions(context.Background(), r)
	if err != nil {
		fmt.Println("fatal error:", err)
	}
	return resp
}

func getRouteName(route maps.Route) string {
	for _, leg := range route.Legs {
		for _, step := range leg.Steps {
			if step.TravelMode == "TRANSIT" {
				return step.TransitDetails.Line.ShortName
			}
		}
	}
	return "Unknown"
}

func getArrivalTime(route maps.Route, arrivalTime int64) int64 {
	lastLeg := route.Legs[len(route.Legs)-1]
	arrive := lastLeg.ArrivalTime.UnixNano() / 1e6
	if arrive < 0 {
		arrive = arrivalTime
	}
	return arrive
}

func getDepartureTime(route maps.Route, arrivalTime int64) int64 {
	firstLeg := route.Legs[0]
	depart := firstLeg.DepartureTime.UnixNano() / 1e6
	if depart < 0 {
		durationMs := int64(firstLeg.Duration / time.Second)
		depart = arrivalTime - durationMs
	}
	return depart
}

func getDescription(route maps.Route) string {
	return route.Summary
}
