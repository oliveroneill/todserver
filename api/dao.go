package api

// DatabaseInterface - a generic interface for database queries
type DatabaseInterface interface {
	UpsertUser(user *UserInfo)
	ScheduleTrip(trip *TripSchedule)
	GetTrips(userID string)
	Copy() DatabaseInterface
	Close()
}
