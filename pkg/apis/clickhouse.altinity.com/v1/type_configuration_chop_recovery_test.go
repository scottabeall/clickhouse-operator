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

package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altinity/clickhouse-operator/pkg/apis/common/types"
)

// TestShouldRecoverAbortedOnPodReady verifies the accessor's behavior across the
// full matrix of possible values for reconcile.recovery.from.aborted.onPodReady.
func TestShouldRecoverAbortedOnPodReady(t *testing.T) {
	tests := []struct {
		name     string
		onReady  *types.String
		expected bool
	}{
		{"nil defaults to retry", nil, true},
		{"empty string defaults to retry", types.NewString(""), true},
		{"retry lowercase", types.NewString("retry"), true},
		{"Retry mixed case", types.NewString("Retry"), true},
		{"RETRY upper case", types.NewString("RETRY"), true},
		{"none lowercase", types.NewString("none"), false},
		{"None mixed case", types.NewString("None"), false},
		{"NONE upper case", types.NewString("NONE"), false},
		{"unknown value treated as no-retry", types.NewString("bogus"), false},
		{"whitespace-only treated as no-retry", types.NewString("  "), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &OperatorConfig{}
			c.Reconcile.Recovery.From.Aborted.OnPodReady = tc.onReady
			require.Equal(t, tc.expected, c.ShouldRecoverAbortedOnPodReady())
		})
	}
}

// TestRecoveryActionConstants documents the stable enum values published in the CRD.
// Changes here would break users' CHOPCONF CRs.
func TestRecoveryActionConstants(t *testing.T) {
	require.Equal(t, "none", RecoveryActionNone)
	require.Equal(t, "retry", RecoveryActionRetry)
}
