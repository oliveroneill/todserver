package main

import (
	"github.com/oliveroneill/todserver/api"
	"testing"
	"time"
)

func TestRoundToNextInterval(t *testing.T) {
	sixteenMinutes := 16 * time.Minute
	fiveMinutes := 5 * time.Minute
	result := roundToNextInterval(sixteenMinutes)
	if result != fiveMinutes {
		t.Error("Expected", result, "to be", fiveMinutes)
	}
}

func TestRoundToNextIntervalRoundsToNearestMinute(t *testing.T) {
	fifteenMinutes := 15 * time.Minute
	fourMinutes := 4 * time.Minute
	result := roundToNextInterval(fifteenMinutes)
	if result != fourMinutes {
		t.Error("Expected", result, "to be", fourMinutes)
	}
}

func TestRoundToNextIntervalLongTimeToGo(t *testing.T) {
	threeHours := 3 * time.Hour
	twoHours := 2 * time.Hour
	result := roundToNextInterval(threeHours)
	if result != twoHours {
		t.Error("Expected", result, "to be", twoHours)
	}
}

func TestRoundToNextIntervalLessThanAMinute(t *testing.T) {
	oneMinute := 1 * time.Minute
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
	now := time.Now()
	trip := &api.TripSchedule{
		Route: &api.RouteOption{
			Description:   "Original description",
			DepartureTime: api.UnixTime{now.Add(2 * time.Hour)},
		},
		RepeatDays: []bool{false, false, false, false, false, false, false},
	}
	route := &api.RouteOption{
		Description:   "Test description",
		DepartureTime: api.UnixTime{now},
	}
	result := watchTrip(trip, NewMockGenerator(route, 0))
	if result != route {
		t.Error("Expected", result, "to equal", route)
	}
}

func TestWatchTripTimesOut(t *testing.T) {
	now := time.Now()
	originalDepartureTime := now.Add(100 * time.Millisecond)
	originalRoute := &api.RouteOption{
		Description: "Original description",
		// Truncate to millisecond level since the API will retrieve trips
		// without nanosecond precision
		DepartureTime: api.UnixTime{originalDepartureTime.Truncate(time.Millisecond)},
	}
	trip := &api.TripSchedule{
		Route:      originalRoute,
		RepeatDays: []bool{false, false, false, false, false, false, false},
	}
	route := &api.RouteOption{
		Description:   "Test description",
		DepartureTime: api.UnixTime{now},
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
	departure := api.UnixTimestampToTime(1500101524000)
	arrival := departure.Add(1 * time.Minute)
	originalRoute := &api.RouteOption{
		Description:   "Original description",
		DepartureTime: api.UnixTime{departure},
		ArrivalTime:   api.UnixTime{arrival},
	}
	// over a year in the future
	newDeparture := departure.Add(500 * 24 * time.Hour)
	newRoute := updateRouteDates(originalRoute, newDeparture)

	// check resulting departure time matches day
	if newRoute.DepartureTime.Year() != newDeparture.Year() {
		t.Error("Expected", newRoute.DepartureTime.Year(), "to equal", newDeparture.Year())
	}
	if newRoute.DepartureTime.Month() != newDeparture.Month() {
		t.Error("Expected", newRoute.DepartureTime.Month(), "to equal", newDeparture.Month())
	}
	if newRoute.DepartureTime.Day() != newDeparture.Day() {
		t.Error("Expected", newRoute.DepartureTime.Day(), "to equal", newDeparture.Day())
	}

	// check resulting arrival time matches day
	if newRoute.ArrivalTime.Year() != newDeparture.Year() {
		t.Error("Expected", newRoute.ArrivalTime.Year(), "to equal", newDeparture.Year())
	}
	if newRoute.ArrivalTime.Month() != newDeparture.Month() {
		t.Error("Expected", newRoute.ArrivalTime.Month(), "to equal", newDeparture.Month())
	}
	if newRoute.ArrivalTime.Day() != newDeparture.Day() {
		t.Error("Expected", newRoute.ArrivalTime.Day(), "to equal", newDeparture.Day())
	}

	// check resulting departure time has same hour, minute, second etc.
	resultTimeOfDay := newRoute.DepartureTime.Format("15:04:05")
	expectedTimeOfDay := departure.Format("15:04:05")
	if resultTimeOfDay != expectedTimeOfDay {
		t.Error("Expected", resultTimeOfDay, "to equal", expectedTimeOfDay)
	}

	resultTimeOfDay = newRoute.ArrivalTime.Format("15:04:05")
	expectedTimeOfDay = arrival.Format("15:04:05")
	if resultTimeOfDay != expectedTimeOfDay {
		t.Error("Expected", resultTimeOfDay, "to equal", expectedTimeOfDay)
	}
}
