package api

import (
	"errors"
	"github.com/oliveroneill/nxtbus-go"
	"googlemaps.github.io/maps"
	"reflect"
	"testing"
	"time"
)

type MockMapsFinder struct {
	GoogleMapsFinder
	options []RouteOption
}

func NewMockMapsFinder(options []RouteOption) *MockMapsFinder {
	finder := new(MockMapsFinder)
	finder.options = options
	return finder
}

func (finder *MockMapsFinder) FindRoutes(originLat, originLng, destLat,
	destLng float64, transportType string, arrivalTime time.Time,
	routeName string) []RouteOption {
	return finder.options
}

type MockNxtBusFinder struct {
	visits []nxtbus.MonitoredStopVisit
}

func NewMockNxtBusFinder(visits []nxtbus.MonitoredStopVisit) *MockNxtBusFinder {
	finder := new(MockNxtBusFinder)
	finder.visits = visits
	return finder
}

func (f *MockNxtBusFinder) GetVisits(stopName string) ([]nxtbus.MonitoredStopVisit, error) {
	if f.visits == nil {
		return nil, errors.New("No data")
	}
	return f.visits, nil
}

func generateValidTransitDetails(departureTime time.Time) *maps.TransitDetails {
	return &maps.TransitDetails{
		Line: maps.TransitLine{
			Agencies: []*maps.TransitAgency{
				&maps.TransitAgency{
					Name: "Transport Canberra",
				},
			},
		},
		DepartureTime: departureTime,
	}
}

func generateInvalidTransitDetails(departureTime time.Time) *maps.TransitDetails {
	return &maps.TransitDetails{
		Line: maps.TransitLine{
			Agencies: []*maps.TransitAgency{
				&maps.TransitAgency{
					Name: "Different Bus Company",
				},
			},
		},
		DepartureTime: departureTime,
	}
}

func dateToNxtbusString(date time.Time) string {
	return date.Format("2006-01-02T15:04:05-07:00")
}

func generateStopVisitInfo(
	now time.Time, name string, scheduledDeparture, realTimeDeparture time.Time,
) []nxtbus.MonitoredStopVisit {
	// Real time data
	realTimeRoute := nxtbus.MonitoredStopVisit{
		LineName:              name,
		AimedDepartureTime:    dateToNxtbusString(scheduledDeparture),
		ExpectedDepartureTime: dateToNxtbusString(realTimeDeparture),
	}
	visits := []nxtbus.MonitoredStopVisit{
		nxtbus.MonitoredStopVisit{
			LineName:              name,
			AimedDepartureTime:    dateToNxtbusString(now.Add(9 * time.Millisecond)),
			ExpectedDepartureTime: dateToNxtbusString(realTimeDeparture.Add(5 * time.Millisecond)),
		},
		realTimeRoute,
		nxtbus.MonitoredStopVisit{
			LineName:              "Different name",
			AimedDepartureTime:    dateToNxtbusString(now.Add(2 * time.Millisecond)),
			ExpectedDepartureTime: dateToNxtbusString(realTimeDeparture.Add(-5 * time.Millisecond)),
		},
	}
	return visits
}

func TestFindRoutesRealTimeThreshold(t *testing.T) {
	now := time.Now()
	arrival := now.Add(100 * time.Minute)
	departure := now.Add(100 * time.Minute)
	route := RouteOption{
		DepartureTime:  departure,
		ArrivalTime:    arrival,
		Name:           "",
		Description:    "",
		transitDetails: generateValidTransitDetails(departure),
	}
	options := []RouteOption{route}
	realTimeDeparture := now.Add(2 * time.Minute)
	visits := generateStopVisitInfo(
		now, "", departure, realTimeDeparture,
	)
	finder := new(NxtBusFinder)
	finder.nxtBusAPI = NewMockNxtBusFinder(visits)
	// make a copy of the options since real time finder will modify
	// without copying
	tmp := make([]RouteOption, len(options))
	copy(tmp, options)
	finder.finder = NewMockMapsFinder(tmp)
	routes := finder.FindRoutes(1, 1, 1, 1, "transit", now, "")
	if !reflect.DeepEqual(routes[0], route) {
		t.Error("Expected", route, "found", routes[0])
	}
}

