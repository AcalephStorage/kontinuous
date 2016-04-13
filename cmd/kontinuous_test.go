package main

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	// test using default
	defVal := "sample"
	actVal := getEnv("test", defVal)
	if actVal != defVal {
		t.Errorf("got %s, should be %s", actVal, defVal)
	}

	// test getting value from OS
	osVal := "changed"
	os.Setenv("test", osVal)
	actVal = getEnv("test", defVal)
	if actVal != osVal {
		t.Errorf("got %s, should be %s", actVal, osVal)
	}
}
