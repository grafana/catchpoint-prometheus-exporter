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
	"bytes"
	"encoding/json"
	"flag"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/promslog"
)

var update = flag.Bool("update", false, "rewrite the golden files under testdata")

// goldenFS scopes golden-file reads to testdata, so the read path cannot escape
// that directory. This is what keeps gosec's G304 quiet without a nolint.
var goldenFS = os.DirFS("testdata")

// alwaysOnMetricCount is the number of metric families the exporter reports even
// with no test data: catchpoint_up, the two webhook counters,
// catchpoint_series_dropped_total and catchpoint_tracked_series.
const alwaysOnMetricCount = 5

func newTestCollector(t *testing.T, cfg *Config) (*Collector, *prometheus.Registry) {
	t.Helper()

	if cfg == nil {
		cfg = &Config{}
	}

	c := NewCollector(promslog.NewNopLogger(), cfg)
	registry := prometheus.NewPedanticRegistry()
	if err := registry.Register(c); err != nil {
		t.Fatal("failed to register collector:", err)
	}
	t.Cleanup(func() { registry.Unregister(c) })

	return c, registry
}

func postWebhook(t *testing.T, c *Collector, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "http://example.com/webhook", strings.NewReader(body))
	w := httptest.NewRecorder()
	c.HandleWebhook(w, req)

	return w
}

// payloadFor re-keys the full sample payload onto a different test/node, so a test
// can deliver several distinct results without repeating 44 summary fields.
func payloadFor(t *testing.T, testID, nodeID, nodeName, testName, totalTime string) string {
	t.Helper()

	var resp Response
	if err := json.Unmarshal([]byte(fullPayload), &resp); err != nil {
		t.Fatalf("failed to parse baseline payload: %v", err)
	}

	resp.TestDetails.TestId = testID
	resp.TestDetails.NodeId = nodeID
	resp.TestDetails.NodeName = nodeName
	resp.TestDetails.TestName = testName
	resp.Summary.TotalTime = totalTime

	body, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to encode payload: %v", err)
	}

	return string(body)
}

// compareGolden asserts the collector's full output matches testdata/<name>, or
// rewrites that file when -update is passed.
func compareGolden(t *testing.T, c *Collector, g prometheus.Gatherer, name string) {
	t.Helper()

	path := filepath.Join("testdata", name)

	if *update {
		// testutil.CollectAndFormat filters out every family when no metric names
		// are given, so the golden output is encoded from the gathered families.
		families, err := g.Gather()
		if err != nil {
			t.Fatalf("gathering metrics failed: %v", err)
		}

		var buf bytes.Buffer
		enc := expfmt.NewEncoder(&buf, expfmt.NewFormat(expfmt.TypeTextPlain))
		for _, mf := range families {
			if err := enc.Encode(mf); err != nil {
				t.Fatalf("encoding metrics failed: %v", err)
			}
		}

		if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		return
	}

	expected, err := fs.ReadFile(goldenFS, name)
	if err != nil {
		t.Fatalf("failed to read expected metrics file: %v", err)
	}

	if err := testutil.CollectAndCompare(c, bytes.NewReader(expected)); err != nil {
		t.Errorf("gathered metrics did not match expected metrics: %v", err)
	}
}

// valuesByTestAndNode maps "<test_id>/<node_id>" to the value exported for the
// named metric.
func valuesByTestAndNode(t *testing.T, g prometheus.Gatherer, name string) map[string]float64 {
	t.Helper()

	families, err := g.Gather()
	if err != nil {
		t.Fatalf("gathering metrics failed: %v", err)
	}

	values := map[string]float64{}
	for _, mf := range families {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			labels := map[string]string{}
			for _, l := range m.GetLabel() {
				labels[l.GetName()] = l.GetValue()
			}
			values[labels[testIDLabel]+"/"+labels[nodeIDLabel]] = m.GetGauge().GetValue()
		}
	}

	return values
}

