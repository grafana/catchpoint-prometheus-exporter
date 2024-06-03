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
	"net/http"
	"strconv"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Metric names
	UpMetric                   = "catchpoint_up"
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
	nodeNameLabel      = "node_name"
	testNameLabel      = "test_name"
	clientIDLabel      = "client_id"
	asnLabel           = "asn"
	divisionIDLabel    = "division_id"
	monitorTypeIDLabel = "monitor_type_id"
	typeIDLabel        = "type_id"
)

type Collector struct {
	latestResponse *Response
	logger         log.Logger
	up             prometheus.Gauge
	cfg            *Config

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

func NewCollector(logger log.Logger, cfg *Config) *Collector {
	if cfg == nil {
		logger.Log("level", "error", "msg", "Initialization failed", "reason", "nil configuration received")
		cfg = NewConfig()
	}

	upMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: UpMetric,
		Help: UpDesc,
	})
	upMetric.Set(1) // Initially set to 1, indicating "up"

	return &Collector{
		logger: logger,
		cfg:    cfg,
		up:     upMetric,
		totalTimeMetric: prometheus.NewDesc(
			TotalTimeMetric,
			TotalTimeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		connectTimeMetric: prometheus.NewDesc(
			ConnectTimeMetric,
			ConnectTimeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		dnsTimeMetric: prometheus.NewDesc(
			DNSTimeMetric,
			DNSTimeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		contentLoadTimeMetric: prometheus.NewDesc(
			ContentLoadTimeMetric,
			ContentLoadTimeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		loadTimeMetric: prometheus.NewDesc(
			LoadTimeMetric,
			LoadTimeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		redirectTimeMetric: prometheus.NewDesc(
			RedirectTimeMetric,
			RedirectTimeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		sslTimeMetric: prometheus.NewDesc(
			SSLTimeMetric,
			SSLTimeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		waitTimeMetric: prometheus.NewDesc(
			WaitTimeMetric,
			WaitTimeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		clientTimeMetric: prometheus.NewDesc(
			ClientTimeMetric,
			ClientTimeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		documentCompleteTimeMetric: prometheus.NewDesc(
			DocumentCompleteTimeMetric,
			DocumentCompleteTimeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		renderStartTimeMetric: prometheus.NewDesc(
			RenderStartTimeMetric,
			RenderStartTimeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		responseContentSizeMetric: prometheus.NewDesc(
			ResponseContentSizeMetric,
			ResponseContentSizeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		responseHeadersSizeMetric: prometheus.NewDesc(
			ResponseHeadersSizeMetric,
			ResponseHeadersSizeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		totalContentSizeMetric: prometheus.NewDesc(
			TotalContentSizeMetric,
			TotalContentSizeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		totalHeadersSizeMetric: prometheus.NewDesc(
			TotalHeadersSizeMetric,
			TotalHeadersSizeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		anyErrorMetric: prometheus.NewDesc(
			AnyErrorMetric,
			AnyErrorDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		connectionErrorMetric: prometheus.NewDesc(
			ConnectionErrorMetric,
			ConnectionErrorDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		dnsErrorMetric: prometheus.NewDesc(
			DNSErrorMetric,
			DNSErrorDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		loadErrorMetric: prometheus.NewDesc(
			LoadErrorMetric,
			LoadErrorDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		timeoutErrorMetric: prometheus.NewDesc(
			TimeoutErrorMetric,
			TimeoutErrorDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		transactionErrorMetric: prometheus.NewDesc(
			TransactionErrorMetric,
			TransactionErrorDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		errorObjectsLoadedMetric: prometheus.NewDesc(
			ErrorObjectsLoadedMetric,
			ErrorObjectsLoadedDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		imageContentTypeMetric: prometheus.NewDesc(
			ImageContentTypeMetric,
			ImageContentTypeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		scriptContentTypeMetric: prometheus.NewDesc(
			ScriptContentTypeMetric,
			ScriptContentTypeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		htmlContentTypeMetric: prometheus.NewDesc(
			HTMLContentTypeMetric,
			HTMLContentTypeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		cssContentTypeMetric: prometheus.NewDesc(
			CSSContentTypeMetric,
			CSSContentTypeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		fontContentTypeMetric: prometheus.NewDesc(
			FontContentTypeMetric,
			FontContentTypeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		mediaContentTypeMetric: prometheus.NewDesc(
			MediaContentTypeMetric,
			MediaContentTypeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		xmlContentTypeMetric: prometheus.NewDesc(
			XMLContentTypeMetric,
			XMLContentTypeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		otherContentTypeMetric: prometheus.NewDesc(
			OtherContentTypeMetric,
			OtherContentTypeDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		connectionsCountMetric: prometheus.NewDesc(
			ConnectionsCountMetric,
			ConnectionsCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		hostsCountMetric: prometheus.NewDesc(
			HostsCountMetric,
			HostsCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		failedRequestsCountMetric: prometheus.NewDesc(
			FailedRequestsCountMetric,
			FailedRequestsCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		requestsCountMetric: prometheus.NewDesc(
			RequestsCountMetric,
			RequestsCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		redirectionsCountMetric: prometheus.NewDesc(
			RedirectionsCountMetric,
			RedirectionsCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		cachedCountMetric: prometheus.NewDesc(
			CachedCountMetric,
			CachedCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		imageCountMetric: prometheus.NewDesc(
			ImageCountMetric,
			ImageCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		scriptCountMetric: prometheus.NewDesc(
			ScriptCountMetric,
			ScriptCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		htmlCountMetric: prometheus.NewDesc(
			HTMLCountMetric,
			HTMLCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		cssCountMetric: prometheus.NewDesc(
			CSSCountMetric,
			CSSCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		fontCountMetric: prometheus.NewDesc(
			FontCountMetric,
			FontCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		xmlCountMetric: prometheus.NewDesc(
			XMLCountMetric,
			XMLCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		mediaCountMetric: prometheus.NewDesc(
			MediaCountMetric,
			MediaCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
		tracepointsCountMetric: prometheus.NewDesc(
			TracepointsCountMetric,
			TracepointsCountDesc,
			[]string{testIDLabel, nodeNameLabel, testNameLabel, clientIDLabel, asnLabel, divisionIDLabel, monitorTypeIDLabel, typeIDLabel},
			nil,
		),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up.Desc()
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

	var resp Response
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&resp); err != nil {
		c.logger.Log("level", "error", "msg", "Failed to decode webhook response", "error", err)
		http.Error(w, fmt.Sprintf("Error decoding response: %v", err), http.StatusBadRequest)
		c.up.Set(0)
		return
	}

	c.up.Set(1)
	if c.cfg.VerboseLogging {
		c.logger.Log("level", "info", "msg", "Webhook processed successfully", "testID", resp.TestDetails.TestId)
	}

	c.latestResponse = &resp
	w.WriteHeader(http.StatusOK)
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	ch <- c.up
	if c.latestResponse == nil {
		if c.cfg.VerboseLogging {
			c.logger.Log("level", "warn", "msg", "No data available to collect")
		}
		return
	}

	resp := c.latestResponse
	if c.cfg.VerboseLogging {
		c.logger.Log("level", "debug", "msg", "Collecting metrics", "responseID", resp.TestDetails.TestId)
	}

	labels := []string{
		resp.TestDetails.TestId,
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
			c.logger.Log("level", "debug", "msg", "Skipping metric emission due to empty value", "metric", metricDesc.String())
		}
		return
	}

	value, err := parseMetricValue(valueStr)
	if err != nil {
		c.logger.Log("level", "error", "msg", "Failed to parse metric value", "metric", metricDesc.String(), "error", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(metricDesc, prometheus.GaugeValue, value, labels...)
}

func parseMetricValue(valueStr string) (float64, error) {
	if valueStr == "False" {
		return 0, nil
	}
	if valueStr == "True" {
		return 1, nil
	}
	return strconv.ParseFloat(valueStr, 64)
}
