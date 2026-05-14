# ClickHouse Installation Helm Chart

This chart deploys a `ClickHouseInstallation` (CHI) resource managed by the Altinity ClickHouse Operator.

## Features

- **Automated Security**: Automatically generates a secure, random admin password on first install.
- **Deployment-Specific Secrets**: Secrets are prefixed with the release name to support multiple installations in the same namespace.
- **Persistence Logic**: Uses Helm `lookup` to ensure passwords remain stable across upgrades.
- **Sidecar Support**: Includes a pre-configured data-checker sidecar for monitoring disk usage.

## Configuration

### Admin User
The `admin` user is configured to use a Kubernetes Secret for its password.

- **Secret Name**: `{{ .Release.Name }}-clickhouse-installation-admin`
- **Secret Key**: `admin`

If `adminPassword` is left empty in `values.yaml`, a random 16-character password is generated during the initial installation.

### Persistence across Upgrades
This chart uses a "lookup-first" pattern. During a `helm upgrade`, Helm checks if the admin secret already exists. If found, it preserves the existing password even if `values.yaml` is changed. This prevents accidental lockouts.

## Usage

### Install the Chart
```bash
helm install ch1 . -n clickstack
```

### Get the Admin Password
To retrieve the automatically generated password:
```bash
kubectl get secret $(helm get notes ch1 | grep "Secret Name" | awk '{print $NF}') -n clickstack -o jsonpath='{.data.admin}' | base64 --decode
```
*(Or manually look up the secret named `<release>-admin`)*

### Upgrade the Chart
```bash
helm upgrade ch1 . -n clickstack
```

## Troubleshooting

### CrashLoopBackOff
If the pods are crashing:
1. **Check Logs**: `kubectl logs <pod-name> -c clickhouse`
2. **Verify Secret**: Ensure the secret exists and contains a valid password.
3. **Zookeeper**: If `replicasCount > 1`, ensure the Zookeeper nodes defined in `values.yaml` are reachable.

### Pending Pods
If pods are stuck in `Pending`:
1. **Describe Pod**: `kubectl describe pod <pod-name>`
2. **PVCs**: Check if the operator has successfully created the PVCs: `kubectl get pvc -l clickhouse.altinity.com/chi={{ .Release.Name }}-clickhouse-installation`
