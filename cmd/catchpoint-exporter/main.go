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

package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/grafana/catchpoint-prometheus-exporter/collector"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
)

func main() {
	promlogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)

	var (
		port         = kingpin.Flag("port", "The port to bind the HTTP server.").Default("9090").Envar("CATCHPOINT_EXPORTER_PORT").String()
		webhookPath  = kingpin.Flag("webhook-path", "The path to receive webhooks.").Default("/catchpoint-webhook").Envar("CATCHPOINT_WEBHOOK_PATH").String()
		verbose      = kingpin.Flag("verbose", "Enable verbose logging").Default("false").Envar("CATCHPOINT_VERBOSE").Bool()
		staleTimeout = kingpin.Flag("stale-timeout", "How long to keep a test/node result after its last webhook. 0 keeps them forever.").
				Default(collector.DefaultStaleTimeout.String()).Envar("CATCHPOINT_STALE_TIMEOUT").Duration()
		maxBodyBytes = kingpin.Flag("max-body-bytes", "The largest webhook body to accept.").
				Default(strconv.Itoa(collector.DefaultMaxBodyBytes)).Envar("CATCHPOINT_MAX_BODY_BYTES").Int64()
		maxSeries = kingpin.Flag("max-series", "The maximum number of test/node combinations to keep. Negative disables the limit.").
				Default(strconv.Itoa(collector.DefaultMaxSeries)).Envar("CATCHPOINT_MAX_SERIES").Int()
	)

	kingpin.Version("1.0.0")
	kingpin.Parse()

	logger := promslog.New(promlogConfig)

	// A negative timeout silently behaves like 0, so reject it rather than
	// keeping results forever.
	if *staleTimeout < 0 {
		logger.Error("Invalid --stale-timeout", "value", *staleTimeout, "reason", "must not be negative")
		os.Exit(1)
	}
	if *maxBodyBytes <= 0 {
		logger.Error("Invalid --max-body-bytes", "value", *maxBodyBytes, "reason", "must be positive")
		os.Exit(1)
	}
	// Zero selects the default rather than "no limit", so reject it as ambiguous.
	if *maxSeries == 0 {
		logger.Error("Invalid --max-series", "value", *maxSeries, "reason", "must be positive, or negative to disable the limit")
		os.Exit(1)
	}

	cfg := &collector.Config{
		VerboseLogging: *verbose,
		Port:           *port,
		WebhookPath:    *webhookPath,
		StaleTimeout:   *staleTimeout,
		MaxBodyBytes:   *maxBodyBytes,
		MaxSeries:      *maxSeries,
	}

	collector := collector.NewCollector(logger, cfg)
	prometheus.MustRegister(collector)

	// HTTP Server setup
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc(*webhookPath, collector.HandleWebhook)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if _, err := fmt.Fprintf(w, landingPageHtml, "/metrics"); err != nil {
			logger.Error("Failed to write landing page response", "error", err)
		}
	})

	logger.Info("Starting Catchpoint Exporter",
		"port", *port, "webhookPath", *webhookPath, "staleTimeout", *staleTimeout, "maxSeries", *maxSeries)
	if *staleTimeout == 0 {
		logger.Warn("Stale eviction disabled; results are kept until restart")
	}
	if *maxSeries < 0 {
		logger.Warn("Series limit disabled; memory grows with the number of test/node combinations seen")
	}
	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("HTTP server terminated", "error", err)
		os.Exit(1)
	}
}

const (
	landingPageHtml = `<html>
<head><title>Catchpoint Exporter</title></head>
<body>
	<h1>Catchpoint Exporter</h1>
	<p><a href='%s'>Metrics</a></p>
</body>
</html>`
)
