package hassiomqtt

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	Client          mqtt.Client
	id              string
	DiscoveryPrefix string
	entities        map[string]*Sensor // Change Sensor to a generic Entity in future
}

func NewClient(broker string, port int, clientId string, user string, password string) *Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID(clientId)
	opts.SetUsername(user)
	opts.SetPassword(password)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)

	c := &Client{
		id:              clientId,
		DiscoveryPrefix: "homeassistant",
		entities:        make(map[string]*Sensor),
	}

	c.Client = mqtt.NewClient(opts)
	return c
}

func (c *Client) Start() {
	go func() {
		for !c.Client.IsConnected() {
			tok := c.Client.Connect()
			ok := tok.WaitTimeout(time.Second)
			if !ok {
				println("timeout connecting to MQTT broker, retrying")
				continue
			}
			err := tok.Error()
			if err != nil {
				fmt.Printf("error connecting to MQTT broker: %v\n", err)
				time.Sleep(5 * time.Second)
			}
		}

		c.Client.Subscribe("homeassistant/status", 0, func(cl mqtt.Client, m mqtt.Message) {
			println("hass status changed:", string(m.Payload()))

			for k, v := range c.entities {
				println("refreshing:", k)
				v.Refresh()
			}
		})
	}()
}
