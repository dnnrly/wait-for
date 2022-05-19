package test

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/confluentinc/bincover"
	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
)

var binPath string = "../wait-for.test"

type stepsData struct {
	assertError error

	statusCode int
	output     string
	duration   time.Duration

	servers   []*http.Server
	serverWG  sync.WaitGroup
	listening bool

	requests     []string
	requestsLock sync.Mutex

	connections     []string
	connectionsLock sync.Mutex
}

func newStepsData() *stepsData {
	s := &stepsData{}
	return s
}

func (s *stepsData) Reset() {
	s.assertError = nil

	s.statusCode = 0
	s.output = ""

	s.requestsLock.Lock()
	defer s.requestsLock.Unlock()
	s.requests = []string{}
}

func (s *stepsData) StopListening() {
	ctx := context.Background()
	for _, server := range s.servers {
		if err := server.Shutdown(ctx); err != nil {
			log.Panicf("error while shutting down listener: %v", err)
		}
	}
	s.serverWG.Wait()
}

func (s *stepsData) startListener(addr string, handler http.Handler) {
	s.serverWG.Add(1)
	server := &http.Server{}
	server.Addr = addr
	server.Handler = handler
	s.listening = true
	s.servers = append(s.servers, server)

	go func() {
		defer s.serverWG.Done()
		server.ConnState = func(conn net.Conn, state http.ConnState) {
			s.addConnection(fmt.Sprintf("%s %s", conn.LocalAddr(), state))
		}
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Panicf("error while listening: %v", err)
		}
	}()
}

func (s *stepsData) addRequest(r *http.Request) {
	s.requestsLock.Lock()
	defer s.requestsLock.Unlock()
	s.requests = append(
		s.requests,
		fmt.Sprintf("%s %s %s", r.Host, r.Method, r.URL),
	)
}

func (s *stepsData) getRequests() []string {
	s.requestsLock.Lock()
	defer s.requestsLock.Unlock()

	var requests []string
	requests = append(requests, s.requests...)

	return requests
}

func (s *stepsData) addConnection(c string) {
	s.connectionsLock.Lock()
	defer s.connectionsLock.Unlock()
	s.connections = append(s.connections, c)
}

func (s *stepsData) getConnections() []string {
	s.connectionsLock.Lock()
	defer s.connectionsLock.Unlock()

	var connections []string
	connections = append(connections, s.connections...)

	return connections
}

func (s *stepsData) Errorf(format string, args ...interface{}) {
	s.assertError = fmt.Errorf(format, args...)
}

func (s *stepsData) iRunWaitforWithParameters(params string) error {
	collector := bincover.NewCoverageCollector("wait-for_coverage.out", true)
	err := collector.Setup()
	if err != nil {
		return err
	}
	defer func() {
		err := collector.TearDown()
		if err != nil {
			panic(err)
		}
	}()

	start := time.Now()

	_, s.statusCode, _ = collector.RunBinary(
		binPath,
		"TestBincoverRunMain",
		[]string{},
		strings.Split(params, " "),
		bincover.PostExec(func(cmd *exec.Cmd, output string, err error) error {
			s.output = output
			return nil
		}),
	)

	s.duration = time.Since(start)

	return nil
}

func (s *stepsData) theOutputContains(expected string) error {
	assert.Contains(s, s.output, expected)
	return s.assertError
}

func (s *stepsData) theOutputDoesNotContain(expected string) error {
	assert.NotContains(s, s.output, expected)
	return s.assertError
}

func (s *stepsData) theTimeTakenIsMoreThan(expected string) error {
	duration, err := time.ParseDuration(expected)
	if err != nil {
		return err
	}

	assert.Greater(s, s.duration.Seconds(), duration.Seconds())
	return s.assertError
}

func (s *stepsData) theTimeTakenIsLessThan(expected string) error {
	duration, err := time.ParseDuration(expected)
	if err != nil {
		return err
	}

	assert.Less(s, s.duration.Seconds(), duration.Seconds())
	return s.assertError
}

