package api

import (
	"testing"
	"time"
)

func TestGetNextRepeatTime(t *testing.T) {
	// Test case: not repeating, the repeat time should be today
	sundayDate, _ := time.Parse("2006-01-02T15:04:05-07:00 MST", "2017-07-02T00:04:05+10:00 AEST")
	now := sundayDate.UTC()
	departureTime := time.Unix(sundayDate.Unix()+200, 0).In(sundayDate.Location())
	repeatDays := []bool{false, false, false, false, false, false, false}
	result := getNextRepeatTimeFromDate(now, sundayDate, departureTime, repeatDays)
	expected := departureTime.Unix() * 1000
	if result != expected {
		t.Error("Expected", result, "to equal", expected)
	}
}

func TestGetNextRepeatTimeRepeatingTomorrow(t *testing.T) {
	// Test case: Repeating tomorrow
	sundayDate, _ := time.Parse("2006-01-02T15:04:05-07:00 MST", "2017-07-02T00:04:05+10:00 AEST")
	// These tests are based on the current time being the last notification
	now := time.Unix(sundayDate.Unix(), 0).UTC()
	// Ensure that we're testing across timezones
	if sundayDate.Weekday() == now.Weekday() {
		t.Error("Expected", sundayDate.Weekday(), "to not equal", now.Weekday())
	}
	// Should repeat tomorrow
	// There are a few extra repeating days in here to ensure it can handle this
	repeatDays := []bool{true, false, false, false, true, true, false}
	result := getNextRepeatTimeFromDate(now, sundayDate, sundayDate, repeatDays)
	// create date which is 1 day later to match repeat day
	dateWithMatchingTime := time.Date(
		sundayDate.Year(), sundayDate.Month(), sundayDate.Day()+1,
		sundayDate.Hour(),
		sundayDate.Minute(), sundayDate.Second(),
		sundayDate.Nanosecond(), sundayDate.Location(),
	)
	expected := dateWithMatchingTime.Unix() * 1000
	if result != expected {
		t.Error("Expected", result, "to equal", expected)
	}
}

func TestGetNextRepeatTimeRepeatingNextWeek(t *testing.T) {
	// Test case: repeating next Thursday across Daylight savings
	// The first sunday of October is wh
	sundayDate, _ := time.Parse("2006-01-02T15:04:05-07:00 MST", "2017-07-02T00:04:05+10:00 AEST")
	// These tests are based on the current time being the last notification
	now := time.Unix(sundayDate.Unix(), 0).UTC()
	// Ensure that we're testing across timezones
	if sundayDate.Weekday() == now.Weekday() {
		t.Error("Expected", sundayDate.Weekday(), "to not equal", now.Weekday())
	}
	// 4 days between Sunday and Thursday
	daysUntilNextRepeat := 4
	// Repeats on Sunday and Thursday and the last notification time is Sunday,
	// therefore Thursday is the next repeat
	repeatDays := []bool{false, false, false, true, false, false, true}
	result := getNextRepeatTimeFromDate(now, sundayDate, sundayDate, repeatDays)
	dateWithMatchingTime := time.Date(
		sundayDate.Year(),
		sundayDate.Month(),
		sundayDate.Day()+daysUntilNextRepeat,
		sundayDate.Hour(),
		sundayDate.Minute(), sundayDate.Second(),
		sundayDate.Nanosecond(), sundayDate.Location(),
	)
	expected := dateWithMatchingTime.Unix() * 1000
	if result != expected {
		t.Error("Expected", result, "to equal", expected)
	}
}

func TestGetNextRepeatTimeOverDaylightSavings(t *testing.T) {
	// Test case: repeating next Wednesday across Daylight savings
	// Daylight savings switches over on the first Sunday of October
	saturdayDate, _ := time.Parse("2006-01-02T15:04:05-07:00 MST", "2017-09-30T00:04:05+10:00 AEST")
	// These tests are based on the current time being the last notification
	now := time.Unix(saturdayDate.Unix(), 0).UTC()
	// Ensure that we're testing across timezones
	if saturdayDate.Weekday() == now.Weekday() {
		t.Error("Expected", saturdayDate.Weekday(), "to not equal", now.Weekday())
	}
	// 4 days between Saturday and Wednesday
	daysUntilNextRepeat := 4
	// Repeats on Saturday and Wednesday and the last notification time is Saturday,
	// therefore Wednesday is the next repeat
	repeatDays := []bool{false, false, true, false, false, true, false}
	result := getNextRepeatTimeFromDate(now, saturdayDate, saturdayDate, repeatDays)
	dateWithMatchingTime := time.Date(
		saturdayDate.Year(),
		saturdayDate.Month(),
		saturdayDate.Day()+daysUntilNextRepeat,
		saturdayDate.Hour(),
		saturdayDate.Minute(), saturdayDate.Second(),
		saturdayDate.Nanosecond(), saturdayDate.Location(),
	)
	expected := dateWithMatchingTime.Unix() * 1000
	if result != expected {
		t.Error("Expected", result, "to equal", expected)
	}
}

