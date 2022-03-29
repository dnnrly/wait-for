package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	waitfor "github.com/dnnrly/wait-for"
	"github.com/spf13/afero"
)

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
		logger = waitfor.NullLogger
	}

	err := waitfor.WaitOn(configFile, fs, logger, timeoutParam, flag.Args())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}
