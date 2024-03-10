package main

import (
	"encoding/json"
	"os"
)

type MQTTSettings struct {
	Broker          string `json:"broker"`
	Port            int    `json:"port"`
	User            string `json:"user"`
	Password        string `json:"password"`
	ClientID        string `json:"clientId"`
	DiscoveryPrefix string `json:"discoveryPrefix"`
}

type Config struct {
	Mqtt MQTTSettings `json:"mqtt"`
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile("home.config")
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
