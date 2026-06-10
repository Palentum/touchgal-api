package httpserver

import (
	"expvar"
	"net/http"
	"net/http/pprof"
	"sync"
	"sync/atomic"
)

type RequestLogMetrics interface {
	Dropped() uint64
	Pending() int
	Capacity() int
}

type requestLogMetricsHolder struct {
	metrics RequestLogMetrics
}

var (
	requestLogMetricsValue       atomic.Value
	requestLogMetricsPublishOnce sync.Once
)

func PublishRequestLogMetrics(writer RequestLogMetrics) {
	if writer == nil {
		return
	}
	requestLogMetricsValue.Store(requestLogMetricsHolder{metrics: writer})
	requestLogMetricsPublishOnce.Do(func() {
		publishExpvar("api_request_log_dropped_total", expvar.Func(func() any {
			metrics := currentRequestLogMetrics()
			if metrics == nil {
				return uint64(0)
			}
			return metrics.Dropped()
		}))
		publishExpvar("api_request_log_queue_pending", expvar.Func(func() any {
			metrics := currentRequestLogMetrics()
			if metrics == nil {
				return 0
			}
			return metrics.Pending()
		}))
		publishExpvar("api_request_log_queue_capacity", expvar.Func(func() any {
			metrics := currentRequestLogMetrics()
			if metrics == nil {
				return 0
			}
			return metrics.Capacity()
		}))
	})
}

func currentRequestLogMetrics() RequestLogMetrics {
	holder, _ := requestLogMetricsValue.Load().(requestLogMetricsHolder)
	return holder.metrics
}

func publishExpvar(name string, variable expvar.Var) {
	if expvar.Get(name) == nil {
		expvar.Publish(name, variable)
	}
}

func NewObservabilityRouter(enablePprof, enableMetrics bool) http.Handler {
	mux := http.NewServeMux()
	if enableMetrics {
		mux.Handle("/debug/vars", expvar.Handler())
	}
	if enablePprof {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	return mux
}
