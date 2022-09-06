package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/kenbell/pi-loratest/sx127x"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/host/v3"
)

func mainImpl() error {
	if _, err := host.Init(); err != nil {
		return err
	}

	port, err := spireg.Open("/dev/spidev0.0")
	if err != nil {
		return err
	}

	rst := gpioreg.ByName("GPIO6")
	dio0 := gpioreg.ByName("GPIO13")

	dev, err := sx127x.New(port, rst, dio0)
	if err != nil {
		return err
	}

	if dev.Detect() {
		fmt.Println("Detected!")
	} else {
		fmt.Println("Not detected!")
	}

	err = dev.Configure(sx127x.Config{
		Frequency: 868100000, // 868.1MHz
		CRC:       sx127x.CrcModeOn,
	})
	if err != nil {
		return err
	}

	for {
		data, err := dev.LoraRx(1000 * 120)
		if err != nil {
			return err
		}

		log.Println("received:")
		log.Println(hex.Dump(data))

		if len(data) >= 11 {
			id := data[0]
			ver := binary.BigEndian.Uint16(data[0x1:])
			battV := binary.BigEndian.Uint16(data[0x3:])
			temp := binary.BigEndian.Uint16(data[0x5:])
			pressure := binary.BigEndian.Uint16(data[0x7:])
			humidity := binary.BigEndian.Uint16(data[0x9:])

			log.Printf("ID: #%d\n", id)
			log.Printf("Ver: %d\n", ver)
			log.Printf("Batt: %f V\n", float32(battV)/1000)
			log.Printf("Temp: %f C\n", float32(temp)/100)
			log.Printf("Pressure: %f mbar\n", float32(pressure)/10)
			log.Printf("Humidity: %f %%\n", float32(humidity)/100)
		}
	}

	return port.Close()
}

func main() {

	fmt.Println("loratest")

	startPrometheus()

	if err := mainImpl(); err != nil {
		fmt.Fprintf(os.Stderr, "loratest: %s.\n", err)
		os.Exit(1)
	}
}
