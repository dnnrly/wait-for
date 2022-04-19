package waitfor

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/spf13/afero"
)

// WaiterFunc is used to implement waiting for a specific type of target.
// The name is used in the error and target is the actual destination being tested.
type WaiterFunc func(name string, target *TargetConfig) error
type Logger func(string, ...interface{})

// NullLogger can be used in place of a real logging function
var NullLogger = func(f string, a ...interface{}) {}

// SupportedWaiters is a mapping of known protocol names to waiter implementations
var SupportedWaiters = map[string]WaiterFunc{
	"http": HTTPWaiter,
	"tcp":  TCPWaiter,
}

// WaitOn implements waiting for many targets, using the location of config file provided with named targets to wait until
// all of those targets are responding as expected
func WaitOn(config *Config, logger Logger, targets []string, waiters map[string]WaiterFunc) error {

	for _, target := range targets {
		if !config.GotTarget(target) {
			err := config.AddFromString(target)
			if err != nil {
				return err
			}
		}
	}
	filtered := config.Filter(targets)
	err := waitOnTargets(logger, filtered.Targets, waiters)
	if err != nil {
		return err
	}

	return nil
}

func OpenConfig(configFile, defaultTimeout, defaultHTTPTimeout string, fs afero.Fs) (*Config, error) {
	var config *Config
	if configFile == "" {
		config = NewConfig()
	} else {
		f, err := fs.Open(configFile)
		if err != nil {
			return nil, fmt.Errorf("unable to open config file: %v", err)
		}

		config, err = NewConfigFromFile(f)
		if err != nil {
			return nil, fmt.Errorf("unable to %v", err)
		}
	}
	timeout, err := time.ParseDuration(defaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("unable to parse timeout: %v", err)
	}
	config.DefaultTimeout = timeout

	httpTimeout, err := time.ParseDuration(defaultHTTPTimeout)
	if err != nil {
		return nil, fmt.Errorf("unable to parse http timeout: %v", err)
	}
	config.DefaultHTTPClientTimeout = httpTimeout

	return config, nil
}

func waitOnTargets(logger Logger, targets map[string]TargetConfig, waiters map[string]WaiterFunc) error {
	var eg errgroup.Group

	for name, target := range targets {
		waiter, found := waiters[target.Type]
		if !found {
			return fmt.Errorf("unknown target type %s", target.Type)
		}

		singleName := name
		singleTarget := target

		eg.Go(func() error {
			logger("started waiting for %s", singleName)
			return waitOnSingleTarget(
				singleName, logger, singleTarget, waiter,
			)
		})
	}

	err := eg.Wait()
	if err != nil {
		return err
	}

	return nil
}

func waitOnSingleTarget(name string, logger Logger, target TargetConfig, waiter WaiterFunc) error {
	end := time.Now().Add(target.Timeout)

	err := waiter(name, &target)
	for err != nil && end.After(time.Now()) {
		logger("error while waiting for %s: %v", name, err)
		time.Sleep(time.Second)
		err = waiter(name, &target)
	}

	if err != nil {
		return fmt.Errorf("timed out waiting for %s: %v", name, err)
	}

	logger("finished waiting for %s", name)

	return nil
}

func TCPWaiter(name string, target *TargetConfig) error {
	conn, err := net.Dial("tcp", target.Target)
	if err != nil {
		return fmt.Errorf("could not connect to %s: %v", name, err)
	}
	defer conn.Close()

	return nil
}

func HTTPWaiter(name string, target *TargetConfig) error {
	client := &http.Client{
		Timeout: target.HTTPClientTimeout,
	}
	req, _ := http.NewRequest("GET", target.Target, nil)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not connect to %s: %v", name, err)
	}

	if !isSuccess(resp.StatusCode) {
		return fmt.Errorf("got %d from %s", resp.StatusCode, name)
	}

	return nil
}

func isSuccess(code int) bool {
	if code < 200 {
		return false
	}

	if code >= 300 {
		return false
	}

	return true
}
