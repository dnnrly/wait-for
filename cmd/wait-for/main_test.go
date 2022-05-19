package main

import (
	"os"
	"testing"

	"github.com/confluentinc/bincover"
)

func TestBincoverRunMain(t *testing.T) {
	os.Setenv("BINCOVER_EXIT", "true")
	bincover.RunTest(main)
}
