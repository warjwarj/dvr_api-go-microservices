package utils

import (
	"time"
)

// used in ws_svr.go to read json messages into structs
type ApiReq_WS struct {
	Messages      []string `json:"Messages"`
	Subscriptions []string `json:"Subscriptions"`
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
	DeviceId   string           `bson:"DeviceId"`
	MsgHistory []Message_Schema `bson:"MsgHistory"`
}

// Device message schema for modelling in mongodb
type Message_Schema struct {
	RecvdTime  time.Time `bson:"ReceivedTime"`
	PacketTime time.Time `bson:"PacketTime"`
	Message    string    `bson:"Message"`
	Direction  string    `bson:"Direction"`
}

// use to represent a message we're sending to an API client
type DeviceMessage_Response struct {
	RecvdTime  time.Time `json:"RecvdTime"`
	PacketTime time.Time `json:"PacketTime"`
	Message    string    `json:"Message"`
	Direction  string    `json:"Direction"`
}

// struct we marshal a http request body, formatted in json, into.
type ApiRequest_HTTP struct {
	Devices []string  `bson:"Devices"`
	Before  time.Time `bson:"Before"`
	After   time.Time `bson:"After"`
}

// used to convey device connection/disconnection events
type DeviceConnectionStateChange struct {
	DeviceId     string `json:"DeviceId"`     // device state change is effect of
	IsConnection bool   `json:"IsConnection"` // if true device is connecting, false device is disconnecting
}

// used to describe video post-processing by ffmpeg, as we will upload it to the cloud.
type VideoDescription struct {
	DeviceId         string `json:DeviceId`
	Channel          string `json:Channel`
	RequestStartTime string `json:RequestStartTime`
	VideoLength      int    `json:VideoLength`
	VideoLink        string `json:VideoLink`
}

// represents a $FILE packet header's values in a struct for easy access.
type FilePacketHeader struct {
	DeviceId         string `json:DeviceId`
	Channel          string `json:Channel`
	SerialNum        string `json:SerialNum`
	RequestStartTime string `json:RequestStartTime`
	RequestLength    string `json:RequestLength`
	VideoStartTime   string `json:VideoStartTime`
	VideoLength      string `json:VideoLength`
	FileLen          string `json:FileLen`
}
