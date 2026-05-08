package chi

import (
	"testing"

	api "github.com/altinity/clickhouse-operator/pkg/apis/clickhouse.altinity.com/v1"
	"github.com/altinity/clickhouse-operator/pkg/apis/common/types"
	"github.com/altinity/clickhouse-operator/pkg/chop"
)

func init() {
	chop.New(nil, nil, "")
}

func Test_shouldEnqueue(t *testing.T) {
	// NB: ShouldEnqueue intentionally does NOT pre-filter on Spec.Suspend.
	// The reconciler itself handles suspend (including marking CHI as Aborted when
	// there are pending changes), so the enqueue step must always let suspended
	// CHIs through. See commit 3d2c80334 "dev: generalize enqueue checker" and
	// the "Bug Fix: Suspend sets Aborted when pending changes exist" change.
	// ShouldEnqueue's sole responsibility is the namespace-watched gate.
	tests := []struct {
		name string
		chi  *api.ClickHouseInstallation
		want bool
	}{
		{
			name: "enqueues a non-suspended CHI",
			chi: &api.ClickHouseInstallation{
				Spec: api.ChiSpec{
					Suspend: types.NewStringBool(false),
				},
			},
			want: true,
		},
		{
			name: "enqueues a suspended CHI (reconciler handles suspend, not ShouldEnqueue)",
			chi: &api.ClickHouseInstallation{
				Spec: api.ChiSpec{
					Suspend: types.NewStringBool(true),
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldEnqueue(tt.chi); got != tt.want {
				t.Errorf("ShouldEnqueue() = %v, want %v", got, tt.want)
			}
		})
	}
}
