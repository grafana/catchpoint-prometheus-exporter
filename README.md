# catchpoint-prometheus-exporter
A golang based Prometheus metrics exporter for Catchpoint

Catchpoint Prometheus Exporter allows you to integrate Catchpoint's Tests Data Webhook into Prometheus, enabling you to monitor performance metrics directly through your Prometheus setup.

## Configuration

The exporter is configurable via command-line flags or environment variables. Here are the key configuration options:

- `--port` or `CATCHPOINT_EXPORTER_PORT`: Sets the port on which the exporter will run (default: `9090`).
- `--webhook-path` or `CATCHPOINT_WEBHOOK_PATH`: Defines the path where the exporter will receive webhook data from Catchpoint (default: `/catchpoint-webhook`).
- `--verbose` or `CATCHPOINT_VERBOSE`: Enables verbose logging to provide more detailed output for debugging purposes (default: `false`).

## Environment Variables

You can also configure the exporter using the following environment variables:

- `CATCHPOINT_EXPORTER_PORT`: Overrides the default port.
- `CATCHPOINT_WEBHOOK_PATH`: Overrides the default webhook path.
- `CATCHPOINT_VERBOSE`: Set to `true` to enable verbose logging.

## Metrics

The exporter provides a range of metrics, reflecting various performance aspects captured by Catchpoint. A complete list of available metrics can be found in the file [/collector/testdata/all_metrics.prom](/collector/testdata/all_metrics.prom).

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
