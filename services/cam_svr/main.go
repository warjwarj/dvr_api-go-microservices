package main

import (
	"fmt"

	config "dvr_api-go-microservices/pkg/config" // our shared config

	amqp "github.com/rabbitmq/amqp091-go" // message broker
	zap "go.uber.org/zap"                 // logger
)

// I think the

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

	// connect to AWS
	awsConn, err := NewAwsConnection(logger)
	if err != nil {
		logger.Fatal("fatal error connecting to AWS", zap.Error(err))
		return
	}

	// TEST CODE - for checking connection to AWS.
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// // upload the video to the cloud and get a link to it
	// if err := awsConn.uploadFile(context.TODO(), config.VIDEO_STORAGE_BUCKET, "testfile", "/app/services/cam_svr/verysmallfile.txt"); err != nil {
	// 	logger.Error("Couldn't upload file to AWS", zap.Error(err))
	// }

	// // get a presigned url
	// presignedUrl, err := awsConn.getPresignedUrl(context.TODO(), config.VIDEO_STORAGE_BUCKET, "testfile", 600)
	// if err != nil {
	// 	logger.Error("Couldn't get presigned URL from AWS", zap.Error(err))
	// }

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// create our device server struct
	devSvr, err := NewCamSvr(logger, config.CAM_SVR_ENDPOINT, 2048, 10, rabbitmqAmqpChannel, *awsConn)
	if err != nil {
		logger.Fatal("fatal error creating cam server: %v", zap.Error(err))
		return
	}

	// run the server
	go devSvr.Run()

	// stop program from exiting
	var forever chan struct{}
	<-forever
}
