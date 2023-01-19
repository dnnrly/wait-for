package waitfor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/credentials/insecure"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/spf13/afero"
)

type Waiter interface {
	Wait(name string, target *TargetConfig) error
}

// WaiterFunc is used to implement waiting for a specific type of target.
// The name is used in the error and target is the actual destination being tested.
type WaiterFunc func(name string, target *TargetConfig) error

func (w WaiterFunc) Wait(name string, target *TargetConfig) error {
	return w(name, target)
}

type Logger func(string, ...interface{})

// NullLogger can be used in place of a real logging function
var NullLogger = func(f string, a ...interface{}) {}

// SupportedWaiters is a mapping of known protocol names to waiter implementations
var SupportedWaiters map[string]Waiter

// WaitOn implements waiting for many targets, using the location of config file provided with named targets to wait until
// all of those targets are responding as expected
func WaitOn(config *Config, logger Logger, targets []string, waiters map[string]Waiter) error {

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

func OpenConfig(configFile, defaultTimeout, defaultHTTPTimeout string, fs afero.Fs, defaultRegexPattern string) (*Config, error) {
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
	config.DefaultRegexStatus = defaultRegexPattern
	return config, nil
}

func waitOnTargets(logger Logger, targets map[string]TargetConfig, waiters map[string]Waiter) error {
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

func waitOnSingleTarget(name string, logger Logger, target TargetConfig, waiter Waiter) error {
	end := time.Now().Add(target.Timeout)

	err := waiter.Wait(name, &target)
	for err != nil && end.After(time.Now()) {
		logger("error while waiting for %s: %v", name, err)
		time.Sleep(time.Second)
		err = waiter.Wait(name, &target)
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
	if target.RegexStatus != "" {
		// simplifies safe initialization for holding compiled regular expressions.
		pattern, err := regexp.Compile(target.RegexStatus)
		if err != nil {
			return fmt.Errorf("invalid Regular Expression %v", err)
		}
		// Check if the given pattern matches the status code
		if !pattern.MatchString(strconv.Itoa(resp.StatusCode)) {
			return fmt.Errorf("%d status Code and %s regex didn't match in %s", resp.StatusCode, pattern.String(), name)
		}
	} else {
		if !isSuccess(resp.StatusCode) {
			return fmt.Errorf("got %d from %s", resp.StatusCode, name)
		}
	}

	return nil
}

func GRPCWaiter(name string, target *TargetConfig) error {
	ctx, cancel := context.WithTimeout(context.TODO(), target.Timeout)
	defer cancel()

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}
	conn, err := grpc.DialContext(ctx, target.Target, dialOpts...)
	if err != nil {
		return fmt.Errorf("could not connect to %s: %v", name, err)
	}
	defer conn.Close()

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

type DNSLookup func(host string) ([]net.IP, error)

type DNSWaiter struct {
	lookup DNSLookup
	logger Logger
}

func NewDNSWaiter(lookup DNSLookup, logger Logger) *DNSWaiter {
	return &DNSWaiter{
		lookup: lookup,
		logger: logger,
	}
}

type IPList []net.IP

func (l IPList) Equals(r IPList) bool {
	return l.String() == r.String()
}

func (l IPList) Len() int {
	return len(l)
}
func (l IPList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l IPList) Less(i, j int) bool { return strings.Compare(l[i].String(), l[j].String()) < 0 }
func (l IPList) String() string {
	sort.Sort(l)
	var s []string
	for _, v := range l {
		s = append(s, v.String())
	}
	return strings.Join(s, ",")
}

func (w *DNSWaiter) Wait(host string, target *TargetConfig) error {
	in, _ := w.lookup(target.Target)
	initial := IPList(in)
	last := initial

	start := time.Now()
	now := start

	for now.Sub(start) < target.Timeout {
		w.logger("got DNS result %s", last)
		time.Sleep(time.Second)
		l, _ := w.lookup(target.Target)
		last = IPList(l)

		if !initial.Equals(last) {
			return nil
		}
		now = time.Now()
	}
	return fmt.Errorf("timed out waiting for DNS update to %s", host)
}