// singleValue returns the value of a metric family that carries exactly one
// unlabelled series, such as catchpoint_up or the webhook counters.
func singleValue(t *testing.T, g prometheus.Gatherer, name string) float64 {
	t.Helper()

	families, err := g.Gather()
	if err != nil {
		t.Fatalf("gathering metrics failed: %v", err)
	}

	for _, mf := range families {
		if mf.GetName() != name {
			continue
		}
		metrics := mf.GetMetric()
		if len(metrics) != 1 {
			t.Fatalf("expected exactly 1 series for %s, got %d", name, len(metrics))
		}
		if gauge := metrics[0].GetGauge(); gauge != nil {
			return gauge.GetValue()
		}
		return metrics[0].GetCounter().GetValue()
	}

	t.Fatalf("metric %s was not exported", name)
	return 0
}

func gatheredNames(t *testing.T, g prometheus.Gatherer) int {
	t.Helper()

	families, err := g.Gather()
	if err != nil {
		t.Fatalf("gathering metrics failed: %v", err)
	}

	return len(families)
}

func TestCollectorWithData(t *testing.T) {
	c, registry := newTestCollector(t, &Config{})

	if got := postWebhook(t, c, fullPayload).Code; got != http.StatusOK {
		t.Fatalf("expected status 200, got %d", got)
	}

	// 44 test metrics plus the exporter's own metrics.
	expectedMetricCount := 44 + alwaysOnMetricCount
	if got := gatheredNames(t, registry); got != expectedMetricCount {
		t.Errorf("expected %d metrics, got %d", expectedMetricCount, got)
	}

	compareGolden(t, c, registry, "all_metrics.prom")
}

func TestCollectorWithEmptyResponse(t *testing.T) {
	c, registry := newTestCollector(t, &Config{})

	if got := postWebhook(t, c, emptySummaryPayload).Code; got != http.StatusOK {
		t.Fatalf("expected status 200, got %d", got)
	}

	// Every summary field is empty, so only the exporter's own metrics remain.
	if got := gatheredNames(t, registry); got != alwaysOnMetricCount {
		t.Errorf("expected %d metrics, got %d", alwaysOnMetricCount, got)
	}

	compareGolden(t, c, registry, "empty_metrics.prom")
}

func TestCollectorWithPartialData(t *testing.T) {
	c, registry := newTestCollector(t, &Config{})

	if got := postWebhook(t, c, partialPayload).Code; got != http.StatusOK {
		t.Fatalf("expected status 200, got %d", got)
	}

	expectedMetricCount := 28 + alwaysOnMetricCount
	if got := gatheredNames(t, registry); got != expectedMetricCount {
		t.Errorf("expected %d metrics, got %d", expectedMetricCount, got)
	}

	compareGolden(t, c, registry, "partial_metrics.prom")
}

// TestCollectorExportsEveryTestAndNode is the regression test for the exporter
// only ever showing the most recently received result: Catchpoint posts one
// webhook per test run per node, and all of them must stay exported.
func TestCollectorExportsEveryTestAndNode(t *testing.T) {
	c, registry := newTestCollector(t, &Config{})

	posts := []struct {
		testID, nodeID, nodeName, testName, totalTime string
	}{
		{"123456", "1", "Bangalore, IN - Tata Teleservices", "My Homepage", "100"},
		{"123456", "2", "London, UK - BT", "My Homepage", "200"},
		{"654321", "1", "Bangalore, IN - Tata Teleservices", "My Checkout", "300"},
		{"654321", "2", "London, UK - BT", "My Checkout", "400"},
	}
	for _, p := range posts {
		body := payloadFor(t, p.testID, p.nodeID, p.nodeName, p.testName, p.totalTime)
		if got := postWebhook(t, c, body).Code; got != http.StatusOK {
			t.Fatalf("test %s node %s: expected status 200, got %d", p.testID, p.nodeID, got)
		}
	}

	if got := testutil.CollectAndCount(c, TotalTimeMetric); got != len(posts) {
		t.Errorf("expected %d %s series, got %d", len(posts), TotalTimeMetric, got)
	}

	want := map[string]float64{
		"123456/1": 100,
		"123456/2": 200,
		"654321/1": 300,
		"654321/2": 400,
	}
	if got := valuesByTestAndNode(t, registry, TotalTimeMetric); !reflect.DeepEqual(got, want) {
		t.Errorf("exported %s series %v, want %v", TotalTimeMetric, got, want)
	}

	if got := singleValue(t, registry, TrackedSeriesMetric); got != float64(len(posts)) {
		t.Errorf("expected %s to be %d, got %v", TrackedSeriesMetric, len(posts), got)
	}

	compareGolden(t, c, registry, "multi_series_metrics.prom")
}

