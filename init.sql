CREATE TABLE users (
    user_id            varchar(240) primary key,     -- unique identifier for device
    notification_token varchar(240),                 -- token used for push notification
    os                 varchar(240)                  -- operating system of device
);

CREATE TABLE trips (
    id                       SERIAL UNIQUE,
    user_id                  varchar(240) references users(user_id),
    description              varchar(240),
    origin                   point,
    dest                     point,
    transport_type           varchar(240),
    input_arrival_time       bigint,
    input_arrival_local_date varchar(240),
    route_arrival_time       bigint,
    route_departure_time     bigint,
    route_name               varchar(240),
    waiting_window           int,
    repeat_days              bool[],
    enabled                  bool,
    last_notification_sent   bigint                  -- timestamp that last notification was sent
);
