package waitfor

import (
	"fmt"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/stretchr/testify/assert"
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
	err := WaitOn("non-existent", afero.NewMemMapFs(), NullLogger, "invalid", []string{"http://localhost"})
	assert.Error(t, err)
}

func TestRun_errorsOnParseFailure(t *testing.T) {
	err := WaitOn("", afero.NewMemMapFs(), NullLogger, "invalid", []string{"http://localhost"})
	assert.Error(t, err)
}

func TestRun_errorsOnConfigFailure(t *testing.T) {
	err := WaitOn("", afero.NewMemMapFs(), NullLogger, "invalid", []string{"localhost"})
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
