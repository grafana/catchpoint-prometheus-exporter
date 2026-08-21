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
	// last webhook. Set it to a few multiples of the slowest test frequency, so a
	// couple of missed runs do not drop a live series. Zero disables eviction,
	// leaving deleted tests exporting their final value indefinitely.
	StaleTimeout time.Duration

	// MaxBodyBytes caps how much of a webhook body is read. Zero or less selects
	// DefaultMaxBodyBytes.
	MaxBodyBytes int64

	// MaxSeries caps how many test/node combinations are held at once. Combinations
	// already tracked keep updating; new ones are rejected. Zero selects
	// DefaultMaxSeries, a negative value disables the limit.
	MaxSeries int
}

// DefaultMaxSeries is far above any realistic Catchpoint account, so reaching it
// means something is wrong rather than that the deployment has outgrown it.
const DefaultMaxSeries = 10000

// DefaultStaleTimeout is deliberately far longer than any realistic test
// frequency: its job is to retire deleted tests, not to detect gaps in reporting.
const DefaultStaleTimeout = 24 * time.Hour

func NewConfig() *Config {
	return &Config{
		VerboseLogging: false,
		Port:           "9090",
		WebhookPath:    "/webhook",
		StaleTimeout:   DefaultStaleTimeout,
		MaxBodyBytes:   DefaultMaxBodyBytes,
		MaxSeries:      DefaultMaxSeries,
	}
}
