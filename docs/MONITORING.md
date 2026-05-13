# Monitoring ClickHouse with Prometheus & Grafana

The Altinity ClickHouse Operator provides deep observability into both the operator itself and the ClickHouse clusters it manages.

## 1. Architecture
1.  **Metrics Exporter:** A sidecar container (deployed by default) that sits next to each ClickHouse pod and translates ClickHouse system tables into Prometheus metrics.
2.  **ServiceMonitor:** A Kubernetes resource that tells Prometheus which pods to scrape.
3.  **Grafana Dashboards:** Pre-configured JSON files that visualize the metrics.

## 2. Enabling Monitoring via Helm

To enable the full monitoring stack, update your `values.yaml`:

```yaml
# 1. Enable Prometheus Scraping
serviceMonitor:
  enabled: true
  # Most Prometheus installations (like kube-prometheus-stack) 
  # require a specific label to discover ServiceMonitors.
  additionalLabels:
    release: prometheus-stack 

# 2. Enable Grafana Dashboards
dashboards:
  enabled: true
  # This label is used by the Grafana sidecar to auto-import the dashboards
  additionalLabels:
    grafana_dashboard: "1"
```

## 3. Available Dashboards
The operator provides three primary dashboards:
*   **ClickHouse Operator:** Tracks reconcile loops, errors, and operator resource usage.
*   **ClickHouse Queries:** Detailed views into query performance, execution times, and errors.
*   **ClickHouse Keeper:** Monitors the health and replication lag of the coordination layer.

## 4. Manual Dashboard Import
If you are not using the Grafana sidecar for auto-import, you can manually upload the JSON files found in the `grafana-dashboard/` directory of this project:
*   `grafana-dashboard/Altinity_ClickHouse_Operator_dashboard.json`
*   `grafana-dashboard/ClickHouse_Queries_dashboard.json`
*   `grafana-dashboard/ClickHouseKeeper_dashboard.json`

## 5. Verifying Metrics
You can verify that metrics are being exported by port-forwarding to a ClickHouse pod on port `8888`:

```bash
kubectl port-forward <clickhouse-pod-name> 8888:8888
curl http://localhost:8888/metrics
```
You should see a long list of metrics starting with `chi_` or `clickhouse_`.

---
[Quickstart](./QUICKSTART.md) | [User Guide](./USER_GUIDE.md) | [Helm Guide](./HELM_GUIDE.md) | [Monitoring](./MONITORING.md) | [Disconnected](./DISCONNECTED.md)
