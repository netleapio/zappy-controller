package main

import (
	"fmt"
	"strings"
	"time"

	hassiomqtt "github.com/netleapio/zappy-controller/hassio-mqtt"
	"github.com/netleapio/zappy-framework/protocol"
)

var hassSensorMetadata = map[protocol.SensorType]struct {
	deviceClass string
	units       string
	icon        *string
}{
	protocol.SensorTypeTemperature: {deviceClass: "temperature", units: "°C"},
	protocol.SensorTypeHumidity:    {deviceClass: "humidity", units: "%"},
	protocol.SensorTypePressure:    {deviceClass: "atmospheric_pressure", units: "Pa"},
	protocol.SensorTypeBattVolts:   {deviceClass: "voltage", units: "V"},
}

type mqttDevice struct {
	hassDevice   *hassiomqtt.Device
	hassEntities map[protocol.SensorType]*hassiomqtt.Sensor
}

type MQTTListener struct {
	network      uint16
	eventChannel chan DeviceChange
	mqtt         *hassiomqtt.Client
	manager      *DeviceManager
	devices      map[uint16]mqttDevice
}

func NewMQTTListener(cfg *MQTTSettings) *MQTTListener {
	return &MQTTListener{
		eventChannel: make(chan DeviceChange, 10),
		mqtt:         hassiomqtt.NewClient(cfg.Broker, cfg.Port, cfg.ClientID, cfg.User, cfg.Password),
		devices:      map[uint16]mqttDevice{},
	}
}

func (l *MQTTListener) Init(manager *DeviceManager, network uint16) {
	l.network = network
	l.manager = manager
}

func (l *MQTTListener) Start() {
	go func() {
		if !l.mqtt.Client.IsConnected() {
			tok := l.mqtt.Client.Connect()
			for {
				ok := tok.WaitTimeout(time.Second)
				if !ok {
					println("timeout connecting to MQTT broker, retrying")
					continue
				}
				err := tok.Error()
				if err != nil {
					fmt.Printf("error connecting to MQTT broker: %v\n", err)
					time.Sleep(time.Second)
				}
				break
			}
		}
	}()

	go func() {
		for {
			change := <-l.eventChannel
			d := l.manager.GetDevice(change.DeviceID)
			if d == nil {
				l.removeDevice(change.DeviceID)
				continue
			} else if change.Changes&ChangeNewDevice != 0 {
				l.newDevice(d)
			}

			l.updateSensorStats(d)
		}
	}()
}

func (l *MQTTListener) newDevice(d *DeviceState) {
	println("new device")
	dev, ok := l.devices[d.id]
	if !ok {
		deviceId := fmt.Sprintf("zappy_%d_%d", l.network, d.id)
		deviceName := fmt.Sprintf("Zappy Environment Sensor #%d", d.id)

		dev = mqttDevice{
			hassDevice: hassiomqtt.NewDevice(l.mqtt, fmt.Sprintf("%d", d.id), &hassiomqtt.DeviceModel{
				Identifiers:  []string{deviceId},
				Manufacturer: "Zappy",
				Model:        "Zappy Environment Sensor",
				Name:         deviceName,
				SerialNumber: fmt.Sprintf("%d", d.id),
			}),
			hassEntities: map[protocol.SensorType]*hassiomqtt.Sensor{},
		}

		for t, _ := range d.sensors {
			md, ok := protocol.SensorMetadata[t]
			if !ok {
				continue
			}

			hassMd, ok := hassSensorMetadata[t]
			if !ok {
				continue
			}

			_, ok = dev.hassEntities[t]
			if !ok {

				sensorId := fmt.Sprintf("%s_%s", deviceId, md.Name)

				s, err := hassiomqtt.NewSensor(dev.hassDevice, "sensor", sensorId, &hassiomqtt.SensorModel{
					EntityModel: hassiomqtt.EntityModel{
						DeviceClass:   hassMd.deviceClass,
						Name:          md.Name,
						ObjectID:      fmt.Sprintf("%s_%s", deviceId, hassMd.deviceClass),
						ValueTemplate: fmt.Sprintf("{{value_json.%s}}", hassMd.deviceClass),
					},
					SuggestedDisplayPrecision: 2,
					UnitOfMeasurement:         hassMd.units,
				})
				if err != nil {
					continue
				}
				dev.hassEntities[t] = s

				l.devices[d.id] = dev
			}

		}
	}

	l.updateSensorStats(d)
}

func (l *MQTTListener) removeDevice(id uint16) {
}

func (l *MQTTListener) updateSensorStats(d *DeviceState) {
	println("updateSensorStats")

	dev, ok := l.devices[d.id]
	if !ok {
		return
	}

	sb := strings.Builder{}
	sb.WriteString("{")
	prefix := ""
	for t, v := range d.sensors {
		md, ok := protocol.SensorMetadata[t]
		if !ok {
			continue
		}

		hassMd, ok := hassSensorMetadata[t]
		if !ok {
			continue
		}

		value := (float32(v) * float32(md.Mult)) / float32(md.Div)

		sb.WriteString(fmt.Sprintf("%s\"%s\":%v", prefix, hassMd.deviceClass, value))
		prefix = ","
	}
	sb.WriteString("}")

	dev.hassDevice.SendStatus(sb.String())
}
