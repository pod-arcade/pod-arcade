package metrics

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/rs/zerolog"
)

var l zerolog.Logger

var Factory promauto.Factory

func init() {
	l = log.NewLogger("Metrics", nil)
	Factory = promauto.With(prometheus.DefaultRegisterer)
}

func GetMetrics() string {
	builder := strings.Builder{}
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		l.Error().Err(err).Msg("Failed to gather metrics, returning empty metrics")
		return ""
	}
	enc := expfmt.NewEncoder(&builder, expfmt.FmtOpenMetrics_1_0_0)
	for _, mf := range metrics {
		enc.Encode(mf)
	}
	return builder.String()
}

func GetMetricsByKey() map[string]string {
	mm := map[string]string{}
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		l.Error().Err(err).Msg("Failed to gather metrics, returning empty metrics")
		return mm
	}
	for _, mf := range metrics {
		for _, m := range mf.Metric {
			metricName := mf.GetName()
			labels := "{"
			prefix := ""
			for _, l := range m.Label {
				labels += prefix + l.GetName() + "=\"" + EscapeKeyValue(l.GetValue()) + "\""
				prefix = ", "
			}
			labels += "}"

			var v float64

			if mf.Type == nil {
				continue
			}
			switch *mf.Type {
			case io_prometheus_client.MetricType_COUNTER:
				v = m.Counter.GetValue()
			case io_prometheus_client.MetricType_GAUGE:
				v = m.Gauge.GetValue()
			case io_prometheus_client.MetricType_HISTOGRAM:
				v = m.Histogram.GetSampleSum()
			case io_prometheus_client.MetricType_SUMMARY:
				v = m.Summary.GetSampleSum()
			default:
				l.Warn().Msgf("Unknown metric type %v", mf.Type)
			}
			mm[metricName] += fmt.Sprintf("%v %v\n", labels, v)
		}
	}
	return mm
}

func EscapeKeyValue(s string) string {
	var toReturn string = s
	toReturn = strings.ReplaceAll(toReturn, "\"", "\\\"")

	return toReturn
}

func Handle() http.Handler {
	return promhttp.Handler()
}