func TestDaysTillNextDay(t *testing.T) {
	expected := 6
	result := daysTillNextDay(0, 6, 7)
	if result != expected {
		t.Error("Expected", result, "to equal", expected)
	}
	expected = 5
	result = daysTillNextDay(3, 1, 7)
	if result != expected {
		t.Error("Expected", result, "to equal", expected)
	}
}

func TestGetNextDay(t *testing.T) {
	expected := 4
	repeatDays := []bool{true, false, false, false, true, true, false}
	result := getNextDay(3, repeatDays)
	if result != expected {
		t.Error("Expected", result, "to equal", expected)
	}
	expected = 1
	repeatDays = []bool{false, true, false, true, false, false, false}
	result = getNextDay(3, repeatDays)
	if result != expected {
		t.Error("Expected", result, "to equal", expected)
	}
}

func TestOriginalAlertSentReturnsTrue(t *testing.T) {
	// Test case: last notification sent recently
	var lastNotification int64 = 1500102324000
	var departureTimestamp int64 = 1500101524000
	trip := &TripSchedule{
		Route: &RouteOption{
			DepartureTime: departureTimestamp,
		},
		LastNotificationSent: lastNotification,
	}
	if !wasOriginalAlertSent(trip) {
		t.Error("Incorrect original alert sent")
	}
}

func TestOriginalAlertSentReturnsFalse(t *testing.T) {
	// Test case: no notifications have been sent at all
	var lastNotification int64 = 0
	var departureTimestamp int64 = 1500102524000
	trip := &TripSchedule{
		Route: &RouteOption{
			DepartureTime: departureTimestamp,
		},
		LastNotificationSent: lastNotification,
	}
	if wasOriginalAlertSent(trip) {
		t.Error("Incorrect original alert sent")
	}
}

func TestGetRouteFromDescription(t *testing.T) {
	// Test case: Multiple matching descriptions
	description := "Drive on this street and then on this one"
	var arrival int64 = 1500101524000
	trip := &TripSchedule{
		Route: &RouteOption{
			Description: description,
			ArrivalTime: arrival,
		},
		RepeatDays: []bool{false, false, false, false, false, false, false},
	}
	expected := RouteOption{
		Description: description,
		ArrivalTime: arrival - 100,
	}
	routes := []RouteOption{
		RouteOption{
			Description: "Test description",
			ArrivalTime: arrival,
		},
		RouteOption{
			Description: description,
			ArrivalTime: arrival - 1000,
		},
		expected,
	}
	result, _ := getRouteFromDescription(trip, routes)
	if result != expected {
		t.Error("Expected", result, "to equal", expected)
	}
}

func TestGetRouteFromDescriptionWithoutMatching(t *testing.T) {
	// Test case: No matching description
	description := "Drive on this street and then on this one"
	var arrival int64 = 1500101524000
	trip := &TripSchedule{
		Route: &RouteOption{
			Description: description,
			ArrivalTime: arrival,
		},
		RepeatDays: []bool{false, false, false, false, false, false, false},
	}
	expected := RouteOption{
		Description: "Test description1",
		ArrivalTime: arrival - 100,
	}
	routes := []RouteOption{
		RouteOption{
			Description: "Test description2",
			ArrivalTime: arrival + 500,
		},
		RouteOption{
			Description: "Test description3",
			ArrivalTime: arrival - 1000,
		},
		expected,
	}
	result, _ := getRouteFromDescription(trip, routes)
	if result != expected {
		t.Error("Expected", result, "to equal", expected)
	}
}

func TestGetRouteFromDescriptionWithNoRoutes(t *testing.T) {
	// Test case: No routes will produce error
	description := "Drive on this street and then on this one"
	var arrival int64 = 1500101524000
	trip := &TripSchedule{
		Route: &RouteOption{
			Description: description,
			ArrivalTime: arrival,
		},
		RepeatDays: []bool{false, false, false, false, false, false, false},
	}
	_, err := getRouteFromDescription(trip, []RouteOption{})
	if err == nil {
		t.Error("No error when no routes")
	}
}

func TestGetRouteFromDescriptionOnRepeatTrip(t *testing.T) {
	// Test case: Match repeating trip
	description := "Drive on this street and then on this one"
	var arrival int64 = 1500101524000
	trip := &TripSchedule{
		Route: &RouteOption{
			Description:   description,
			ArrivalTime:   arrival,
			DepartureTime: arrival - 10,
		},
		LastNotificationSent: arrival + 10,
		InputArrivalTime: &Date{
			String:    "2017-07-14T17:05:24-7:00",
			Timestamp: arrival,
		},
		RepeatDays: []bool{true, true, true, true, true, true, true},
	}
	// 24 hours from arrival
	expected := RouteOption{
		Description: description,
		ArrivalTime: arrival + 60*60*24*1000,
	}
	routes := []RouteOption{
		RouteOption{
			Description: description,
			ArrivalTime: arrival - 100,
		},
		RouteOption{
			Description: "Test description",
			ArrivalTime: arrival,
		},
		expected,
	}
	result, _ := getRouteFromDescription(trip, routes)
	if result != expected {
		t.Error("Expected", result, "to equal", expected)
	}
}
