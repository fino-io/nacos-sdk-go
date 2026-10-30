/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package naming_client

import (
	"math/rand"
	"sort"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
)

type Chooser struct {
	data   []model.Instance
	totals []float64
	max    float64
}

// newChooser receives positive finite weights from selectOneHealthyInstances.
func newChooser(instances []model.Instance) Chooser {
	maxWeight := 0.0
	for _, c := range instances {
		if c.Weight > maxWeight {
			maxWeight = c.Weight
		}
	}
	totals := make([]float64, len(instances))
	total := 0.0
	for i, c := range instances {
		// Scaling preserves ratios without overflowing when weights are very large.
		total += c.Weight / maxWeight
		totals[i] = total
	}
	return Chooser{data: instances, totals: totals, max: total}
}

func (chs Chooser) pick() model.Instance {
	r := rand.Float64() * chs.max
	i := sort.Search(len(chs.totals), func(i int) bool { return chs.totals[i] > r })
	// Floating-point multiplication can round up to the upper endpoint.
	if i == len(chs.data) {
		i--
	}
	return chs.data[i]
}
