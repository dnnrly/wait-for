FROM golang:1.17

RUN mkdir /app
WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download
RUN go install github.com/cucumber/godog/cmd/godog@v0.12.4

ADD . .
RUN go test ./cmd/wait-for -covermode=atomic -coverpkg=./... -c -o wait-for.test

WORKDIR /app/test/
CMD . files/godog.sh
