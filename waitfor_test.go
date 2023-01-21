package waitfor

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/phayes/freeport"
	"google.golang.org/grpc"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	ip1 = net.IPv4(byte(0x01), byte(0x02), byte(0x03), byte(0x04))
	ip2 = net.IPv4(byte(0x11), byte(0x12), byte(0x13), byte(0x14))
	ip3 = net.IPv4(byte(0x21), byte(0x22), byte(0x23), byte(0x24))
	ip4 = net.IPv4(byte(0x04), byte(0x05), byte(0x06), byte(0x07))
	ip5 = net.IPv4(byte(0x14), byte(0x15), byte(0x16), byte(0x17))
	ip6 = net.IPv4(byte(0x24), byte(0x22), byte(0x23), byte(0x24))
)

func TestStatusPattern200(t *testing.T) {
	err := checkStatus("200", 200)
	assert.Nil(t, err)
}

func TestInvalidRegex(t *testing.T) {
	err := checkStatus("[", 200)
	assert.Error(t, err)
}

func TestRegexMatch(t *testing.T) {
	err := checkStatus("2[0-9]{2}", 200)
	assert.Nil(t, err)
}

func TestRegexNotMatch(t *testing.T) {
	err := checkStatus("2[0-9]{2}", 404)
	assert.Error(t, err)
}

func Test_isSuccess(t *testing.T) {
	assert.True(t, isSuccess(200))
	assert.True(t, isSuccess(214))
	assert.False(t, isSuccess(300))
	assert.False(t, isSuccess(199))
	assert.False(t, isSuccess(100))
	assert.False(t, isSuccess(500))
	assert.False(t, isSuccess(407))
}

func TestOpenConfig_errorOnFileOpenFailure(t *testing.T) {
	mockFS := afero.NewMemMapFs()

	config, err := OpenConfig("./wait-for.yaml", "", "", afero.NewReadOnlyFs(mockFS), "")
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestOpenConfig_errorOnFileParsingFailure(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte("this isn't yaml!"), 0444)

	config, err := OpenConfig("./wait-for.yaml", "", "", afero.NewReadOnlyFs(mockFS), "")
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestOpenConfig_errorOnParsingDefaultTimeout(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte(defaultConfigYaml()), 0444)

	config, err := OpenConfig("./wait-for.yaml", "invalid duration", "1s", afero.NewReadOnlyFs(mockFS), "")
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestOpenConfig_errorOnParsingDefaultHTTPTimeout(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte(defaultConfigYaml()), 0444)

	config, err := OpenConfig("./wait-for.yaml", "10s", "invalid duration", afero.NewReadOnlyFs(mockFS), "")
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestOpenConfig_defaultTimeoutCanBeSet(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte(defaultConfigYaml()), 0444)

	config, err := OpenConfig("./wait-for.yaml", "19s", "1s", afero.NewReadOnlyFs(mockFS), "")
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, time.Second*19, config.DefaultTimeout)
}

func TestOpenConfig_defaultHTTPTimeoutCanBeSet(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte(defaultConfigYaml()), 0444)

	config, err := OpenConfig("./wait-for.yaml", "19s", "20s", afero.NewReadOnlyFs(mockFS), "")
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, time.Second*20, config.DefaultHTTPClientTimeout)
}

func TestOpenConfig_defaultRegexCanBeSet(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte(defaultConfigYaml()), 0444)

	config, err := OpenConfig("./wait-for.yaml", "5s", "5s", afero.NewReadOnlyFs(mockFS), "[0-9]+")
	assert.NoError(t, err)
	assert.NotNil(t, config)
}

func TestWaitOn_errorsInvalidTarget(t *testing.T) {
	err := WaitOn(NewConfig(), NullLogger, []string{"localhost"}, map[string]Waiter{})
	assert.Error(t, err)
}

func TestRun_errorsOnParseFailure(t *testing.T) {
	err := WaitOn(NewConfig(), NullLogger, []string{"http://localhost"}, map[string]Waiter{})
	assert.Error(t, err)
}

func TestWaitOnSingleTarget_succeedsImmediately(t *testing.T) {
	var logs []string
	doLog := func(f string, p ...interface{}) { logs = append(logs, fmt.Sprintf(f, p...)) }

	err := waitOnSingleTarget(
		"name",
		doLog,
		TargetConfig{Timeout: time.Second * 2},
		WaiterFunc(func(name string, target *TargetConfig) error { return nil }),
	)

	assert.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "finished waiting for name", logs[0])
}

