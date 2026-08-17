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

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/promslog"
)

// maxWebhookBodyBytes caps how much of a webhook body is read, so a malformed or
// hostile request cannot exhaust the exporter's memory.
const maxWebhookBodyBytes = 1 << 20 // 1 MiB

const (
	// Metric names
	UpMetric                   = "catchpoint_up"
	WebhookRequestsMetric      = "catchpoint_webhook_requests_total"
	WebhookErrorsMetric        = "catchpoint_webhook_errors_total"
	TrackedSeriesMetric        = "catchpoint_tracked_series"
	TotalTimeMetric            = "catchpoint_total_time"
	ConnectTimeMetric          = "catchpoint_connect_time"
	DNSTimeMetric              = "catchpoint_dns_time"
	ContentLoadTimeMetric      = "catchpoint_content_load_time"
	LoadTimeMetric             = "catchpoint_load_time"
	RedirectTimeMetric         = "catchpoint_redirect_time"
	SSLTimeMetric              = "catchpoint_ssl_time"
	WaitTimeMetric             = "catchpoint_wait_time"
	ClientTimeMetric           = "catchpoint_client_time"
	DocumentCompleteTimeMetric = "catchpoint_document_complete_time"
	RenderStartTimeMetric      = "catchpoint_render_start_time"
	ResponseContentSizeMetric  = "catchpoint_response_content_size"
	ResponseHeadersSizeMetric  = "catchpoint_response_headers_size"
	TotalContentSizeMetric     = "catchpoint_total_content_size"
	TotalHeadersSizeMetric     = "catchpoint_total_headers_size"
	AnyErrorMetric             = "catchpoint_any_error"
	ConnectionErrorMetric      = "catchpoint_connection_error"
	DNSErrorMetric             = "catchpoint_dns_error"
	LoadErrorMetric            = "catchpoint_load_error"
	TimeoutErrorMetric         = "catchpoint_timeout_error"
	TransactionErrorMetric     = "catchpoint_transaction_error"
	ErrorObjectsLoadedMetric   = "catchpoint_error_objects_loaded"
	ImageContentTypeMetric     = "catchpoint_image_content_type"
	ScriptContentTypeMetric    = "catchpoint_script_content_type"
	HTMLContentTypeMetric      = "catchpoint_html_content_type"
	CSSContentTypeMetric       = "catchpoint_css_content_type"
	FontContentTypeMetric      = "catchpoint_font_content_type"
	MediaContentTypeMetric     = "catchpoint_media_content_type"
	XMLContentTypeMetric       = "catchpoint_xml_content_type"
	OtherContentTypeMetric     = "catchpoint_other_content_type"
	ConnectionsCountMetric     = "catchpoint_connections_count"
	HostsCountMetric           = "catchpoint_hosts_count"
	FailedRequestsCountMetric  = "catchpoint_failed_requests_count"
	RequestsCountMetric        = "catchpoint_requests_count"
	RedirectionsCountMetric    = "catchpoint_redirections_count"
	CachedCountMetric          = "catchpoint_cached_count"
	ImageCountMetric           = "catchpoint_image_count"
	ScriptCountMetric          = "catchpoint_script_count"
	HTMLCountMetric            = "catchpoint_html_count"
	CSSCountMetric             = "catchpoint_css_count"
	FontCountMetric            = "catchpoint_font_count"
	XMLCountMetric             = "catchpoint_xml_count"
	MediaCountMetric           = "catchpoint_media_count"
	TracepointsCountMetric     = "catchpoint_tracepoints_count"

	// Metric descriptions
	UpDesc                   = "Catchpoint exporter is up and running."
	WebhookRequestsDesc      = "Total number of webhook requests received from Catchpoint."
	WebhookErrorsDesc        = "Total number of webhook requests that could not be processed."
	TrackedSeriesDesc        = "Number of test/node combinations currently exported."
	TotalTimeDesc            = "Total time it took to load the webpage in milliseconds."
	ConnectTimeDesc          = "Time taken to connect to the URL in milliseconds."
	DNSTimeDesc              = "Time taken to resolve the domain name in milliseconds."
	ContentLoadTimeDesc      = "Time taken to load content in milliseconds."
	LoadTimeDesc             = "Time taken to load the first and last byte of the primary URL in milliseconds."
	RedirectTimeDesc         = "Time taken for HTTP redirects in milliseconds."
	SSLTimeDesc              = "Time taken to establish SSL handshake in milliseconds."
	WaitTimeDesc             = "Time from successful connection to receiving the first byte in milliseconds."
	ClientTimeDesc           = "Client processing time in milliseconds."
	DocumentCompleteTimeDesc = "Time taken for the browser to fully render the page after all resources are downloaded in milliseconds."
	RenderStartTimeDesc      = "Time taken to start rendering the webpage in milliseconds."
	ResponseContentSizeDesc  = "Size of the HTTP response content in bytes."
	ResponseHeadersSizeDesc  = "Size of the HTTP response headers in bytes."
	TotalContentSizeDesc     = "Total size of the HTTP response content and headers in bytes."
	TotalHeadersSizeDesc     = "Total size of the HTTP response headers in bytes."
	AnyErrorDesc             = "Indicates if any error occurred during the test."
	ConnectionErrorDesc      = "Indicates if a connection error occurred during the test."
	DNSErrorDesc             = "Indicates if a DNS error occurred during the test."
	LoadErrorDesc            = "Indicates if a load error occurred during the test."
	TimeoutErrorDesc         = "Indicates if a timeout error occurred during the test."
	TransactionErrorDesc     = "Indicates if a transaction error occurred during the test."
	ErrorObjectsLoadedDesc   = "Indicates if error objects were loaded during the test."
	ImageContentTypeDesc     = "Size of image content loaded during the test in bytes."
	ScriptContentTypeDesc    = "Size of script content loaded during the test in bytes."
	HTMLContentTypeDesc      = "Size of HTML content loaded during the test in bytes."
	CSSContentTypeDesc       = "Size of CSS content loaded during the test in bytes."
	FontContentTypeDesc      = "Size of font content loaded during the test in bytes."
	MediaContentTypeDesc     = "Size of media content loaded during the test in bytes."
	XMLContentTypeDesc       = "Size of XML content loaded during the test in bytes."
	OtherContentTypeDesc     = "Size of other content loaded during the test in bytes."
	ConnectionsCountDesc     = "Total number of connections made during the test."
	HostsCountDesc           = "Total number of hosts contacted during the test."
	FailedRequestsCountDesc  = "Number of failed requests during the test."
	RequestsCountDesc        = "Number of requests made during the test."
	RedirectionsCountDesc    = "Number of HTTP redirections encountered during the test."
	CachedCountDesc          = "Number of cached elements accessed during the test."
	ImageCountDesc           = "Number of image elements loaded during the test."
	ScriptCountDesc          = "Number of script elements loaded during the test."
	HTMLCountDesc            = "Number of HTML documents loaded during the test."
	CSSCountDesc             = "Number of CSS documents loaded during the test."
	FontCountDesc            = "Number of font resources loaded during the test."
	XMLCountDesc             = "Number of XML documents loaded during the test."
	MediaCountDesc           = "Number of media elements loaded during the test."
	TracepointsCountDesc     = "Number of tracepoints hit during the test."
)

