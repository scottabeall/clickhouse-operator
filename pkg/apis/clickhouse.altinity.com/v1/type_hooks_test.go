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
)

// TestHookActionMatchesAnyEvent exercises the matcher that decides whether a hook should
// fire given the events emitted by the reconcile classifier.
func TestHookActionMatchesAnyEvent(t *testing.T) {
	t.Run("nil action does not match", func(t *testing.T) {
		var a *HookAction
		require.False(t, a.MatchesAnyEvent([]HookEvent{HookEventHostUpdate}))
	})

	t.Run("empty On list does not match (invalid input — schema enforces non-empty)", func(t *testing.T) {
		a := &HookAction{}
		require.False(t, a.MatchesAnyEvent([]HookEvent{HookEventHostUpdate}))
	})

	t.Run("Always matches everything, including empty fired set", func(t *testing.T) {
		a := &HookAction{Events: []HookEvent{HookEventAlways}}
		require.True(t, a.MatchesAnyEvent(nil))
		require.True(t, a.MatchesAnyEvent([]HookEvent{HookEventHostUpdate}))
		require.True(t, a.MatchesAnyEvent([]HookEvent{HookEventHostShutdown, HookEventHostStop}))
	})

	t.Run("specific event matches when fired", func(t *testing.T) {
		a := &HookAction{Events: []HookEvent{HookEventHostCreate}}
		require.True(t, a.MatchesAnyEvent([]HookEvent{HookEventHostCreate}))
		require.False(t, a.MatchesAnyEvent([]HookEvent{HookEventHostUpdate}))
	})

	t.Run("any of several listed events matching is enough", func(t *testing.T) {
		a := &HookAction{Events: []HookEvent{HookEventHostStop, HookEventHostRollout}}
		// Fired set includes HostRollout — second item in On matches.
		require.True(t, a.MatchesAnyEvent([]HookEvent{HookEventHostUpdate, HookEventHostRollout}))
	})

	t.Run("aggregate HostShutdown is just another event — matched if classifier emits it", func(t *testing.T) {
		a := &HookAction{Events: []HookEvent{HookEventHostShutdown}}
		// classifier emits HostShutdown alongside any of Stop/ConfigRestart/Rollout
		require.True(t, a.MatchesAnyEvent([]HookEvent{HookEventHostConfigRestart, HookEventHostShutdown}))
		// no shutdown-class event in the fired set
		require.False(t, a.MatchesAnyEvent([]HookEvent{HookEventHostUpdate}))
	})

	t.Run("no overlap fires nothing", func(t *testing.T) {
		a := &HookAction{Events: []HookEvent{HookEventHostCreate}}
		require.False(t, a.MatchesAnyEvent([]HookEvent{HookEventHostUpdate, HookEventHostStop}))
	})
}
