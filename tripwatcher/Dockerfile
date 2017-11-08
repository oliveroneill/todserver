FROM golang:1.8

RUN apt-get update
RUN apt-get dist-upgrade -y

RUN go get github.com/appleboy/gorush
RUN go get gopkg.in/alecthomas/kingpin.v2
RUN go get googlemaps.github.io/maps
RUN go get github.com/lib/pq
RUN go get github.com/oliveroneill/nxtbus-go

ADD . /go/src/github.com/oliveroneill/todserver/
WORKDIR /go/src/github.com/oliveroneill/todserver/tripwatcher

RUN go install
