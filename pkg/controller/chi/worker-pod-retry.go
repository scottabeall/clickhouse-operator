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

	core "k8s.io/api/core/v1"

	api "github.com/altinity/clickhouse-operator/pkg/apis/clickhouse.altinity.com/v1"
	"github.com/altinity/clickhouse-operator/pkg/chop"
	"github.com/altinity/clickhouse-operator/pkg/controller/chi/cmd_queue"
	"github.com/altinity/clickhouse-operator/pkg/controller/chi/metrics"
	a "github.com/altinity/clickhouse-operator/pkg/controller/common/announcer"
	"github.com/altinity/clickhouse-operator/pkg/model/k8s"
)

// recoverAbortedReconcileOnPodReady inspects a pod update event and re-enqueues the parent
// CHI for reconcile when the pod transitioned NotReady → Ready and the CHI is Aborted.
// Controlled by reconcile.recovery.from.aborted.onPodReady config option (default: retry).
// The decision to re-enqueue is based on the CHI's Status alone, not on ActionPlan —
// see shouldTriggerAutoRecovery for the rationale.
func (w *worker) recoverAbortedReconcileOnPodReady(ctx context.Context, oldPod, newPod *core.Pod) {
	if !chop.Config().ShouldRecoverAbortedOnPodReady() {
		return
	}

	if !isPodNotReadyToReadyTransition(oldPod, newPod) {
		return
	}

	// Reverse-lookup parent CHI via the clickhouse.altinity.com/chi label.
	cr, err := w.c.GetCR(&newPod.ObjectMeta)
	if err != nil || cr == nil {
		return
	}

	if !shouldTriggerAutoRecovery(cr) {
		return
	}

	// Pod and CHI share the namespace (CHI-owned pods), so log it once.
	w.a.V(1).M(cr).F().
		WithEvent(cr, a.EventActionReconcile, a.EventReasonAutoRecoveryTriggered).
		Info(
			"Auto-recovery: pod %s became Ready while CHI %s/%s is Aborted — re-enqueuing reconcile",
			newPod.Name, cr.Namespace, cr.Name,
		)
	metrics.CRAutoRecoveriesTriggered(ctx, cr)

	// Use ReconcileAdd path: it does not require a spec diff, bypasses isGenerationTheSame,
	// and relies on the ActionPlan preserved in status to drive the remaining work.
	w.c.enqueueObject(cmd_queue.NewReconcileCHI(cmd_queue.ReconcileAdd, nil, cr))
}

// shouldTriggerAutoRecovery reports whether the given CHI is a valid auto-recovery target:
// status is Aborted and the CHI is not being deleted. Package-private and pure so it can
// be unit-tested without constructing a worker or controller.
//
// We deliberately do not check ActionPlan.HasActionsToDo() here: on a fresh CHI fetch the
// Runtime.ActionPlan is empty (it gets rebuilt from ancestor+current during reconcile).
// The reconcile path itself checks HasReconcileWork() and exits cleanly if there's no
// work left, so triggering a no-op reconcile is harmless.
func shouldTriggerAutoRecovery(cr *api.ClickHouseInstallation) bool {
	if cr == nil {
		return false
	}
	if cr.EnsureStatus().GetStatus() != api.StatusAborted {
		return false
	}
	// Skip CHIs being deleted.
	if !cr.GetDeletionTimestamp().IsZero() {
		return false
	}
	return true
}

// isPodNotReadyToReadyTransition reports whether the pod transitioned from "some container
// not ready" to "all containers ready". We intentionally use container readiness rather than
// IsPodOK() so the transition fires as soon as containers become Ready — no need to wait for
// full pod-phase bookkeeping.
//
// Edge case: a pod with zero container statuses is treated by PodHasNotReadyContainers as
// "ready" (zero-length loop returns false). So a first pod event that has already-populated
// ready statuses never fires a transition. This is intentional: we only react to *observed*
// NotReady → Ready flips, not to pods that were already Ready when we first saw them.
func isPodNotReadyToReadyTransition(oldPod, newPod *core.Pod) bool {
	if oldPod == nil || newPod == nil {
		return false
	}
	wasNotReady := k8s.PodHasNotReadyContainers(oldPod)
	isReadyNow := !k8s.PodHasNotReadyContainers(newPod)
	return wasNotReady && isReadyNow
}
