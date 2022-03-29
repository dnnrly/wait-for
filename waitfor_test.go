package waitfor

import (
	"errors"
	"fmt"
	"testing"
	"time"

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

	config, err := openConfig("./wait-for.yaml", "", afero.NewReadOnlyFs(mockFS))
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestOpenConfig_errorOnFileParsingFailure(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte("this isn't yaml!"), 0444)

	config, err := openConfig("./wait-for.yaml", "", afero.NewReadOnlyFs(mockFS))
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestOpenConfig_errorOnParsingDefaultTimeout(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte(defaultConfigYaml()), 0444)

	config, err := openConfig("./wait-for.yaml", "invalid duration", afero.NewReadOnlyFs(mockFS))
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestOpenConfig_defaultTimeoutCanBeSet(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	_ = afero.WriteFile(mockFS, "./wait-for.yaml", []byte(defaultConfigYaml()), 0444)

	config, err := openConfig("./wait-for.yaml", "19s", afero.NewReadOnlyFs(mockFS))
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, time.Second*19, config.DefaultTimeout)
}

func TestRun_errorsOnConfigFileFailure(t *testing.T) {
	err := WaitOn("non-existent", afero.NewMemMapFs(), NullLogger, "invalid", []string{"http://localhost"}, map[string]WaiterFunc{})
	assert.Error(t, err)
}

func TestRun_errorsOnParseFailure(t *testing.T) {
	err := WaitOn("", afero.NewMemMapFs(), NullLogger, "invalid", []string{"http://localhost"}, map[string]WaiterFunc{})
	assert.Error(t, err)
}

func TestRun_errorsOnConfigFailure(t *testing.T) {
	err := WaitOn("", afero.NewMemMapFs(), NullLogger, "invalid", []string{"localhost"}, map[string]WaiterFunc{})
	assert.Error(t, err)
}

func TestWaitOnSingleTarget_succeedsImmediately(t *testing.T) {
	var logs []string
	doLog := func(f string, p ...interface{}) { logs = append(logs, fmt.Sprintf(f, p...)) }

	err := waitOnSingleTarget(
		"name",
		doLog,
		TargetConfig{Timeout: time.Second * 2},
		func(name string, target string) error { return nil },
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
		func(name string, target string) error {
			if waitUntil.After(time.Now()) {
				return fmt.Errorf("there was an error")
			}
			return nil
		},
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
		func(name string, target string) error {
			return fmt.Errorf("")
		},
	)

	assert.Error(t, err)
	assert.NotContains(t, logs, "finished waiting for name")
}

func TestWaitOnTargets_failsForUnknownType(t *testing.T) {
	err := waitOnTargets(
		NullLogger,
		map[string]TargetConfig{"unkown": {Type: "unknown type"}},
		map[string]WaiterFunc{"type": func(string, string) error { return errors.New("") }},
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
		map[string]WaiterFunc{
			"t1": func(string, string) error { return nil },
			"t2": func(string, string) error { return errors.New("an error") },
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
		map[string]WaiterFunc{
			"t1": func(string, string) error { return nil },
			"t2": func(string, string) error { return errors.New("an error") },
		},
	)

	require.Error(t, err)
	assert.Equal(t, "timed out waiting for type 2: an error", err.Error())
}