func TestWaitOnSingleTarget_succeedsAfterWaiting(t *testing.T) {
	var logs []string
	doLog := func(f string, p ...interface{}) { logs = append(logs, fmt.Sprintf(f, p...)) }

	waitUntil := time.Now().Add(time.Millisecond * 1100)

	err := waitOnSingleTarget(
		"name",
		doLog,
		TargetConfig{Timeout: time.Second * 2},
		WaiterFunc(func(name string, target *TargetConfig) error {
			if waitUntil.After(time.Now()) {
				return fmt.Errorf("there was an error")
			}
			return nil
		}),
	)

	assert.NoError(t, err)
	assert.Contains(t, logs, "error while waiting for name: there was an error")
	assert.Contains(t, logs, "finished waiting for name")
}

func TestWaitOnSingleTarget_failsIfRegexInvalid(t *testing.T) {
	var logs []string
	doLog := func(f string, p ...interface{}) { logs = append(logs, fmt.Sprintf(f, p...)) }

	err := waitOnSingleTarget(
		"name",
		doLog,
		TargetConfig{StatusPattern: "{5-2}"},
		WaiterFunc(func(name string, target *TargetConfig) error {
			return fmt.Errorf("")
		}),
	)

	assert.Error(t, err)
	assert.NotContains(t, logs, "finished waiting for name")
}
func TestWaitOnSingleTarget_failsIfTimerExpires(t *testing.T) {
	var logs []string
	doLog := func(f string, p ...interface{}) { logs = append(logs, fmt.Sprintf(f, p...)) }

	err := waitOnSingleTarget(
		"name",
		doLog,
		TargetConfig{Timeout: time.Second * 2},
		WaiterFunc(func(name string, target *TargetConfig) error {
			return fmt.Errorf("")
		}),
	)

	assert.Error(t, err)
	assert.NotContains(t, logs, "finished waiting for name")
}

func TestWaitOnTargets_failsForUnknownType(t *testing.T) {
	err := waitOnTargets(
		NullLogger,
		map[string]TargetConfig{"unkown": {Type: "unknown type"}},
		map[string]Waiter{"type": WaiterFunc(func(string, *TargetConfig) error { return errors.New("") })},
	)

	require.Error(t, err)
	assert.Equal(t, "unknown target type unknown type", err.Error())
}

func TestWaitOnTargets_selectsCorrectWaiter(t *testing.T) {
	err := waitOnTargets(
		NullLogger,
		map[string]TargetConfig{
			"type 1": {Type: "t1"},
		},
		map[string]Waiter{
			"t1": WaiterFunc(func(string, *TargetConfig) error { return nil }),
			"t2": WaiterFunc(func(string, *TargetConfig) error { return errors.New("an error") }),
		},
	)

	require.NoError(t, err)
}

func TestWaitOnTargets_failsWhenWaiterFails(t *testing.T) {
	err := waitOnTargets(
		NullLogger,
		map[string]TargetConfig{
			"type 1": {Type: "t1"},
			"type 2": {Type: "t2"},
		},
		map[string]Waiter{
			"t1": WaiterFunc(func(string, *TargetConfig) error { return nil }),
			"t2": WaiterFunc(func(string, *TargetConfig) error { return errors.New("an error") }),
		},
	)

	require.Error(t, err)
	assert.Equal(t, "timed out waiting for type 2: an error", err.Error())
}

func setupGrpcServer(t *testing.T) (*grpc.Server, net.Listener, error) {
	port, err := freeport.GetFreePort()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get free tcp port: %v", err)
	}

	addr := fmt.Sprintf("localhost:%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		lis.Close()
		return nil, nil, fmt.Errorf("failed to listen on %s: %v", addr, err)
	}

	server := grpc.NewServer()
	go func() {
		err = server.Serve(lis)
		if err != nil {
			t.Errorf("failed to serve grpc on addr %s: %v", lis.Addr().String(), err)
			return
		}
	}()

	// If server.Serve threw an error, fail now
	if t.Failed() {
		t.FailNow()
	}
	return server, lis, nil
}

func TestGRPCWaiter_succeedsImmediately(t *testing.T) {
	server, lis, err := setupGrpcServer(t)
	if err != nil {
		t.Fatalf("failed to create grpc server: %v", err)
	}
	defer server.Stop()

	err = waitOnSingleTarget(lis.Addr().String(), NullLogger, TargetConfig{
		Target:  lis.Addr().String(),
		Timeout: DefaultTimeout,
		Type:    "grpc",
	}, WaiterFunc(GRPCWaiter))

	assert.Nil(t, err, "error waiting for grpc: %v", err)
}

