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
	s := &Sensor{
		device:      device,
		model:       *model,
		component:   component,
		configTopic: fmt.Sprintf("%s/%s/%s/%s/config", device.client.DiscoveryPrefix, component, device.client.id, id),
	}

	s.model.StateTopic = device.statusTopic
	s.model.UniqueID = id
	s.model.Device = &device.model

	err := s.Refresh()
	if err != nil {
		return nil, err
	}

	println("storing:", id)
	device.client.entities[id] = s

	return s, nil
}

func (s *Sensor) Refresh() error {
	data, err := json.Marshal(s.model)
	if err != nil {
		return err
	}

	fmt.Printf("send: %s\n%s\n", s.configTopic, string(data))

	tok := s.device.client.Client.Publish(s.configTopic, 1, false, data)
	tok.WaitTimeout(time.Second)
	return tok.Error()
}
