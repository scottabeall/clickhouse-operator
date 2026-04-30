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
	"fmt"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricRegexpFilter(t *testing.T) {
	filter := newMetricRegexpFilter([]string{
		`^metric\.OS.*CPU[0-9]+$`,
		`^table_parts$`,
		`[`,
		``,
	})

	if !filter.isExcluded("metric.OSUserTimeCPU12") {
		t.Fatal("expected CPU metric to be excluded")
	}
	if !filter.isExcluded("table_parts") {
		t.Fatal("expected table_parts metric to be excluded")
	}
	if filter.isExcluded("metric.OSUserTime") {
		t.Fatal("expected non-CPU metric to be included")
	}
	if len(filter.validPatterns()) != 2 {
		t.Fatalf("expected 2 valid patterns, got %d", len(filter.validPatterns()))
	}
}

func TestMetricsSQLWithoutFilterIsBackwardCompatible(t *testing.T) {
	fetcher := NewMetricsFetcher(nil, "^(metrics|custom_metrics)$", nil)
	expected := fmt.Sprintf(queryMetricsSQLTemplate, "merge('system','^(metrics|custom_metrics)$')")

	if got := fetcher.buildMetricsSQL(); got != expected {
		t.Fatalf("expected unfiltered SQL to match previous template\nexpected:\n%s\ngot:\n%s", expected, got)
	}
}

func TestMetricsSQLWithFilter(t *testing.T) {
	fetcher := NewMetricsFetcher(
		nil,
		"metrics",
		newMetricRegexpFilter([]string{`^metric\.OS.*CPU[0-9]+$`, `^table_parts$`}),
	)
	sql := fetcher.buildMetricsSQL()

	assertContains(t, sql, "FROM (")
	assertContains(t, sql, "WHERE NOT (")
	assertContains(t, sql, `match(metric, '^metric\\.OS.*CPU[0-9]+$')`)
	assertContains(t, sql, `match(metric, '^table_parts$')`)
}

func TestEscapeClickHouseString(t *testing.T) {
	got := escapeClickHouseString(`^metric\.Foo'Bar\\Baz$`)
	expected := `^metric\\.Foo\'Bar\\\\Baz$`
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestPrometheusWriterSkipsExcludedMetric(t *testing.T) {
	out := make(chan prometheus.Metric, 1)
	writer := &CHIPrometheusWriter{
		out:          out,
		metricFilter: newMetricRegexpFilter([]string{`^metric\.OS.*CPU[0-9]+$`}),
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

func assertContains(t *testing.T, s string, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("expected SQL to contain %q\nSQL:\n%s", substr, s)
	}
}
