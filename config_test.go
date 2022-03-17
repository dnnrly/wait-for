package main

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

func TestConfig_fromYAML(t *testing.T) {
	config, err := NewConfigFromFile(strings.NewReader(defaultConfigYaml()))

	assert.NoError(t, err)
	assert.Equal(t, time.Second*5, config.DefaultTimeout)
	assert.Equal(t, 2, len(config.Targets))
	assert.Equal(t, "http://localhost/health", config.Targets["http-connection"].Target)
	assert.Equal(t, time.Second*10, config.Targets["http-connection"].Timeout)
	assert.Equal(t, "http", config.Targets["http-connection"].Type)
	assert.Equal(t, "localhost:80", config.Targets["tcp-connection"].Target)
	assert.Equal(t, "tcp", config.Targets["tcp-connection"].Type)
	assert.Equal(t, time.Second*5, config.Targets["tcp-connection"].Timeout)
}

func TestConfig_incorrectTimeDurationFails(t *testing.T) {
	config, err := NewConfigFromFile(strings.NewReader(`targets:
  timeout-connection:
    type: http
    target: http://localhost/health
    timeout: not parsable`))

	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestConfig_settingDefaultTimeoutWorks(t *testing.T) {
	config, err := NewConfigFromFile(strings.NewReader(`
default-timeout: 18s
targets:
  http-connection:
    type: http
    target: http://localhost/health`))

	assert.NoError(t, err)
	assert.Equal(t, time.Second*18, config.Targets["http-connection"].Timeout)
}

func TestConfig_GotTarget(t *testing.T) {
	config, _ := NewConfigFromFile(strings.NewReader(defaultConfigYaml()))

	assert.True(t, config.GotTarget("http-connection"))
	assert.False(t, config.GotTarget("non-existent"))
}

func TestConfig_AddFromString(t *testing.T) {
	config := NewConfig()

	assert.NoError(t, config.AddFromString("http://some-host/endpoint"))
	assert.NoError(t, config.AddFromString("https://some-host/endpoint"))
	assert.NoError(t, config.AddFromString("http://another-host/endpoint"))
	assert.NoError(t, config.AddFromString("tcp:listener-tcp:9090"))
	assert.Error(t, config.AddFromString("udp:some-listener:9090"))

	assert.Equal(t, 4, len(config.Targets))

	assert.Equal(t, "http://some-host/endpoint", config.Targets["http://some-host/endpoint"].Target)
	assert.Equal(t, "http", config.Targets["http://some-host/endpoint"].Type)
	assert.Equal(t, time.Second*5, config.Targets["http://some-host/endpoint"].Timeout)

	assert.Equal(t, "https://some-host/endpoint", config.Targets["https://some-host/endpoint"].Target)
	assert.Equal(t, "http", config.Targets["https://some-host/endpoint"].Type)
	assert.Equal(t, time.Second*5, config.Targets["https://some-host/endpoint"].Timeout)

	assert.Equal(t, "http://another-host/endpoint", config.Targets["http://another-host/endpoint"].Target)
	assert.Equal(t, "http", config.Targets["http://another-host/endpoint"].Type)
	assert.Equal(t, time.Second*5, config.Targets["http://some-host/endpoint"].Timeout)

	assert.Equal(t, "listener-tcp:9090", config.Targets["tcp:listener-tcp:9090"].Target)
	assert.Equal(t, "tcp", config.Targets["tcp:listener-tcp:9090"].Type)
	assert.Equal(t, time.Second*5, config.Targets["tcp:listener-tcp:9090"].Timeout)
}

func TestConfig_Filters(t *testing.T) {
	config := NewConfig()

	config.AddFromString("http://some-host/endpoint")
	config.AddFromString("https://some-host/endpoint")
	config.AddFromString("http://another-host/endpoint")
	config.Targets["listener-tcp"] = TargetConfig{
		Target: "tcp:listener-tcp:9090",
	}

	filtered := config.Filter([]string{"http://some-host/endpoint", "listener-tcp"})
	assert.Equal(t, 2, len(filtered.Targets))
}

func defaultConfigYaml() string {
	return `targets:
  http-connection:
    type: http
    target: http://localhost/health
    timeout: 10s
  tcp-connection:
    type: tcp
    target: localhost:80
`
}
