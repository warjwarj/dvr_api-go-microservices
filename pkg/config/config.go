/*

TODO:


*/

package config

// Globals
const (
	// endopints
	DEVICE_SVR_ENDPOINT  string = "127.0.0.1:9047"           // endpoint for dev svr
	WEBSOCK_SVR_ENDPOINT string = "127.0.0.1:9046"           // endpoint for api websock svr
	HTTP_SVR_ENDPOINT    string = "127.0.0.1:9045"           // endpoint for api REST svr
	MONGODB_ENDPOINT     string = "mongodb://0.0.0.0:27017/" // database uri

	// just use this for the logger atm
	PROD bool = false

	// server configuration variables
	CAPACITY        int = 200  // how many devices can connect to the server
	BUF_SIZE        int = 1024 // how much memory will you allocate to IO operations
	SVR_MSGBUF_SIZE int = 40   // capacity of message queue

	// rabbitmq variables
	DEVICEMSG_OUTPUT_CHANNEL = "devicemsg_output_channel"
	DEVICEMSG_INPUT_CHANNEL  = "devicemsg_input_channel"
)
