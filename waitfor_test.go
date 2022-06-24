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

	config, err := OpenConfig("./wait-for.yaml", "", "", afero.NewReadOnlyFs(mockFS))
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestOpenConfig_errorOnFileParsingFailure(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte("this isn't yaml!"), 0444)

	config, err := OpenConfig("./wait-for.yaml", "", "", afero.NewReadOnlyFs(mockFS))
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestOpenConfig_errorOnParsingDefaultTimeout(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte(defaultConfigYaml()), 0444)

	config, err := OpenConfig("./wait-for.yaml", "invalid duration", "1s", afero.NewReadOnlyFs(mockFS))
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestOpenConfig_errorOnParsingDefaultHTTPTimeout(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte(defaultConfigYaml()), 0444)

	config, err := OpenConfig("./wait-for.yaml", "10s", "invalid duration", afero.NewReadOnlyFs(mockFS))
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestOpenConfig_defaultTimeoutCanBeSet(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte(defaultConfigYaml()), 0444)

	config, err := OpenConfig("./wait-for.yaml", "19s", "1s", afero.NewReadOnlyFs(mockFS))
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, time.Second*19, config.DefaultTimeout)
}

func TestOpenConfig_defaultHTTPTimeoutCanBeSet(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte(defaultConfigYaml()), 0444)

	config, err := OpenConfig("./wait-for.yaml", "19s", "20s", afero.NewReadOnlyFs(mockFS))
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, time.Second*20, config.DefaultHTTPClientTimeout)
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
	}, SupportedWaiters["grpc"])

	assert.Nil(t, err, "error waiting for grpc: %v", err)
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
	}, SupportedWaiters["grpc"])

	assert.NotNil(t, err, "expected error but error was nil")
	fmt.Println(err)
}