// TestCollectorLatestResultWinsPerSeries checks a repeat run of the same
// test/node updates its series in place rather than adding a second one.
func TestCollectorLatestResultWinsPerSeries(t *testing.T) {
	c, registry := newTestCollector(t, &Config{})

	postWebhook(t, c, payloadFor(t, "123456", "1", "Bangalore, IN", "My Homepage", "100"))
	postWebhook(t, c, payloadFor(t, "123456", "1", "Bangalore, IN", "My Homepage", "500"))

	want := map[string]float64{"123456/1": 500}
	if got := valuesByTestAndNode(t, registry, TotalTimeMetric); !reflect.DeepEqual(got, want) {
		t.Errorf("exported %s series %v, want %v", TotalTimeMetric, got, want)
	}
}

func TestCollectorEvictsStaleSeries(t *testing.T) {
	c, registry := newTestCollector(t, &Config{StaleTimeout: 10 * time.Minute})

	clock := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return clock }

	postWebhook(t, c, payloadFor(t, "123456", "1", "Bangalore, IN", "My Homepage", "100"))

	// Still fresh.
	if got := singleValue(t, registry, TrackedSeriesMetric); got != 1 {
		t.Fatalf("expected 1 tracked series, got %v", got)
	}

	clock = clock.Add(11 * time.Minute)
	postWebhook(t, c, payloadFor(t, "654321", "2", "London, UK", "My Checkout", "300"))

	want := map[string]float64{"654321/2": 300}
	if got := valuesByTestAndNode(t, registry, TotalTimeMetric); !reflect.DeepEqual(got, want) {
		t.Errorf("after expiry exported %s series %v, want %v", TotalTimeMetric, got, want)
	}
	if got := singleValue(t, registry, TrackedSeriesMetric); got != 1 {
		t.Errorf("expected 1 tracked series after expiry, got %v", got)
	}
}

func TestCollectorKeepsSeriesWhenStaleTimeoutDisabled(t *testing.T) {
	c, registry := newTestCollector(t, &Config{})

	clock := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return clock }

	postWebhook(t, c, payloadFor(t, "123456", "1", "Bangalore, IN", "My Homepage", "100"))
	clock = clock.Add(30 * 24 * time.Hour)

	if got := singleValue(t, registry, TrackedSeriesMetric); got != 1 {
		t.Errorf("expected series to be retained with expiry disabled, got %v tracked", got)
	}
}

// TestDefaultStaleTimeoutRetiresDeletedTests pins the default retention: a result
// must survive well past any realistic test frequency, and must not survive
// forever. The exact window matters to operators, so a change here should be a
// deliberate one.
func TestDefaultStaleTimeoutRetiresDeletedTests(t *testing.T) {
	if DefaultStaleTimeout != 24*time.Hour {
		t.Errorf("DefaultStaleTimeout = %v, want 24h", DefaultStaleTimeout)
	}
	if got := NewConfig().StaleTimeout; got != DefaultStaleTimeout {
		t.Errorf("NewConfig().StaleTimeout = %v, want %v", got, DefaultStaleTimeout)
	}

	c, registry := newTestCollector(t, &Config{StaleTimeout: DefaultStaleTimeout})

	clock := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return clock }

	postWebhook(t, c, payloadFor(t, "123456", "1", "Bangalore, IN", "My Homepage", "100"))

	// A long gap in reporting must not drop a live series.
	clock = clock.Add(12 * time.Hour)
	if got := singleValue(t, registry, TrackedSeriesMetric); got != 1 {
		t.Errorf("series dropped after 12h, want it retained; got %v tracked", got)
	}

	// A deleted test must stop being exported within a day.
	clock = clock.Add(13 * time.Hour)
	if got := singleValue(t, registry, TrackedSeriesMetric); got != 0 {
		t.Errorf("series still exported after 25h, want it evicted; got %v tracked", got)
	}
}

