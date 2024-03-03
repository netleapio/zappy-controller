package hassiomqtt

import (
	"encoding/json"
	"fmt"
	"time"
)

type Sensor struct {
	device      *Device
	model       SensorModel
	component   string
	configTopic string
}

func NewSensor(device *Device, component string, id string, model *SensorModel) (*Sensor, error) {
	e := &Sensor{
		device:      device,
		model:       *model,
		component:   component,
		configTopic: fmt.Sprintf("%s/%s/%s/%s/config", device.client.DiscoveryPrefix, component, device.client.id, id),
	}

	e.model.StateTopic = device.statusTopic
	e.model.UniqueID = id
	e.model.Device = &device.model

	data, err := json.Marshal(e.model)
	if err != nil {
		return nil, err
	}

	fmt.Printf("send: %s\n%s\n", e.configTopic, string(data))

	tok := device.client.Client.Publish(e.configTopic, 1, false, data)
	tok.WaitTimeout(time.Second)
	err = tok.Error()
	if err != nil {
		panic(err)
	}

	return e, nil
}
