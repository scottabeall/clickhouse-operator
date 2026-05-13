# Running ClickHouse Operator in Disconnected (Air-Gapped) Environments

This guide provides instructions for deploying the operator and ClickHouse clusters in environments without internet access.

## 1. Mirroring Docker Images
You must mirror the following images to your internal private registry:

| Original Image | Purpose |
| :--- | :--- |
| `altinity/clickhouse-operator:0.26.3` | The main operator |
| `altinity/metrics-exporter:0.26.3` | Metrics collection sidecar |
| `bitnami/kubectl:latest` | Required for CRD installation hooks |
| `clickhouse/clickhouse-server:<version>` | Your chosen ClickHouse version |

**Example Mirroring Commands:**
```bash
REGISTRY="your-registry.internal"
for img in "altinity/clickhouse-operator:0.26.3" "altinity/metrics-exporter:0.26.3" "bitnami/kubectl:latest"; do
  docker pull $img
  docker tag $img $REGISTRY/$img
  docker push $REGISTRY/$img
done
```

## 2. Helm Configuration for Air-Gap
Update your `values.yaml` to point to your private registry.

```yaml
operator:
  image:
    registry: "your-registry.internal"
    repository: altinity/clickhouse-operator
    tag: "0.26.3"

metrics:
  image:
    registry: "your-registry.internal"
    repository: altinity/metrics-exporter
    tag: "0.26.3"

crdHook:
  image:
    repository: "your-registry.internal/bitnami/kubectl"
    tag: "latest"
```

## 3. Local Manifest Generation
If you cannot use Helm directly in the target environment, generate a static "bundle" manifest from a machine with access to this repository:

```bash
# Generate a single standalone YAML file
helm template clickhouse-operator ./deploy/helm/clickhouse-operator \
  --namespace kube-system \
  --set operator.image.registry="your-registry.internal" \
  --set metrics.image.registry="your-registry.internal" \
  > operator-bundle.yaml
```
You can then transfer `operator-bundle.yaml` to your air-gapped cluster and apply it:
```bash
kubectl apply -f operator-bundle.yaml
```

## 4. ClickHouse Cluster Images
When defining your `ClickHouseInstallation` (CHI), ensure you explicitly set the image to your private registry:

```yaml
spec:
  templates:
    podTemplates:
      - name: standard
        spec:
          containers:
            - name: clickhouse
              image: "your-registry.internal/clickhouse/clickhouse-server:24.8"
```

## 5. Disconnected Storage (MinIO)
In Step 10 of the [User Guide](./USER_GUIDE.md), ensure your MinIO endpoint uses an internal FQDN (e.g., `http://minio.storage.svc.cluster.local:9000`) rather than an external URL.

---
[Quickstart](./QUICKSTART.md) | [User Guide](./USER_GUIDE.md) | [Helm Guide](./HELM_GUIDE.md) | [Monitoring](./MONITORING.md) | [Disconnected](./DISCONNECTED.md)
