package utils

import (
	"time"
)

// used in ws_svr.go to read json messages into structs
type ApiReq_WS struct {
	Messages            []string `json:"messages"`
	Subscriptions       []string `json:"subscriptions"`
	GetConnectedDevices bool     `json:"getConnectedDevices"`
}

// used in ws_svr.go to send a websocket message containing all
type ApiRes_WS struct {
	ConnectedDevicesList []string `json:"connectedDevicesList"`
}

// used in ws_svr.go - use to convey subscription requests to the handler from the server
type SubReqWrapper struct {
	ClientId   *string
	NewDevlist []string
	OldDevlist []string
}

// represent a message as it was sent from a client, before adulteration.
type MessageWrapper struct {
	Message   string    // text the tcp client sent
	ClientId  *string   // index which the message sender with in the connIndex of the server
	RecvdTime time.Time // recvd time
}

// Device schema for modelling in mongodb
type Device_Schema struct {
	DeviceId   string                 `bson:"device_id"`
	MsgHistory []DeviceMessage_Schema `bson:"msg_history"`
}

// Device message schema for modelling in mongodb
type DeviceMessage_Schema struct {
	RecvdTime  time.Time `bson:"receivedTime"`
	PacketTime time.Time `bson:"packetTime"`
	Message    string    `bson:"message"`
	Direction  string    `bson:"direction"`
}

// use to represent a message we're sending to an API client
type DeviceMessage_Response struct {
	RecvdTime  time.Time `json:"receivedTime"`
	PacketTime time.Time `json:"packetTime"`
	Message    string    `json:"message"`
	Direction  string    `json:"direction"`
}

// struct we marshal a http request body, formatted in json, into.
type ApiRequest_HTTP struct {
	Devices []string  `bson:"devices"`
	Before  time.Time `bson:"before"`
	After   time.Time `bson:"after"`
}
