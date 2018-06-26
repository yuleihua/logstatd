package metric

import (
	"net/http"
	"time"

	st "airman.com/logstatd/pkg/stat"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "logstatd"
)

type Exporter struct {
	timeout       time.Duration
	up            *prometheus.Desc
	uptime        *prometheus.Desc
	version       *prometheus.Desc
	request       *prometheus.Desc
	success       *prometheus.Desc
	jobq          *prometheus.Desc
	failed        *prometheus.Desc
	del_failed    *prometheus.Desc
	recovery_init *prometheus.Desc
	recovry_succ  *prometheus.Desc
	recovery      *prometheus.Desc
}

func NewExporter(timeout time.Duration) *Exporter {
	return &Exporter{
		timeout: timeout,
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "up"),
			"Could the server be reached.",
			nil,
			nil,
		),
		uptime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "uptime_seconds"),
			"Number of seconds since the server started.",
			nil,
			nil,
		),
		version: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "version"),
			"The version of this lion server.",
			[]string{"version"},
			nil,
		),
		request: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "request_total"),
			"Total number of request.",
			nil,
			nil,
		),
		jobq: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "job_total"),
			"Total number of job.",
			nil,
			nil,
		),
		success: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "success_total"),
			"Total number of success.",
			nil,
			nil,
		),
		failed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "failed_total"),
			"Total number of failed.",
			nil,
			nil,
		),
		del_failed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "del_failed_total"),
			"Total number of delete failed.",
			nil,
			nil,
		),
		recovery_init: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "recovey_init_total"),
			"Total number of recovery init.",
			nil,
			nil,
		),
		recovry_succ: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "recovey_succ_total"),
			"Total number of recovery success.",
			nil,
			nil,
		),
		recovery: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "recovey_job_total"),
			"Total number of recovery failed.",
			nil,
			nil,
		),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up
	ch <- e.uptime
	ch <- e.version
	ch <- e.jobq
	ch <- e.request
	ch <- e.success
	ch <- e.failed
	ch <- e.del_failed
	ch <- e.recovery_init
	ch <- e.recovry_succ
	ch <- e.recovery
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(e.up, prometheus.CounterValue, float64(st.GetStat().Status))
	ch <- prometheus.MustNewConstMetric(e.uptime, prometheus.GaugeValue, time.Now().Sub(st.GetStat().UpTime).Seconds())
	ch <- prometheus.MustNewConstMetric(e.version, prometheus.GaugeValue, 1.0)

	ch <- prometheus.MustNewConstMetric(e.jobq, prometheus.CounterValue, float64(st.GetStat().JobNum))
	ch <- prometheus.MustNewConstMetric(e.request, prometheus.CounterValue, float64(st.GetStat().ReqNum))
	ch <- prometheus.MustNewConstMetric(e.success, prometheus.CounterValue, float64(st.GetStat().SuccNum))
	ch <- prometheus.MustNewConstMetric(e.failed, prometheus.CounterValue, float64(st.GetStat().ErrNum))
	ch <- prometheus.MustNewConstMetric(e.del_failed, prometheus.CounterValue, float64(st.GetStat().DelErrNum))
	ch <- prometheus.MustNewConstMetric(e.recovery_init, prometheus.CounterValue, float64(st.GetStat().RecovInitNum))
	ch <- prometheus.MustNewConstMetric(e.recovry_succ, prometheus.CounterValue, float64(st.GetStat().RecovSuccNum))
	ch <- prometheus.MustNewConstMetric(e.recovery, prometheus.CounterValue, float64(st.GetStat().RecovNum))
}

func WebMetricHandler(timeout time.Duration) http.Handler {
	prometheus.MustRegister(NewExporter(timeout))
	return prometheus.Handler()
}