func (s *stepsData) waitforExitsWithAnError() error {
	assert.NotEqual(s, 0, s.statusCode)
	return s.assertError
}

func (s *stepsData) waitforExitsWithoutError() error {
	assert.Equal(s, 0, s.statusCode)
	return s.assertError
}

func (s *stepsData) iCanSeeThatAnHTTPRequestWasMadeFor(r string) error {
	assert.Contains(s, s.getRequests(), r)
	return s.assertError
}

func (s *stepsData) iCanSeeAConnectionEvent(e string) error {
	assert.Contains(s, s.getConnections(), e)
	return s.assertError
}

func (s *stepsData) iHaveAnHTTPServerOnPortWithStatus(port, status int) error {
	recordRequest := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		s.addRequest(r)
	})
	s.startListener(fmt.Sprintf(":%d", port), recordRequest)
	time.Sleep(time.Millisecond * 250)
	return nil
}

func (s *stepsData) listeningServerStatusThenStatus(port, startCode int, duration string, endCode int) error {
	waitTime, err := time.ParseDuration(duration)
	if err != nil {
		return err
	}

	changeTime := time.Now().Add(waitTime)
	recordRequest := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.addRequest(r)

		if time.Now().Before(changeTime) {
			w.WriteHeader(startCode)
			return
		}

		w.WriteHeader(endCode)
	})
	s.startListener(fmt.Sprintf(":%d", port), recordRequest)
	time.Sleep(time.Millisecond * 250)
	return nil
}

func (s *stepsData) listeningServerWaitsThenResponds(port int, duration string, status int) error {
	waitTime, err := time.ParseDuration(duration)
	if err != nil {
		return err
	}

	recordRequest := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.addRequest(r)
		time.Sleep(waitTime)
		w.WriteHeader(status)
	})
	s.startListener(fmt.Sprintf(":%d", port), recordRequest)
	time.Sleep(time.Millisecond * 250)
	return nil
}

func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
	})
	ctx.AfterSuite(func() {
	})
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	data := newStepsData()
	ctx.BeforeScenario(func(s *godog.Scenario) {
		data.Reset()
	})
	ctx.AfterScenario(func(s *godog.Scenario, err error) {
		data.StopListening()

		if data.assertError != nil {
			fmt.Println("Got connections:")
			for _, c := range data.getConnections() {
				fmt.Println("\t" + c)
			}
			fmt.Println("Got requests:")
			for _, r := range data.getRequests() {
				fmt.Println("\t" + r)
			}

			fmt.Printf("Output is:\n%s\n", data.output)
		}
	})
	ctx.Step(`^I run wait-for with parameters "([^"]*)"$`, data.iRunWaitforWithParameters)
	ctx.Step(`^wait-for exits without error$`, data.waitforExitsWithoutError)
	ctx.Step(`^wait-for exits with an error$`, data.waitforExitsWithAnError)
	ctx.Step(`^the output contains "([^"]*)"$`, data.theOutputContains)
	ctx.Step(`^the output does not contain "([^"]*)"$`, data.theOutputDoesNotContain)
	ctx.Step(`^I can see that an HTTP request was made for "([^"]*)"$`, data.iCanSeeThatAnHTTPRequestWasMadeFor)
	ctx.Step(`^I have an HTTP server running on port (\d+) that responds with (\d+)$`, data.iHaveAnHTTPServerOnPortWithStatus)
	ctx.Step(`^the time taken is more than "([^"]*)"`, data.theTimeTakenIsMoreThan)
	ctx.Step(`^the time taken is less than "([^"]*)"$`, data.theTimeTakenIsLessThan)
	ctx.Step(`^I have an HTTP server running on port (\d+) that responds with (\d+) for "([^"]*)" then responds with (\d+)$`, data.listeningServerStatusThenStatus)
	ctx.Step(`^I have an HTTP server running on port (\d+) that waits "([^"]*)" then responds with (\d+)$`, data.listeningServerWaitsThenResponds)
	ctx.Step(`^I can see a connection event "([^"]*)"$`, data.iCanSeeAConnectionEvent)

}