func TestFindRoutesUsesRealTimeData(t *testing.T) {
	name := "729"
	now := time.Now()
	scheduledArrival := now.Add(11 * time.Minute)
	departure := now.Add(10 * time.Minute)
	details := generateValidTransitDetails(departure)
	route := RouteOption{
		DepartureTime:  departure,
		ArrivalTime:    scheduledArrival,
		Name:           name,
		Description:    "",
		transitDetails: details,
	}
	options := []RouteOption{route}
	realTimeDeparture := now.Add(2 * time.Minute)
	visits := generateStopVisitInfo(
		now, name, departure, realTimeDeparture,
	)
	finder := new(NxtBusFinder)
	finder.nxtBusAPI = NewMockNxtBusFinder(visits)
	// Expected route option after real time update
	arrival := scheduledArrival.Add(-(departure.Sub(realTimeDeparture)))
	expected := RouteOption{
		DepartureTime:  realTimeDeparture.Truncate(time.Second),
		ArrivalTime:    arrival.Truncate(time.Second),
		Name:           name,
		Description:    "",
		transitDetails: details,
	}
	// make a copy of the options since real time finder will modify
	// without copying
	tmp := make([]RouteOption, len(options))
	copy(tmp, options)
	finder.finder = NewMockMapsFinder(tmp)
	routes := finder.FindRoutes(1, 1, 1, 1, "transit", now, "")
	if len(routes) != 1 {
		t.Error("Expected length", 1, "found", len(routes))
	}
	result := routes[0]
	if reflect.DeepEqual(expected, result) {
		t.Error("Expected", expected, "found", result)
	}
}

func TestFindRoutesFallsBackWhenMissingStopInfo(t *testing.T) {
	name := "729"
	now := time.Now()
	scheduledArrival := now.Add(11 * time.Minute)
	departure := now.Add(10 * time.Minute)
	route := RouteOption{
		DepartureTime:  departure,
		ArrivalTime:    scheduledArrival,
		Name:           name,
		Description:    "",
		transitDetails: generateValidTransitDetails(departure),
	}
	options := []RouteOption{route}
	finder := new(NxtBusFinder)
	// the stop info will be nil
	finder.nxtBusAPI = NewMockNxtBusFinder(nil)
	// make a copy of the options since real time finder will modify
	// without copying
	tmp := make([]RouteOption, len(options))
	copy(tmp, options)
	finder.finder = NewMockMapsFinder(tmp)
	routes := finder.FindRoutes(1, 1, 1, 1, "transit", now, "")
	if !reflect.DeepEqual(routes[0], route) {
		t.Error("Expected", route, "found", routes[0])
	}
}

func TestFindRoutesFallsBackWhenMissingExpectedDates(t *testing.T) {
	name := "729"
	now := time.Now()
	scheduledArrival := now.Add(11 * time.Minute)
	departure := now.Add(10 * time.Minute)
	route := RouteOption{
		DepartureTime:  departure,
		ArrivalTime:    scheduledArrival,
		Name:           name,
		Description:    "",
		transitDetails: generateValidTransitDetails(departure),
	}
	options := []RouteOption{route}
	realTimeDeparture := now.Add(2 * time.Minute)
	visits := generateStopVisitInfo(
		now, name, departure, realTimeDeparture,
	)
	// remove relevant info
	for i := range visits {
		visits[i].ExpectedDepartureTime = ""
		visits[i].ExpectedArrivalTime = ""
	}
	finder := new(NxtBusFinder)
	finder.nxtBusAPI = NewMockNxtBusFinder(visits)
	// make a copy of the options since real time finder will modify
	// without copying
	tmp := make([]RouteOption, len(options))
	copy(tmp, options)
	finder.finder = NewMockMapsFinder(tmp)
	routes := finder.FindRoutes(1, 1, 1, 1, "transit", now, "")
	if !reflect.DeepEqual(routes[0], route) {
		t.Error("Expected", route, "found", routes[0])
	}
}

