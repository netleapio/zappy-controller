//go:build !simulated

package main

import (
	"fmt"

	"github.com/netleapio/zappy-controller/sx127x"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/host/v3"
)

type radio struct {
	port spi.PortCloser
	dev  sx127x.Device
}

func (r *radio) Init() error {
	_, err := host.Init()
	if err != nil {
		return err
	}

	r.port, err = spireg.Open("/dev/spidev0.1")
	if err != nil {
		return err
	}

	rst := gpioreg.ByName("GPIO22")
	dio0 := gpioreg.ByName("GPIO25")

	dev, err := sx127x.New(r.port, rst, dio0)
	if err != nil {
		return err
	}
	d.radio = dev

	if dev.Detect() {
		fmt.Println("Detected!")
	} else {
		fmt.Println("Not detected!")
	}

	err = dev.Configure(sx127x.Config{
		Frequency: 868100000, // 868.1MHz
		CRC:       sx127x.CrcModeOn,
	})
	return err
}

func (r *radio) Close() error {
	return r.port.Close()
}

func (r *radio) Rx(timeoutMs uint32, buf []byte) (int, error) {
	return r.dev.LoraRxTo(timeoutMs, buf)
}
