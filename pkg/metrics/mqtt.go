package metrics

import (
	"context"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func StartMQTTMetricsPublisher(ctx context.Context, desktopId string, mqttClient *mqtt.Client, interval time.Duration) {
	client := *mqttClient
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				client.Publish("desktops/"+desktopId+"/metrics", 0, true, GetMetrics())
			}
		}
	}()
}
