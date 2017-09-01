package main

import (
	"encoding/json"
	"github.com/oliveroneill/todserver/api"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type TodServer struct {
	finder api.RouteFinder
}

func registerUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method.", 405)
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Couldn't read body", 500)
	}
	var user *api.UserInfo
	err = json.Unmarshal(body, &user)
	if err != nil {
		http.Error(w, "Invalid json body.", 400)
	}
	err = api.UpsertUser(user)
	if err != nil {
		http.Error(w, "Failed to register", 500)
	}
}

func getTripsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid request method.", 405)
	}
	params := r.URL.Query()
	userId := params.Get("user_id")
	trips, err := api.GetScheduledTrips(userId)
	if err != nil {
		http.Error(w, "Couldn't get trips.", 500)
	}
	json.NewEncoder(w).Encode(trips)
}

func scheduleTripHandler(w http.ResponseWriter, r *http.Request) {
	// Check method type
	if r.Method != "POST" {
		http.Error(w, "Invalid request method.", 405)
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Couldn't read body", 500)
	}
	route := &api.TripSchedule{
		Enabled: true,
	}
	err = json.Unmarshal(body, &route)
	if err != nil {
		http.Error(w, "Invalid json body.", 400)
	}
	err = api.ScheduleTrip(route)
	if err != nil {
		http.Error(w, "Couldn't schedule trip.", 500)
	}
}

func enableDisableTripHandler(w http.ResponseWriter, r *http.Request) {
	// Check method type
	if r.Method != "POST" {
		http.Error(w, "Invalid request method.", 405)
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Couldn't read body", 500)
	}
	var route *api.TripSchedule
	err = json.Unmarshal(body, &route)
	if err != nil {
		http.Error(w, "Invalid json body.", 400)
	}
	err = api.EnableDisableTrip(route.ID, route.User.ID)
	if err != nil {
		http.Error(w, "Couldn't enable/disable trip.", 500)
	}
}

func deleteTripHandler(w http.ResponseWriter, r *http.Request) {
	// Check method type
	if r.Method != "DELETE" {
		http.Error(w, "Invalid request method.", 405)
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Couldn't read body", 500)
	}
	var route *api.TripSchedule
	err = json.Unmarshal(body, &route)
	if err != nil {
		http.Error(w, "Invalid json body.", 400)
	}
	err = api.DeleteTrip(route.ID, route.User.ID)
	if err != nil {
		http.Error(w, "Couldn't delete trip.", 500)
	}
}

func (s *TodServer) getRoutesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid request method.", 405)
	}
	params := r.URL.Query()
	originLat, err := strconv.ParseFloat(params.Get("origin_lat"), 64)
	if err != nil {
		http.Error(w, "Invalid origin latitude", 400)
		return
	}
	originLng, err := strconv.ParseFloat(params.Get("origin_lng"), 64)
	if err != nil {
		http.Error(w, "Invalid origin longitude", 400)
		return
	}
	destLat, err := strconv.ParseFloat(params.Get("dest_lat"), 64)
	if err != nil {
		http.Error(w, "Invalid destination latitude", 400)
		return
	}
	destLng, err := strconv.ParseFloat(params.Get("dest_lng"), 64)
	if err != nil {
		http.Error(w, "Invalid destination longitude", 400)
		return
	}
	transportType := params.Get("transport_type")
	arrivalTime, err := strconv.ParseInt(params.Get("arrival_time"), 0, 64)
	if err != nil {
		http.Error(w, "Invalid arrival time", 400)
		return
	}
	routeName := params.Get("route_name")
	routes := s.finder.FindRoutes(originLat, originLng, destLat,
		destLng, transportType, arrivalTime,
		routeName)
	json.NewEncoder(w).Encode(routes)
}

func main() {
	keyArg := kingpin.Arg("googlemapskey", "Google Maps API key for querying routes").Required().String()
	kingpin.Parse()
	apiKey := *keyArg
	if len(apiKey) == 0 {
		log.Fatal("No api key set.")
	}
	server := &TodServer{finder: api.NewGoogleMapsFinder(apiKey)}
	http.HandleFunc("/api/register-user", registerUserHandler)
	http.HandleFunc("/api/get-scheduled-trips", getTripsHandler)
	http.HandleFunc("/api/schedule-trip", scheduleTripHandler)
	http.HandleFunc("/api/enable-disable-trip", enableDisableTripHandler)
	http.HandleFunc("/api/delete-trip", deleteTripHandler)
	http.HandleFunc("/api/get-routes", server.getRoutesHandler)
	log.Fatal(http.ListenAndServe(":80", nil))
}
