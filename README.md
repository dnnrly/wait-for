# `wait-for`

This is a tool that waits for an event before continuing. Simples. But it does it
cross platform and as a single dependency that can be downloaded in to your container
or environment.

Typically, you would use this to wait on another resource (such as an HTTP resource)
to become available before continuing - or timeout and exit with an error.

## How this tool is build

First off, this tool uses the [Standard Package Layout](https://github.com/golang-standards/project-layout) and
[avoids function `main`](https://pace.dev/blog/2020/02/12/why-you-shouldnt-use-func-main-in-golang-by-mat-ryer.html)
as much as possible.

## Using `wait-for`

### Waiting for arbitrary HTTP services

```shell script
$ wait-for http://your-service-here:8080/health https://another-service/
``` 


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
  another-service:
    type: http
    target: https://another-one
  snmp-service:
    type: tcp
    target: snmp-trap-dns:514
```

## Developing `wait-for`

### Building the tool

To build the tool so that you can run it locally, you can use the following
command. Please note that when releasing and publishing the tool, you won't
actually be using the version built for the local platform - you will be
creating artifacts using Docker for the different targets being used.

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
