package main

import (
	"github.com/oliveroneill/todserver/api"
	"testing"
	"time"
)

func TestRoundToNextInterval(t *testing.T) {
	var sixteenMinutes int64 = 16 * 60 * 1000
	var fiveMinutes int64 = 5 * 60 * 1000
	result := roundToNextInterval(sixteenMinutes)
	if result != fiveMinutes {
		t.Error("Expected", result, "to be", fiveMinutes)
	}
}

func TestRoundToNextIntervalRoundsToNearestMinute(t *testing.T) {
	var fifteenMinutes int64 = 15 * 60 * 1000
	var fourMinutes int64 = 4 * 60 * 1000
	result := roundToNextInterval(fifteenMinutes)
	if result != fourMinutes {
		t.Error("Expected", result, "to be", fourMinutes)
	}
}

func TestRoundToNextIntervalLongTimeToGo(t *testing.T) {
	var threeHours int64 = 3 * 60 * 60 * 1000
	var twoHours int64 = 2 * 60 * 60 * 1000
	result := roundToNextInterval(threeHours)
	if result != twoHours {
		t.Error("Expected", result, "to be", twoHours)
	}
}

func TestRoundToNextIntervalLessThanAMinute(t *testing.T) {
	var oneMinute int64 = 1 * 60 * 100
	result := roundToNextInterval(oneMinute)
	if result != oneMinute {
		t.Error("Expected", result, "to be", oneMinute)
	}
}

type MockGenerator struct {
	mockRoute *api.RouteOption
	delay     int
}

func NewMockGenerator(route *api.RouteOption, delay int) *MockGenerator {
	return &MockGenerator{
		mockRoute: route,
		delay:     delay,
	}
}

func (g *MockGenerator) GenerateRoute(trip *api.TripSchedule) <-chan *api.RouteOption {
	channel := make(chan *api.RouteOption)
	go func() {
		defer close(channel)
		if g.delay > 0 {
			time.Sleep(time.Duration(g.delay) * time.Millisecond)
		}
		channel <- g.mockRoute
	}()
	return channel
}

func TestWatchTrip(t *testing.T) {
	now := time.Now().Unix() * 1000
	trip := &api.TripSchedule{
		Route: &api.RouteOption{
			Description:   "Original description",
			DepartureTime: now + 2*60*60*1000,
		},
		RepeatDays: []bool{false, false, false, false, false, false, false},
	}
	route := &api.RouteOption{
		Description:   "Test description",
		DepartureTime: now,
	}
	result := watchTrip(trip, NewMockGenerator(route, 0))
	if result != route {
		t.Error("Expected", result, "to equal", route)
	}
}

func TestWatchTripTimesOut(t *testing.T) {
	now := time.Now().Unix() * 1000
	originalRoute := &api.RouteOption{
		Description:   "Original description",
		DepartureTime: now + 100,
	}
	trip := &api.TripSchedule{
		Route:      originalRoute,
		RepeatDays: []bool{false, false, false, false, false, false, false},
	}
	route := &api.RouteOption{
		Description:   "Test description",
		DepartureTime: now,
	}
	// The route will be returned after 200ms but the watcher will timeout
	// at 100ms
	result := watchTrip(trip, NewMockGenerator(route, 200))
	if result != originalRoute {
		t.Error("Expected", result, "to equal", originalRoute)
	}
}

func TestUpdateRouteDates(t *testing.T) {
	// Ensure that it updates days
	var departure int64 = 1500101524000
	arrival := departure + 60*1000
	arrivalTime := time.Unix(arrival/1000, 0)
	departureTime := time.Unix(departure/1000, 0)
	originalRoute := &api.RouteOption{
		Description:   "Original description",
		DepartureTime: departure,
		ArrivalTime:   arrival,
	}
	// over a year in the future
	newDeparture := departure + 1000*60*60*24*500
	newDepartureTime := time.Unix(newDeparture/1000, 0)
	newRoute := updateRouteDates(originalRoute, newDeparture)
	resultArrivalTime := time.Unix(newRoute.ArrivalTime/1000, 0)
	resultDepartureTime := time.Unix(newRoute.DepartureTime/1000, 0)

	// check resulting departure time matches day
	if resultDepartureTime.Year() != newDepartureTime.Year() {
		t.Error("Expected", resultDepartureTime.Year(), "to equal", newDepartureTime.Year())
	}
	if resultDepartureTime.Month() != newDepartureTime.Month() {
		t.Error("Expected", resultDepartureTime.Month(), "to equal", newDepartureTime.Month())
	}
	if resultDepartureTime.Day() != newDepartureTime.Day() {
		t.Error("Expected", resultDepartureTime.Day(), "to equal", newDepartureTime.Day())
	}

	// check resulting arrival time matches day
	if resultArrivalTime.Year() != newDepartureTime.Year() {
		t.Error("Expected", resultArrivalTime.Year(), "to equal", newDepartureTime.Year())
	}
	if resultArrivalTime.Month() != newDepartureTime.Month() {
		t.Error("Expected", resultArrivalTime.Month(), "to equal", newDepartureTime.Month())
	}
	if resultArrivalTime.Day() != newDepartureTime.Day() {
		t.Error("Expected", resultArrivalTime.Day(), "to equal", newDepartureTime.Day())
	}

	// check resulting departure time has same hour, minute, second etc.
	resultTimeOfDay := resultDepartureTime.Format("15:04:05")
	expectedTimeOfDay := departureTime.Format("15:04:05")
	if resultTimeOfDay != expectedTimeOfDay {
		t.Error("Expected", resultTimeOfDay, "to equal", expectedTimeOfDay)
	}

	resultTimeOfDay = resultArrivalTime.Format("15:04:05")
	expectedTimeOfDay = arrivalTime.Format("15:04:05")
	if resultTimeOfDay != expectedTimeOfDay {
		t.Error("Expected", resultTimeOfDay, "to equal", expectedTimeOfDay)
	}
}
