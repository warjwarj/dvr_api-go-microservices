package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	config "dvr_api-go-microservices/pkg/config"
	utils "dvr_api-go-microservices/pkg/utils"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	zap "go.uber.org/zap"
)

type DbProxySvr struct {
	logger              *zap.Logger
	client              *mongo.Client
	rabbitmqAmqpChannel *amqp.Channel
	dbName              string
	restEndpoint        string
}

func NewDbProxySvr(
	logger *zap.Logger,
	rabbitmqAmqpChannel *amqp.Channel,
	client *mongo.Client,
	dbName string,
	restEndpoint string,
) (*DbProxySvr, error) {
	return &DbProxySvr{
		logger:              logger,
		client:              client,
		rabbitmqAmqpChannel: rabbitmqAmqpChannel,
		dbName:              dbName,
		restEndpoint:        restEndpoint}, nil
}

func (dps *DbProxySvr) Run() {

	// start piping messages into the database
	go dps.pipeMessagesFromQueueIntoDatabase("from_device", config.MSGS_FROM_DEVICE_SVR)
	go dps.pipeMessagesFromQueueIntoDatabase("from_api_client", config.MSGS_FROM_API_SVR)

	// start receiving device connection and disconnection events
	go dps.pipeConnectedDevicesFromBroker(config.DEVICE_CONNECTION_STATE_CHANGES)

	// listen tcp
	l, err := net.Listen("tcp", dps.restEndpoint)
	if err != nil {
		dps.logger.Fatal("error listening on %v: %v", zap.String("dps.restEndpoint", dps.restEndpoint), zap.Error(err))
	} else {
		dps.logger.Info("http dbproxy_svr server listening on:", zap.String("dps.restEndpoint", dps.restEndpoint))
	}

	// accept http on tcp port we've just ran
	HttpApiSvr := &http.Server{
		Handler: dps,
	}
	err = HttpApiSvr.Serve(l)
	if err != nil {
		dps.logger.Fatal("error serving http on dbproxy_svr: %v", zap.Error(err))
	}
}

// serve the http API
func (dps *DbProxySvr) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// // check edge cases
	// if r.Method != http.MethodPost {
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }

	// check cors if in prod
	if !config.PROD {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		dps.logger.Fatal("cors enabled on http server, disable in prod")
	}

	// read the bytes
	body, err := io.ReadAll(r.Body)
	if err != nil {
		dps.logger.Error("Unable to read request body", zap.Error(err))
		return
	}

	// parse the request type reqType
	query := r.URL.Query()
	reqType := query.Get("reqType")

	if reqType == "MessageHistory" {
		// unmarshal bytes into a struct we can work with
		var req utils.ApiRequest_HTTP
		err = json.Unmarshal(body, &req)
		if err != nil {
			dps.logger.Warn("failed to unmarshal json: \n%v", zap.String("body", string(body)))
		}

		// query the database
		res, err := dps.QueryMsgHistory(req.Devices, req.Before, req.After)
		if err != nil {
			dps.logger.Error("failed to query msg history: %v", zap.Error(err))
		}

		// marshal back into json
		bytes, err := json.Marshal(res)
		if err != nil {
			dps.logger.Error("failed to marshal golang struct into json bytes: %v", zap.Error(err))
			return
		}

		// set the response header Content-Type to application/json
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes)

	} else if reqType == "GetConnectedDevices" {
		// query the database
		res, err := dps.QueryConnectedDevices()
		if err != nil {
			dps.logger.Error("failed to query msg history: %v", zap.Error(err))
		}

		// marshal back into json
		bytes, err := json.Marshal(res)
		if err != nil {
			dps.logger.Error("failed to marshal golang struct into json bytes: %v", zap.Error(err))
			return
		}

		// set the response header Content-Type to application/json
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes)
	}
}

func (dps *DbProxySvr) QueryConnectedDevices() (bson.M, error) {
	// might be worth storing this to avoid redeclaration upon each function call
	coll := dps.client.Database(dps.dbName).Collection(config.MONGODB_GLOBAL_CONNECTED_DEVICES_LIST)

	// Find all documents (empty filter)
	cursor, err := coll.Find(context.TODO(), bson.D{}, options.Find().SetProjection(bson.M{"_id": 0}))
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.TODO())

	// iterate over the cursor returned and return docuements that match the query
	var documents []bson.M
	for cursor.Next(context.TODO()) {
		var result bson.M
		err := cursor.Decode(&result)
		if err != nil {
			panic(err)
		}
		documents = append(documents, result)
	}
	return documents[0], nil
}

