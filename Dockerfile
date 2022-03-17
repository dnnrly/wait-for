FROM golang:1.18-alpine AS alpine-build

COPY go.mod .
COPY go.sum .
RUN go mod download

ADD . .

RUN go build ./cmd/wait-for

FROM golang:1.18 AS bionic-build

COPY go.mod .
COPY go.sum .
RUN go mod download

ADD . .

RUN go build ./cmd/wait-for

FROM alpine-build AS godog

RUN go get github.com/cucumber/godog/cmd/godog

WORKDIR /app/test
