package mqtt

import (
	"context"
	"fmt"
)

type LocalMQTTConfigurator struct {
	Host        string
	Password    string
	Username    string
	TopicPrefix string
}

func NewLocalMQTTConfigurator(host, desktopPSK, desktopID string) *LocalMQTTConfigurator {
	return &LocalMQTTConfigurator{
		Host:        host,
		Password:    desktopPSK,
		Username:    "desktop:" + desktopID,
		TopicPrefix: fmt.Sprintf("desktops/%v/", desktopID),
	}
}

func (c *LocalMQTTConfigurator) GetConfiguration(ctx context.Context) *MQTTConfig {
	return &MQTTConfig{
		Host:        c.Host,
		Password:    c.Password,
		Username:    c.Username,
		TopicPrefix: c.TopicPrefix,
	}
}
