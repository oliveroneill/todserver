package api

import (
	"fmt"
	"time"
	"testing"
	"reflect"
	"strings"
	"encoding/json"
)

func TestRouteJSONDecoding(t *testing.T) {
	var departureTimeUnix int64 = 1500101524000
	departureTime := time.Unix(0, departureTimeUnix * 1e6)
	var arrivalTimeUnix int64 = 1500101584000
	arrivalTime := time.Unix(0, arrivalTimeUnix * 1e6)
	routeName := "Example Route Name"
	description := "Description of Route"
	expected := NewRouteOption(departureTime, arrivalTime, routeName, description)
	jsonString := fmt.Sprintf(
		`{"departure_time":%d,"arrival_time":%d,"name":"%s","description":"%s"}`,
		departureTimeUnix, arrivalTimeUnix, routeName, description,
	)
	b := []byte(jsonString)
	var r RouteOption
	err := json.Unmarshal(b, &r)
	if err != nil {
		t.Error("Unexpected error", err)
	}
	if !reflect.DeepEqual(r, expected) {
		t.Error("Expected", expected, "found", r)
	}
}

func TestRouteJSONEncoding(t *testing.T) {
	var departureTimeUnix int64 = 1500101524000
	departureTime := time.Unix(0, departureTimeUnix * 1e6)
	var arrivalTimeUnix int64 = 1500101584000
	arrivalTime := time.Unix(0, arrivalTimeUnix * 1e6)
	routeName := "Example Route Name"
	description := "Description of Route"
	expected := fmt.Sprintf(
		`{"departure_time":%d,"arrival_time":%d,"name":"%s","description":"%s"}`,
		departureTimeUnix, arrivalTimeUnix, routeName, description,
	)
	r := NewRouteOption(departureTime, arrivalTime, routeName, description)
	b, err := json.Marshal(&r)
	if err != nil {
		t.Error("Unexpected error", err)
	}
	result := string(b[:])
	if strings.Compare(result, expected) != 0 {
		t.Error("Expected", expected, "found", result)
	}
}
