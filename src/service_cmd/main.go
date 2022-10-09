package main

import (
	"github.com/envoyproxy/ratelimit/src/service_cmd/runner"
	"github.com/envoyproxy/ratelimit/src/settings"
)

func main() {
	s := settings.NewSettings()
	settings.TracingEnabled = s.TracingEnabled
	runner := runner.NewRunner(s)
	runner.Run()
}