// Labels
var (
	testIDLabel        = "test_id"
	nodeIDLabel        = "node_id"
	nodeNameLabel      = "node_name"
	testNameLabel      = "test_name"
	clientIDLabel      = "client_id"
	asnLabel           = "asn"
	divisionIDLabel    = "division_id"
	monitorTypeIDLabel = "monitor_type_id"
	typeIDLabel        = "type_id"
)

// metricLabels is the label set carried by every Catchpoint metric. Values must be
// supplied in this order; see (*Collector).emitResponse.
var metricLabels = []string{
	testIDLabel,
	nodeIDLabel,
	nodeNameLabel,
	testNameLabel,
	clientIDLabel,
	asnLabel,
	divisionIDLabel,
	monitorTypeIDLabel,
	typeIDLabel,
}

// seriesKey identifies one exported series. Catchpoint delivers one webhook per
// test run per node, so a test alone is not a unique identity: the same test
// reports independently from every node it runs on.
type seriesKey struct {
	testID string
	nodeID string
}

// sample is the most recent payload seen for a seriesKey, plus the time it was
// received so it can be aged out.
type sample struct {
	response   *Response
	receivedAt time.Time
}

type Collector struct {
	// mu guards samples. Webhooks arrive on HTTP handler goroutines while
	// Collect runs on the scrape goroutine. Collect also evicts, so every holder
	// takes the write lock.
	mu      sync.Mutex
	samples map[seriesKey]sample

	// now is overridable in tests to exercise staleness expiry.
	now func() time.Time

	// warnMissingNodeID keeps the "template has no nodeid" warning to once per
	// process rather than once per webhook.
	warnMissingNodeID sync.Once

	logger          *slog.Logger
	up              prometheus.Gauge
	webhookRequests prometheus.Counter
	webhookErrors   prometheus.Counter
	cfg             *Config

	trackedSeriesMetric *prometheus.Desc

	totalTimeMetric            *prometheus.Desc
	connectTimeMetric          *prometheus.Desc
	dnsTimeMetric              *prometheus.Desc
	contentLoadTimeMetric      *prometheus.Desc
	loadTimeMetric             *prometheus.Desc
	redirectTimeMetric         *prometheus.Desc
	sslTimeMetric              *prometheus.Desc
	waitTimeMetric             *prometheus.Desc
	clientTimeMetric           *prometheus.Desc
	documentCompleteTimeMetric *prometheus.Desc
	renderStartTimeMetric      *prometheus.Desc
	responseContentSizeMetric  *prometheus.Desc
	responseHeadersSizeMetric  *prometheus.Desc
	totalContentSizeMetric     *prometheus.Desc
	totalHeadersSizeMetric     *prometheus.Desc
	anyErrorMetric             *prometheus.Desc
	connectionErrorMetric      *prometheus.Desc
	dnsErrorMetric             *prometheus.Desc
	loadErrorMetric            *prometheus.Desc
	timeoutErrorMetric         *prometheus.Desc
	transactionErrorMetric     *prometheus.Desc
	errorObjectsLoadedMetric   *prometheus.Desc
	imageContentTypeMetric     *prometheus.Desc
	scriptContentTypeMetric    *prometheus.Desc
	htmlContentTypeMetric      *prometheus.Desc
	cssContentTypeMetric       *prometheus.Desc
	fontContentTypeMetric      *prometheus.Desc
	mediaContentTypeMetric     *prometheus.Desc
	xmlContentTypeMetric       *prometheus.Desc
	otherContentTypeMetric     *prometheus.Desc
	connectionsCountMetric     *prometheus.Desc
	hostsCountMetric           *prometheus.Desc
	failedRequestsCountMetric  *prometheus.Desc
	requestsCountMetric        *prometheus.Desc
	redirectionsCountMetric    *prometheus.Desc
	cachedCountMetric          *prometheus.Desc
	imageCountMetric           *prometheus.Desc
	scriptCountMetric          *prometheus.Desc
	htmlCountMetric            *prometheus.Desc
	cssCountMetric             *prometheus.Desc
	fontCountMetric            *prometheus.Desc
	xmlCountMetric             *prometheus.Desc
	mediaCountMetric           *prometheus.Desc
	tracepointsCountMetric     *prometheus.Desc
}

