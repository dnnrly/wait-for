# `wait-for`

This is a CLI tool that waits for an event before continuing. Simples. But it does it
cross platform and as a single dependency that can be downloaded into your container
or environment.

Typically, you would use this to wait on another resource (such as an HTTP resource)
to become available before continuing - or timeout and exit with an error.

At the moment, you can wait for a few different kinds of thing. They are:

* HTTP or HTTPS success response or any expected response following regular expressions
* TCP or GRPC connection
* DNS IP resolve address change

[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/dnnrly/wait-for)](https://github.com/dnnrly/wait-for/releases/latest)
[![GitHub Workflow Status](https://img.shields.io/github/workflow/status/dnnrly/wait-for/Release%20workflow)](https://github.com/dnnrly/wait-for/actions?query=workflow%3A%22Release+workflow%22)
[![codecov](https://codecov.io/gh/dnnrly/wait-for/branch/main/graph/badge.svg?token=s0OfKkTFuI)](https://codecov.io/gh/dnnrly/wait-for)
[![report card](https://goreportcard.com/badge/github.com/dnnrly/wait-for)](https://goreportcard.com/report/github.com/dnnrly/wait-for)
[![Go Reference](https://pkg.go.dev/badge/github.com/dnnrly/wait-for.svg)](https://pkg.go.dev/github.com/dnnrly/wait-for)

![GitHub watchers](https://img.shields.io/github/watchers/dnnrly/wait-for?style=social)
![GitHub stars](https://img.shields.io/github/stars/dnnrly/wait-for?style=social)
[![Twitter URL](https://img.shields.io/twitter/url?style=social&url=https%3A%2F%2Fgithub.com%2Fdnnrly%2Fwait-for)](https://twitter.com/intent/tweet?url=https://github.com/dnnrly/wait-for)

## Installing `wait-for`

Using the `go` command:

```shell
go install github.com/dnnrly/wait-for/cmd/wait-for@latest
```

If you don't have Go installed (in a Docker container, for example) then you can take advantage of the pre-built versions. Check out the [releases](https://github.com/dnnrly/wait-for/releases) and check out the links for direct downloads. You can download and unpack a release like so:

```shell
wget https://github.com/dnnrly/wait-for/releases/download/v0.0.5/wait-for_0.0.5_linux_386.tar.gz
gunzip wait-for_0.0.5_linux_386.tar.gz
tar -xfv wait-for_0.0.5_linux_386.tar
```

In your Dockerfile, you can do this:
```docker
ADD https://github.com/dnnrly/wait-for/releases/download/v0.0.1/wait-for_0.0.5_linux_386.tar.gz wait-for.tar.gz
RUN gunzip wait-for.tar.gz && tar -xf wait-for.tar
```

Feel free to choose from any of the other releases though.

## Using `wait-for`

### Waiting for arbitrary HTTP services

```shell script
$ wait-for http://your-service-here:8080/health https://another-service/
``` 

### Waiting for HTTP services with expected response status

```shell script
$ wait-for -status=[0-2]{3} http://your-service-here:8080/health 
```  

### Waiting for gRPC services

```shell script
$ wait-for grpc-server:8092 other-grpc-server:9091
```

### Waiting for DNS changes

```shell script
$ wait-for dns:google.com
```

This will wait for the list of IP addresses bound to that DNS name to be
updated, regardless of order. You can use this to wait for a DNS update
such as failover or other similar operations.

### Preconfiguring services to connect to

```shell script
$ wait-for preconfigured-service
```

By default, `wait-for` will look for a file in the current directory called
`.wait-for.yml`. In this, you can define the names of services that you would
like to wait on.

```yaml
wait-for:
  preconfigured-service:
    type: http
    target: http://the-service:8080/health?reload=true
    interval: 5s
    timeout: 60s
    http-client-timeout: 3s
  another-service:
    type: http
    target: https://another-one
  grpcService:
    type: grpc
    target: localhost:9092
  snmp-service:
    type: tcp
    target: snmp-trap-dns:514
  dns-thing:
    type: dns
    target: your.r53-entry.com
```

### Using `wait-for` in Docker Compose

You can use `wait-for` to do some of the orchestration for you in your compose file. A good example
would be something like this:

```yaml
version: '3'
services:
  web:
    build: .
    ports:
      - "8080"
    command: sh -c 'wait-for tcp:db:5432 && ./your-api
    depends_on:
      - db
  db:
    image: "postgres:13-alpine"
    command: "-c log_statement=all"
    environment:
      POSTGRES_DB: weallvote-api
      POSTGRES_USER: ${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-postgres}
```

## Developing `wait-for`

### Building the tool

To build the tool so that you can run it locally, you can use the following
command.

```shell script
$ make build
```

### Unit tests

You can run the tests as the build system would, using the following command:

```shell script
$ make test
```

You can also run the Go tests in the 'usual' way with the following command:

```shell script
$ go test ./...
```

### Acceptance tests

There is a suite of GoDog tests that check that the built tooling works as
expected.

```shell script
$ make acceptance-test
```

Depending on how your system is set up, it might not be possible for you to
open up the necessary ports to run the acceptance tests. To get around this
you can run those same tests in Docker


```shell script
$ make acceptance-test-docker
```
