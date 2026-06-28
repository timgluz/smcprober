# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

All common tasks use [Task](https://taskfile.dev):

```bash
task run:exporter       # Run Prometheus exporter locally (port 8080)
task run:job            # Run one-shot job check
task run:downloader     # Download device data from API
task lint:go            # Run golangci-lint
task lint:go:fix        # Run golangci-lint with auto-fix
task lint:all           # Run all linters (go, helm, markdown)
task test:alerts        # Run Prometheus alert rule tests (uses mise-managed promtool; run `mise install` first)
task build:docker       # Build Docker image via nerdctl
task template:helm      # Template Helm chart and dry-run apply
task generate:dashboards # Regenerate Grafana dashboard JSON
```

Run a single binary directly:
```bash
go run cmd/smcexporter/main.go --config configs/config-exporter-dev.json --dotenv .env
go run cmd/smcjob/main.go --config configs/config.json --dotenv .env
```

## Architecture

The project produces four binaries from `cmd/`: `smcexporter` (Prometheus exporter), `smcjob` (scheduled alert job), `smcdownload` (data downloader), and `gen-device-dashboard` (Grafana dashboard generator).

### Core packages

**`smartcitizen/`** — SmartCitizen API layer
- `Provider` interface: `GetMe`, `GetDevice`, `Authenticate`, `Ping` — the `HTTPProvider` implementation uses instrumented HTTP
- `APIExporter` — polls the API on a ticker, converts device data to Prometheus metrics via `Converter`s registered in a `CombinedConverter`
- Converters: `DeviceInfoConverter`, `DeviceStateConverter`, `DeviceSensorConverter`, `DeviceSensorInfoConverter` — each implements `metric.Converter`

**`metric/`** — Prometheus metric abstraction
- `Registry` interface + `NamespacedRegistry` implementation — thread-safe, lazy GetOrCreate pattern; all metrics are namespaced (default: `smartcitizen`)
- `Converter` interface: `Convert(Registry, any) error` — typed converters registered per Go type via `CombinedConverter`
- `SensorMetricMapping` — maps SmartCitizen sensor names to Prometheus metric names and categories (configured via JSON `sensor_mapping`)

**`alert/`** — In-process alerting engine (used by `smcjob`)
- `AlertingEngine` evaluates `AlertRule`s against a `Metric` value; each rule has a `Condition` func and `Action` func
- Separate from Prometheus alerting rules (which live in `helm/alerts/` and are tested via `promtool`)

**`ntfy/`** — NTFY.sh push notification provider used by alert actions

**`httpclient/`** — Wraps `http.Client` with an instrumented transport that records Prometheus metrics for outbound HTTP calls

### Configuration

Each binary takes `--config <json-file>` and optional `--dotenv <env-file>`. The exporter's `AppConfig` embeds `smartcitizen.Config` under the `smartcitizen` key and a `sensor_mapping` map for metric name translation. Credentials are always resolved from env vars (`SMARTCITIZEN_USERNAME`, `SMARTCITIZEN_TOKEN`).

### Deployment

- Docker image uses `nerdctl` (not `docker`) — see `DOCKER_BIN` var in Taskfile
- Helm chart in `helm/` with ServiceMonitor support for Prometheus Operator
- Tekton CI pipelines in `helm/pipelines/` for multi-arch builds
- Grafana dashboards generated from code (`cmd/gen-device-dashboard`) and committed to `helm/dashboards/`
- Alert YAML rules in `helm/alerts/`, tested with `promtool test rules tests/alerts/test_*.yaml`
