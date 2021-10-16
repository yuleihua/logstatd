// copyright carousell
// this file coming from : https://github.com/carousell/fasthttp-prometheus-middleware/blob/master/prometheus.go

package middleware

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// Prometheus contains the metrics gathered by the instance and its path
type Prometheus struct {
	uri      string
	reqCnt   *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// NewPrometheus generates a new set of metrics with a certain subsystem name
func NewPrometheus(subsystem, uri string) *Prometheus {
	p := &Prometheus{uri: uri}
	p.registerMetrics(subsystem)
	return p
}

func (p *Prometheus) registerMetrics(subsystem string) {
	p.duration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: subsystem,
			Name:      "request_duration_seconds",
			Help:      "request latencies",
			Buckets:   []float64{.005, .01, .02, 0.04, .06, 0.08, .1, 0.15, .25, 0.4, .6, .8, 1, 1.5, 2, 3, 5},
		},
		[]string{"code", "path"},
	)

	p.reqCnt = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "requests_total",
			Help:      "The HTTP request counts processed.",
		},
		[]string{"code", "method"},
	)

	prometheus.MustRegister(p.reqCnt, p.duration)
}

func (p *Prometheus) Handle() func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			uri := string(ctx.Request.URI().Path())
			if uri == p.uri {
				// next
				h(ctx)
				return
			}

			start := time.Now()
			// next
			h(ctx)

			status := strconv.Itoa(ctx.Response.StatusCode())
			elapsed := float64(time.Since(start)) / float64(time.Second)
			ep := string(ctx.Method()) + "_" + uri
			p.duration.WithLabelValues(status, ep).Observe(elapsed)
			p.reqCnt.WithLabelValues(status, string(ctx.Method())).Inc()
		}
	}
}

// since prometheus/client_golang use net/http we need this net/http adapter for fasthttp
func PrometheusHandler() fasthttp.RequestHandler {
	return fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())
}