func NewCollector(logger *slog.Logger, cfg *Config) *Collector {
	if logger == nil {
		logger = promslog.NewNopLogger()
	}

	if cfg == nil {
		logger.Error("Initialization failed", "reason", "nil configuration received")
		cfg = NewConfig()
	}

	upMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: UpMetric,
		Help: UpDesc,
	})
	// The exporter is up whenever it can serve a scrape. Individual bad payloads
	// are reported through catchpoint_webhook_errors_total, not by flipping this.
	upMetric.Set(1)

	return &Collector{
		samples: make(map[seriesKey]sample),
		now:     time.Now,
		logger:  logger,
		cfg:     cfg,
		up:      upMetric,
		webhookRequests: prometheus.NewCounter(prometheus.CounterOpts{
			Name: WebhookRequestsMetric,
			Help: WebhookRequestsDesc,
		}),
		webhookErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: WebhookErrorsMetric,
			Help: WebhookErrorsDesc,
		}),
		trackedSeriesMetric: prometheus.NewDesc(
			TrackedSeriesMetric,
			TrackedSeriesDesc,
			nil,
			nil,
		),
		totalTimeMetric: prometheus.NewDesc(
			TotalTimeMetric,
			TotalTimeDesc,
			metricLabels,
			nil,
		),
		connectTimeMetric: prometheus.NewDesc(
			ConnectTimeMetric,
			ConnectTimeDesc,
			metricLabels,
			nil,
		),
		dnsTimeMetric: prometheus.NewDesc(
			DNSTimeMetric,
			DNSTimeDesc,
			metricLabels,
			nil,
		),
		contentLoadTimeMetric: prometheus.NewDesc(
			ContentLoadTimeMetric,
			ContentLoadTimeDesc,
			metricLabels,
			nil,
		),
		loadTimeMetric: prometheus.NewDesc(
			LoadTimeMetric,
			LoadTimeDesc,
			metricLabels,
			nil,
		),
		redirectTimeMetric: prometheus.NewDesc(
			RedirectTimeMetric,
			RedirectTimeDesc,
			metricLabels,
			nil,
		),
		sslTimeMetric: prometheus.NewDesc(
			SSLTimeMetric,
			SSLTimeDesc,
			metricLabels,
			nil,
		),
		waitTimeMetric: prometheus.NewDesc(
			WaitTimeMetric,
			WaitTimeDesc,
			metricLabels,
			nil,
		),
		clientTimeMetric: prometheus.NewDesc(
			ClientTimeMetric,
			ClientTimeDesc,
			metricLabels,
			nil,
		),
		documentCompleteTimeMetric: prometheus.NewDesc(
			DocumentCompleteTimeMetric,
			DocumentCompleteTimeDesc,
			metricLabels,
			nil,
		),
		renderStartTimeMetric: prometheus.NewDesc(
			RenderStartTimeMetric,
			RenderStartTimeDesc,
			metricLabels,
			nil,
		),
		responseContentSizeMetric: prometheus.NewDesc(
			ResponseContentSizeMetric,
			ResponseContentSizeDesc,
			metricLabels,
			nil,
		),
		responseHeadersSizeMetric: prometheus.NewDesc(
			ResponseHeadersSizeMetric,
			ResponseHeadersSizeDesc,
			metricLabels,
			nil,
		),
		totalContentSizeMetric: prometheus.NewDesc(
			TotalContentSizeMetric,
			TotalContentSizeDesc,
			metricLabels,
			nil,
		),
		totalHeadersSizeMetric: prometheus.NewDesc(
			TotalHeadersSizeMetric,
			TotalHeadersSizeDesc,
			metricLabels,
			nil,
		),
		anyErrorMetric: prometheus.NewDesc(
			AnyErrorMetric,
			AnyErrorDesc,
			metricLabels,
			nil,
		),
		connectionErrorMetric: prometheus.NewDesc(
			ConnectionErrorMetric,
			ConnectionErrorDesc,
			metricLabels,
			nil,
		),
		dnsErrorMetric: prometheus.NewDesc(
			DNSErrorMetric,
			DNSErrorDesc,
			metricLabels,
			nil,
		),
		loadErrorMetric: prometheus.NewDesc(
			LoadErrorMetric,
			LoadErrorDesc,
			metricLabels,
			nil,
		),
		timeoutErrorMetric: prometheus.NewDesc(
			TimeoutErrorMetric,
			TimeoutErrorDesc,
			metricLabels,
			nil,
		),
		transactionErrorMetric: prometheus.NewDesc(
			TransactionErrorMetric,
			TransactionErrorDesc,
			metricLabels,
			nil,
		),
		errorObjectsLoadedMetric: prometheus.NewDesc(
			ErrorObjectsLoadedMetric,
			ErrorObjectsLoadedDesc,
			metricLabels,
			nil,
		),
		imageContentTypeMetric: prometheus.NewDesc(
			ImageContentTypeMetric,
			ImageContentTypeDesc,
			metricLabels,
			nil,
		),
		scriptContentTypeMetric: prometheus.NewDesc(
			ScriptContentTypeMetric,
			ScriptContentTypeDesc,
			metricLabels,
			nil,
		),
		htmlContentTypeMetric: prometheus.NewDesc(
			HTMLContentTypeMetric,
			HTMLContentTypeDesc,
			metricLabels,
			nil,
		),
		cssContentTypeMetric: prometheus.NewDesc(
			CSSContentTypeMetric,
			CSSContentTypeDesc,
			metricLabels,
			nil,
		),
		fontContentTypeMetric: prometheus.NewDesc(
			FontContentTypeMetric,
			FontContentTypeDesc,
			metricLabels,
			nil,
		),
		mediaContentTypeMetric: prometheus.NewDesc(
			MediaContentTypeMetric,
			MediaContentTypeDesc,
			metricLabels,
			nil,
		),
		xmlContentTypeMetric: prometheus.NewDesc(
			XMLContentTypeMetric,
			XMLContentTypeDesc,
			metricLabels,
			nil,
		),
		otherContentTypeMetric: prometheus.NewDesc(
			OtherContentTypeMetric,
			OtherContentTypeDesc,
			metricLabels,
			nil,
		),
		connectionsCountMetric: prometheus.NewDesc(
			ConnectionsCountMetric,
			ConnectionsCountDesc,
			metricLabels,
			nil,
		),
		hostsCountMetric: prometheus.NewDesc(
			HostsCountMetric,
			HostsCountDesc,
			metricLabels,
			nil,
		),
		failedRequestsCountMetric: prometheus.NewDesc(
			FailedRequestsCountMetric,
			FailedRequestsCountDesc,
			metricLabels,
			nil,
		),
		requestsCountMetric: prometheus.NewDesc(
			RequestsCountMetric,
			RequestsCountDesc,
			metricLabels,
			nil,
		),
		redirectionsCountMetric: prometheus.NewDesc(
			RedirectionsCountMetric,
			RedirectionsCountDesc,
			metricLabels,
			nil,
		),
		cachedCountMetric: prometheus.NewDesc(
			CachedCountMetric,
			CachedCountDesc,
			metricLabels,
			nil,
		),
		imageCountMetric: prometheus.NewDesc(
			ImageCountMetric,
			ImageCountDesc,
			metricLabels,
			nil,
		),
		scriptCountMetric: prometheus.NewDesc(
			ScriptCountMetric,
			ScriptCountDesc,
			metricLabels,
			nil,
		),
		htmlCountMetric: prometheus.NewDesc(
			HTMLCountMetric,
			HTMLCountDesc,
			metricLabels,
			nil,
		),
		cssCountMetric: prometheus.NewDesc(
			CSSCountMetric,
			CSSCountDesc,
			metricLabels,
			nil,
		),
		fontCountMetric: prometheus.NewDesc(
			FontCountMetric,
			FontCountDesc,
			metricLabels,
			nil,
		),
		xmlCountMetric: prometheus.NewDesc(
			XMLCountMetric,
			XMLCountDesc,
			metricLabels,
			nil,
		),
		mediaCountMetric: prometheus.NewDesc(
			MediaCountMetric,
			MediaCountDesc,
			metricLabels,
			nil,
		),
		tracepointsCountMetric: prometheus.NewDesc(
			TracepointsCountMetric,
			TracepointsCountDesc,
			metricLabels,
			nil,
		),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up.Desc()
	ch <- c.webhookRequests.Desc()
	ch <- c.webhookErrors.Desc()
	ch <- c.trackedSeriesMetric
	ch <- c.totalTimeMetric
	ch <- c.connectTimeMetric
	ch <- c.dnsTimeMetric
	ch <- c.contentLoadTimeMetric
	ch <- c.loadTimeMetric
	ch <- c.redirectTimeMetric
	ch <- c.sslTimeMetric
	ch <- c.waitTimeMetric
	ch <- c.clientTimeMetric
	ch <- c.documentCompleteTimeMetric
	ch <- c.renderStartTimeMetric
	ch <- c.responseContentSizeMetric
	ch <- c.responseHeadersSizeMetric
	ch <- c.totalContentSizeMetric
	ch <- c.totalHeadersSizeMetric
	ch <- c.anyErrorMetric
	ch <- c.connectionErrorMetric
	ch <- c.dnsErrorMetric
	ch <- c.loadErrorMetric
	ch <- c.timeoutErrorMetric
	ch <- c.transactionErrorMetric
	ch <- c.errorObjectsLoadedMetric
	ch <- c.imageContentTypeMetric
	ch <- c.scriptContentTypeMetric
	ch <- c.htmlContentTypeMetric
	ch <- c.cssContentTypeMetric
	ch <- c.fontContentTypeMetric
	ch <- c.mediaContentTypeMetric
	ch <- c.xmlContentTypeMetric
	ch <- c.otherContentTypeMetric
	ch <- c.connectionsCountMetric
	ch <- c.hostsCountMetric
	ch <- c.failedRequestsCountMetric
	ch <- c.requestsCountMetric
	ch <- c.redirectionsCountMetric
	ch <- c.cachedCountMetric
	ch <- c.imageCountMetric
	ch <- c.scriptCountMetric
	ch <- c.htmlCountMetric
	ch <- c.cssCountMetric
	ch <- c.fontCountMetric
	ch <- c.xmlCountMetric
	ch <- c.mediaCountMetric
	ch <- c.tracepointsCountMetric
}

