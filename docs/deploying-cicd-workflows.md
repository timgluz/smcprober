# Deploying CI/CD Workflows

This guide explains how to deploy and run Tekton pipelines for building multi-architecture Docker images of the
smcprober project.

## Overview

The CI/CD workflows use Tekton pipelines to:

- Clone the repository from GitHub
- Build Docker images for multiple architectures (amd64, arm64)
- Push images to a container registry (public Docker Hub by default)
- Create multi-architecture manifests

## Prerequisites

Before starting, ensure you have:

- `kubectl` CLI installed and configured with access to your Kubernetes cluster
- `tkn` (Tekton CLI) installed
- Access to create resources in the cluster
- A GitHub personal access token with appropriate permissions
- A container registry account and credentials (Docker Hub by default)

## Initial Setup

### 1. Create GitHub Personal Access Token

Create a fine-grained personal access token with the following permissions:

1. Go to GitHub: Settings → Developer settings → Personal access tokens → Fine-grained tokens
2. Click "Generate new token"
3. Configure the token:
   - **Repository access**: Choose specific repos or all repos
   - **Permissions**:
     - Contents: Read
     - Metadata: Read (automatically included)
     - Pull requests: Read (if you need PR checkout functionality)

Export the token as an environment variable:

```bash
export GH_TOKEN=<your_github_token>
```

### 2. Create Kubernetes Namespace

```bash
kubectl create namespace smc-cicd
```

### 3. Create GitHub Token Secret

```bash
kubectl create secret generic github-token \
  --from-literal=token=$GH_TOKEN \
  -n smc-cicd
```

### 4. Create Container Registry Secret

This secret is required for pushing images. The example below targets Docker Hub;
for a private/internal registry use its host for `--docker-server`:

```bash
export DOCKER_PASSWORD=<your_registry_password>

kubectl create secret docker-registry docker-config \
  --docker-username=<your_registry_username> \
  --docker-password=$DOCKER_PASSWORD \
  --docker-server=docker.io \
  -n smc-cicd
```

Alternatively, set `REGISTRY_HOST` (plus `DOCKER_USERNAME` / `DOCKER_PASSWORD`) in
`.env` and run `task deploy:ci:credentials`.

## Deploying Tasks and Pipelines

### Deploy Individual Task

To deploy a single task (e.g., git-clone-and-build):

```bash
kubectl apply -f helm/tasks/git-clone-and-build.yaml -n smc-cicd
```

### Deploy All Tasks

To deploy all available tasks:

```bash
kubectl apply -f helm/tasks/ -n smc-cicd
```

### Deploy the Build Pipeline

```bash
kubectl apply -f helm/pipelines/build-multiarch-image.yaml -n smc-cicd
```

## Running Workflows

### Option 1: Run Individual Clone-and-Build Task

To run a single architecture build:

```bash
tkn task start git-clone-and-build \
  --param repo=timgluz/smcprober \
  --param revision=main \
  --param image=docker.io/tauho/smcprober:latest-dev-amd64 \
  --param platform=linux/amd64 \
  --workspace name=dockerconfig,secret=docker-config \
  --showlog \
  -n smc-cicd
```

For a private registry served over plain HTTP, add
`--param registry-insecure=true`; leave it off for TLS registries.

### Option 2: Release Helm Chart

- deploys the chart

```bash
k apply -f helm/tasks/release-helm.yaml
```

- start task

```bash
tkn task start package-helm \
  --param repo=timgluz/smcprober \
  --param revision=main \
  --param version=latest \
  --param oci-registry=oci://registry-1.docker.io/tauho \
  --workspace name=dockerconfig,secret=docker-config \
  --showlog \
  -n smc-cicd
```

Set `--param oci-registry` to your own `oci://<host>/<namespace>` and add
`--param registry-insecure=true` only for a plain-HTTP registry.

### Option 3: Run Multi-Architecture Build Pipeline

To build for both amd64 and arm64 architectures:

```bash
kubectl create -f helm/runs/start-multiarch-build.yaml -n smc-cicd
```

Edit the `image`, `registry` and `registry-insecure` params in
`helm/runs/start-multiarch-build.yaml` to target a different registry.

## Monitoring Pipeline Runs

### List Recent Pipeline Runs

```bash
tkn pipelinerun list -n smc-cicd
```

### View Pipeline Run Logs

Follow logs in real-time (replace `<pipelinerun-name>` with the actual name from the list command):

```bash
tkn pipelinerun logs <pipelinerun-name> -f -n smc-cicd
```

Example:

```bash
tkn pipelinerun logs build-multiarch-image-run-abc123 -f -n smc-cicd
```

### Check Detailed Pipeline Status

```bash
tkn pipelinerun describe <pipelinerun-name> -n smc-cicd
```

### Access Tekton Dashboard (Optional)

If you have a Taskfile task configured to expose the Tekton dashboard:

```bash
task expose:tekton
```

This will make the Tekton web UI accessible for visual monitoring of pipeline runs.

## Troubleshooting

### View Failed Task Logs

```bash
kubectl logs <pod-name> -n smc-cicd
```

### Check Task Status

```bash
tkn taskrun list -n smc-cicd
tkn taskrun describe <taskrun-name> -n smc-cicd
```

### Verify Secrets

```bash
kubectl get secrets -n smc-cicd
kubectl describe secret github-token -n smc-cicd
kubectl describe secret docker-config -n smc-cicd
```
