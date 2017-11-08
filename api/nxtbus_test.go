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
	destLng float64, transportType string, arrivalTime int64,
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

func generateValidTransitDetails() *maps.Route {
	return &maps.Route{
		Legs: []*maps.Leg{
			&maps.Leg{
				Steps: []*maps.Step{
					&maps.Step{
						TravelMode: "TRANSIT",
						TransitDetails: &maps.TransitDetails{
							Line: maps.TransitLine{
								Agencies: []*maps.TransitAgency{
									&maps.TransitAgency{
										Name: "Transport Canberra",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func generateInvalidTransitDetails() *maps.Route {
	return &maps.Route{
		Legs: []*maps.Leg{
			&maps.Leg{
				Steps: []*maps.Step{
					&maps.Step{
						TravelMode: "TRANSIT",
						TransitDetails: &maps.TransitDetails{
							Line: maps.TransitLine{
								Agencies: []*maps.TransitAgency{
									&maps.TransitAgency{
										Name: "Different Bus Company",
									},
								},
							},
						},
					},
				},
			},
		},
	}

}

func convertUnixTimestampToNxtBusDate(ms int64) string {
	date := time.Unix(0, ms*1e6).In(time.UTC)
	return date.Format("2006-01-02T15:04:05.000000")
}

func generateStopVisitInfo(
	now int64, name string, scheduledArrival, realTimeArrival, realTimeDeparture int64,
) []nxtbus.MonitoredStopVisit {
	// Real time data
	realTimeRoute := nxtbus.MonitoredStopVisit{
		LineName:              name,
		AimedArrivalTime:      convertUnixTimestampToNxtBusDate(scheduledArrival),
		ExpectedDepartureTime: convertUnixTimestampToNxtBusDate(realTimeDeparture),
		ExpectedArrivalTime:   convertUnixTimestampToNxtBusDate(realTimeArrival),
	}
	visits := []nxtbus.MonitoredStopVisit{
		nxtbus.MonitoredStopVisit{
			LineName:              name,
			AimedArrivalTime:      convertUnixTimestampToNxtBusDate(now + 90),
			ExpectedDepartureTime: convertUnixTimestampToNxtBusDate(realTimeDeparture + 5),
			ExpectedArrivalTime:   convertUnixTimestampToNxtBusDate(realTimeArrival - 10),
		},
		realTimeRoute,
		nxtbus.MonitoredStopVisit{
			LineName:              "Different name",
			AimedArrivalTime:      convertUnixTimestampToNxtBusDate(now + 29),
			ExpectedDepartureTime: convertUnixTimestampToNxtBusDate(realTimeDeparture - 5),
			ExpectedArrivalTime:   convertUnixTimestampToNxtBusDate(realTimeArrival + 10),
		},
	}
	return visits
}

func TestFindRoutesRealTimeThreshold(t *testing.T) {
	now := time.Now().Unix() * 1000
	arrival := now + 100*60*1000
	route := RouteOption{
		DepartureTime: now + 100*60*1000,
		ArrivalTime:   arrival,
		Name:          "",
		Description:   "",
		MapsDetails:   generateValidTransitDetails(),
	}
	options := []RouteOption{route}
	realTimeArrival := now + 4*60*1000
	realTimeDeparture := now + 2*60*1000
	visits := generateStopVisitInfo(
		now, "", arrival, realTimeArrival, realTimeDeparture,
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
	now := time.Now().Unix() * 1000
	scheduledArrival := now + 11*60*1000
	route := RouteOption{
		DepartureTime: now + 10*60*1000,
		ArrivalTime:   scheduledArrival,
		Name:          name,
		Description:   "",
		MapsDetails:   generateValidTransitDetails(),
	}
	options := []RouteOption{route}
	realTimeArrival := now + 4*60*1000
	realTimeDeparture := now + 2*60*1000
	visits := generateStopVisitInfo(
		now, name, scheduledArrival, realTimeArrival, realTimeDeparture,
	)
	finder := new(NxtBusFinder)
	finder.nxtBusAPI = NewMockNxtBusFinder(visits)
	// Expected route option after real time update
	expected := RouteOption{
		DepartureTime: realTimeDeparture,
		ArrivalTime:   realTimeArrival,
		Name:          name,
		Description:   "",
		MapsDetails:   generateValidTransitDetails(),
	}
	// make a copy of the options since real time finder will modify
	// without copying
	tmp := make([]RouteOption, len(options))
	copy(tmp, options)
	finder.finder = NewMockMapsFinder(tmp)
	routes := finder.FindRoutes(1, 1, 1, 1, "transit", now, "")
	if !reflect.DeepEqual(routes[0], expected) {
		t.Error("Expected", expected, "found", routes[0])
	}
}

func TestFindRoutesFallsBackWhenMissingStopInfo(t *testing.T) {
	name := "729"
	now := time.Now().Unix() * 1000
	scheduledArrival := now + 11*60*1000
	route := RouteOption{
		DepartureTime: now + 10*60*1000,
		ArrivalTime:   scheduledArrival,
		Name:          name,
		Description:   "",
		MapsDetails:   generateValidTransitDetails(),
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
	now := time.Now().Unix() * 1000
	scheduledArrival := now + 11*60*1000
	route := RouteOption{
		DepartureTime: now + 10*60*1000,
		ArrivalTime:   scheduledArrival,
		Name:          name,
		Description:   "",
		MapsDetails:   generateValidTransitDetails(),
	}
	options := []RouteOption{route}
	realTimeArrival := now + 4*60*1000
	realTimeDeparture := now + 2*60*1000
	visits := generateStopVisitInfo(
		now, name, scheduledArrival, realTimeArrival, realTimeDeparture,
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
	now := time.Now().Unix() * 1000
	scheduledArrival := now + 11*60*1000
	route := RouteOption{
		DepartureTime: now + 10*60*1000,
		ArrivalTime:   scheduledArrival,
		Name:          name,
		Description:   "",
		// invalid details
		MapsDetails: generateInvalidTransitDetails(),
	}
	options := []RouteOption{route}
	realTimeArrival := now + 4*60*1000
	realTimeDeparture := now + 2*60*1000
	visits := generateStopVisitInfo(
		now, name, scheduledArrival, realTimeArrival, realTimeDeparture,
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
	now := time.Now().Unix() * 1000
	scheduledArrival := now + 11*60*1000
	route := RouteOption{
		DepartureTime: now + 10*60*1000,
		ArrivalTime:   scheduledArrival,
		Name:          name,
		Description:   "",
		MapsDetails:   generateValidTransitDetails(),
	}
	options := []RouteOption{route}
	realTimeArrival := now + 4*60*1000
	realTimeDeparture := now + 2*60*1000
	visits := generateStopVisitInfo(
		now, name, scheduledArrival, realTimeArrival, realTimeDeparture,
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
