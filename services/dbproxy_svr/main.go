package main

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	config "dvr_api-go-microservices/pkg/config" // our shared config

	amqp "github.com/rabbitmq/amqp091-go" // message broker
	zap "go.uber.org/zap"                 // logger
)

func main() {
	// set up our zap logger
	var logger *zap.Logger
	if config.PROD {
		tmp, err := zap.NewProduction()
		if err != nil {
			fmt.Println("error initing logger: ", err)
			return
		}
		logger = tmp
	} else {
		tmp, err := zap.NewDevelopment()
		if err != nil {
			fmt.Println("error initing logger: ", err)
			return
		}
		logger = tmp
	}
	defer logger.Sync() // flushes buffer, if any

	// dial the rabbitmq server
	rabbitmqAmqpConnection, err := amqp.Dial(config.RABBITMQ_AMQP_ENDPOINT)
	if err != nil {
		logger.Fatal("fatal error dialing rabbitmq server: %v", zap.Error(err))
		return
	}
	defer rabbitmqAmqpConnection.Close()

	// open the channel (the tcp connection)
	rabbitmqAmqpChannel, err := rabbitmqAmqpConnection.Channel()
	if err != nil {
		logger.Fatal("fatal error creating rabbitmq channel: %v", zap.Error(err))
		return
	}
	defer rabbitmqAmqpChannel.Close()

	// connect to the mongodb instance
	mongoClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(config.MONGODB_ENDPOINT))
	if err != nil {
		logger.Fatal("fatal error connecting to mongo database: %v", zap.Error(err))
	}

	// ping the specific db in the mongo instance to check the connection
	var result bson.M
	err = mongoClient.Database(config.MONGODB_MESSAGE_DB).RunCommand(context.TODO(), bson.D{{"ping", 1}}).Decode(&result)
	if err != nil {
		logger.Fatal("fatal error pinging database: %v", zap.Error(err))
	}
	logger.Info("database %v connected successfully", zap.String("config.MONGODB_MESSAGE_DB", config.MONGODB_MESSAGE_DB))

	// create our database proxy server
	dbps, err := NewDbProxySvr(logger, rabbitmqAmqpChannel, mongoClient, config.MONGODB_MESSAGE_DB, config.DBPROXY_SVR_ENDPOINT)
	if err != nil {
		logger.Fatal("fatal error creating database proxy server")
	}

	// run the server
	dbps.Run()

	// hang indefinitely
	var forever chan struct{}
	<-forever
}
