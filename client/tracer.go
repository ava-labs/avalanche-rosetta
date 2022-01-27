package client

import (
	"fmt"
	"io/ioutil"

	"github.com/ava-labs/coreth/eth/tracers"
)

const tracerPath = "client/call_tracer.js"

var tracerTimeout = "180s"

func loadTraceConfig() (*tracers.TraceConfig, error) {
	loadedFile, err := ioutil.ReadFile(tracerPath)
	if err != nil {
		return nil, fmt.Errorf("%w: could not load tracer file", err)
	}

	loadedTracer := string(loadedFile)

	return &tracers.TraceConfig{
		Timeout: &tracerTimeout,
		Tracer:  &loadedTracer,
	}, nil
}
