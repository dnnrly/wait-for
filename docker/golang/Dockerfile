FROM golang:1.18

COPY go.mod .
COPY go.sum .
RUN go mod download

ADD . .

RUN go build ./cmd/wait-for
