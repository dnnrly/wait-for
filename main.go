package main

import (
	"flag"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/spf13/afero"
)

type WaiterFunc func(string, string) error
type Logger func(string, ...interface{})

var nullLogger = func(f string, a ...interface{}) {}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	timeoutParam := "5s"
	configFile := ""
	var quiet bool

	flag.StringVar(&timeoutParam, "timeout", timeoutParam, "time to wait for services to become available")
	flag.StringVar(&configFile, "config", "", "configuration file to use")
	flag.BoolVar(&quiet, "quiet", false, "reduce output to the minimum")
	flag.Parse()

	fs := afero.NewOsFs()

	logger := func(f string, a ...interface{}) {
		log.Printf(f, a...)
	}

	if quiet {
		logger = nullLogger
	}

	err := run(configFile, fs, logger, timeoutParam, flag.Args())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

func run(configFile string, fs afero.Fs, logger Logger, timeoutParam string, targets []string) error {
	config, err := openConfig(configFile, timeoutParam, fs)
	if err != nil {
		return err
	}

	for _, target := range targets {
		if !config.GotTarget(target) {
			err := config.AddFromString(target)
			if err != nil {
				return err
			}
		}
	}
	filtered := config.Filter(targets)

	err = waitOnTargets(logger, filtered.Targets)
	if err != nil {
		return err
	}

	return nil
}

func openConfig(configFile, defaultTimeout string, fs afero.Fs) (*Config, error) {
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

	return config, nil
}

func waitOnTargets(logger Logger, targets map[string]TargetConfig) error {
	var eg errgroup.Group

	for name, target := range targets {
		var waiter WaiterFunc
		switch target.Type {
		case "tcp":
			waiter = tcpWait
		case "http":
			waiter = httpWait
		default:
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

	err := waiter(name, target.Target)
	for err != nil && end.After(time.Now()) {
		logger("error while waiting for %s: %v", name, err)
		time.Sleep(time.Second)
		err = waiter(name, target.Target)
	}

	if err != nil {
		return fmt.Errorf("timed out waiting for %s: %v", name, err)
	}

	logger("finished waiting for %s", name)

	return nil
}

func tcpWait(name string, target string) error {
	conn, err := net.Dial("tcp", target)
	if err != nil {
		return fmt.Errorf("could not connect to %s: %v", name, err)
	}
	defer conn.Close()

	return nil
}

func httpWait(name string, target string) error {
	client := &http.Client{
		Timeout: time.Second * 1,
	}
	req, _ := http.NewRequest("GET", target, nil)
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
