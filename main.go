package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/netleapio/zappy-framework/protocol"
)

const (
	NetworkID = 0
)

// buffer for protocol packets
var pkt protocol.Packet

func mainImpl() error {
	mgr := NewDeviceManager()
	metrics := NewPrometheusListener()
	metrics.Init(mgr, NetworkID)
	mgr.AddListener(metrics.eventChannel)

	metrics.Start()
	mgr.Start()

	radio := radio{}
	err := radio.Init()
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

		log.Printf("Network: #%04x\n", pkt.NetworkID())
		log.Printf("Device: #%04x\n", pkt.DeviceID())
		log.Printf("Version: %d\n", pkt.Version())
		log.Printf("Alerts: %#v\n", pkt.Alerts())
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
