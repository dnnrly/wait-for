package main

import (
	"errors"
	"io"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultTimeout = time.Second * 5

type TargetConfig struct {
	Type    string
	Target  string
	Timeout time.Duration
}

type Config struct {
	DefaultTimeout time.Duration `yaml:"default-timeout"`
	Targets        map[string]TargetConfig
}

func NewConfig() *Config {
	return &Config{
		DefaultTimeout: DefaultTimeout,
		Targets:        map[string]TargetConfig{},
	}
}

func NewConfigFromFile(r io.Reader) (*Config, error) {
	config := Config{}
	err := yaml.NewDecoder(r).Decode(&config)
	if err != nil {
		return nil, err
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = DefaultTimeout
	}
	for t := range config.Targets {
		if config.Targets[t].Timeout == 0 {
			target := config.Targets[t]
			target.Timeout = config.DefaultTimeout
			config.Targets[t] = target
		}
	}
	return &config, nil
}

func (c *Config) GotTarget(t string) bool {
	_, ok := c.Targets[t]
	return ok
}

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
			Target:  t,
			Type:    "http",
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
