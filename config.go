package waitfor

import (
	"errors"
	"io"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

// DefaultTimeout is the amount of time to wait for target before failing
const DefaultTimeout = time.Second * 5

// DefaultHTTPClientTimeout a default value for a time limit for requests made by http client
const DefaultHTTPClientTimeout = time.Second

// TargetConfig is the configuration of a single target
type TargetConfig struct {
	// Type is the kind of target being described
	Type string
	// Target is the location of the target to be tested
	Target string
	// Timeout is the timeout to use for this specific target if it is different to DefaultTimeout
	Timeout time.Duration
	// HTTPClientTimeout is the timeout for requests made by a http client
	HTTPClientTimeout time.Duration `yaml:"http-client-timeout"`
}

// Config represents all of the config that can be defined in a config file
type Config struct {
	DefaultTimeout           time.Duration `yaml:"default-timeout"`
	Targets                  map[string]TargetConfig
	DefaultHTTPClientTimeout time.Duration `yaml:"default-http-client-timeout"`
}

// NewConfig creates an empty Config
func NewConfig() *Config {
	return &Config{
		DefaultTimeout:           DefaultTimeout,
		Targets:                  map[string]TargetConfig{},
		DefaultHTTPClientTimeout: DefaultHTTPClientTimeout,
	}
}

// NewConfigFromFile reads configuration from the file provided
func NewConfigFromFile(r io.Reader) (*Config, error) {
	config := Config{}
	err := yaml.NewDecoder(r).Decode(&config)
	if err != nil {
		return nil, err
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = DefaultTimeout
	}
	if config.DefaultHTTPClientTimeout == 0 {
		config.DefaultHTTPClientTimeout = DefaultHTTPClientTimeout
	}
	for t := range config.Targets {
		target := config.Targets[t]
		if config.Targets[t].Timeout == 0 {
			target.Timeout = config.DefaultTimeout
		}
		if config.Targets[t].HTTPClientTimeout == 0 {
			target.HTTPClientTimeout = config.DefaultHTTPClientTimeout
		}
		config.Targets[t] = target
	}
	return &config, nil
}

// GotTarget returns true if the target exists in this config
func (c *Config) GotTarget(t string) bool {
	_, ok := c.Targets[t]
	return ok
}

// AddFromString adds a new target from a string using the format <type>:<target location>
func (c *Config) AddFromString(t string) error {
	if strings.HasPrefix(t, "tcp:") {
		c.Targets[t] = TargetConfig{
			Target:  strings.Replace(t, "tcp:", "", 1),
			Type:    "tcp",
			Timeout: c.DefaultTimeout,
		}
		return nil
	}

	if strings.HasPrefix(t, "http:") || strings.HasPrefix(t, "https:") {
		c.Targets[t] = TargetConfig{
			Target:            t,
			Type:              "http",
			Timeout:           c.DefaultTimeout,
			HTTPClientTimeout: c.DefaultHTTPClientTimeout,
		}
		return nil
	}

	if strings.HasPrefix(t, "dns:") {
		c.Targets[t] = TargetConfig{
			Target:  strings.Replace(t, "dns:", "", 1),
			Type:    "dns",
			Timeout: c.DefaultTimeout,
		}
		return nil
	}

	return errors.New("unable to understand target " + t)
}

func (c *Config) Filter(targets []string) *Config {
	result := NewConfig()

	for _, target := range targets {
		if c.GotTarget(target) {
			result.Targets[target] = c.Targets[target]
		}
	}

	return result
}