func TestFindRoutesFallsBackWhenDifferentBusCompany(t *testing.T) {
	name := "729"
	now := time.Now()
	scheduledArrival := now.Add(11 * time.Minute)
	departure := now.Add(10 * time.Minute)
	route := RouteOption{
		DepartureTime: departure,
		ArrivalTime:   scheduledArrival,
		Name:          name,
		Description:   "",
		// invalid details
		transitDetails: generateInvalidTransitDetails(departure),
	}
	options := []RouteOption{route}
	realTimeDeparture := now.Add(2 * time.Minute)
	visits := generateStopVisitInfo(
		now, name, departure, realTimeDeparture,
	)
	finder := new(NxtBusFinder)
	finder.nxtBusAPI = NewMockNxtBusFinder(visits)
	// make a copy of the options since real time finder will modify
	// without copying
	tmp := make([]RouteOption, len(options))
	copy(tmp, options)
	finder.finder = NewMockMapsFinder(tmp)
	routes := finder.FindRoutes(1, 1, 1, 1, "transit", now, "")
	if !reflect.DeepEqual(routes[0], route) {
		t.Error("Expected", route, "found", routes[0])
	}
}

func TestFindRoutesFallsBackWhenNotTransit(t *testing.T) {
	name := "729"
	now := time.Now()
	scheduledArrival := now.Add(11 * time.Minute)
	departure := now.Add(10 * time.Minute)
	route := RouteOption{
		DepartureTime:  departure,
		ArrivalTime:    scheduledArrival,
		Name:           name,
		Description:    "",
		transitDetails: generateValidTransitDetails(departure),
	}
	options := []RouteOption{route}
	realTimeDeparture := now.Add(2 * time.Minute)
	visits := generateStopVisitInfo(
		now, name, departure, realTimeDeparture,
	)
	finder := new(NxtBusFinder)
	finder.nxtBusAPI = NewMockNxtBusFinder(visits)
	// make a copy of the options since real time finder will modify
	// without copying
	tmp := make([]RouteOption, len(options))
	copy(tmp, options)
	finder.finder = NewMockMapsFinder(tmp)
	// set transport mode to driving
	routes := finder.FindRoutes(1, 1, 1, 1, "driving", now, "")
	if !reflect.DeepEqual(routes[0], route) {
		t.Error("Expected", route, "found", routes[0])
	}
}

func TestFindRoutesFallsBackWhenOutOfThreshold(t *testing.T) {
	name := "729"
	now := time.Now()
	scheduledArrival := now.Add(11 * time.Minute)
	departure := now.Add(10 * time.Minute)
	route := RouteOption{
		DepartureTime:  departure,
		ArrivalTime:    scheduledArrival,
		Name:           name,
		Description:    "",
		transitDetails: generateValidTransitDetails(departure),
	}
	options := []RouteOption{route}
	realTimeDeparture := now.Add(2 * time.Minute)
	// the aimed time is 10 minutes later and therefore should not
	// match up with the Google Maps data
	aimedDeparture := departure.Add(10 * time.Minute)
	visits := generateStopVisitInfo(
		now, name, aimedDeparture, realTimeDeparture,
	)
	finder := new(NxtBusFinder)
	finder.nxtBusAPI = NewMockNxtBusFinder(visits)
	// make a copy of the options since real time finder will modify
	// without copying
	tmp := make([]RouteOption, len(options))
	copy(tmp, options)
	finder.finder = NewMockMapsFinder(tmp)
	// set transport mode to driving
	routes := finder.FindRoutes(1, 1, 1, 1, "transit", now, "")
	if !reflect.DeepEqual(routes[0], route) {
		t.Error("Expected", route, "found", routes[0])
	}
}
