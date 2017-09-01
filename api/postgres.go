package api

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"log"
)

const DaysAWeek = 7

// PostgresInterface - a mongodb implementation of `DatabaseInterface`
type PostgresInterface struct {
	DatabaseInterface
	conn *sql.DB
}

// NewPostgresInterface - use to create a new mongo connection
func NewPostgresInterface() PostgresInterface {
	db := new(PostgresInterface)
	conn, err := sql.Open("postgres", "host=postgres user=docker dbname=docker sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	db.conn = conn
	return *db
}

func (db *PostgresInterface) ScheduleTrip(trip *TripSchedule) error {
	sqlStatement := `
		INSERT INTO trips
		(user_id, description, origin, dest, input_arrival_time, input_arrival_local_date,
		route_arrival_time, route_departure_time, waiting_window, transport_type,
		route_name, repeat_days, enabled, last_notification_sent)
		VALUES ($1, $2, point($3, $4), point($5, $6), $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`
	_, err := db.conn.Exec(sqlStatement, trip.User.ID, trip.Route.Description,
		trip.Origin.Lat, trip.Origin.Lng,
		trip.Destination.Lat, trip.Destination.Lng,
		trip.InputArrivalTime.Timestamp, trip.InputArrivalTime.String,
		trip.Route.ArrivalTime, trip.Route.DepartureTime,
		trip.WaitingWindowMs, trip.TransportType,
		trip.Route.Name, pq.Array(trip.RepeatDays),
		trip.Enabled, trip.LastNotificationSent)
	return err
}

func (db *PostgresInterface) UpsertUser(user *UserInfo) error {
	sqlStatement := `
		INSERT INTO users (user_id, notification_token, os)
		VALUES ($1, $2, $3) ON CONFLICT (user_id) DO UPDATE SET notification_token=EXCLUDED.notification_token`
	_, err := db.conn.Exec(sqlStatement, user.ID, user.NotificationToken, user.DeviceOS)
	if err != nil {
		return err
	}
	return nil
}

func (db *PostgresInterface) GetTrips(userId string) ([]TripSchedule, error) {
	sqlStatement := `
		SELECT
		trips.id, users.user_id, users.notification_token, users.os, trips.description,
		trips.origin, trips.dest, trips.input_arrival_time, trips.input_arrival_local_date,
		trips.route_arrival_time, trips.route_departure_time,
		trips.waiting_window, trips.transport_type, trips.route_name, trips.repeat_days,
		trips.enabled, trips.last_notification_sent
		FROM users, trips
		WHERE trips.user_id = users.user_id AND trips.user_id=$1`
	rows, err := db.conn.Query(sqlStatement, userId)
	if err != nil {
		return nil, err
	}
	trips := []TripSchedule{}
	for rows.Next() {
		var t TripSchedule
		t.Route = &RouteOption{}
		t.User = &UserInfo{}
		t.InputArrivalTime = &Date{}
		var origin string
		var dest string
		err = rows.Scan(&t.ID, &t.User.ID, &t.User.NotificationToken, &t.User.DeviceOS,
			&t.Route.Description,
			&origin, &dest,
			&t.InputArrivalTime.Timestamp, &t.InputArrivalTime.String,
			&t.Route.ArrivalTime,
			&t.Route.DepartureTime, &t.WaitingWindowMs,
			&t.TransportType, &t.Route.Name,
			pq.Array(&t.RepeatDays), &t.Enabled, &t.LastNotificationSent)
		if err != nil {
			continue
		}
		_, err = fmt.Sscanf(origin, "(%f,%f)", &t.Origin.Lat, &t.Origin.Lng)
		if err != nil {
			continue
		}
		_, err = fmt.Sscanf(dest, "(%f,%f)", &t.Destination.Lat, &t.Destination.Lng)
		if err != nil {
			continue
		}
		trips = append(trips, t)
	}
	return trips, nil
}

func (db *PostgresInterface) SetLastNotificationTime(trip *TripSchedule, timestamp int64) error {
	sqlStatement := `UPDATE trips SET last_notification_sent = $1 WHERE id = $2`
	_, err := db.conn.Exec(sqlStatement, timestamp, trip.ID)
	return err
}

func (db *PostgresInterface) IsEnabled(trip *TripSchedule) bool {
	sqlStatement := `SELECT enabled trips WHERE id = $1`
	var enabled bool
	err := db.conn.QueryRow(sqlStatement, trip.ID).Scan(&enabled)
	if err != nil {
		return trip.Enabled
	}
	return enabled
}

func (db *PostgresInterface) EnableDisableTrip(tripId string, userId string) error {
	sqlStatement := `UPDATE trips SET enabled = NOT enabled WHERE id = $1 AND user_id = $2`
	_, err := db.conn.Exec(sqlStatement, tripId, userId)
	return err
}

func (db *PostgresInterface) DeleteTrip(tripId string, userId string) error {
	sqlStatement := `DELETE FROM trips WHERE id = $1 AND user_id = $2`
	_, err := db.conn.Exec(sqlStatement, tripId, userId)
	return err
}

// Close will close the current mongo connection
func (db *PostgresInterface) Close() {
	db.conn.Close()
}

func (db *PostgresInterface) GetAllScheduledTrips() ([]*TripSchedule, error) {
	sqlStatement := `
		SELECT
		trips.id, users.user_id, users.notification_token, users.os, trips.description,
		trips.origin, trips.dest, trips.input_arrival_time, trips.input_arrival_local_date,
		trips.route_arrival_time, trips.route_departure_time,
		trips.waiting_window, trips.transport_type, trips.route_name, trips.repeat_days,
		trips.enabled, trips.last_notification_sent
		FROM users, trips
		WHERE trips.user_id = users.user_id`
	rows, err := db.conn.Query(sqlStatement)
	if err != nil {
		return nil, err
	}
	trips := []*TripSchedule{}
	for rows.Next() {
		var t TripSchedule
		t.Route = &RouteOption{}
		t.User = &UserInfo{}
		t.InputArrivalTime = &Date{}
		var origin string
		var dest string
		err = rows.Scan(&t.ID, &t.User.ID, &t.User.NotificationToken, &t.User.DeviceOS,
			&t.Route.Description,
			&origin, &dest,
			&t.InputArrivalTime.Timestamp, &t.InputArrivalTime.String,
			&t.Route.ArrivalTime,
			&t.Route.DepartureTime, &t.WaitingWindowMs,
			&t.TransportType, &t.Route.Name,
			pq.Array(&t.RepeatDays), &t.Enabled, &t.LastNotificationSent)
		if err != nil {
			fmt.Println(err)
			continue
		}
		_, err = fmt.Sscanf(origin, "(%f,%f)", &t.Origin.Lat, &t.Origin.Lng)
		if err != nil {
			fmt.Println(err)
			continue
		}
		_, err = fmt.Sscanf(dest, "(%f,%f)", &t.Destination.Lat, &t.Destination.Lng)
		if err != nil {
			fmt.Println(err)
			continue
		}
		trips = append(trips, &t)
	}
	return trips, nil
}
