# SmartCitizen Prober

SmartCitizen Prober is a small service to check the status of
SmartCitizen devices and report their connectivity and data status.

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

#### Release

To release a new version:

```bash
task release
```

This will build and push both the Docker image and Helm chart.
The version is read from the `VERSION` file.
