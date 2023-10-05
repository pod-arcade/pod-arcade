package metrics

import (
	"context"

	"github.com/pion/webrtc/v4"
)

type WebRTCSessionMetrics struct {
	// This will contain a pointer to each different type of metric we want to expose.
	// We need to keep track of and cache these, since it's a very bad practice of calling NewCounter/NewGauge/etc every
	// time we do it.
}

var WebRTCSessionMetricsCache = map[string]WebRTCSessionMetrics{}

func ProcessWebRTCStats(ctx context.Context, sessionId string, statsReport webrtc.StatsReport) {
	// TODO: actually do something.
}