func (dps *DbProxySvr) QueryMsgHistory(devices []string, before time.Time, after time.Time) ([]bson.M, error) {

	// might be worth storing this to avoid redeclaration upon each function call
	coll := dps.client.Database(dps.dbName).Collection(config.MONGODB_MESSAGE_DEFAULT_COLLECTION)

	// query filter. Device id in devices, and packet_time between the two dates passed
	filter := bson.D{
		{"DeviceId", bson.D{
			{"$in", devices},
		}},
		{"MsgHistory", bson.D{
			{"$elemMatch", bson.D{
				{"ReceivedTime", bson.D{
					{"$gte", after},
					{"$lt", before},
				}},
			}},
		}},
	}

	// query using above. Exclude _id field
	cursor, err := coll.Find(context.Background(), filter, options.Find().SetProjection(bson.M{"_id": 0}))
	if err != nil {
		return nil, fmt.Errorf("error querying database: %v", err)
	}

	// iterate over the cursor returned and return docuements that match the query
	var documents []bson.M
	for cursor.Next(context.Background()) {
		var result bson.M
		err := cursor.Decode(&result)
		if err != nil {
			panic(err)
		}
		documents = append(documents, result)
	}
	return documents, nil
}

func (dps *DbProxySvr) pipeMessagesFromQueueIntoDatabase(directionDescriptor string, exchangeName string) {

	// declare exchange we are taking messages from.
	err := dps.rabbitmqAmqpChannel.ExchangeDeclare(
		exchangeName, // name
		"fanout",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		dps.logger.Fatal("error piping messages from message broker %v", zap.Error(err))
	}

	// declare a queue onto the exchange above
	q, err := dps.rabbitmqAmqpChannel.QueueDeclare(
		"",    // name -
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		dps.logger.Fatal("error declaring queue %v", zap.Error(err))
	}

	// bind the queue onto the exchange
	err = dps.rabbitmqAmqpChannel.QueueBind(
		q.Name,       // queue name
		"",           // routing key
		exchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		dps.logger.Error("error binding queue %v", zap.Error(err))
	}

	// get a golang channel we can read messages from.
	msgs, err := dps.rabbitmqAmqpChannel.Consume(
		q.Name, // name - WE DEPEND ON THE QUEUE BEING NAMED THE SAME AS THE EXCHANGE - ONLY ONE Q CONSUMING FROM THIS EXCHANGE
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		dps.logger.Error("error consuming from queue %v", zap.Error(err))
	}

	// this is the collection we're going to be putting the messages into
	coll := dps.client.Database(dps.dbName).Collection(config.MONGODB_MESSAGE_DEFAULT_COLLECTION)

	// reuse the template
	var msgSchema utils.Message_Schema

	// loop over the device message channel and put it into the database
	for d := range msgs {

		// this is the message revceived from broker
		var msgWrap utils.MessageWrapper
		err := json.Unmarshal(d.Body, &msgWrap)
		if err != nil {
			dps.logger.Error("received erroneous value from message broker, couldn't unmarshal into message wrapper")
		}

		// parse id from message
		devId, err := utils.GetIdFromMessage(&msgWrap.Message)
		if err != nil {
			dps.logger.Error("error consuming from queue", zap.Error(err))
			continue
		}

		// parse the time in the packet
		packetTime, err := utils.GetDateFromMessage(msgWrap.Message)
		if err != nil {
			dps.logger.Error("error consuming from queue", zap.Error(err))
		}

		// Create a new message
		msgSchema = utils.Message_Schema{
			RecvdTime:  msgWrap.RecvdTime,
			PacketTime: packetTime,
			Message:    msgWrap.Message,
			Direction:  directionDescriptor,
		}

		// find the field in the document into which we're going to be inserting the message
		filter := bson.M{"DeviceId": devId}
		update := bson.M{
			"$push": bson.M{
				"MsgHistory": msgSchema,
			},
		}

		// perform update/insertion
		_, err = coll.UpdateOne(context.TODO(), filter, update, options.Update().SetUpsert(true))
		if err != nil {
			dps.logger.Error("error inserting message into database", zap.Error(err))
		}
	}
}

