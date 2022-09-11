package main

import (
	"log"
	"sync"
	"time"

	"github.com/netleapio/zappy-framework/protocol"
)

const (
	// Constant for now - may be adaptable in future
	DeviceUpdatePeriod = 1 * time.Minute
)

type DeviceChangeTypes int

const (
	ChangeNone      DeviceChangeTypes = 0
	ChangeNewDevice                   = 1 << iota
	ChangeDeviceUpdate
	ChangeDeviceGone
)

type DeviceChange struct {
	Changes  DeviceChangeTypes
	DeviceID uint16
}

// DeviceManager keeps track of all known devices and their current state.
//
// DeviceManager is specific to a given 'network', so a device ID is
// considered unique.
//
// DeviceManager will stop tracking devices that have not been seen for
// three update periods.
type DeviceManager struct {
	lock      sync.Mutex
	devices   map[uint16]*DeviceState
	listeners []chan DeviceChange
}

type DeviceState struct {
	id       uint16
	lastSeen time.Time
	alerts   protocol.Alerts
	sensors  map[protocol.SensorType]uint16
}

func NewDeviceManager() *DeviceManager {
	return &DeviceManager{
		lock:      sync.Mutex{},
		devices:   make(map[uint16]*DeviceState),
		listeners: make([]chan DeviceChange, 0),
	}
}

func (m *DeviceManager) Start() {
	go m.cleanupDevices()
}

func (m *DeviceManager) DeviceSensorUpdate(rpt *protocol.SensorReport) {
	changes := ChangeNone

	d := m.getOrCreate(&changes, rpt.Packet().DeviceID())

	d.lastSeen = time.Now()
	d.alerts = rpt.Packet().Alerts()

	// Add to existing sensor readings in case device sends an incomplete
	// set of readings
	for k, v := range rpt.AllReadings() {
		d.sensors[k] = v
	}

	changes |= ChangeDeviceUpdate

	m.notifyListeners(rpt.Packet().DeviceID(), changes)
}

func (m *DeviceManager) GetDevice(id uint16) *DeviceState {
	return m.get(id)
}

func (m *DeviceManager) AddListener(ch chan DeviceChange) {
	m.listeners = append(m.listeners, ch)
}

func (m *DeviceManager) get(id uint16) *DeviceState {
	var device *DeviceState

	m.doLocked(func() error {
		d, ok := m.devices[id]
		if ok {
			device = d
		}
		return nil
	})

	return device
}

func (m *DeviceManager) getOrCreate(changes *DeviceChangeTypes, id uint16) *DeviceState {
	var device *DeviceState

	m.doLocked(func() error {
		d, ok := m.devices[id]
		if !ok {
			*changes |= ChangeNewDevice
			d = &DeviceState{
				id:      id,
				sensors: map[protocol.SensorType]uint16{},
			}
			m.devices[id] = d
		}
		device = d
		return nil
	})

	return device
}

func (m *DeviceManager) cleanupDevices() {
	for {
		time.Sleep(DeviceUpdatePeriod)
		now := time.Now()
		m.doLocked(func() error {
			toRemove := []*DeviceState{}

			for _, d := range m.devices {
				if now.Sub(d.lastSeen) > 2*DeviceUpdatePeriod {
					toRemove = append(toRemove, d)
				}
			}

			for _, d := range toRemove {
				log.Printf("Device #%04x timed-out", d.id)
				delete(m.devices, d.id)
				m.notifyListeners(d.id, ChangeDeviceGone)
			}

			return nil
		})
	}
}

func (m *DeviceManager) notifyListeners(id uint16, changes DeviceChangeTypes) {
	notification := DeviceChange{DeviceID: id, Changes: changes}

	for _, ch := range m.listeners {
		select {
		case ch <- notification:
		default:
		}
	}
}

func (m *DeviceManager) doLocked(fn func() error) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return fn()
}
