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
	"regexp"
	"strings"

	log "github.com/golang/glog"
)

type metricRegexpFilter struct {
	patterns []string
	regexps  []*regexp.Regexp
}

func newMetricRegexpFilter(patterns []string) *metricRegexpFilter {
	filter := &metricRegexpFilter{}
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Warningf("Ignore invalid ClickHouse metric exclusion regexp %q: %v", pattern, err)
			continue
		}
		filter.patterns = append(filter.patterns, pattern)
		filter.regexps = append(filter.regexps, re)
	}
	return filter
}

func (f *metricRegexpFilter) hasPatterns() bool {
	return f != nil && len(f.patterns) > 0
}

func (f *metricRegexpFilter) validPatterns() []string {
	if f == nil {
		return nil
	}
	return f.patterns
}

func (f *metricRegexpFilter) isExcluded(name string) bool {
	if f == nil {
		return false
	}
	for _, re := range f.regexps {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

func (f *metricRegexpFilter) buildExcludeMetricsWhere() string {
	conditions := make([]string, 0, len(f.validPatterns()))
	for _, pattern := range f.validPatterns() {
		conditions = append(conditions, fmt.Sprintf("match(metric, '%s')", escapeClickHouseString(pattern)))
	}
	return strings.Join(conditions, " OR ")
}

func escapeClickHouseString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `\'`)
	return value
}
