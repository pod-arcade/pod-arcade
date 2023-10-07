package metrics

import (
	"net/http"
	"strings"

	"github.com/pod-arcade/pod-arcade/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/expfmt"
	"github.com/rs/zerolog"
)

var l zerolog.Logger

var Factory promauto.Factory

func init() {
	l = logger.CreateLogger(map[string]string{
		"Component": "Metrics",
	})
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

func Handle() http.Handler {
	return promhttp.Handler()
}
