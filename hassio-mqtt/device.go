package hassiomqtt

import (
	"fmt"
	"time"
)

type Device struct {
	client      *Client
	id          string
	statusTopic string
	model       DeviceModel
}

// NewDevice creates a new device with a unique id
func NewDevice(client *Client, id string, model *DeviceModel) *Device {
	return &Device{
		client:      client,
		id:          id,
		statusTopic: fmt.Sprintf("%s/%s/state", client.DiscoveryPrefix, id),
		model:       *model,
	}
}

func (d *Device) SendStatus(status interface{}) error {
	tok := d.client.Client.Publish(d.statusTopic, 0, false, status)
	for {
		ok := tok.WaitTimeout(time.Second)
		if ok {
			break
		}
		println("sendstatus: retry")
	}

	return tok.Error()
}
