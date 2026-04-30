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
	"testing"

	"github.com/stretchr/testify/require"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/altinity/clickhouse-operator/pkg/apis/clickhouse.altinity.com/v1"
	chiLabeler "github.com/altinity/clickhouse-operator/pkg/model/chi/tags/labeler"
	commonLabeler "github.com/altinity/clickhouse-operator/pkg/model/common/tags/labeler"
)

// makeCompletedCR builds a minimal CHI with one cluster, one shard, one host —
// enough for FindHost(cluster, shard, replica) to resolve. ChiShard.FindHost matches
// against host.Runtime.Address.HostName (not host.Name), so the fixture must populate
// the Runtime.Address fields rather than just the top-level Name.
func makeCompletedCR(clusterName, shardName, replicaName string) *api.ClickHouseInstallation {
	host := &api.Host{Name: replicaName}
	host.Runtime.Address.HostName = replicaName
	shard := &api.ChiShard{
		Name:  shardName,
		Hosts: []*api.Host{host},
	}
	cluster := &api.Cluster{
		Name: clusterName,
		Layout: &api.ChiClusterLayout{
			Shards: []*api.ChiShard{shard},
		},
	}
	cr := &api.ClickHouseInstallation{
		Spec: api.ChiSpec{
			Configuration: &api.Configuration{
				Clusters: []*api.Cluster{cluster},
			},
		},
	}
	return cr
}

// objWithLabels wraps a label map in a meta.ObjectMeta so it satisfies meta.Object
// for findHostInCompletedFromLabels.
func objWithLabels(labels map[string]string) meta.Object {
	return &meta.ObjectMeta{Labels: labels}
}

// TestFindHostInCompletedFromLabels exercises the orphan-host resolver: given a
// StatefulSet's labels, can we recover the corresponding *api.Host from the
// previously-completed CR? Covers the skip cases (missing labels, no match) and
// the happy path (labels match a real host in completed).
func TestFindHostInCompletedFromLabels(t *testing.T) {
	completed := makeCompletedCR("c1", "s1", "r1")
	labeler := chiLabeler.New(completed)

	clusterKey := labeler.Get(commonLabeler.LabelClusterName)
	shardKey := labeler.Get(commonLabeler.LabelShardName)
	replicaKey := labeler.Get(commonLabeler.LabelReplicaName)

	t.Run("nil labels → nil", func(t *testing.T) {
		got := findHostInCompletedFromLabels(completed, labeler, &meta.ObjectMeta{Labels: nil})
		require.Nil(t, got)
	})

	t.Run("missing cluster label → nil", func(t *testing.T) {
		obj := objWithLabels(map[string]string{
			shardKey:   "s1",
			replicaKey: "r1",
		})
		require.Nil(t, findHostInCompletedFromLabels(completed, labeler, obj))
	})

	t.Run("missing shard label → nil", func(t *testing.T) {
		obj := objWithLabels(map[string]string{
			clusterKey: "c1",
			replicaKey: "r1",
		})
		require.Nil(t, findHostInCompletedFromLabels(completed, labeler, obj))
	})

	t.Run("missing replica label → nil", func(t *testing.T) {
		obj := objWithLabels(map[string]string{
			clusterKey: "c1",
			shardKey:   "s1",
		})
		require.Nil(t, findHostInCompletedFromLabels(completed, labeler, obj))
	})

	t.Run("labels point to non-existent cluster → nil (true orphan)", func(t *testing.T) {
		obj := objWithLabels(map[string]string{
			clusterKey: "ghost",
			shardKey:   "s1",
			replicaKey: "r1",
		})
		require.Nil(t, findHostInCompletedFromLabels(completed, labeler, obj))
	})

	t.Run("labels point to non-existent shard → nil", func(t *testing.T) {
		obj := objWithLabels(map[string]string{
			clusterKey: "c1",
			shardKey:   "ghost",
			replicaKey: "r1",
		})
		require.Nil(t, findHostInCompletedFromLabels(completed, labeler, obj))
	})

	t.Run("labels point to non-existent replica → nil", func(t *testing.T) {
		obj := objWithLabels(map[string]string{
			clusterKey: "c1",
			shardKey:   "s1",
			replicaKey: "ghost",
		})
		require.Nil(t, findHostInCompletedFromLabels(completed, labeler, obj))
	})

	t.Run("labels match a real host → returns that host", func(t *testing.T) {
		obj := objWithLabels(map[string]string{
			clusterKey: "c1",
			shardKey:   "s1",
			replicaKey: "r1",
		})
		got := findHostInCompletedFromLabels(completed, labeler, obj)
		require.NotNil(t, got)
		require.Equal(t, "r1", got.GetName())
	})
}
