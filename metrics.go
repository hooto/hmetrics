// Copyright 2023 Eryx <evorui at gmail dot com>, All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hmetrics

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/ServiceWeaver/weaver/runtime/metrics"
	"github.com/ServiceWeaver/weaver/runtime/protos"
)

type Label struct {
	Name string
	Item string
}

type MetricCounterMap interface {
	Add(string, string, float64)
	Set(string, string, float64)
}

type MetricGaugeMap interface {
	Add(string, string, float64)
	Set(string, string, float64)
}

type MetricHistogramMap interface {
	Add(string, string, float64)
}

type MetricComplexMap interface {
	Add(name, item string, c float64, g float64, t time.Duration)
}

var (
	mu             sync.Mutex
	complexMetrics = map[string]MetricComplexMap{}
)

func RegisterCounterMap(name, help string) MetricCounterMap {
	return &counterMap{
		metric: metrics.RegisterMap[Label](
			protos.MetricType_COUNTER,
			name,
			help,
			nil,
		),
	}
}

func RegisterGaugeMap(name, help string) MetricGaugeMap {
	return &gaugeMap{
		metric: metrics.RegisterMap[Label](
			protos.MetricType_GAUGE,
			name,
			help,
			nil,
		),
	}
}

func RegisterHistogramMap(name, help string, buckets []float64) MetricHistogramMap {
	return &histogramMap{
		metric: metrics.RegisterMap[Label](
			protos.MetricType_HISTOGRAM,
			name,
			help,
			buckets,
		),
	}
}

func NewBuckets(start, factor float64, count int) []float64 {
	if count < 1 {
		panic("NewBuckets needs a positive count")
	}
	if start <= 0 {
		panic("NewBuckets needs a positive start value")
	}
	if factor <= 1 {
		panic("NewBuckets needs a factor greater than 1")
	}
	buckets := make([]float64, count)
	for i := range buckets {
		buckets[i] = start
		start *= factor
	}
	return buckets
}

func HttpHandler(w http.ResponseWriter, _ *http.Request) {
	var buf bytes.Buffer
	translateMetricsToPrometheusTextFormat(&buf, metrics.Snapshot())
	w.Write(buf.Bytes())
}

type counter struct {
	metric *metrics.Metric
}

func (it *counter) Add(v float64) {
	it.metric.Add(v)
}

type counterMap struct {
	metric *metrics.MetricMap[Label]
}

func (it *counterMap) Add(name, item string, v float64) {
	it.metric.Get(Label{name, item}).Add(v)
}

func (it *counterMap) Set(name, item string, v float64) {
	it.metric.Get(Label{name, item}).Set(v)
}

type gaugeMap struct {
	metric *metrics.MetricMap[Label]
}

func (it *gaugeMap) Add(name, item string, v float64) {
	it.metric.Get(Label{name, item}).Add(v)
}

func (it *gaugeMap) Set(name, item string, v float64) {
	it.metric.Get(Label{name, item}).Set(v)
}

type histogramMap struct {
	metric *metrics.MetricMap[Label]
}

func (it *histogramMap) Add(name, item string, v float64) {
	it.metric.Get(Label{name, item}).Put(v)
}

type complexMap struct {
	counter   *metrics.MetricMap[Label]
	gauge     *metrics.MetricMap[Label]
	histogram *metrics.MetricMap[Label]
}

func RegisterComplexMap(name, help string, buckets []float64) MetricComplexMap {
	mu.Lock()
	defer mu.Unlock()
	m, ok := complexMetrics[name]
	if !ok {
		m = &complexMap{
			counter: metrics.RegisterMap[Label](
				protos.MetricType_COUNTER,
				name+"_counter",
				help,
				nil,
			),
			gauge: metrics.RegisterMap[Label](
				protos.MetricType_GAUGE,
				name+"_gauge",
				help,
				nil,
			),
			histogram: metrics.RegisterMap[Label](
				protos.MetricType_HISTOGRAM,
				name+"_histogram",
				help,
				buckets,
			),
		}
		complexMetrics[name] = m
	}
	return m
}

func (it *complexMap) Add(name, item string, c, g float64, h time.Duration) {
	l := Label{name, item}
	if c > 0 {
		it.counter.Get(l).Add(c)
	}
	if g != 0 {
		it.gauge.Get(l).Add(g)
	}
	if h >= 0 {
		// it.histogram.Get(l).Put(float64(h / 1e3))
		it.histogram.Get(l).Put(h.Seconds())
	}
}
