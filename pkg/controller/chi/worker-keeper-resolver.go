// Copyright 2019 Altinity Ltd and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package chi

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/altinity/clickhouse-operator/pkg/announcer"
	chkApi "github.com/altinity/clickhouse-operator/pkg/apis/clickhouse-keeper.altinity.com/v1"
	api "github.com/altinity/clickhouse-operator/pkg/apis/clickhouse.altinity.com/v1"
	"github.com/altinity/clickhouse-operator/pkg/apis/common/types"
	"github.com/altinity/clickhouse-operator/pkg/chop"
	"github.com/altinity/clickhouse-operator/pkg/controller"
	"github.com/altinity/clickhouse-operator/pkg/controller/common/poller"
	"github.com/altinity/clickhouse-operator/pkg/interfaces"
	chkNamer "github.com/altinity/clickhouse-operator/pkg/model/chk/namer"
	chkLabeler "github.com/altinity/clickhouse-operator/pkg/model/chk/tags/labeler"
	commonLabeler "github.com/altinity/clickhouse-operator/pkg/model/common/tags/labeler"
)

var (
	// ErrKeeperRefResolve indicates a failure to resolve a keeper reference (e.g., service not found).
	ErrKeeperRefResolve = errors.New("failed to resolve keeper ref")
	// ErrKeeperRefNoNodes indicates a keeper reference resolved successfully but produced zero nodes.
	ErrKeeperRefNoNodes = errors.New("keeper ref resolved to 0 nodes")
	// ErrKeeperNotReady indicates the referenced keeper's pods are not all Running yet.
	ErrKeeperNotReady = errors.New("keeper not ready")
)

