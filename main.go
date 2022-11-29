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
	gonum "gonum.org/v1/gonum/stat/distuv"
)

var oscillationFactor func() float64

type metrics struct {
	rpcDurationsHistogram       prometheus.Histogram
	rpcDurationsNativeHistogram prometheus.Histogram
}

func NewMetrics(reg prometheus.Registerer, normMean, normDomain float64) *metrics {
	m := &metrics{
		rpcDurationsHistogram: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "rpc_durations_histogram_seconds",
			Help:    "RPC latency distributions.",
			Buckets: prometheus.ExponentialBuckets(2, 2, 5),
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

func observer(ls []gonum.LogNormal, m *metrics) {
	for {

		for _, l := range ls {
			v := l.Rand()
			m.rpcDurationsHistogram.(prometheus.ExemplarObserver).ObserveWithExemplar(
				v, prometheus.Labels{"dummyID": fmt.Sprint(rand.Intn(100000))},
			)
			m.rpcDurationsNativeHistogram.(prometheus.ExemplarObserver).ObserveWithExemplar(
				v, prometheus.Labels{"dummyID": fmt.Sprint(rand.Intn(100000))},
			)
		}
		wait := time.Duration(75*oscillationFactor()) * time.Millisecond
		time.Sleep(wait)
	}
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
		// expRate = flag.Float64(
		// 	"exponential.rate",
		// 	0.05,
		// 	"The rate for the exponential distribution.",
		// )
		// logNormNu = flag.Float64(
		// 	"lognormal.nu",
		// 	2,
		// 	"The domain for the normal distribution.",
		// )
		// logNormSigma = flag.Float64(
		// 	"lognormal.sigma",
		// 	0.20,
		// 	"The domain for the normal distribution.",
		// )
		shouldOscillate = flag.Bool(
			"should-oscilate",
			false,
			"",
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

	oscillationFactor = func() float64 {
		if *shouldOscillate {
			return 2 + math.Sin(
				math.Sin(2*math.Pi*float64(time.Since(start))/float64(*oscillationPeriod)),
			)
		}
		return 1
	}

	ls := []gonum.LogNormal{
		{
			Mu:    2.7,
			Sigma: 0.1,
			Src:   nil,
		},
		{
			Mu:    1.6,
			Sigma: 0.2,
			Src:   nil,
		},
	}
	go observer(ls, m)

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