func TestCollectorRejectsPayloadWithoutTestID(t *testing.T) {
	c, registry := newTestCollector(t, &Config{})

	body := `{"TestDetails":{"NodeId":"1"},"Summary":{"TotalTime":"100"}}`
	if got := postWebhook(t, c, body).Code; got != http.StatusBadRequest {
		t.Errorf("expected status 400 for a payload with no TestId, got %d", got)
	}

	if got := singleValue(t, registry, TrackedSeriesMetric); got != 0 {
		t.Errorf("expected no tracked series, got %v", got)
	}
	if got := singleValue(t, registry, WebhookRequestsMetric); got != 1 {
		t.Errorf("expected 1 webhook request, got %v", got)
	}
	if got := singleValue(t, registry, WebhookErrorsMetric); got != 1 {
		t.Errorf("expected 1 webhook error, got %v", got)
	}
}

func TestCollectorRejectsNonPOST(t *testing.T) {
	c, registry := newTestCollector(t, &Config{})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/webhook", nil)
	w := httptest.NewRecorder()
	c.HandleWebhook(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET, got %d", w.Code)
	}
	if got := singleValue(t, registry, WebhookErrorsMetric); got != 1 {
		t.Errorf("expected 1 webhook error, got %v", got)
	}
}

// TestCollectorGoesDownOnBadPayloadAndRecovers covers the point of the up gauge:
// a template the exporter cannot ingest has to show as down, and fixing it has to
// show as up again without a restart.
func TestCollectorGoesDownOnBadPayloadAndRecovers(t *testing.T) {
	c, registry := newTestCollector(t, &Config{})

	if got := singleValue(t, registry, UpMetric); got != 1 {
		t.Errorf("expected %s to be 1 before any webhook, got %v", UpMetric, got)
	}

	if got := postWebhook(t, c, `{"TestDetails":`).Code; got != http.StatusBadRequest {
		t.Errorf("expected status 400 for malformed JSON, got %d", got)
	}

	if got := singleValue(t, registry, UpMetric); got != 0 {
		t.Errorf("expected %s to be 0 after a bad payload, got %v", UpMetric, got)
	}
	if got := singleValue(t, registry, WebhookErrorsMetric); got != 1 {
		t.Errorf("expected 1 webhook error, got %v", got)
	}

	// A good payload after a bad one is exported, and clears the down state.
	postWebhook(t, c, payloadFor(t, "123456", "1", "Bangalore, IN", "My Homepage", "100"))
	if got := singleValue(t, registry, TrackedSeriesMetric); got != 1 {
		t.Errorf("expected 1 tracked series, got %v", got)
	}
	if got := singleValue(t, registry, UpMetric); got != 1 {
		t.Errorf("expected %s to return to 1 after a good payload, got %v", UpMetric, got)
	}
}

