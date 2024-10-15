package main

import (
	"fmt"

	"dvr_api-go-microservices/pkg/config"

	zap "go.uber.org/zap"

	// our shared utils package
	amqp "github.com/rabbitmq/amqp091-go" // rabbitmq import
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
	rabbitmqAmqpConnection, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
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

	// create our device server struct
	apiSvr, err := NewWsApiSvr(logger, config.API_SVR_ENDPOINT, rabbitmqAmqpChannel)
	if err != nil {
		logger.Fatal("fatal error creating device server: %v", zap.Error(err))
		return
	}

	// create the subscription handler
	subHandler, err := NewPubSubHandler(logger, apiSvr, rabbitmqAmqpChannel)
	if err != nil {
		logger.Fatal("fatal error creating relay struct: %v", zap.Error(err))
	}

	// handle subscriptions and run the server
	go subHandler.Run()
	go apiSvr.Run()

	// stop program from exiting
	var forever chan struct{}
	<-forever
}
