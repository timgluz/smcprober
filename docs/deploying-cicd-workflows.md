# Deploying CI/CD Workflows

This guide explains how to deploy and run Tekton pipelines for building multi-architecture Docker images of the
smcprober project.

## Overview

The CI/CD workflows use Tekton pipelines to:

- Clone the repository from GitHub
- Resolve the version from the `VERSION` file
- Build Docker images for multiple architectures (amd64, arm64)
- Push images to a container registry (public Docker Hub by default)
- Create multi-architecture manifests
- Publish the Helm chart for tagged releases

## Versioning and Releases

The `VERSION` file in the repository root is the **source of truth** for the
current release (for example `v0.0.2`). Git tags and the file keep the `v` prefix,
but published artifacts use **bare SemVer** (`0.0.2`) because Helm rejects a `v`
prefix in a chart version. The pipeline refuses to build a release whose git tag
does not match `VERSION`, or whose `helm/Chart.yaml` `version` does not match the
bare version, so the image tag and chart version can never drift.

| Trigger | Image tags published |
| --- | --- |
| Push to `main` | `latest` |
| Push tag `vX.Y.Z` | `X.Y.Z` (immutable) and `latest` |

To cut a release:

1. Bump `VERSION` (for example to `v0.0.3`), and set `0.0.3` as both `version`
   and `appVersion` in `helm/Chart.yaml`.
2. Add a matching entry to `CHANGELOG.md`.
3. Merge to `main`.
4. Tag and push — either `task release:tag`, or manually:

   ```bash
   git tag -a v0.0.3 -m "Release v0.0.3"
   git push origin v0.0.3
   ```

The tag push triggers the pipeline, which publishes
`<registry>/<namespace>/smcprober:0.0.3` and the Helm chart as
`smcprober-0.0.3` (a unique chart version per release, rather than overwriting
the previous one). Deployments can then pin that fixed version:

```bash
helm upgrade --install smcprober ./helm -n smcprober --set image.tag=0.0.3
```

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

This includes `verify-dockerhub-creds`, `resolve-version`, `git-clone-and-build`,
`create-docker-manifest`, and `package-helm`.

### Deploy the Build Pipeline

```bash
kubectl apply -f helm/pipelines/build-multiarch-image.yaml -n smc-cicd
```

## Running Workflows

### Option 1: Run Individual Clone-and-Build Task

To run a single architecture build, pass the repository without a tag plus the
version to publish (bare SemVer — see above):

```bash
tkn task start git-clone-and-build \
  --param repo=timgluz/smcprober \
  --param revision=main \
  --param image=docker.io/tauho/smcprober \
  --param version=0.0.2-dev-amd64 \
  --param platform=linux/amd64 \
  --workspace name=dockerconfig,secret=docker-config \
  --showlog \
  -n smc-cicd
```

`image` is the repository and `version` is the tag, so the pushed reference is
`<image>:<version>`. For a private registry served over plain HTTP, add
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
  --param revision=v0.0.2 \
  --param version=0.0.2 \
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

For a non-release build this publishes `latest`. To publish a fixed version,
set `release-tag` to the git tag (it must match `VERSION`) and optionally
`additional-tag` to also move `latest`:

```yaml
params:
  - name: revision
    value: v0.0.3
  - name: release-tag
    value: v0.0.3
  - name: additional-tag
    value: latest
```

Or from the CLI:

```bash
tkn pipeline start build-multiarch-image \
  --namespace smc-cicd \
  --param repo=timgluz/smcprober \
  --param revision=v0.0.3 \
  --param image=docker.io/tauho/smcprober \
  --param release-tag=v0.0.3 \
  --param additional-tag=latest \
  --workspace name=dockerconfig,secret=docker-config \
  --showlog
```

`revision` and `release-tag` use the git tag (`v0.0.3`); the published image tag
drops the `v` and becomes `0.0.3`.

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