// TestCollectorEnforcesSeriesLimit covers the memory bound: once the limit is
// reached, tests already being exported keep updating but unseen ones are
// dropped rather than growing the map without limit.
func TestCollectorEnforcesSeriesLimit(t *testing.T) {
	c, registry := newTestCollector(t, &Config{MaxSeries: 2})

	postWebhook(t, c, payloadFor(t, "1", "1", "Bangalore, IN", "First", "100"))
	postWebhook(t, c, payloadFor(t, "2", "1", "Sydney, AU", "Second", "200"))

	if got := postWebhook(t, c, payloadFor(t, "3", "1", "Perth, AU", "Third", "300")).Code; got != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 once the series limit is reached, got %d", got)
	}
	if got := singleValue(t, registry, TrackedSeriesMetric); got != 2 {
		t.Errorf("expected the series limit to hold tracked series at 2, got %v", got)
	}
	if got := singleValue(t, registry, SeriesDroppedMetric); got != 1 {
		t.Errorf("expected 1 dropped series, got %v", got)
	}

	// A series already tracked keeps updating at the limit.
	postWebhook(t, c, payloadFor(t, "1", "1", "Bangalore, IN", "First", "999"))
	if got := singleValue(t, registry, TrackedSeriesMetric); got != 2 {
		t.Errorf("expected tracked series to stay at 2, got %v", got)
	}
	if got := singleValue(t, registry, SeriesDroppedMetric); got != 1 {
		t.Errorf("expected no further drops, got %v", got)
	}
}

// TestCollectorAppliesDefaultSeriesLimit checks that a zero-valued Config picks up
// the default rather than silently running uncapped.
func TestCollectorAppliesDefaultSeriesLimit(t *testing.T) {
	c, _ := newTestCollector(t, &Config{})
	if c.maxSeries != DefaultMaxSeries {
		t.Errorf("expected the default series limit of %d, got %d", DefaultMaxSeries, c.maxSeries)
	}
}

// TestCollectorSeriesLimitDisabled checks that a negative limit is honoured as
// "unlimited" rather than falling back to the default.
func TestCollectorSeriesLimitDisabled(t *testing.T) {
	c, registry := newTestCollector(t, &Config{MaxSeries: -1})

	for _, id := range []string{"1", "2", "3", "4"} {
		if got := postWebhook(t, c, payloadFor(t, id, "1", "Bangalore, IN", "Test", "100")).Code; got != http.StatusOK {
			t.Fatalf("expected status 200 with the limit disabled, got %d", got)
		}
	}

	if got := singleValue(t, registry, TrackedSeriesMetric); got != 4 {
		t.Errorf("expected 4 tracked series with the limit disabled, got %v", got)
	}
	if got := singleValue(t, registry, SeriesDroppedMetric); got != 0 {
		t.Errorf("expected no dropped series with the limit disabled, got %v", got)
	}
}

func TestCollectorRejectsOversizedBody(t *testing.T) {
	c, _ := newTestCollector(t, &Config{})

	body := `{"TestDetails":{"TestName":"` + strings.Repeat("a", DefaultMaxBodyBytes+1) + `"}}`
	if got := postWebhook(t, c, body).Code; got != http.StatusBadRequest {
		t.Errorf("expected status 400 for an oversized body, got %d", got)
	}
}

// TestCollectorHonoursConfiguredBodyLimit checks that MaxBodyBytes overrides the
// default in both directions, and that an unset value falls back to the default.
func TestCollectorHonoursConfiguredBodyLimit(t *testing.T) {
	padded := func(n int) string {
		return `{"TestDetails":{"TestId":"1","TestName":"` + strings.Repeat("a", n) + `"}}`
	}

	t.Run("below the configured limit", func(t *testing.T) {
		c, _ := newTestCollector(t, &Config{MaxBodyBytes: 4096})
		if got := postWebhook(t, c, padded(1024)).Code; got != http.StatusOK {
			t.Errorf("expected status 200, got %d", got)
		}
	})

	t.Run("above the configured limit", func(t *testing.T) {
		c, _ := newTestCollector(t, &Config{MaxBodyBytes: 4096})
		if got := postWebhook(t, c, padded(8192)).Code; got != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", got)
		}
	})

	t.Run("a body the default would reject", func(t *testing.T) {
		c, _ := newTestCollector(t, &Config{MaxBodyBytes: 4 * DefaultMaxBodyBytes})
		if got := postWebhook(t, c, padded(DefaultMaxBodyBytes+1)).Code; got != http.StatusOK {
			t.Errorf("expected status 200, got %d", got)
		}
	})

	t.Run("unset falls back to the default", func(t *testing.T) {
		c, _ := newTestCollector(t, &Config{})
		if got := c.maxBodyBytes; got != DefaultMaxBodyBytes {
			t.Errorf("maxBodyBytes = %d, want %d", got, DefaultMaxBodyBytes)
		}
	})
}

