# catchpoint-prometheus-exporter
A golang based Prometheus metrics exporter for Catchpoint

Catchpoint Prometheus Exporter allows you to integrate Catchpoint's Tests Data Webhook into Prometheus, enabling you to monitor performance metrics directly through your Prometheus setup.

## Configuration

The exporter is configurable via command-line flags or environment variables. Here are the key configuration options:

- `--port` or `CATCHPOINT_EXPORTER_PORT`: Sets the port on which the exporter will run (default: `9090`).
- `--webhook-path` or `CATCHPOINT_WEBHOOK_PATH`: Defines the path where the exporter will receive webhook data from Catchpoint (default: `/catchpoint-webhook`).
- `--verbose` or `CATCHPOINT_VERBOSE`: Enables verbose logging to provide more detailed output for debugging purposes (default: `false`).
- `--stale-timeout` or `CATCHPOINT_STALE_TIMEOUT`: How long a test/node result is kept after its last webhook (default: `24h`; `0s` keeps them forever). See [Staleness](#staleness).
- `--max-body-bytes` or `CATCHPOINT_MAX_BODY_BYTES`: Largest webhook body accepted, in bytes (default: `1048576`).
- `--max-series` or `CATCHPOINT_MAX_SERIES`: Maximum number of test/node combinations kept (default: `10000`; a negative value disables the limit). See [Series limit](#series-limit).

## Environment Variables

You can also configure the exporter using the following environment variables:

- `CATCHPOINT_EXPORTER_PORT`: Overrides the default port.
- `CATCHPOINT_WEBHOOK_PATH`: Overrides the default webhook path.
- `CATCHPOINT_VERBOSE`: Set to `true` to enable verbose logging.
- `CATCHPOINT_STALE_TIMEOUT`: Overrides the staleness timeout, e.g. `90m`.
- `CATCHPOINT_MAX_BODY_BYTES`: Overrides the maximum webhook body size.
- `CATCHPOINT_MAX_SERIES`: Overrides the maximum number of test/node combinations.

## Metrics

The exporter provides a range of metrics, reflecting various performance aspects captured by Catchpoint. A complete list of available metrics can be found in the file [/collector/testdata/all_metrics.prom](/collector/testdata/all_metrics.prom).

Catchpoint sends one webhook per test run **per node**, and the exporter keeps the most recent result for every test/node combination it has seen. Each result becomes its own set of series, identified by the `test_id` and `node_id` labels, so a scrape of `/metrics` returns every test from every node — not only whichever one reported last. See [/collector/testdata/multi_series_metrics.prom](/collector/testdata/multi_series_metrics.prom) for what two tests across two nodes look like.

Every test metric carries these labels: `test_id`, `node_id`, `node_name`, `test_name`, `client_id`, `asn`, `division_id`, `monitor_type_id`, `type_id`.

The exporter also reports on itself:

- `catchpoint_up`: `1` while the exporter is running; `0` if the last webhook body could not be decoded, until one decodes again.
- `catchpoint_webhook_requests_total`: webhook requests received.
- `catchpoint_webhook_errors_total`: webhook requests that could not be processed (wrong method, undecodable body, a payload with no `TestId`, or the series limit being reached).
- `catchpoint_series_dropped_total`: results dropped after reaching the series limit. See [Series limit](#series-limit).
- `catchpoint_tracked_series`: number of test/node combinations currently being exported.

Compare `catchpoint_tracked_series` against what you expect, using [template.json](/template.json) as the reference:

- Stuck at `0` while `catchpoint_webhook_requests_total` climbs: the template is not sending `TestId`. Those payloads are rejected with HTTP 400 and counted in `catchpoint_webhook_errors_total`.
- Equal to your number of tests, rather than tests × nodes: the template is not sending `NodeId`, so every node of a test collapses onto one series and overwrites the others.

## Series limit

The exporter keeps one entry per test/node combination it has seen, so an unsupervised deployment grows with whatever Catchpoint sends it. `--max-series` bounds that.

At the limit, combinations already tracked keep updating; only unseen ones are rejected, with HTTP 503 and a bump to `catchpoint_series_dropped_total`. Existing tests keep reporting rather than the exporter going blind all at once.

The default of `10000` is far above any realistic Catchpoint account, so reaching it usually means something is wrong — a template sending unexpected `TestId` values, or `--stale-timeout=0s` letting retired tests accumulate. A negative value disables the limit and logs a warning at startup.

## Staleness

The exporter re-exports the last result of every test/node on each scrape, timestamped at scrape time rather than test-run time. Nothing in the exposed data shows that a test has stopped reporting, so without eviction a deleted test keeps exporting its final value forever.

`--stale-timeout` drops a result once that long has passed since its last webhook. Set it to a few multiples of your slowest test frequency — `90m` suits tests running every 15–30 minutes. Below that frequency, live series flap between runs.

`--stale-timeout=0s` disables eviction and logs a warning at startup. Eviction happens during a scrape, so results still accumulate if nothing scrapes `/metrics`.

> **Changed in this release:** the default was previously `0s` (keep everything until restart) and is now `24h`. Tests that stopped reporting over a day ago will disappear from `/metrics` after upgrading. Set `--stale-timeout=0s` to restore the old behaviour.

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

