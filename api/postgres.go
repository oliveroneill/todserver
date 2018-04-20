package api

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"log"
)

// DaysAWeek is the number of days in a week
const DaysAWeek = 7

// PostgresInterface - a mongodb implementation of `DatabaseInterface`
type PostgresInterface struct {
	DatabaseInterface
	conn *sql.DB
}

// NewPostgresInterface - use to create a new mongo connection
func NewPostgresInterface() *PostgresInterface {
	db := new(PostgresInterface)
	conn, err := sql.Open("postgres", "host=postgres user=docker dbname=docker sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	db.conn = conn
	return db
}

// ScheduleTrip will store this trip in a Postgres database under the trips
// table
func (db *PostgresInterface) ScheduleTrip(trip *TripSchedule) error {
	sqlStatement := `
		INSERT INTO trips
		(user_id, description, origin, dest, input_arrival_time, input_arrival_local_date,
		route_arrival_time, route_departure_time, waiting_window, transport_type,
		route_name, repeat_days, enabled, last_notification_sent, timezone_location)
		VALUES ($1, $2, point($3, $4), point($5, $6), $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`
	_, err := db.conn.Exec(sqlStatement, trip.User.ID, trip.Route.Description,
		trip.Origin.Lat, trip.Origin.Lng,
		trip.Destination.Lat, trip.Destination.Lng,
		trip.InputArrivalTime.Timestamp, trip.InputArrivalTime.String,
		TimeToUnixTimestamp(trip.Route.ArrivalTime),
		TimeToUnixTimestamp(trip.Route.DepartureTime),
		trip.WaitingWindowMs, trip.TransportType,
		trip.Route.Name, pq.Array(trip.RepeatDays),
		trip.Enabled, trip.LastNotificationSent, trip.InputArrivalTime.TimezoneLocation)
	return err
}

// UpsertUser will insert this user if they don't exist, otherwise it will
// update the user with this notification token
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

// GetTrips will return all trips scheduled for this user
func (db *PostgresInterface) GetTrips(userID string) ([]TripSchedule, error) {
	sqlStatement := `
		SELECT
		trips.id, users.user_id, users.notification_token, users.os, trips.description,
		trips.origin, trips.dest, trips.input_arrival_time, trips.input_arrival_local_date,
		trips.route_arrival_time, trips.route_departure_time,
		trips.waiting_window, trips.transport_type, trips.route_name, trips.repeat_days,
		trips.enabled, trips.last_notification_sent, trips.timezone_location
		FROM users, trips
		WHERE trips.user_id = users.user_id AND trips.user_id=$1`
	rows, err := db.conn.Query(sqlStatement, userID)
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
		var departureTime int64
		var arrivalTime int64
		err = rows.Scan(&t.ID, &t.User.ID, &t.User.NotificationToken, &t.User.DeviceOS,
			&t.Route.Description,
			&origin, &dest,
			&t.InputArrivalTime.Timestamp, &t.InputArrivalTime.String,
			&arrivalTime,
			&departureTime, &t.WaitingWindowMs,
			&t.TransportType, &t.Route.Name,
			pq.Array(&t.RepeatDays), &t.Enabled, &t.LastNotificationSent,
			&t.InputArrivalTime.TimezoneLocation)
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
		t.Route.DepartureTime = UnixTime{UnixTimestampToTime(departureTime)}
		t.Route.ArrivalTime = UnixTime{UnixTimestampToTime(arrivalTime)}
		trips = append(trips, t)
	}
	return trips, nil
}

// SetLastNotificationTime will store the time of the last notification
func (db *PostgresInterface) SetLastNotificationTime(trip *TripSchedule, timestamp int64) error {
	sqlStatement := `UPDATE trips SET last_notification_sent = $1 WHERE id = $2`
	_, err := db.conn.Exec(sqlStatement, timestamp, trip.ID)
	return err
}

// IsEnabled will return true if the specified trip is enabled
func (db *PostgresInterface) IsEnabled(trip *TripSchedule) bool {
	sqlStatement := `SELECT enabled FROM trips WHERE id = $1`
	var enabled bool
	err := db.conn.QueryRow(sqlStatement, trip.ID).Scan(&enabled)
	if err != nil {
		// If we can't find the trip then it's deleted, therefore
		// it's disabled
		// TODO: postgres shoudln't be making decisions
		if err == sql.ErrNoRows {
			return false
		}
		// any other error and we return the previously set value
		return trip.Enabled
	}
	return enabled
}

// EnableDisableTrip will toggle the current setting of the trip
// ie. enabled goes to disabled or disabled goes to enabled
func (db *PostgresInterface) EnableDisableTrip(tripID string, userID string) error {
	sqlStatement := `UPDATE trips SET enabled = NOT enabled WHERE id = $1 AND user_id = $2`
	_, err := db.conn.Exec(sqlStatement, tripID, userID)
	return err
}

// DeleteTrip will delete the specified trip
func (db *PostgresInterface) DeleteTrip(tripID string, userID string) error {
	sqlStatement := `DELETE FROM trips WHERE id = $1 AND user_id = $2`
	_, err := db.conn.Exec(sqlStatement, tripID, userID)
	return err
}

// Close will close the current postgres connection
func (db *PostgresInterface) Close() {
	db.conn.Close()
}

// GetAllScheduledTrips will list of trips stored in the trips table
func (db *PostgresInterface) GetAllScheduledTrips() ([]*TripSchedule, error) {
	sqlStatement := `
		SELECT
		trips.id, users.user_id, users.notification_token, users.os, trips.description,
		trips.origin, trips.dest, trips.input_arrival_time, trips.input_arrival_local_date,
		trips.route_arrival_time, trips.route_departure_time,
		trips.waiting_window, trips.transport_type, trips.route_name, trips.repeat_days,
		trips.enabled, trips.last_notification_sent, trips.timezone_location
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
		var departureTime int64
		var arrivalTime int64
		err = rows.Scan(&t.ID, &t.User.ID, &t.User.NotificationToken, &t.User.DeviceOS,
			&t.Route.Description,
			&origin, &dest,
			&t.InputArrivalTime.Timestamp, &t.InputArrivalTime.String,
			&arrivalTime,
			&departureTime, &t.WaitingWindowMs,
			&t.TransportType, &t.Route.Name,
			pq.Array(&t.RepeatDays), &t.Enabled, &t.LastNotificationSent,
			&t.InputArrivalTime.TimezoneLocation)
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
		t.Route.DepartureTime = UnixTime{UnixTimestampToTime(departureTime)}
		t.Route.ArrivalTime = UnixTime{UnixTimestampToTime(arrivalTime)}
		trips = append(trips, &t)
	}
	return trips, nil
}