func TestIPList_Equality(t *testing.T) {
	l1 := IPList([]net.IP{ip1, ip2, ip3})
	l2 := IPList([]net.IP{ip1, ip3, ip2})
	l3 := IPList([]net.IP{ip3, ip3, ip2})
	l4 := IPList([]net.IP{ip1, ip2, ip3, ip3})

	assert.Truef(t, l1.Equals(l2), "%s != %s", l1, l2)
	assert.Truef(t, l2.Equals(l1), "%s != %s", l2, l1)
	assert.Falsef(t, l1.Equals(l3), "%s == %s", l1, l3)
	assert.Falsef(t, l1.Equals(l4), "%s == %s", l1, l4)
}

func TestIPList_String(t *testing.T) {
	assert.Equal(t, "1.2.3.4,17.18.19.20,33.34.35.36", IPList{ip1, ip2, ip3}.String())
}

func TestGRPCWaiter_failsToConnect(t *testing.T) {
	server, lis, err := setupGrpcServer(t)
	if err != nil {
		t.Fatalf("failed to create grpc server: %v", err)
	}
	defer server.Stop()

	err = waitOnSingleTarget(lis.Addr().String(), NullLogger, TargetConfig{
		Target:  "localhost:8081",
		Timeout: DefaultTimeout,
		Type:    "grpc",
	}, WaiterFunc(GRPCWaiter))

	assert.NotNil(t, err, "expected error but error was nil")
	fmt.Println(err)
}

func TestDNSWaiter_resolvesCorrectDNSName(t *testing.T) {
	name := ""
	w := NewDNSWaiter(func(host string) ([]net.IP, error) {
		name = host
		return []net.IP{ip1, ip2, ip3}, nil
	}, NullLogger)

	_ = w.Wait("dns1", &TargetConfig{
		Target: "dns.name",
	})
	assert.Equal(t, "dns.name", name)
}

func TestDNSWaiter_timesOutOnSameDNS(t *testing.T) {
	w := NewDNSWaiter(func(host string) ([]net.IP, error) { return []net.IP{ip1, ip2, ip3}, nil }, NullLogger)

	start := time.Now()
	err := w.Wait("dns1", &TargetConfig{
		Target:  "dns.name",
		Timeout: time.Second,
	})
	end := time.Now()
	require.Error(t, err)
	assert.Equal(t, "timed out waiting for DNS update to dns1", err.Error())
	assert.GreaterOrEqual(t, end.Sub(start), time.Second)
}

func TestDNSWaiter_successAfterDNSChange(t *testing.T) {
	ips := [][]net.IP{
		{ip1, ip2, ip3},
		{ip1, ip2, ip3},
		{ip4, ip5, ip6},
	}
	w := NewDNSWaiter(func(host string) ([]net.IP, error) {
		next := ips[0]
		if len(ips) > 0 {
			ips = ips[1:]
		}
		return next, nil
	}, NullLogger)

	err := w.Wait("dns1", &TargetConfig{
		Target:  "dns.name",
		Type:    "dns",
		Timeout: time.Second * 3,
	})
	require.NoError(t, err)
}

func TestDNSWaiter_allowsAddressrderChange(t *testing.T) {
	ips := [][]net.IP{
		{ip1, ip2, ip3},
		{ip2, ip1, ip3},
		{ip1, ip3, ip2},
	}
	w := NewDNSWaiter(func(host string) ([]net.IP, error) {
		next := ips[0]
		if len(ips) > 0 {
			ips = ips[1:]
		}
		return next, nil
	}, NullLogger)

	err := w.Wait("dns1", &TargetConfig{
		Target:  "dns.name",
		Type:    "dns",
		Timeout: time.Second * 2,
	})
	require.Error(t, err)
}

func TestDNSWaiter_returnsErrorOnStart(t *testing.T) {
	w := NewDNSWaiter(func(host string) ([]net.IP, error) {
		return nil, fmt.Errorf("some error")
	}, NullLogger)

	err := w.Wait("dns1", &TargetConfig{
		Target:  "dns.name",
		Type:    "dns",
		Timeout: time.Second * 2,
	})
	assert.Error(t, err)
}

func TestDNSWaiter_returnsErrorWhenWaitingz(t *testing.T) {
	errs := []error{nil, nil, fmt.Errorf("some error")}
	w := NewDNSWaiter(func(host string) ([]net.IP, error) {
		next := errs[0]
		if len(errs) > 0 {
			errs = errs[1:]
		}
		return nil, next
	}, NullLogger)

	err := w.Wait("dns1", &TargetConfig{
		Target:  "dns.name",
		Type:    "dns",
		Timeout: time.Second * 2,
	})
	assert.Error(t, err)
}
