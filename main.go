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

// TodServer is used for sharing a RouteFinder between requests
type TodServer struct {
	finder api.RouteFinder
	db     api.DatabaseInterface
}

func (s *TodServer) registerUserHandler(w http.ResponseWriter, r *http.Request) {
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
	err = api.UpsertUser(s.db, user)
	if err != nil {
		http.Error(w, "Failed to register", 500)
	}
}

func (s *TodServer) getTripsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid request method.", 405)
	}
	params := r.URL.Query()
	userID := params.Get("user_id")
	trips, err := api.GetScheduledTrips(s.db, userID)
	if err != nil {
		http.Error(w, "Couldn't get trips.", 500)
	}
	json.NewEncoder(w).Encode(trips)
}

func (s *TodServer) scheduleTripHandler(w http.ResponseWriter, r *http.Request) {
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
	err = api.ScheduleTrip(s.db, route)
	if err != nil {
		http.Error(w, "Couldn't schedule trip.", 500)
	}
}

func (s *TodServer) enableDisableTripHandler(w http.ResponseWriter, r *http.Request) {
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
	err = api.EnableDisableTrip(s.db, route.ID, route.User.ID)
	if err != nil {
		http.Error(w, "Couldn't enable/disable trip.", 500)
	}
}

func (s *TodServer) deleteTripHandler(w http.ResponseWriter, r *http.Request) {
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
	err = api.DeleteTrip(s.db, route.ID, route.User.ID)
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
		destLng, transportType, api.UnixTimestampToTime(arrivalTime),
		routeName)
	json.NewEncoder(w).Encode(routes)
}

func main() {
	mapsKeyArg := kingpin.Arg("googlemapskey", "Google Maps API key for querying routes").Required().String()
	nxtBusKeyArg := kingpin.Flag("nxtbuskey", "NXTBUS API key for real time data in Canberra").String()
	kingpin.Parse()
	mapsAPIKey := *mapsKeyArg
	if len(mapsAPIKey) == 0 {
		log.Fatal("No api key set.")
	}
	nxtBusAPIKey := *nxtBusKeyArg
	mapsFinder := api.NewGoogleMapsFinder(mapsAPIKey)
	var finder api.RouteFinder = mapsFinder
	if len(nxtBusAPIKey) > 0 {
		finder = api.NewNxtBusFinder(nxtBusAPIKey, mapsFinder)
	}
	db := api.NewPostgresInterface()
	defer db.Close()
	server := &TodServer{finder: finder, db: db}
	http.HandleFunc("/api/register-user", server.registerUserHandler)
	http.HandleFunc("/api/get-scheduled-trips", server.getTripsHandler)
	http.HandleFunc("/api/schedule-trip", server.scheduleTripHandler)
	http.HandleFunc("/api/enable-disable-trip", server.enableDisableTripHandler)
	http.HandleFunc("/api/delete-trip", server.deleteTripHandler)
	http.HandleFunc("/api/get-routes", server.getRoutesHandler)
	log.Fatal(http.ListenAndServe(":80", nil))
}
