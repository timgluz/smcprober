# Deploying workflows for CI/CD

DRAFT

## Deploying Workflows

Create github token with repo and workflow permissions.
For checking out repos with gh CLI, you need the contents: `read` permission (for public repos) or contents: `read` + `metadata: read` (for private repos).

```markdown
Go to GitHub
Settings → Developer settings → Personal access tokens → Fine-grained tokens

Repository access: Choose specific repos or all repos
Permissions:
✓ Contents: Read
✓ Metadata: Read (automatically included)
✓ Pull requests: Read (if you need pr checkout specifically)
```

```bash
export GH_TOKEN=<your_github_token>
```

Create namespace and create secret with the token

```bash
# Create namespace
kubectl create namespace smc-cicd

# Create GitHub token secret
kubectl create secret generic github-token \
  --from-literal=token=$GH_TOKEN \
  -n smc-cicd
```

### Trigger checkout-pr task

- deploy task

```bash
k apply -f helm/workflows/checkout-pr-task.yaml
```

- trigger taskrun

```bash
tkn task start checkout-pr \
  --param repo=timgluz/smcprober \
  --param pr-number=42 \
  --workspace name=source,emptyDir="" \
  --showlog \
  -n smc-cicd
```
