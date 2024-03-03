package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/netleapio/zappy-framework/protocol"
)

const (
	NetworkID = 0
)

type status struct {
	Temperature float32 `json:"temperature"`
	Humidity    float32 `json:"humidity"`
}

func mainImpl() error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	mgr := NewDeviceManager()

	metrics := NewPrometheusListener()
	metrics.Init(mgr, NetworkID)
	mgr.AddListener(metrics.eventChannel)

	websocket := NewWebSocketListener()
	websocket.Init(mgr, NetworkID)
	mgr.AddListener(websocket.eventChannel)

	mqttBroker := NewMQTTListener(&cfg.Mqtt)
	mqttBroker.Init(mgr, NetworkID)
	mgr.AddListener(mqttBroker.eventChannel)

	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
	mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)

	websocket.Start()
	metrics.Start()
	mqttBroker.Start()
	mgr.Start()

	radio := radio{}
	err = radio.Init()
	if err != nil {
		return err
	}

	pkt := protocol.Packet{}

	for {
		pkt.SetLength(255)
		n, err := radio.Rx(1000*120, pkt.AsBytes())
		if err != nil {
			return err
		}
		pkt.SetLength(uint8(n))
		if n == 0 {
			continue
		}

		log.Println("received:")
		log.Println(hex.Dump(pkt.AsBytes()))

		msg := protocol.DetectMessage(&pkt)

		if msg == nil {
			log.Println("unknown packet")
			continue
		}

		if pkt.NetworkID() != NetworkID {
			log.Println("packet for other network, skipping")
			continue
		}

		log.Printf("Network: 0x%04x\n", pkt.NetworkID())
		log.Printf("Device: 0x%04x\n", pkt.DeviceID())
		log.Printf("Version: %d\n", pkt.Version())
		log.Printf("Alerts: 0x%04x %s\n", uint16(pkt.Alerts()), pkt.Alerts())
		log.Printf("Type: #%#v\n", pkt.Type())

		switch msg.Packet().Type() {
		case protocol.TypeSensorReport:
			rpt := msg.(*protocol.SensorReport)
			if rpt.HasBatteryVoltage() {
				log.Printf("Batt: %.3f V\n", float32(rpt.BatteryVoltage())/1000)
			}
			if rpt.HasTemperature() {
				log.Printf("Temp: %.2f C\n", float32(rpt.Temperature())/100)
			}
			if rpt.HasPressure() {
				log.Printf("Pressure: %.1f mbar\n", float32(rpt.Pressure())/10)
			}
			if rpt.HasHumidity() {
				log.Printf("Humidity: %.2f %%\n", float32(rpt.Humidity())/100)
			}
			mgr.DeviceSensorUpdate(rpt)
		}
	}

	return radio.Close()
}

func main() {
	fmt.Println("zappy-controller")

	if err := mainImpl(); err != nil {
		fmt.Fprintf(os.Stderr, "zappy-controller: %s.\n", err)
		os.Exit(1)
	}
}
