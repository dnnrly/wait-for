package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	waitfor "github.com/dnnrly/wait-for"
	"github.com/spf13/afero"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	timeoutParam := "5s"
	httpTimeoutParam := "1s"
	configFile := ""
	var quiet bool
	regexStatus := ""

	flag.StringVar(&timeoutParam, "timeout", timeoutParam, "time to wait for services to become available")
	flag.StringVar(&httpTimeoutParam, "http_timeout", httpTimeoutParam, "timeout for requests made by a http client")
	flag.StringVar(&configFile, "config", "", "configuration file to use")
	flag.BoolVar(&quiet, "quiet", false, "reduce output to the minimum")
	flag.StringVar(&regexStatus, "regex", "", "Use regex to match the expected result in HTTP status codes")
	flag.Parse()

	fs := afero.NewOsFs()

	logger := func(f string, a ...interface{}) {
		log.Printf(f, a...)
	}

	if quiet {
		logger = waitfor.NullLogger
	}

	config, err := waitfor.OpenConfig(configFile, timeoutParam, httpTimeoutParam, fs, regexStatus)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	waitfor.SupportedWaiters = map[string]waitfor.Waiter{
		"http": waitfor.WaiterFunc(waitfor.HTTPWaiter),
		"tcp":  waitfor.WaiterFunc(waitfor.TCPWaiter),
		"grpc": waitfor.WaiterFunc(waitfor.GRPCWaiter),
		"dns":  waitfor.NewDNSWaiter(net.LookupIP, logger),
	}

	err = waitfor.WaitOn(config, logger, flag.Args(), waitfor.SupportedWaiters)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}
