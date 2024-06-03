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

// TestDetails represents detailed information about a test run.
type TestDetails struct {
	TestName      string `json:"TestName"`
	TypeId        string `json:"TypeId"`
	MonitorTypeId string `json:"MonitorTypeId"`
	TestId        string `json:"TestId"`
	ReportWindow  string `json:"ReportWindow"`
	NodeId        string `json:"NodeId"`
	NodeName      string `json:"NodeName"`
	Asn           string `json:"Asn"`
	DivisionId    string `json:"DivisionId"`
	ClientId      string `json:"ClientId"`
}

// Summary represents the 44 aggregated results of a test run.
type Summary struct {
	Timestamp           string `json:"Timestamp"`
	TotalTime           string `json:"TotalTime"`
	Connect             string `json:"Connect"`
	Dns                 string `json:"Dns"`
	ContentLoad         string `json:"ContentLoad"`
	Load                string `json:"Load"`
	Redirect            string `json:"Redirect"`
	SSL                 string `json:"SSL"`
	Wait                string `json:"Wait"`
	Client              string `json:"Client"`
	DocumentComplete    string `json:"DocumentComplete"`
	RenderStart         string `json:"RenderStart"`
	ResponseContent     string `json:"ResponseContent"`
	ResponseHeaders     string `json:"ResponseHeaders"`
	TotalContent        string `json:"TotalContent"`
	TotalHeaders        string `json:"TotalHeaders"`
	AnyError            string `json:"AnyError"`
	ConnectionError     string `json:"ConnectionError"`
	DNSError            string `json:"DNSError"`
	LoadError           string `json:"LoadError"`
	TimeoutError        string `json:"TimeoutError"`
	TransactionError    string `json:"TransactionError"`
	ErrorObjectsLoaded  string `json:"ErrorObjectsLoaded"`
	ImageContentType    string `json:"ImageContentType"`
	ScriptContentType   string `json:"ScriptContentType"`
	HTMLContentType     string `json:"HTMLContentType"`
	CSSContentType      string `json:"CSSContentType"`
	FontContentType     string `json:"FontContentType"`
	MediaContentType    string `json:"MediaContentType"`
	XMLContentType      string `json:"XMLContentType"`
	OtherContentType    string `json:"OtherContentType"`
	ConnectionsCount    string `json:"ConnectionsCount"`
	HostsCount          string `json:"HostsCount"`
	FailedRequestsCount string `json:"FailedRequestsCount"`
	RequestsCount       string `json:"RequestsCount"`
	RedirectionsCount   string `json:"RedirectionsCount"`
	CachedCount         string `json:"CachedCount"`
	ImageCount          string `json:"ImageCount"`
	ScriptCount         string `json:"ScriptCount"`
	HTMLCount           string `json:"HTMLCount"`
	CSSCount            string `json:"CSSCount"`
	FontCount           string `json:"FontCount"`
	XMLCount            string `json:"XMLCount"`
	MediaCount          string `json:"MediaCount"`
	TracepointsCount    string `json:"TracepointsCount"`
}

// Response represents the full response structure from the API.
type Response struct {
	TestDetails TestDetails `json:"TestDetails"`
	Summary     Summary     `json:"Summary"`
}
