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
	// last webhook. It should be a few multiples of the slowest test frequency, so
	// that a couple of missed runs do not drop a live series but a deleted test
	// stops being exported within a day. Zero disables eviction, which means a test
	// that is deleted or stops reporting keeps exporting its final value
	// indefinitely; metric timestamps are scrape time, so nothing downstream can
	// tell that value is stale.
	StaleTimeout time.Duration
}

// DefaultStaleTimeout is the retention applied when --stale-timeout is not set.
// It is deliberately far longer than any realistic test frequency: its job is to
// retire deleted tests, not to detect gaps in reporting.
const DefaultStaleTimeout = 24 * time.Hour

func NewConfig() *Config {
	return &Config{
		VerboseLogging: false,
		Port:           "9090",
		WebhookPath:    "/webhook",
		StaleTimeout:   DefaultStaleTimeout,
	}
}
