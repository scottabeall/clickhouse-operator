# Connecting to ClickHouse

This guide describes the different ways to connect to ClickHouse instances managed by the Altinity ClickHouse Operator.

## 1. Internal CLI Access (Quickest)
Use `kubectl exec` to run the native `clickhouse-client` directly inside a pod. This is ideal for quick queries and administration.

```bash
# Get the pod name
POD_NAME=$(kubectl get pods -l clickhouse.altinity.com/chi=<chi-name> -o jsonpath='{.items[0].metadata.name}')

# Connect
kubectl exec -it $POD_NAME -- clickhouse-client -u <user> --password <password>
```
*Note: If no user is specified in the CHI manifest, try user `default` with password `default`.*

## 2. Local Desktop Access (GUI & Tools)
To use tools like **DBeaver**, **DataGrip**, or your local **clickhouse-client**, use Kubernetes port-forwarding to bridge your local machine to the cluster.

### For HTTP / Web Interface (ClickHouse Play)
```bash
kubectl port-forward svc/clickhouse-<chi-name> 8123:8123
```
*   **URL:** [http://localhost:8123/play](http://localhost:8123/play)
*   **Default User:** `default`
*   **Default Password:** `default`

### For Native TCP Client
```bash
kubectl port-forward svc/clickhouse-<chi-name> 9000:9000
```
*   **Host:** `127.0.0.1`
*   **Port:** `9000`

## 3. Application Access (Internal to K8s)
Applications running inside the same Kubernetes cluster should use the Service names created by the operator.

*   **Cluster-wide Service:** `clickhouse-<chi-name>.<namespace>.svc.cluster.local` (Port 8123 for HTTP, 9000 for TCP).
*   **Shard-specific Service:** `service-<chi-name>-<cluster>-<shard>.<namespace>.svc.cluster.local`.

To see all available services:
```bash
kubectl get svc -l clickhouse.altinity.com/chi=<chi-name>
```

## 4. External Access (Outside K8s)
To expose ClickHouse to the internet or an external network, you have two main options:

### Option A: LoadBalancer Service
Update your `ClickHouseInstallation` manifest to include a `serviceTemplate` with `type: LoadBalancer`.

```yaml
spec:
  templates:
    serviceTemplates:
      - name: external-service
        spec:
          type: LoadBalancer
          ports:
            - name: http
              port: 8123
            - name: client
              port: 9000
```

### Option B: Ingress
Configure an Ingress resource (like NGINX or Traefik) to route a domain name to the ClickHouse HTTP port (8123).

## Troubleshooting Credentials
If you are unsure of the password:
1.  **Check the CHI Manifest:** `kubectl get chi <chi-name> -o yaml` and look under `spec.configuration.users`.
2.  **Check ConfigMaps:** The operator often stores user XMLs in ConfigMaps named `chi-<chi-name>-common-usersd`.
3.  **Operator Defaults:** The default password set by the operator if none is provided is usually `default`.
