FROM golang:1.18

COPY go.mod .
COPY go.sum .
RUN go mod download
RUN go get github.com/cucumber/godog/cmd/godog

ADD . .
RUN go build ./cmd/wait-for

WORKDIR /app/test
ENTRYPOINT . files/godog.sh
