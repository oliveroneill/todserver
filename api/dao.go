package api

// DatabaseInterface is a generic interface for database queries for Tod
type DatabaseInterface interface {
	// ScheduleTrip will store this trip
	ScheduleTrip(trip *TripSchedule) error
	// UpsertUser will insert this user if they don't exist, otherwise it will
	// update the user with this notification token
	UpsertUser(user *UserInfo) error
	// GetTrips will return all trips scheduled for this user
	GetTrips(userID string) ([]TripSchedule, error)
	// SetLastNotificationTime will store the time of the last notification
	SetLastNotificationTime(trip *TripSchedule, timestamp int64) error
	// EnableDisableTrip will toggle the current setting of the trip
	// ie. enabled goes to disabled or disabled goes to enabled
	EnableDisableTrip(tripID string, userID string) error
	// DeleteTrip will delete the specified trip
	DeleteTrip(tripID string, userID string) error
	// GetAllScheduledTrips will list of trips currently persisted
	GetAllScheduledTrips() ([]*TripSchedule, error)
	// IsEnabled will return true if the specified trip is enabled
	IsEnabled(trip *TripSchedule) bool
	// Close the database connection
	Close()
}
