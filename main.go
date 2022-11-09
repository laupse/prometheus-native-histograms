// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// A simple example exposing fictional RPC latencies with different types of
// random distributions (uniform, normal, and exponential) as Prometheus
// metrics.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metrics struct {
	rpcDurationsHistogram       prometheus.Histogram
	rpcDurationsNativeHistogram prometheus.Histogram
}

func NewMetrics(reg prometheus.Registerer, normMean, normDomain float64) *metrics {
	m := &metrics{
		rpcDurationsHistogram: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "rpc_durations_histogram_seconds",
			Help: "RPC latency distributions.",
			Buckets: prometheus.LinearBuckets(
				normMean-5*normDomain,
				.5*normDomain,
				20,
			),
		}),
		rpcDurationsNativeHistogram: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:                        "rpc_durations_native_histogram_seconds",
			Help:                        "RPC latency distributions.",
			Buckets:                     nil,
			NativeHistogramBucketFactor: 1.1,
		}),
	}
	reg.MustRegister(m.rpcDurationsHistogram)
	reg.MustRegister(m.rpcDurationsNativeHistogram)
	return m
}

func main() {
	var (
		addr = flag.String(
			"listen-address",
			":8080",
			"The address to listen on for HTTP requests.",
		)
		normDomain = flag.Float64(
			"normal.domain",
			2,
			"The domain for the normal distribution.",
		)
		normMean = flag.Float64(
			"normal.mean",
			1,
			"The mean for the normal distribution.",
		)
		oscillationPeriod = flag.Duration(
			"oscillation-period",
			10*time.Minute,
			"The duration of the rate oscillation period.",
		)
	)

	flag.Parse()

	// Create a non-global registry.
	reg := prometheus.NewRegistry()

	// Create new metrics and register them using the custom registry.
	m := NewMetrics(reg, *normMean, *normDomain)
	// Add Go module build info.
	reg.MustRegister(collectors.NewBuildInfoCollector())

	start := time.Now()

	oscillationFactor := func() float64 {
		return 2 + math.Sin(
			math.Sin(2*math.Pi*float64(time.Since(start))/float64(*oscillationPeriod)),
		)
	}

	go func() {
		for {
			v := (rand.NormFloat64() * *normDomain) + *normMean
			// m.rpcDurations.WithLabelValues("normal").Observe(v)
			// Demonstrate exemplar support with a dummy ID. This
			// would be something like a trace ID in a real
			// application.  Note the necessary type assertion. We
			// already know that rpcDurationsHistogram implements
			// the ExemplarObserver interface and thus don't need to
			// check the outcome of the type assertion.
			m.rpcDurationsHistogram.(prometheus.ExemplarObserver).ObserveWithExemplar(
				v, prometheus.Labels{"dummyID": fmt.Sprint(rand.Intn(100000))},
			)
			m.rpcDurationsNativeHistogram.(prometheus.ExemplarObserver).ObserveWithExemplar(
				v, prometheus.Labels{"dummyID": fmt.Sprint(rand.Intn(100000))},
			)
			time.Sleep(time.Duration(75*oscillationFactor()) * time.Millisecond)
		}
	}()

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
			// Pass custom registry
			Registry: reg,
		},
	))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
