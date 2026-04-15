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
	"time"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	log "github.com/altinity/clickhouse-operator/pkg/announcer"
	api "github.com/altinity/clickhouse-operator/pkg/apis/clickhouse.altinity.com/v1"
	"github.com/altinity/clickhouse-operator/pkg/chop"
	cmd_queue "github.com/altinity/clickhouse-operator/pkg/controller/chi/cmd_queue"
)

var chkGVR = schema.GroupVersionResource{
	Group:    "clickhouse-keeper.altinity.com",
	Version:  "v1",
	Resource: "clickhousekeeperinstallations",
}

const (
	chkWatcherResyncPeriod = 60 * time.Second
)

// StartCHKWatcher starts a dynamic informer for CHK resources.
// When a CHK transitions to Completed status, finds dependent CHIs and enqueues reconcile for them.
// Respects the reconcile.coordination.keeper.onKeeperResourceUpdate config setting.
func (c *Controller) StartCHKWatcher(ctx context.Context) {
	if !c.isKeeperWatchEnabled() {
		log.V(1).Info("CHK watcher disabled (reconcile.coordination.keeper.onKeeperResourceUpdate != 'reconcile')")
		return
	}

	if c.dynamicClient == nil {
		log.V(1).Info("CHK watcher disabled: no dynamic client available")
		return
	}

	ns := chop.Config().GetInformerNamespace()
	listWatcher := &cache.ListWatch{
		ListFunc: func(options meta.ListOptions) (runtime.Object, error) {
			return c.dynamicClient.Resource(chkGVR).Namespace(ns).List(ctx, options)
		},
		WatchFunc: func(options meta.ListOptions) (watch.Interface, error) {
			return c.dynamicClient.Resource(chkGVR).Namespace(ns).Watch(ctx, options)
		},
	}

	_, informer := cache.NewInformer(listWatcher, &unstructured.Unstructured{}, chkWatcherResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				c.onCHKUpdate(old, new)
			},
		},
	)

	log.V(1).Info("Starting CHK watcher for keeper resource changes")
	go informer.Run(ctx.Done())
}

// isKeeperWatchEnabled checks if the CHK watch is configured.
func (c *Controller) isKeeperWatchEnabled() bool {
	policy := chop.Config().Reconcile.Coordination.Keeper.OnKeeperResourceUpdate
	return policy.HasValue() && policy.Value() == api.KeeperOnResourceUpdateReconcile
}

// onCHKUpdate handles CHK update events. Triggers CHI reconcile only when CHK transitions to Completed.
func (c *Controller) onCHKUpdate(old, new interface{}) {
	newObj, ok := new.(*unstructured.Unstructured)
	if !ok {
		return
	}
	oldObj, ok := old.(*unstructured.Unstructured)
	if !ok {
		return
	}

	newStatus := getUnstructuredStatus(newObj)
	oldStatus := getUnstructuredStatus(oldObj)

	// Only react when CHK transitions TO Completed (not on every update while already Completed)
	if newStatus != "Completed" || oldStatus == "Completed" {
		return
	}

	chkName := newObj.GetName()
	chkNamespace := newObj.GetNamespace()
	log.V(1).Info("CHK %s/%s reached Completed — looking for dependent CHIs", chkNamespace, chkName)

	c.enqueueDependentCHIs(chkNamespace, chkName)
}

// getUnstructuredStatus extracts .status.status from an unstructured object.
func getUnstructuredStatus(obj *unstructured.Unstructured) string {
	status, found, err := unstructured.NestedString(obj.Object, "status", "status")
	if err != nil || !found {
		return ""
	}
	return status
}

// enqueueDependentCHIs finds all CHIs in the same namespace that reference the given CHK
// and enqueues reconcile for each of them.
func (c *Controller) enqueueDependentCHIs(chkNamespace, chkName string) {
	chiList, err := c.chopClient.ClickhouseV1().ClickHouseInstallations(chkNamespace).List(
		context.Background(),
		meta.ListOptions{},
	)
	if err != nil {
		log.V(1).Info("Failed to list CHIs in namespace %s: %v", chkNamespace, err)
		return
	}

	for i := range chiList.Items {
		chi := &chiList.Items[i]
		if chiReferencesKeeper(chi, chkName, chkNamespace) {
			log.V(1).Info("Triggering reconcile for CHI %s/%s due to CHK %s/%s completing",
				chi.Namespace, chi.Name, chkNamespace, chkName)
			c.enqueueObject(cmd_queue.NewReconcileCHI(cmd_queue.ReconcileUpdate, chi, chi))
		}
	}
}

// chiReferencesKeeper checks if a CHI references the given CHK by name.
func chiReferencesKeeper(chi *api.ClickHouseInstallation, chkName, chkNamespace string) bool {
	// Check top-level zookeeper.keeper
	if chi.Spec.Configuration != nil && chi.Spec.Configuration.Zookeeper != nil {
		keeper := chi.Spec.Configuration.Zookeeper.Keeper
		if keeper.HasName() && keeper.Name == chkName && keeper.GetNamespace(chi.Namespace) == chkNamespace {
			return true
		}
	}

	// Check per-cluster zookeeper.keeper
	if chi.Spec.Configuration != nil {
		for _, cluster := range chi.Spec.Configuration.Clusters {
			if cluster != nil && cluster.Zookeeper != nil {
				keeper := cluster.Zookeeper.Keeper
				if keeper.HasName() && keeper.Name == chkName && keeper.GetNamespace(chi.Namespace) == chkNamespace {
					return true
				}
			}
		}
	}

	return false
}
