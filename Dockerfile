FROM golang:1.8

RUN apt-get update
RUN apt-get dist-upgrade -y

ADD . /go/src/github.com/oliveroneill/todserver/
WORKDIR /go/src/github.com/oliveroneill/todserver/

RUN go get ./...
RUN go install