func (c *Collector) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if c == nil {
		http.Error(w, "Collector instance is uninitialized", http.StatusInternalServerError)
		return
	}

	c.webhookRequests.Inc()

	if r.Method != http.MethodPost {
		c.webhookErrors.Inc()
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Only POST is supported on the webhook path", http.StatusMethodNotAllowed)
		return
	}

	var resp Response
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes))
	if err := decoder.Decode(&resp); err != nil {
		c.webhookErrors.Inc()
		c.logger.Error("Failed to decode webhook response", "error", err)
		http.Error(w, fmt.Sprintf("Error decoding response: %v", err), http.StatusBadRequest)
		return
	}

	// Without a TestId every payload would collapse onto the same series and each
	// test would silently overwrite the previous one, so reject it loudly instead.
	if resp.TestDetails.TestId == "" {
		c.webhookErrors.Inc()
		c.logger.Error("Rejecting webhook payload with no TestId",
			"hint", "the Test Data Webhook template must map TestDetails.TestId to ${testid}")
		http.Error(w, "payload must set TestDetails.TestId", http.StatusBadRequest)
		return
	}

	if resp.TestDetails.NodeId == "" {
		c.warnMissingNodeID.Do(func() {
			c.logger.Warn("Webhook payload has no NodeId; results from all nodes of a test will overwrite each other",
				"testID", resp.TestDetails.TestId,
				"hint", "the Test Data Webhook template must map TestDetails.NodeId to ${nodeid}")
		})
	}

	key := seriesKey{testID: resp.TestDetails.TestId, nodeID: resp.TestDetails.NodeId}

	c.mu.Lock()
	c.samples[key] = sample{response: &resp, receivedAt: c.now()}
	tracked := len(c.samples)
	c.mu.Unlock()

	if c.cfg.VerboseLogging {
		c.logger.Info("Webhook processed successfully",
			"testID", key.testID, "nodeID", key.nodeID, "trackedSeries", tracked)
	}

	w.WriteHeader(http.StatusOK)
}

