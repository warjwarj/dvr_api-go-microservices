package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	uri                 string
	dbName              string
}

func NewDbProxySvr(
	logger *zap.Logger, // logger
	rabbitmqAmqpChannel *amqp.Channel, // rabbitmq channel
	uri string, // network location of the database
	dbName string, // name of the database inside mongo
) (*DbProxySvr, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	var result bson.M
	// ping db to check the connection
	err = client.Database(dbName).RunCommand(context.TODO(), bson.D{{"ping", 1}}).Decode(&result)
	if err != nil {
		return nil, err
	}
	logger.Info("database %v connected successfully", zap.String("dbName", dbName))
	return &DbProxySvr{
		logger:              logger,
		client:              client,
		rabbitmqAmqpChannel: rabbitmqAmqpChannel,
		uri:                 uri,
		dbName:              dbName}, nil
}

func (dps *DbProxySvr) Run() {

	// start piping messages into the database
	go dps.pipeMessagesFromExchangeIntoDatabase(config.MSGS_FROM_DEVICE_SVR)
	go dps.pipeMessagesFromExchangeIntoDatabase(config.MSGS_FROM_API_SVR)

	// listen tcp
	l, err := net.Listen("tcp", dps.uri)
	if err != nil {
		dps.logger.Fatal("error listening on %v: %v", zap.String("s.endpoint", dps.uri), zap.Error(err))
	} else {
		dps.logger.Info("http dbproxy_svr server listening on: %v", zap.String("s.endpoint", dps.uri))
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
	// check edge cases
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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
}

func (s *DbProxySvr) QueryMsgHistory(devices []string, before time.Time, after time.Time) ([]bson.M, error) {

	// might be worth storing this to avoid redeclaration upon each function call
	coll := s.client.Database(s.dbName).Collection("devices")

	// query filter. Device id in devices, and packet_time between the two dates passed
	filter := bson.D{
		{"DeviceId", bson.D{
			{"$in", devices},
		}},
		{"MsgHistory", bson.D{
			{"$elemMatch", bson.D{
				{"receivedTime", bson.D{
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

// insert a record into the database
func (dps *DbProxySvr) RecordMessage_ToFromDevice(fromDevice bool, msg *utils.MessageWrapper) (*mongo.UpdateResult, error) {

	// record direction
	var directionDescriptor string
	switch fromDevice {
	// msg sent by the device
	case true:
		directionDescriptor = "from device"
	// msg sent by an API client to a device
	case false:
		directionDescriptor = "to device"
	}

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	// might be worth storing this to avoid redeclaration upon each function call
	coll := dps.client.Database(dps.dbName).Collection("devices")

	// parse the time in the packet
	packetTime, err := utils.GetDateFromMessage(msg.Message)

	// Create a new message
	newMessage := utils.DeviceMessage_Schema{
		RecvdTime:  msg.RecvdTime,
		PacketTime: packetTime,
		Message:    msg.Message,
		Direction:  directionDescriptor,
	}

	devId, err := utils.GetIdFromMessage(&msg.Message)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse device id from message: %v", msg.Message)
	}

	// Update the document for the device, adding the new message to the messages array
	filter := bson.M{"DeviceId": devId}
	update := bson.M{
		"$push": bson.M{
			"MsgHistory": newMessage,
		},
	}

	// perform update/insertion
	updateResult, err := coll.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return updateResult, err
	}

	// return results of update/insertion
	return updateResult, nil
}

func (dps *DbProxySvr) pipeMessagesFromExchangeIntoDatabase(exchangeName string) {
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

	for msg := range msgs {
		fmt.Println(msg)
	}
}
