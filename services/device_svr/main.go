package main

import (
	"fmt"

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
	devSvr, err := NewDeviceSvr(logger, config.DEVICE_SVR_ENDPOINT, config.CAPACITY, config.BUF_SIZE, config.SVR_MSGBUF_SIZE, rabbitmqAmqpChannel)
	if err != nil {
		logger.Fatal("fatal error creating device server: %v", zap.Error(err))
		return
	}

	// run the server
	go devSvr.Run()

	// stop program from exiting
	var forever chan struct{}
	<-forever
}