// TestCollectorConcurrentWebhooksAndScrapes exercises the lock between the HTTP
// handler goroutines and the scrape goroutine. Meaningful under -race.
func TestCollectorConcurrentWebhooksAndScrapes(t *testing.T) {
	c, registry := newTestCollector(t, &Config{StaleTimeout: time.Hour})

	const writers, readers, iterations = 4, 4, 50

	// Payloads are built up front: the helpers call t.Fatalf, which must not run
	// off the test's own goroutine.
	bodies := make([]string, writers)
	for w := range bodies {
		nodeID := strconv.Itoa(w + 1)
		bodies[w] = payloadFor(t, "123456", nodeID, "Node "+nodeID, "My Homepage", "100")
	}

	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(body string) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				req := httptest.NewRequest(http.MethodPost, "http://example.com/webhook", strings.NewReader(body))
				c.HandleWebhook(httptest.NewRecorder(), req)
			}
		}(bodies[w])
	}
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				if _, err := registry.Gather(); err != nil {
					t.Errorf("gathering metrics failed: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	if got := singleValue(t, registry, TrackedSeriesMetric); got != writers {
		t.Errorf("expected %d tracked series, got %v", writers, got)
	}
}

func TestParseMetricValue(t *testing.T) {
	cases := []struct {
		in      string
		want    float64
		wantErr bool
	}{
		{in: "6591", want: 6591},
		{in: "0", want: 0},
		{in: "1.5", want: 1.5},
		{in: "False", want: 0},
		{in: "false", want: 0},
		{in: "True", want: 1},
		{in: "true", want: 1},
		{in: "", wantErr: true},
		{in: "not-a-number", wantErr: true},
	}

	for _, tc := range cases {
		got, err := parseMetricValue(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseMetricValue(%q): expected an error, got %v", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseMetricValue(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseMetricValue(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

const fullPayload = `{
    "TestDetails": {
        "TestName": "My Homepage",
        "TypeId": "0",
        "MonitorTypeId": "11",
        "TestId": "123456",
        "ReportWindow": "123123123210000000",
        "NodeId": "12345",
        "NodeName": "Bangalore, IN - Tata Teleservices",
        "Asn": "12345",
        "DivisionId": "1234",
        "ClientId": "123"
    },
    "Summary": {
        "Timestamp": "20240502212044798",
        "TotalTime": "6591",
        "Connect": "11",
        "Dns": "24",
        "ContentLoad": "6285",
        "Load": "598",
        "Redirect": "309",
        "SSL": "19",
        "Wait": "517",
        "Client": "167",
        "DocumentComplete": "4406",
        "RenderStart": "1554",
        "ResponseContent": "101392",
        "ResponseHeaders": "2315",
        "TotalContent": "1567691",
        "TotalHeaders": "54175",
        "AnyError": "False",
        "ConnectionError": "False",
        "DNSError": "False",
        "LoadError": "False",
        "TimeoutError": "False",
        "TransactionError": "False",
        "ErrorObjectsLoaded": "False",
        "ImageContentType": "542482",
        "ScriptContentType": "593927",
        "HTMLContentType": "104773",
        "CSSContentType": "237942",
        "FontContentType": "138677",
        "MediaContentType": "123456",
        "XMLContentType": "123456",
        "OtherContentType": "123456",
        "ConnectionsCount": "15",
        "HostsCount": "22",
        "FailedRequestsCount": "0",
        "RequestsCount": "59",
        "RedirectionsCount": "4",
        "CachedCount": "0",
        "ImageCount": "11",
        "ScriptCount": "21",
        "HTMLCount": "4",
        "CSSCount": "4",
        "FontCount": "4",
        "XMLCount": "0",
        "MediaCount": "0",
        "TracepointsCount": "0"
    }
}`

const emptySummaryPayload = `{
    "TestDetails": {
        "TestName": "My Homepage",
        "TypeId": "0",
        "MonitorTypeId": "11",
        "TestId": "123456",
        "ReportWindow": "123123123210000000",
        "NodeId": "12345",
        "NodeName": "Bangalore, IN - Tata Teleservices",
        "Asn": "12345",
        "DivisionId": "1234",
        "ClientId": "123"
    },
    "Summary": {
        "Timestamp": "20240502212044798",
        "TotalTime": "",
        "Connect": "",
        "Dns": "",
        "ContentLoad": "",
        "Load": "",
        "Redirect": "",
        "SSL": "",
        "Wait": "",
        "Client": "",
        "DocumentComplete": "",
        "RenderStart": "",
        "ResponseContent": "",
        "ResponseHeaders": "",
        "TotalContent": "",
        "TotalHeaders": "",
        "AnyError": "",
        "ConnectionError": "",
        "DNSError": "",
        "LoadError": "",
        "TimeoutError": "",
        "TransactionError": "",
        "ErrorObjectsLoaded": "",
        "ImageContentType": "",
        "ScriptContentType": "",
        "HTMLContentType": "",
        "CSSContentType": "",
        "FontContentType": "",
        "MediaContentType": "",
        "XMLContentType": "",
        "OtherContentType": "",
        "ConnectionsCount": "",
        "HostsCount": "",
        "FailedRequestsCount": "",
        "RequestsCount": "",
        "RedirectionsCount": "",
        "CachedCount": "",
        "ImageCount": "",
        "ScriptCount": "",
        "HTMLCount": "",
        "CSSCount": "",
        "FontCount": "",
        "XMLCount": "",
        "MediaCount": "",
        "TracepointsCount": ""
    }
}`

const partialPayload = `{
    "TestDetails": {
        "TestName": "My Homepage",
        "TypeId": "0",
        "MonitorTypeId": "11",
        "TestId": "123456",
        "ReportWindow": "123123123210000000",
        "NodeId": "12345",
        "NodeName": "Bangalore, IN - Tata Teleservices",
        "Asn": "12345",
        "DivisionId": "1234",
        "ClientId": "123"
    },
    "Summary": {
        "Timestamp": "20240502212044798",
        "TotalTime": "6591",
        "Connect": "11",
        "Dns": "",
        "ContentLoad": "6285",
        "Load": "",
        "Redirect": "309",
        "SSL": "",
        "Wait": "517",
        "Client": "",
        "DocumentComplete": "",
        "RenderStart": "1554",
        "ResponseContent": "",
        "ResponseHeaders": "2315",
        "TotalContent": "",
        "TotalHeaders": "",
        "AnyError": "False",
        "ConnectionError": "False",
        "DNSError": "False",
        "LoadError": "False",
        "TimeoutError": "False",
        "TransactionError": "False",
        "ErrorObjectsLoaded": "False",
        "ImageContentType": "",
        "ScriptContentType": "",
        "HTMLContentType": "",
        "CSSContentType": "",
        "FontContentType": "",
        "MediaContentType": "",
        "XMLContentType": "",
        "OtherContentType": "",
        "ConnectionsCount": "15",
        "HostsCount": "22",
        "FailedRequestsCount": "0",
        "RequestsCount": "59",
        "RedirectionsCount": "4",
        "CachedCount": "0",
        "ImageCount": "11",
        "ScriptCount": "21",
        "HTMLCount": "4",
        "CSSCount": "4",
        "FontCount": "4",
        "XMLCount": "0",
        "MediaCount": "0",
        "TracepointsCount": "0"
    }
}`