// snapshot returns the live samples, evicting any that have aged past the
// configured stale timeout. A zero timeout disables eviction.
func (c *Collector) snapshot() []sample {
	now := c.now()

	c.mu.Lock()
	defer c.mu.Unlock()

	live := make([]sample, 0, len(c.samples))
	for key, s := range c.samples {
		if c.cfg.StaleTimeout > 0 && now.Sub(s.receivedAt) > c.cfg.StaleTimeout {
			delete(c.samples, key)
			if c.cfg.VerboseLogging {
				c.logger.Debug("Evicting stale series",
					"testID", key.testID, "nodeID", key.nodeID, "age", now.Sub(s.receivedAt))
			}
			continue
		}
		live = append(live, s)
	}
	return live
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	ch <- c.up
	ch <- c.webhookRequests
	ch <- c.webhookErrors

	live := c.snapshot()
	ch <- prometheus.MustNewConstMetric(c.trackedSeriesMetric, prometheus.GaugeValue, float64(len(live)))

	if len(live) == 0 {
		if c.cfg.VerboseLogging {
			c.logger.Warn("No data available to collect")
		}
		return
	}

	// Every test/node combination that has reported is exported, not only the one
	// that reported most recently.
	for _, s := range live {
		c.emitResponse(ch, s.response)
	}
}