// Take messages from the broker and send them to clients. Messages from devices
func (dps *DbProxySvr) pipeConnectedDevicesFromBroker(exchangeName string) {

	// declare exchange we are taking messages from.
	err := dps.rabbitmqAmqpChannel.ExchangeDeclare(
		exchangeName, // name
		"fanout",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		dps.logger.Fatal("error piping messages from message broker %v", zap.Error(err))
	}

	// declare a queue onto the exchange above
	q, err := dps.rabbitmqAmqpChannel.QueueDeclare(
		"",    // name -
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		dps.logger.Fatal("error declaring queue %v", zap.Error(err))
	}

	// bind the queue onto the exchange
	err = dps.rabbitmqAmqpChannel.QueueBind(
		q.Name,       // queue name
		"",           // routing key
		exchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		dps.logger.Error("error binding queue %v", zap.Error(err))
	}

	// get a golang channel we can read messages from.
	msgs, err := dps.rabbitmqAmqpChannel.Consume(
		q.Name, // name - WE DEPEND ON THE QUEUE BEING NAMED THE SAME AS THE EXCHANGE - ONLY ONE Q CONSUMING FROM THIS EXCHANGE
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		dps.logger.Error("error consuming from queue %v", zap.Error(err))
	}

	// declare collection, insert/initialise empty document for dev list
	coll := dps.client.Database(dps.dbName).Collection(config.MONGODB_GLOBAL_CONNECTED_DEVICES_LIST)
	_, err = coll.UpdateOne(
		context.TODO(),
		bson.D{},
		bson.D{{Key: "$set", Value: bson.D{{Key: "ConnectedDevices", Value: []string{}}}}}, // set ConnectedDevices to an empty array
		options.Update().SetUpsert(true))
	if err != nil {
		dps.logger.Error("error inserting initial document to mongo %v", zap.Error(err))
	}

	// reuse - struct which informs of a device connection/disconneciton
	var devConnStateChange utils.DeviceConnectionStateChange

	// connection loop
	for d := range msgs {

		// this is the message revceived from broker
		err := json.Unmarshal(d.Body, &devConnStateChange)
		if err != nil {
			dps.logger.Error("received erroneous value from message broker, couldn't unmarshal")
			fmt.Println(devConnStateChange)
		}

		// either add or remove the device id from the connected devices list
		if devConnStateChange.IsConnection {
			// if the device is connecting
			_, err = coll.UpdateOne(
				context.TODO(),
				bson.D{}, // Match all documents (since we only have one)
				bson.D{{Key: "$push", Value: bson.D{{Key: "ConnectedDevices", Value: devConnStateChange.DeviceId}}}},
			)
			if err != nil {
				dps.logger.Error("error inserting device to connected device list document in mongo %v", zap.Error(err))
			}
		} else {
			// else the device is disconnecting
			_, err = coll.UpdateOne(
				context.TODO(),
				bson.D{}, // Match all documents (since we only have one)
				bson.D{{Key: "$pull", Value: bson.D{{Key: "ConnectedDevices", Value: devConnStateChange.DeviceId}}}},
			)
			if err != nil {
				dps.logger.Error("error removing device from connected device list document in mongo %v", zap.Error(err))
			}
		}

	}
}

// func insertString(newString string) {
//     collection := client.Database(dbName).Collection(collectionName)
//     _, err := collection.UpdateOne(
//         context.TODO(),
//         bson.D{}, // Match all documents (since we only have one)
//         bson.D{{Key: "$push", Value: bson.D{{Key: "strings", Value: newString}}}},
//     )
//     if err != nil {
//         log.Fatal(err)
//     }
//     fmt.Printf("Inserted: %s\n", newString)
// }

// func deleteString(stringToDelete string) {
//     collection := client.Database(dbName).Collection(collectionName)
//     _, err := collection.UpdateOne(
//         context.TODO(),
//         bson.D{}, // Match all documents
//         bson.D{{Key: "$pull", Value: bson.D{{Key: "strings", Value: stringToDelete}}}},
//     )
//     if err != nil {
//         log.Fatal(err)
//     }
//     fmt.Printf("Deleted: %s\n", stringToDelete)
// }
