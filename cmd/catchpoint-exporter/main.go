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
	"fmt"
	"net/http"

	"catchpoint-prometheus-exporter/collector"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
)

func main() {
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)

	var (
		port        = kingpin.Flag("port", "The port to bind the HTTP server.").Default("9090").Envar("CATCHPOINT_EXPORTER_PORT").String()
		webhookPath = kingpin.Flag("webhook-path", "The path to receive webhooks.").Default("/webhook").String()
		verbose     = kingpin.Flag("verbose", "Enable verbose logging").Default("false").Bool()
	)

	kingpin.Version("1.0.0")
	kingpin.Parse()

	logger := promlog.New(promlogConfig)
	cfg := &collector.Config{
		VerboseLogging: *verbose,
		Port:           *port,
		WebhookPath:    *webhookPath,
	}

	collector := collector.NewCollector(logger, cfg)
	prometheus.MustRegister(collector)

	// HTTP Server setup
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc(*webhookPath, collector.HandleWebhook)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, landingPageHtml, "/metrics")
	})

	level.Info(logger).Log("msg", "Starting Catchpoint Exporter", "port", *port)
	level.Error(logger).Log("msg", http.ListenAndServe(":"+*port, nil))
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
