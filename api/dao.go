package api

// DatabaseInterface - a generic interface for database queries
type DatabaseInterface interface {
	UpsertUser(user *UserInfo)
	ScheduleTrip(trip *TripSchedule)
	GetTrips(userId string)
	Copy() DatabaseInterface
	Close()
}
