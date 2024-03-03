package main

import (
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/netleapio/zappy-framework/protocol"
)

type jsonDeviceUpdate struct {
	DeviceID string
	Alerts   []string
	Sensors  map[string]float64
}

type WebSocket struct {
	network      uint16
	eventChannel chan DeviceChange
	manager      *DeviceManager
	upgrader     websocket.Upgrader
}

func NewWebSocketListener() *WebSocket {
	return &WebSocket{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		eventChannel: make(chan DeviceChange, 10),
	}
}

func (ws *WebSocket) Init(manager *DeviceManager, network uint16) {
	ws.network = network
	ws.manager = manager
}

func (ws *WebSocket) Start() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := ws.upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity

		for {
			change := <-ws.eventChannel

			if change.Changes|ChangeDeviceUpdate != 0 {
				device := ws.manager.GetDevice(change.DeviceID)

				msg := jsonDeviceUpdate{
					DeviceID: strconv.Itoa(int(change.DeviceID)),
					Alerts:   device.alerts.Strings(),
					Sensors:  map[string]float64{},
				}

				for k, v := range device.sensors {
					t := protocol.SensorType(k)
					md := protocol.SensorMetadata[t]

					msg.Sensors[md.Name] = float64(v) * float64(md.Mult) / float64(md.Div)
				}

				err := conn.WriteJSON(msg)
				if err != nil {
					break
				}
			}
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	go http.ListenAndServe(":3456", nil)

}
