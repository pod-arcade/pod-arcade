package mqtt

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/rs/zerolog"
)

type CloudMQTTConfigurator struct {
	cloudURL string
	authKey  string

	nextFetchTime time.Time
	lastConfig    *MQTTConfig

	l zerolog.Logger

	fetchLock sync.Mutex
}

type CloudConfigResponse struct {
	Password    string `json:"password"`
	Username    string `json:"username"`
	BrokerURL   string `json:"broker_url"`
	TopicPrefix string `json:"topic_prefix"`
}

func NewCloudMQTTConfigurator(cloudURL, authKey string) *CloudMQTTConfigurator {
	configEndpoint := cloudURL + "/api/v1/desktop/config"

	return &CloudMQTTConfigurator{
		cloudURL: configEndpoint,
		authKey:  authKey,

		l: log.NewLogger("mqtt-cloud-configurator", map[string]string{"cloudURL": cloudURL}),
	}
}

func (c *CloudMQTTConfigurator) GetConfiguration(ctx context.Context) *MQTTConfig {
	// Prevent multiple fetches from happening at the same time.
	c.fetchLock.Lock()
	defer c.fetchLock.Unlock()

	// retry forever until we successfully get the config.
	// exponentially backoff until we hit a max of 60 seconds
	if c.lastConfig == nil || time.Now().After(c.nextFetchTime) {
		config := &CloudConfigResponse{}
		delay := time.Second * time.Duration(60*rand.Float32())

		for {
			select {
			case <-ctx.Done():
				break
			default:
			}

			req, err := http.NewRequest("GET", c.cloudURL, nil)
			req.Header.Set("Authorization", c.authKey)
			if err != nil {
				c.l.Fatal().Msgf("Failed to create cloud config request. %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				c.l.Error().Msgf("Failed to get cloud config. %v", err)
				time.Sleep(delay)
				continue
			} else if resp.StatusCode != http.StatusOK {
				c.l.Error().Msgf("Failed to get cloud config with status code %v", resp.StatusCode)
				time.Sleep(delay)
				continue
			}

			err = json.NewDecoder(resp.Body).Decode(config)
			if err != nil {
				c.l.Error().Msgf("Failed to decode cloud config. %v", err)
				time.Sleep(delay)
				continue
			}

			break
		}
		c.lastConfig = &MQTTConfig{
			Host:        config.BrokerURL,
			Password:    config.Password,
			Username:    config.Username,
			TopicPrefix: config.TopicPrefix,
		}
		c.nextFetchTime = time.Now().Add(time.Second * 60)
	}
	return c.lastConfig
}
