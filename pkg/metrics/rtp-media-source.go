package metrics

import (
	"github.com/pion/rtp"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/media"
	"github.com/prometheus/client_golang/prometheus"
)

func CaptureMetricsForMediaSource(ms media.RTPMediaSource) {
	ms.OnDroppedRTPPacket(func(p *rtp.Packet) {
		metricName := "rtp_dropped_packets"
		mediaName := ms.GetName()
		mediaType := ms.GetType()
		mediaTypeString := ""

		switch mediaType {
		case media.TYPE_VIDEO:
			mediaTypeString = "video"
		case media.TYPE_AUDIO:
			mediaTypeString = "audio"
		}

		labels := prometheus.Labels{
			"media_name": mediaName,
			"media_type": mediaTypeString,
		}

		counter := GlobalMetricCache.GetCounter(metricName, labels)
		counter.Inc()
	})
}
