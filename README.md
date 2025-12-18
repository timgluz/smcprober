# SmartCitizen Prober

SmartCitizen Prober is a small service for checking the status of
SmartCitizen devices and report their connectivity and sensors' status.

It's splitted into 3 different binaries based on use case:

1. smcdownloader: Download data from SmartCitizen devices and store it
   locally for further processing.

2. smcjob: Tool that could be scheduled periodically to check device status
   and send notifications if devices are down or not sending data.

3. smcprober: Prometheus exporter to expose device metrics for monitoring.
   It is designed to be deployed in Kubernetes using Helm and could be used
   for advanced monitoring and alerting.

## Status

Project is in early development stage and developed as sideproject.
Features and APIs may change without notice.

## Features

- Periodically checks the battery level of SmartCitizen devices.

## Getting Started

### Prerequisites

- [Go 1.x](https://golang.org/doc/install) (for local development)
- [Task](https://taskfile.dev) - Task runner (install via
  `brew install go-task/tap/go-task` or see
  [installation guide](https://taskfile.dev/installation/))
- [Docker](https://docs.docker.com/get-docker/) or
  [nerdctl](https://github.com/containerd/nerdctl)
  (for containerized deployment)
- [Helm](https://helm.sh/docs/intro/install/) (for Kubernetes deployment)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
  (for Kubernetes deployment)

### Installation

1. Clone the repository:

```bash
git clone <repository-url>
cd smcprober
```

1. Install Go dependencies:

```bash
go mod download
```

### Configuration

1. Create a `.env` file in the project root with your credentials:

```bash
SMARTCITIZEN_USERNAME="your-email@example.com"
SMARTCITIZEN_TOKEN="your-smartcitizen-api-token"
NTFY_TOKEN="your-ntfy-token"
```

1. Configure the application settings in `configs/config.json`:

```json
{
  "dotenv_path": ".env",
  "log_level": "INFO",
  "battery_sensor_name": "Battery SCK",
  "ntfy": {
    "host": "https://ntfy.sh",
    "topic": "your_topic_name",
    "token_env": "NTFY_TOKEN"
  },
  "smc": {
    "endpoint": "https://api.smartcitizen.me",
    "api_version": "v0",
    "username_env": "SMARTCITIZEN_USERNAME",
    "password_env": "SMARTCITIZEN_PASSWORD"
  }
}
```

### Running the Application

#### Run Locally

Run the application directly with Go:

```bash
task run
```

This executes: `go run main.go --config configs/config.json`

#### Run with Docker

Build and run the application in a container:

```bash
task run:docker
```

Or manually:

```bash
task build:docker
```

### Development

Available Task commands:

- `task run` - Run the application locally
- `task build:docker` - Build Docker image
- `task run:docker` - Build and run Docker container
- `task template:helm` - Template and validate Helm chart
- `task release:docker` - Release Docker image to registry
- `task release:helm` - Package and push Helm chart
- `task release` - Release both Docker and Helm
- `task deploy:credentials` - Deploy credentials to Kubernetes

View all available tasks:

```bash
task --list
```

### Deployment

#### Kubernetes with Helm

1. Create the namespace:

```bash
kubectl create namespace smcprober
```

1. Deploy credentials (requires `DOCKER_USERNAME` and `DOCKER_PASSWORD`
   environment variables):

```bash
DOCKER_USERNAME=your-username DOCKER_PASSWORD=your-password task deploy:credentials
```

Note: The `deploy:credentials` task also creates the namespace if it
doesn't exist.

1. Template and preview the Helm deployment:

```bash
task template:helm
```

1. Deploy using Helm:

```bash
helm install smcprober ./helm \
  --set "imagePullSecrets[0].name"="smcprober-registry" \
  --set-file=config=configs/config-k8s.json \
  --set-file=secret.env=.env
```

#### Prometheus Monitoring

The Helm chart includes optional ServiceMonitor support for automatic metrics
discovery by Prometheus Operator.

**Prerequisites:**

- [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator)
  installed in your cluster

**Enable ServiceMonitor:**

```bash
helm install smcprober ./helm \
  --set "imagePullSecrets[0].name"="smcprober-registry" \
  --set-file=config=configs/config-k8s.json \
  --set-file=secret.env=.env \
  --set serviceMonitor.enabled=true
```

**Configure ServiceMonitor:**

You can customize the ServiceMonitor settings in your `values.yaml`:

```yaml
serviceMonitor:
  enabled: true
  interval: 30s # Scrape interval
  scrapeTimeout: 10s # Scrape timeout
  path: /metrics # Metrics endpoint path
  honorLabels: true # Honor labels from scraped metrics
  labels: {} # Additional labels for ServiceMonitor
  annotations: {} # Additional annotations
```

The application exposes Prometheus metrics at `/metrics` endpoint on port 8080.

#### Release

To release a new version:

```bash
task release
```

This will build and push both the Docker image and Helm chart.
The version is read from the `VERSION` file.
