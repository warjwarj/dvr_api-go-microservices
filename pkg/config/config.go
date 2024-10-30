/*

TODO:

*/

package config

// Globals
const (

	// just use this for the logger atm
	PROD bool = false

	// Endpoints visible to the outside.
	DEVICE_SVR_ENDPOINT  string = "device_svr:9047"   // endpoint for dev svr
	API_SVR_ENDPOINT     string = "ws_api_svr:9046"   // endpoint for api websock svr
	HTTP_SVR_ENDPOINT    string = "http_api_svr:9045" // endpoint for api REST svr
	DBPROXY_SVR_ENDPOINT string = "dbproxy_svr:9044"  // endpoint for database proxy REST svr

	// these are used in the device server
	CAPACITY        int = 200  // how many devices can connect to the server
	BUF_SIZE        int = 1024 // how much memory will you allocate to IO operations
	SVR_MSGBUF_SIZE int = 40   // capacity of message queue

	// RABBITMQ - endpoint and variables
	RABBITMQ_AMQP_ENDPOINT          string = "amqp://guest:guest@rabbitmq:5672" // rabbitmq amqp uri
	MSGS_FROM_DEVICE_SVR            string = "msgs_from_device_svr"             // sub to this to receive every message from every device
	MSGS_FROM_API_SVR               string = "msgs_from_api_svr"                // sub to this to receive every message from every device
	DEVICE_CONNECTION_STATE_CHANGES string = "device_connection_state_changes"  // each time a device connects ot disconnects this exchange is supplied with a fresh list

	// MONGODB - endpoints and general variables
	MONGODB_ENDPOINT                      string = "mongodb://mongodb:27017"    // database uri
	MONGODB_MESSAGE_DB                    string = "message_db"                 // message store db
	MONGODB_MESSAGE_DEFAULT_COLLECTION    string = "default_message_collection" // collection to store the messages in, in the database
	MONGODB_GLOBAL_CONNECTED_DEVICES_LIST string = "global_connected_devices_list"
)
