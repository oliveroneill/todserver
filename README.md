# tod server
[![Build Status](https://travis-ci.org/oliveroneill/todserver.svg?branch=master)](https://travis-ci.org/oliveroneill/todserver)

A server that schedules trips so that a notification can be sent when it's
time to leave. This was designed to be used with the
[tod app](https://github.com/oliveroneill/tod). The server tracks trip
duration in real-time using Google Maps so that transit information is updated
and users are alerted so that they never miss their bus. Tod is short for
'time of departure'.

Google Maps does not offer real-time updates for all transit systems but the
code is extensible so that additional systems can be added.
See the [Development section](#development). Don't rely on this server for
transit updates unless you're sure that Google Maps Directions offers regular
updates to schedules in your city.

**WARNING**: This is a personal side project that is still in alpha
development, you are responsible for your own appointments and this
server will not be held accountable for any tardiness.

## Dependencies
* [Docker](https://docs.docker.com/engine/installation/)
* [docker-compose](https://docs.docker.com/compose/install/)

## Usage
The tod server is made up of:
* Trip watcher (tripwatcher in docker-compose) - this watches trips to ensure that notifications are sent based on real-time data.
* Web server (todserver in docker-compose) - this schedules trips and returns search results to the client

These can be started by calling:
```bash
docker-compose build && docker-compose up
```
from the base directory. A Google Maps Directions API key as well as push
notification configuration is required, see the
[Configuration section](#configuration) below.

### Configuration
Before calling `docker-compose up` you will need to enter API keys into the
`command` keys for each docker container in the docker-compose file. Like so:

```yaml
  todserver:
    build:
      context: .
      dockerfile: Dockerfile
    ports: ["80:80"]
    links: ['postgres']
    command: todserver XXXXXXXXXXXXXXXXX

  tripwatcher:
    build:
      context: .
      dockerfile: tripwatcher/Dockerfile
    links: ['postgres']
    command: tripwatcher XXXXXXXXXXXXXXXXX
```
Where `XXXXXXXXXXXXXXXXX` is your Google Maps Directions API key. If you don't
have a key see
[Directions API](https://developers.google.com/maps/documentation/directions/)
for more details.

You will also need to need to set up a `config.yml` in `tripwatcher/`.
Here you'll configure the `apikey` key from Firebase for `android` and
`key_path` for `ios` to point to a .p12 certificate for APNS.

You may also be required to specify a topic for APNS if setting the server
up with a production certificate. The topic will be your bundle identifier.
This should be set in `tripwatcher/main.go` in the `sendNotification` function.

## Development
Tripwatcher works by regularly searching Google Maps for routes that match the
user's query, if the trip duration suddenly takes a lot longer (due to traffic,
for example) then the notification will be sent early. As another example, if
the bus is running late, the notification will be delayed accordingly.

`api/routes.go` lists the basic API for routes and how the server will search
for them using the `RouteFinder` interface. This is currently implemented in
`googlemaps.go` using the `GoogleMapsFinder` implementation. This is then used
in `main.go` and in `tripwatcher/main.go` but could easily be replaced for
another data source.

## Testing
All tests can be run using the command `go test ./...`

## Database
The database is a Postgres database, this is configured via the `init.sql` file.
All queries and commands run to the database are in `api/postgres.go`.

## TODO
This is a list of features or issues I'd like to work on in the future.
* Real-time transit info: I'm based in Canberra and plan to include a
NXTBUS finder extension on the current Google Maps finder.
* Setup APNS feedback service checks
