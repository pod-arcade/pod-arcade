package metrics

import (
	"context"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func StartSimpleMQTTMetricsPublisher(ctx context.Context, topicPrefix string, mqttClient *mqtt.Client, interval time.Duration) {
	client := *mqttClient
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				client.Publish(topicPrefix+"metrics", 0, false, GetMetrics())
			}
		}
	}()
}

func StartAdvancedMQTTMetricsPublisher(ctx context.Context, topicPrefix string, mqttClient *mqtt.Client, interval time.Duration) {
	client := *mqttClient
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m := GetMetricsByKey()
				for k, v := range m {
					client.Publish(topicPrefix+"metrics/"+k, 0, false, v)
				}
			}
		}
	}()
}
