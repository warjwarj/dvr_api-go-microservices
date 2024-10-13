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

	// rabbitmq variables for Qs and Xchanges. Since we only have one queue on the exchange going in from the message broker we can have it named the same as the exchange.
	MSGS_TO_DEVICE_SVR   string = "msgs_to_device_svr"
	MSGS_FROM_DEVICE_SVR string = "msgs_from_device_svr"

	MSGS_TO_API_SVR   string = "msgs_to_api_svr"
	MSGS_FROM_API_SVR string = "msgs_from_api_svr"
)
