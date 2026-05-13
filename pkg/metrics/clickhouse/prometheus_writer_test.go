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

package clickhouse

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// stubMetricsFilter excludes any metric whose name appears in the set.
// Decouples the writer test from the concrete filter implementation in the
// `filters` sub-package — the interface contract is what matters here.
type stubMetricsFilter struct {
	excluded map[string]bool
}

func (s *stubMetricsFilter) IsExcluded(name string) bool {
	return s.excluded[name]
}

func TestPrometheusWriterSkipsExcludedMetric(t *testing.T) {
	out := make(chan prometheus.Metric, 1)
	writer := &CHIPrometheusWriter{
		out:           out,
		metricsFilter: &stubMetricsFilter{excluded: map[string]bool{"metric.OSUserTimeCPU12": true}},
	}

	writer.writeSingleMetricToPrometheus(
		"metric.OSUserTimeCPU12",
		"",
		prometheus.GaugeValue,
		"1",
		nil,
	)

	if len(out) != 0 {
		t.Fatal("expected excluded metric not to be written")
	}
}
