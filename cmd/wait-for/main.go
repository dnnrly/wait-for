package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/confluentinc/bincover"
	waitfor "github.com/dnnrly/wait-for"
	"github.com/spf13/afero"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	timeoutParam := "5s"
	httpTimeoutParam := "1s"
	configFile := ""
	var quiet bool

	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)
	flag.StringVar(&timeoutParam, "timeout", timeoutParam, "time to wait for services to become available")
	flag.StringVar(&httpTimeoutParam, "http_timeout", httpTimeoutParam, "timeout for requests made by a http client")
	flag.StringVar(&configFile, "config", "", "configuration file to use")
	flag.BoolVar(&quiet, "quiet", false, "reduce output to the minimum")
	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			exit(0)
		} else {
			exit(1)
		}
		return
	}

	fs := afero.NewOsFs()

	logger := func(f string, a ...interface{}) {
		log.Printf(f, a...)
	}

	if quiet {
		logger = waitfor.NullLogger
	}

	config, err := waitfor.OpenConfig(configFile, timeoutParam, httpTimeoutParam, fs)
	if err != nil {
		_, _ = fmt.Printf("%v", err)
		exit(1)
		return
	}

	err = waitfor.WaitOn(config, logger, flag.Args(), waitfor.SupportedWaiters)
	if err != nil {
		_, _ = fmt.Printf("%v", err)
		exit(1)
		return
	}
}

func exit(code int) {
	if val, found := os.LookupEnv("BINCOVER_EXIT"); found {
		if strings.ToLower(val) == "true" {
			bincover.ExitCode = code
			return
		}
	}

	os.Exit(code)
}
