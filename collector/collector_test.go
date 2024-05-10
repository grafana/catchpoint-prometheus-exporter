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
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/promlog"
)

func TestCollectorWithData(t *testing.T) {
	logger := promlog.New(&promlog.Config{})
	collector := NewCollector(logger, &Config{})
	registry := prometheus.NewPedanticRegistry()
	if err := registry.Register(collector); err != nil {
		t.Fatal("failed to register collector:", err)
	}

	jsonData := `{
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
	req := httptest.NewRequest("POST", "http://example.com/webhook", strings.NewReader(jsonData))
	w := httptest.NewRecorder()
	collector.HandleWebhook(w, req)

	// Gather all metrics and assert the count
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("gathering metrics failed: %v", err)
	}

	// Define expected metric count
	expectedMetricCount := 45 // 44 metrics + 1(up) for the collector
	if len(metrics) != expectedMetricCount {
		t.Errorf("expected %d metrics, got %d", expectedMetricCount, len(metrics))
	}

	// Read expected metrics from file
	expectedFile, err := os.Open(filepath.Join("testdata", "all_metrics.prom"))
	if err != nil {
		t.Fatalf("failed to open expected metrics file: %v", err)
	}
	defer expectedFile.Close()

	// Compare gathered metrics against expected metrics
	if err := testutil.GatherAndCompare(registry, expectedFile); err != nil {
		t.Errorf("gathered metrics did not match expected metrics: %v", err)
	}

	// Unregister to clean up after the test
	registry.Unregister(collector)
}

func TestCollectorWithEmptyResponse(t *testing.T) {
	logger := promlog.New(&promlog.Config{})
	collector := NewCollector(logger, &Config{})
	registry := prometheus.NewPedanticRegistry()
	if err := registry.Register(collector); err != nil {
		t.Fatal("failed to register collector:", err)
	}

	jsonData := `{
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
	req := httptest.NewRequest("POST", "http://example.com/webhook", strings.NewReader(jsonData))
	w := httptest.NewRecorder()
	httpHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	collector.HandleWebhook(w, req)

	// Serve HTTP to trigger metric collection
	httpHandler.ServeHTTP(w, req)

	// Gather all metrics and assert the count
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("gathering metrics failed: %v", err)
	}

	// Define expected metric count
	expectedMetricCount := 1 // Assuming 1 'up' metric for the collector
	if len(metrics) != expectedMetricCount {
		t.Errorf("expected %d metrics, got %d", expectedMetricCount, len(metrics))
	}

	// Read expected metrics from file
	expectedFile, err := os.Open(filepath.Join("testdata", "empty_metrics.prom"))
	if err != nil {
		t.Fatalf("failed to open expected metrics file: %v", err)
	}
	defer expectedFile.Close()

	// Compare gathered metrics against expected metrics
	if err := testutil.GatherAndCompare(registry, expectedFile); err != nil {
		t.Errorf("gathered metrics did not match expected metrics: %v", err)
	}

	// Unregister to clean up after the test
	registry.Unregister(collector)
}
