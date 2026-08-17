// Copyright 2024 Grafana Labs
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import "time"

type Config struct {
	VerboseLogging bool
	Port           string
	WebhookPath    string

	// StaleTimeout is how long a test/node result keeps being exported after its
	// last webhook. Zero keeps results forever, which means a test that is deleted
	// or stops reporting will keep exporting its final value indefinitely. Set it
	// to a few multiples of your slowest test frequency to have those series
	// disappear instead.
	StaleTimeout time.Duration
}

func NewConfig() *Config {
	return &Config{
		VerboseLogging: false,
		Port:           "9090",
		WebhookPath:    "/webhook",
		StaleTimeout:   0,
	}
}
