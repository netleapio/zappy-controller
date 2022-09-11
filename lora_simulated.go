//go:build simulated

package main

import (
	"log"
	"net"
)

const (
	srvAddr         = "224.0.0.1:9999"
	maxDatagramSize = 256
)

type radio struct {
	lc *net.UDPConn
	bc *net.UDPConn
}

func (r *radio) Init() error {
	addr, err := net.ResolveUDPAddr("udp", srvAddr)
	if err != nil {
		log.Fatal(err)
	}
	bc, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	r.bc = bc

	lc, err := net.ListenMulticastUDP("udp", nil, addr)
	lc.SetReadBuffer(maxDatagramSize)
	if err != nil {
		return err
	}
	r.lc = lc

	return nil
}

func (r *radio) Close() error {
	if r.lc != nil {
		r.lc.Close()
	}

	if r.bc != nil {
		r.bc.Close()
	}

	return nil
}

func (r *radio) Rx(timeoutMs uint32, buf []byte) (int, error) {
	n, _, err := r.lc.ReadFromUDP(buf)
	return n, err
}