// emitResponse emits the full metric set for a single test/node result.
func (c *Collector) emitResponse(ch chan<- prometheus.Metric, resp *Response) {
	if c.cfg.VerboseLogging {
		c.logger.Debug("Collecting metrics",
			"testID", resp.TestDetails.TestId, "nodeID", resp.TestDetails.NodeId)
	}

	labels := []string{
		resp.TestDetails.TestId,
		resp.TestDetails.NodeId,
		resp.TestDetails.NodeName,
		resp.TestDetails.TestName,
		resp.TestDetails.ClientId,
		resp.TestDetails.Asn,
		resp.TestDetails.DivisionId,
		resp.TestDetails.MonitorTypeId,
		resp.TestDetails.TypeId,
	}

	// Emit metrics
	c.emitMetric(ch, c.totalTimeMetric, resp.Summary.TotalTime, labels)
	c.emitMetric(ch, c.connectTimeMetric, resp.Summary.Connect, labels)
	c.emitMetric(ch, c.dnsTimeMetric, resp.Summary.Dns, labels)
	c.emitMetric(ch, c.contentLoadTimeMetric, resp.Summary.ContentLoad, labels)
	c.emitMetric(ch, c.loadTimeMetric, resp.Summary.Load, labels)
	c.emitMetric(ch, c.redirectTimeMetric, resp.Summary.Redirect, labels)
	c.emitMetric(ch, c.sslTimeMetric, resp.Summary.SSL, labels)
	c.emitMetric(ch, c.waitTimeMetric, resp.Summary.Wait, labels)
	c.emitMetric(ch, c.clientTimeMetric, resp.Summary.Client, labels)
	c.emitMetric(ch, c.documentCompleteTimeMetric, resp.Summary.DocumentComplete, labels)
	c.emitMetric(ch, c.renderStartTimeMetric, resp.Summary.RenderStart, labels)
	c.emitMetric(ch, c.responseContentSizeMetric, resp.Summary.ResponseContent, labels)
	c.emitMetric(ch, c.responseHeadersSizeMetric, resp.Summary.ResponseHeaders, labels)
	c.emitMetric(ch, c.totalContentSizeMetric, resp.Summary.TotalContent, labels)
	c.emitMetric(ch, c.totalHeadersSizeMetric, resp.Summary.TotalHeaders, labels)
	c.emitMetric(ch, c.anyErrorMetric, resp.Summary.AnyError, labels)
	c.emitMetric(ch, c.connectionErrorMetric, resp.Summary.ConnectionError, labels)
	c.emitMetric(ch, c.dnsErrorMetric, resp.Summary.DNSError, labels)
	c.emitMetric(ch, c.loadErrorMetric, resp.Summary.LoadError, labels)
	c.emitMetric(ch, c.timeoutErrorMetric, resp.Summary.TimeoutError, labels)
	c.emitMetric(ch, c.transactionErrorMetric, resp.Summary.TransactionError, labels)
	c.emitMetric(ch, c.errorObjectsLoadedMetric, resp.Summary.ErrorObjectsLoaded, labels)
	c.emitMetric(ch, c.imageContentTypeMetric, resp.Summary.ImageContentType, labels)
	c.emitMetric(ch, c.scriptContentTypeMetric, resp.Summary.ScriptContentType, labels)
	c.emitMetric(ch, c.htmlContentTypeMetric, resp.Summary.HTMLContentType, labels)
	c.emitMetric(ch, c.cssContentTypeMetric, resp.Summary.CSSContentType, labels)
	c.emitMetric(ch, c.fontContentTypeMetric, resp.Summary.FontContentType, labels)
	c.emitMetric(ch, c.mediaContentTypeMetric, resp.Summary.MediaContentType, labels)
	c.emitMetric(ch, c.xmlContentTypeMetric, resp.Summary.XMLContentType, labels)
	c.emitMetric(ch, c.otherContentTypeMetric, resp.Summary.OtherContentType, labels)
	c.emitMetric(ch, c.connectionsCountMetric, resp.Summary.ConnectionsCount, labels)
	c.emitMetric(ch, c.hostsCountMetric, resp.Summary.HostsCount, labels)
	c.emitMetric(ch, c.failedRequestsCountMetric, resp.Summary.FailedRequestsCount, labels)
	c.emitMetric(ch, c.requestsCountMetric, resp.Summary.RequestsCount, labels)
	c.emitMetric(ch, c.redirectionsCountMetric, resp.Summary.RedirectionsCount, labels)
	c.emitMetric(ch, c.cachedCountMetric, resp.Summary.CachedCount, labels)
	c.emitMetric(ch, c.imageCountMetric, resp.Summary.ImageCount, labels)
	c.emitMetric(ch, c.scriptCountMetric, resp.Summary.ScriptCount, labels)
	c.emitMetric(ch, c.htmlCountMetric, resp.Summary.HTMLCount, labels)
	c.emitMetric(ch, c.cssCountMetric, resp.Summary.CSSCount, labels)
	c.emitMetric(ch, c.fontCountMetric, resp.Summary.FontCount, labels)
	c.emitMetric(ch, c.xmlCountMetric, resp.Summary.XMLCount, labels)
	c.emitMetric(ch, c.mediaCountMetric, resp.Summary.MediaCount, labels)
	c.emitMetric(ch, c.tracepointsCountMetric, resp.Summary.TracepointsCount, labels)
}

func (c *Collector) emitMetric(ch chan<- prometheus.Metric, metricDesc *prometheus.Desc, valueStr string, labels []string) {
	if valueStr == "" {
		if c.cfg.VerboseLogging {
			c.logger.Debug("Skipping metric emission due to empty value", "metric", metricDesc.String())
		}
		return
	}

	value, err := parseMetricValue(valueStr)
	if err != nil {
		c.logger.Error("Failed to parse metric value", "metric", metricDesc.String(), "error", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(metricDesc, prometheus.GaugeValue, value, labels...)
}

func parseMetricValue(valueStr string) (float64, error) {
	if strings.EqualFold(valueStr, "False") {
		return 0, nil
	}
	if strings.EqualFold(valueStr, "True") {
		return 1, nil
	}
	return strconv.ParseFloat(valueStr, 64)
}
