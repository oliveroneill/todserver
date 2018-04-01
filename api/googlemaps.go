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

// FindRoutes will use Google Maps Directions API to search for routes based on
// input
// @param originLat - the starting position latitude
// @param originLng - the starting position longitude
// @param destLat - the destination latitude
// @param destLng - the destination longitude
// @param transportType - transit, driving, walking etc.
// @param arrivalTime - the time of arrival to the destination
// @param routeName - optionally specify the description. This could be the bus
//        number for example
func (finder *GoogleMapsFinder) FindRoutes(originLat, originLng, destLat, destLng float64,
	transportType string, arrivalTime time.Time,
	routeName string) []RouteOption {
	routes := getRoutes(finder.apiKey, originLat, originLng, destLat, destLng,
		transportType, arrivalTime)
	options := []RouteOption{}
	for _, route := range routes {
		depart := getDepartureTime(route, arrivalTime)
		arrive := getArrivalTime(route, arrivalTime)
		desc := getDescription(route)
		details := getTransitDetails(route)
		if len(routeName) > 0 {
			if getRouteName(route) == routeName {
				option := NewRouteOption(depart, arrive, routeName, desc)
				option.transitDetails = details
				options = append(options, option)
			}
		} else {
			option := NewRouteOption(depart, arrive, getRouteName(route), desc)
			option.transitDetails = details
			options = append(options, option)
		}
	}
	return options
}

func getRoutes(apiKey string, originLat float64, originLng float64, destLat float64,
	destLng float64, transportType string, arrivalTime time.Time) []maps.Route {
	c, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		fmt.Println("fatal error:", err)
	}
	r := &maps.DirectionsRequest{
		Alternatives: true,
		Origin:       fmt.Sprintf("%f, %f", originLat, originLng),
		Destination:  fmt.Sprintf("%f, %f", destLat, destLng),
		Mode:         maps.Mode(transportType),
		ArrivalTime:  fmt.Sprintf("%d", arrivalTime.UnixNano()/1e9),
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

func getArrivalTime(route maps.Route, arrivalTime time.Time) time.Time {
	lastLeg := route.Legs[len(route.Legs)-1]
	arrive := lastLeg.ArrivalTime
	var zero time.Time
	if arrive.Equal(zero) {
		arrive = arrivalTime
	}
	return arrive
}

func getDepartureTime(route maps.Route, arrivalTime time.Time) time.Time {
	firstLeg := route.Legs[0]
	depart := firstLeg.DepartureTime
	var zero time.Time
	if depart.Equal(zero) {
		depart = arrivalTime.Add(-firstLeg.Duration)
	}
	return depart
}

func getTransitDetails(route maps.Route) *maps.TransitDetails {
	for _, leg := range route.Legs {
		for _, step := range leg.Steps {
			if step.TravelMode == "TRANSIT" {
				return step.TransitDetails
			}
		}
	}
	return nil
}

func getDescription(route maps.Route) string {
	return route.Summary
}
