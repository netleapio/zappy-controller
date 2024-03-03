package hassiomqtt

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	Client          mqtt.Client
	id              string
	DiscoveryPrefix string
}

func NewClient(broker string, port int, clientId string, user string, password string) *Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID(clientId)
	opts.SetUsername(user)
	opts.SetPassword(password)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)

	return &Client{
		Client:          mqtt.NewClient(opts),
		id:              clientId,
		DiscoveryPrefix: "homeassistant",
	}
}
