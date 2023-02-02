FROM golang:1.20

RUN mkdir /app
WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download
RUN go install github.com/cucumber/godog/cmd/godog@v0.12.6

ADD . .
RUN go build -cover -o wait-for ./cmd/wait-for

ENV GOCOVERDIR=/app/coverage

WORKDIR /app/test/
ENTRYPOINT . files/godog.sh