// resolveAllKeeperReferences resolves all keeper references in the CHI spec to ZooKeeper nodes.
// Must be called before normalization.
//
// Each ZookeeperConfig level is resolved independently — resolved nodes are appended to
// that level's own Nodes slice, not to a shared one:
//   - Top-level spec.configuration.zookeeper.keeper → resolved into spec.configuration.zookeeper.nodes
//   - Cluster-level clusters[i].zookeeper.keeper    → resolved into clusters[i].zookeeper.nodes
//
// Inheritance between levels is NOT handled here — that happens later in the normalizer
// via InheritZookeeperFrom (type_cluster.go). The inheritance rule is either/or:
//   - If a cluster has NO zookeeper config (empty) → it inherits the top-level config entirely
//   - If a cluster has ANY zookeeper config (own keeper ref or own nodes) → top-level is ignored
//     for that cluster (InheritZookeeperFrom returns early when cluster.Zookeeper is non-empty)
func (w *worker) resolveAllKeeperReferences(ctx context.Context, cr *api.ClickHouseInstallation) error {
	// Pass CR's namespace domain pattern to the resolver
	domainPattern := cr.Spec.GetNamespaceDomainPattern()

	// Resolve top-level zookeeper keeper ref into top-level nodes
	if cr.Spec.Configuration != nil && cr.Spec.Configuration.Zookeeper != nil {
		if err := w.resolveKeeperReference(ctx, cr.GetNamespace(), domainPattern, cr.Spec.Configuration.Zookeeper); err != nil {
			return err
		}
	}

	// Resolve each cluster's zookeeper keeper ref into that cluster's own nodes
	if cr.Spec.Configuration != nil {
		for i := range cr.Spec.Configuration.Clusters {
			// Convenience wrapper
			cluster := cr.Spec.Configuration.Clusters[i]
			if cluster != nil && cluster.Zookeeper != nil {
				if err := w.resolveKeeperReference(ctx, cr.GetNamespace(), domainPattern, cluster.Zookeeper); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// resolveKeeperReference resolves a single KeeperRef into ZookeeperNodes within the given ZookeeperConfig.
func (w *worker) resolveKeeperReference(ctx context.Context, chiNamespace string, domainPattern *types.String, zkc *api.ZookeeperConfig) error {
	// Sanity check: if no keeper reference, nothing to do
	if !zkc.HasKeeper() {
		return nil
	}

	keeper := zkc.Keeper
	ns := keeper.GetNamespace(chiNamespace)
	name := keeper.Name

	// Wait until the referenced keeper (CHK) is ready before resolving it to zookeeper nodes
	if err := w.waitKeeperReady(ctx, ns, name); err != nil {
		return err
	}

	var nodes api.ZookeeperNodes
	var err error

	// Resolve keeper to either per-host services or CR-level service nodes based on service type
	switch keeper.GetServiceType() {
	case api.KeeperServiceTypeService:
		nodes, err = w.resolveKeeperByService(ctx, ns, name, domainPattern)
	case api.KeeperServiceTypeReplicas:
		nodes, err = w.resolveKeeperByReplicas(ctx, ns, name, domainPattern)
		if err != nil || len(nodes) == 0 {
			// Fallback to CR-level service if replica discovery fails
			log.V(1).Info("Keeper replica discovery failed or returned 0 nodes, falling back to CR service: %v", err)
			nodes, err = w.resolveKeeperByService(ctx, ns, name, domainPattern)
		}
	default:
		return fmt.Errorf("%w: invalid keeper serviceType %q for %s/%s", ErrKeeperRefResolve, keeper.GetServiceType(), ns, name)
	}

	// Handle errors
	switch {
	case err != nil:
		return fmt.Errorf("%w %s/%s: %w", ErrKeeperRefResolve, ns, name, err)
	case len(nodes) == 0:
		return fmt.Errorf("%w %s/%s", ErrKeeperRefNoNodes, ns, name)
	}

	// Looks good, append service or nodes to the CHI's ZookeeperConfig
	log.V(1).Info("Resolved keeper ref %s/%s to %d node(s)", ns, name, len(nodes))
	zkc.Nodes = zkc.Nodes.Append(nodes...)
	return nil
}

// chkListOptions builds meta.ListOptions to select all resources belonging to a CHK by name.
func chkListOptions(name, namespace string) meta.ListOptions {
	chk := chkApi.NewClickHouseKeeperInstallation(name, namespace)
	l := chkLabeler.New(chk)
	return controller.NewListOptions(map[string]string{
		l.Get(commonLabeler.LabelCRName): name,
	})
}

// chkHostServiceListOptions builds meta.ListOptions to select CHK per-host services.
func chkHostServiceListOptions(name, namespace string) meta.ListOptions {
	chk := chkApi.NewClickHouseKeeperInstallation(name, namespace)
	l := chkLabeler.New(chk)
	return controller.NewListOptions(map[string]string{
		l.Get(commonLabeler.LabelCRName):  name,
		l.Get(commonLabeler.LabelService): l.Get(commonLabeler.LabelServiceValueHost),
	})
}

const (
	// keeperReadyPollInterval is how often to re-check CHK pod readiness.
	keeperReadyPollInterval = 5 * time.Second
)

// waitKeeperReady blocks until the referenced CHK has all pods in Running phase.
// Uses the project's standard poller infrastructure.
// Timeout is configurable via CHOP config: clickhouse.keeperReadyTimeout (seconds).
// Returns ErrKeeperNotReady if the timeout expires.
func (w *worker) waitKeeperReady(ctx context.Context, namespace, name string) error {
	opts := chkListOptions(name, namespace)

	timeout := time.Duration(chop.Config().Reconcile.Coordination.Keeper.ReadyTimeout) * time.Second

	pollerName := fmt.Sprintf("keeper-ready/%s/%s", namespace, name)
	return poller.New(ctx, pollerName).
		WithOptions(&poller.Options{
			Timeout:                    timeout,
			MainInterval:               keeperReadyPollInterval,
			StartBotheringAfterTimeout: 30 * time.Second,
		}).
		WithFunctions(&poller.Functions{
			Get: func(ctx context.Context) (any, error) {
				return w.c.kubeClient.CoreV1().Pods(namespace).List(ctx, opts)
			},
			IsDone: func(_ context.Context, item any) bool {
				pods, ok := item.(*core.PodList)
				if !ok || len(pods.Items) == 0 {
					return false
				}
				for _, pod := range pods.Items {
					if pod.Status.Phase != core.PodRunning {
						return false
					}
				}
				return true
			},
			ShouldContinueOnGetError: func(_ context.Context, _ any, _ error) bool {
				return true
			},
		}).
		Poll()
}

// resolveKeeperByService resolves using the CR-level service (single endpoint).
func (w *worker) resolveKeeperByService(ctx context.Context, namespace, name string, domainPattern *types.String) (api.ZookeeperNodes, error) {
	// Find the CHK CR-level service in k8s
	chk := chkApi.NewClickHouseKeeperInstallation(name, namespace)
	namer := chkNamer.New()
	serviceName := namer.Name(interfaces.NameCRService, chk)
	svc, err := w.c.kube.Service().Get(ctx, namespace, serviceName)
	if err != nil {
		return nil, fmt.Errorf("%w: CR service %s/%s: %w", ErrKeeperRefResolve, namespace, serviceName, err)
	}

	// CHK service found, extract ZK port (with TLS auto-detection) and FQDN
	portInfo := api.ExtractZKPortInfo(svc.Spec.Ports)
	fqdn := namer.Name(interfaces.NameCRServiceFQDN, chk, domainPattern)

	return api.NewZookeeperNodes(api.NewZookeeperNodeFromPortInfo(fqdn, portInfo)), nil
}

// resolveKeeperByReplicas resolves using per-host services (one node per replica).
// CHK host services are discovered by CHK labeler's LabelCRName and LabelService/LabelServiceValueHost labels.
func (w *worker) resolveKeeperByReplicas(ctx context.Context, namespace, name string, domainPattern *types.String) (api.ZookeeperNodes, error) {
	// Discover CHK host services by label selector
	opts := chkHostServiceListOptions(name, namespace)
	services, err := w.c.kubeClient.CoreV1().Services(namespace).List(ctx, opts)

	// Handle errors and empty list
	switch {
	case err != nil:
		return nil, fmt.Errorf("%w: host services %s/%s: %w", ErrKeeperRefResolve, namespace, name, err)
	case len(services.Items) == 0:
		return nil, fmt.Errorf("%w: host services %s/%s", ErrKeeperRefNoNodes, namespace, name)
	}

	// We have list of host services, now build ZookeeperNodes

	// Sort for deterministic order
	sort.Slice(services.Items, func(i, j int) bool {
		return services.Items[i].Name < services.Items[j].Name
	})

	namer := chkNamer.New()
	nodes := make(api.ZookeeperNodes, 0, len(services.Items))
	for _, svc := range services.Items {
		portInfo := api.ExtractZKPortInfo(svc.Spec.Ports)
		fqdn := namer.ServiceFQDN(svc.Name, namespace, domainPattern)
		nodes = nodes.Append(api.NewZookeeperNodeFromPortInfo(fqdn, portInfo))
	}

	return nodes, nil
}
