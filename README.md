# catchpoint-prometheus-exporter
A golang based Prometheus metrics exporter for Catchpoint

Catchpoint Prometheus Exporter allows you to integrate Catchpoint's Tests Data Webhook into Prometheus, enabling you to monitor performance metrics directly through your Prometheus setup.

## Configuration

The exporter is configurable via command-line flags or environment variables. Here are the key configuration options:

- `--port` or `CATCHPOINT_EXPORTER_PORT`: Sets the port on which the exporter will run (default: `9090`).
- `--webhook-path` or `CATCHPOINT_WEBHOOK_PATH`: Defines the path where the exporter will receive webhook data from Catchpoint (default: `/catchpoint-webhook`).
- `--verbose` or `CATCHPOINT_VERBOSE`: Enables verbose logging to provide more detailed output for debugging purposes (default: `false`).
- `--stale-timeout` or `CATCHPOINT_STALE_TIMEOUT`: Stops exporting a test/node result this long after its last webhook (default: `24h`; `0s` keeps results forever). See [Staleness](#staleness).

## Environment Variables

You can also configure the exporter using the following environment variables:

- `CATCHPOINT_EXPORTER_PORT`: Overrides the default port.
- `CATCHPOINT_WEBHOOK_PATH`: Overrides the default webhook path.
- `CATCHPOINT_VERBOSE`: Set to `true` to enable verbose logging.
- `CATCHPOINT_STALE_TIMEOUT`: Overrides the staleness timeout, e.g. `90m`. Defaults to `24h`.

## Metrics

The exporter provides a range of metrics, reflecting various performance aspects captured by Catchpoint. A complete list of available metrics can be found in the file [/collector/testdata/all_metrics.prom](/collector/testdata/all_metrics.prom).

Catchpoint sends one webhook per test run **per node**, and the exporter keeps the most recent result for every test/node combination it has seen. Each result becomes its own set of series, identified by the `test_id` and `node_id` labels, so a scrape of `/metrics` returns every test from every node — not only whichever one reported last. See [/collector/testdata/multi_series_metrics.prom](/collector/testdata/multi_series_metrics.prom) for what two tests across two nodes look like.

Every test metric carries these labels: `test_id`, `node_id`, `node_name`, `test_name`, `client_id`, `asn`, `division_id`, `monitor_type_id`, `type_id`.

The exporter also reports on itself:

- `catchpoint_up`: always `1` while the exporter is able to serve a scrape.
- `catchpoint_webhook_requests_total`: webhook requests received.
- `catchpoint_webhook_errors_total`: webhook requests that could not be processed (wrong method, undecodable body, or a payload with no `TestId`).
- `catchpoint_tracked_series`: number of test/node combinations currently being exported.

If `catchpoint_tracked_series` stays at `1` while you expect many tests, the webhook template is most likely not sending `TestId`/`NodeId` — check that your template matches [template.json](/template.json), since those two fields are what separate one test's results from another's. Payloads without a `TestId` are rejected with HTTP 400 and counted in `catchpoint_webhook_errors_total`.

## Staleness

The exporter keeps the last result of every test/node it has seen and re-exports it on every scrape. Metric timestamps are always scrape time, not the time of the Catchpoint test run, so a result that arrived hours ago is indistinguishable downstream from one that arrived seconds ago — nothing in the exposed data reveals that a test has stopped reporting.

`--stale-timeout` bounds that. A test/node result stops being exported once this long has passed since its last webhook, so a test you delete in Catchpoint disappears from `/metrics` rather than freezing at its final value forever. The default is `24h`: long enough that a run of missed test executions never drops a live series, short enough that deleted tests retire within a day.

Set it lower if you want deleted or broken tests to disappear sooner — a few multiples of your slowest test frequency is the rule of thumb, so `--stale-timeout=90m` suits tests running every 15–30 minutes. **Never set it below your slowest test frequency**, or live series will flap in and out between runs.

`--stale-timeout=0s` disables eviction entirely and restores the pre-`24h` behaviour of keeping every result until the process restarts. The exporter logs a warning at startup when you do. Note that eviction happens during a scrape, so an exporter that receives webhooks while nothing scrapes `/metrics` accumulates results regardless of this setting.

## Webhook Setup

To receive data from Catchpoint, you need to set up a webhook that points to the URL where this exporter is running. Follow these steps to configure the webhook in Catchpoint:
1. Log in to your Catchpoint account.
2. Navigate to Settings > API > Test Data Webhooks
3. Click Add URL
4. Set the "URL" to `http://<your_exporter_address>:<port>/catchpoint-webhook`, where `<your_exporter_address>` is the IP address or domain of your server where the exporter is running, and `<port>` is configured as per the `CATCHPOINT_EXPORTER_PORT`.
5. Add a [template](/template.json) json to target the selected metrics used in this Prometheus exporter.
6. Save the webhook configuration.

Next you need to set up the webhook for tests:
1. Navigate to Control Center > Tests > Select the Product Properties(or multiple) within the nav section
2. Under the Product Properties section, enable the Test Data Webhook and select the Template you just created
3. Next, under Navigate to Control Center > Tests you will see a list of test names
4. Click on each test name you wish to monitor which brings a window up
5. Under More Settings, enable the `Test Data Webhook`
6. Under Targeting & Scheduling, set the desired Frequency

## Running the Exporter

To start the exporter, you can use the following command:

```bash
go build -o catchpoint-exporter ./cmd/catchpoint-exporter/main.go

./catchpoint-exporter  --port="9090" --webhook-path="/catchpoint-webhook"
```

This command starts the exporter on port 9090, sets up `/catchpoint-webhook` as the endpoint for receiving webhook data, and enables verbose logging.

