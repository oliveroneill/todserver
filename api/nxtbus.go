package api

import (
	"github.com/oliveroneill/nxtbus-go"
	"math"
	"time"
)

// NxtBusThresholdMs is used so that NXTBUS will track 90 minutes ahead
const NxtBusThresholdMs = 90 * 60 * 1000

// TransportCanberraName is the name stored in NXTBUS API response
const TransportCanberraName = "Transport Canberra"

// StopTimeThreshold is a threshold to determine whether a stop date from
// Google Maps is close enough to the NXTBUS date to be the same route
const StopTimeThresholdMs = 2 * 60 * 1000

// NxtBusFinder - an implementation of RouteFinder that uses GoogleMaps
// and NXTBUS for accurate departure times in Canberra
type NxtBusFinder struct {
	RouteFinder
	nxtBusAPI RealTimeAPI
	// this should always be a GoogleMapsFinder since we will
	// use the information that it provides, but for testing
	// purposes its easiest to use the interface
	finder RouteFinder
}

// NewNxtBusFinder - create a NxtBusFinder with api key
func NewNxtBusFinder(apiKey string, mapsFinder *GoogleMapsFinder) *NxtBusFinder {
	finder := new(NxtBusFinder)
	finder.nxtBusAPI = NewNxtBusAPI(apiKey)
	finder.finder = mapsFinder
	return finder
}

// RealTimeAPI is an interface for finding routes at a specific transit stop
// Currently this is only used for NXTBUS, and will return nxtbus API
// types
type RealTimeAPI interface {
	GetVisits(stopName string) ([]nxtbus.MonitoredStopVisit, error)
}

// NxtBusAPI is an implementation of RealTimeAPI using NXTBUS
type NxtBusAPI struct {
	RealTimeAPI
	apiKey string
}

// NewNxtBusAPI will create a new NxtBusAPI with the specified NXTBUS
// API key
func NewNxtBusAPI(apiKey string) *NxtBusAPI {
	i := new(NxtBusAPI)
	i.apiKey = apiKey
	return i
}

// GetVisits will return all routes going through the specified stop name
func (api *NxtBusAPI) GetVisits(stopName string) ([]nxtbus.MonitoredStopVisit, error) {
	id, err := nxtbus.StopNameToID(stopName)
	if err != nil {
		return nil, err
	}
	resp, err := nxtbus.MakeStopMonitoringRequest(api.apiKey, id)
	if err != nil {
		return nil, err
	}
	if resp.StopMonitoringDelivery == nil {
		return []nxtbus.MonitoredStopVisit{}, nil
	}
	if resp.StopMonitoringDelivery.MonitoredStopVisits == nil {
		return []nxtbus.MonitoredStopVisit{}, nil
	}
	return resp.StopMonitoringDelivery.MonitoredStopVisits, nil
}

// FindRoutes will return real-time transit data and fallback to standard
// Google Maps data when this data is unavailable or irrelevant
func (finder *NxtBusFinder) FindRoutes(originLat, originLng, destLat,
	destLng float64, transportType string, arrivalTime int64,
	routeName string) []RouteOption {
	options := finder.finder.FindRoutes(originLat, originLng, destLat, destLng, transportType, arrivalTime, routeName)
	if transportType != "transit" {
		return options
	}
	for i, option := range options {
		now := time.Now().UnixNano() / 1e6
		// if its more than 90 minutes then skip
		if option.DepartureTime-now >= NxtBusThresholdMs {
			continue
		}
		if option.transitDetails == nil {
			continue
		}
		details := option.transitDetails
		// if the transitDetails indicate not Transport Canberra then skip
		if details.Line.Agencies[0].Name != TransportCanberraName {
			continue
		}
		finder.updateUsingRealTimeData(&options[i])
	}
	return options
}

// NOTE: This will modify the option passed in without copying
func (finder *NxtBusFinder) updateUsingRealTimeData(option *RouteOption) {
	visits, err := finder.nxtBusAPI.GetVisits(option.transitDetails.DepartureStop.Name)
	if err != nil {
		return
	}
	// use the transit details departure time, since there may be other legs
	// on the trip
	mapsDeparture := int64(option.transitDetails.DepartureTime.UnixNano() / 1e6)
	var closest float64 = -1
	var bestChoice *nxtbus.MonitoredStopVisit
	// find MonitoredStopVisit with closest scheduled departure to option's departure time
	for i, data := range visits {
		if data.LineName != option.Name {
			continue
		}
		date, err := nxtbus.ParseDate(data.AimedDepartureTime)
		// some responses seem to be missing data
		if err != nil {
			continue
		}
		aimedDeparture := int64(date.UnixNano() / 1e6)
		diff := math.Abs(float64(mapsDeparture - aimedDeparture))
		// make sure the times aren't too far apart
		if diff > StopTimeThresholdMs {
			continue
		}
		if bestChoice == nil || diff < closest {
			closest = diff
			bestChoice = &visits[i]
		}
	}
	if bestChoice == nil {
		return
	}
	expectedDeparture, err := nxtbus.ParseDate(bestChoice.ExpectedDepartureTime)
	// ensure we aren't missing data
	if err != nil {
		return
	}
	// Figure out how much time we've gained or lost compared to the schedule.
	// We use the Google Maps departure time to compensate if it has
	// an incorrect transit departure time
	diff := mapsDeparture - (expectedDeparture.UnixNano() / 1e6)
	// move the trip start and end accordingly
	option.DepartureTime -= diff
	option.ArrivalTime -= diff
}
