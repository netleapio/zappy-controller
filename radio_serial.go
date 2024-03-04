//go:build !(simulated || sx127x)

package main

import (
	"io"

	"go.bug.st/serial"
)

type radio struct {
	port serial.Port
}

func (r *radio) Init(port string) error {
	p, err := serial.Open(port, &serial.Mode{BaudRate: 115200})
	if err != nil {
		return err
	}

	r.port = p

	return nil
}

func (r *radio) Rx(timeoutMs uint32, buf []byte) (int, error) {

	// len(marker) + len(len) + pkt
	hdr := make([]byte, 4)

	_, err := io.ReadAtLeast(r.port, hdr, 4)
	if err != nil {
		return 0, err
	}

	// If not synchronized, wipe out the read buffer
	for hdr[0] != 'P' || hdr[1] != 'K' || hdr[2] != 'T' {
		r.port.ResetInputBuffer()

		_, err = io.ReadAtLeast(r.port, hdr, 4)
		if err != nil {
			return 0, err
		}
	}

	pktlen := hdr[3]
	_, err = io.ReadAtLeast(r.port, buf, int(pktlen))
	if err != nil {
		return 0, err
	}

	return int(pktlen), nil
}

func (r *radio) Close() error {
	return r.port.Close()
}
