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
// client id is the api client's id.
type SubReqWrapper struct {
	ClientId       *string
	NewDevlist     []string
	OldDevlist     []string
	ReqMatchString string
}

// represent a message as it was sent from a client, before adulteration.
type MessageWrapper struct {
	Message          string            // text the tcp client sent
	ClientId         *string           // index which the message sender with in the connIndex of the server
	RecvdTime        time.Time         // recvd time
	VideoDescription *VideoDescription // video description from cam server
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
	DeviceId               string `json:DeviceId`
	Channel                string `json:Channel`
	RequestLength          string `json:RequestLength`
	RequestStartTime       string `json:RequestStartTime`
	VideoLength            int    `json:VideoLength`
	VideoLink              string `json:VideoLink`
	VideoDescriptionString string `json:VideoDescriptionString`
}

// $FILE;V03;1221223799;1;1;20241114-170853;240;20241114-170810;60;4742224
// [0 $FILE];[1 protocol];[2 DeviceID];[3 SN];[4 camera];[5 RStart];[6 Rlen];[7 VStart];[8 Vlen];[9 filelen]<CR>[bytes]
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

// $VIDEO;85000;all;0;20181127-091000;86400<CR>
// [0 $VIDEO];[1 DeviceID];[2 type];[3 camera];[4 start];[5 time length]<CR>
// represents a $VIDEO packet header's values in a struct for easy access.
type VideoPacketHeader struct {
	DeviceId         string `json:DeviceId`
	RequestStartTime string `json:RequestStartTime`
	RequestLength    string `json:RequestLength`
}
