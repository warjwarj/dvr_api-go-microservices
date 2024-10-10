package main

import (
	"fmt"

	"dvr_api-go-microservices/pkg/config" // our shared config

	"go.uber.org/zap"
)

func main() {

	// set up our zap logger
	var logger *zap.Logger
	if config.PROD {
		tmp, err := zap.NewProduction()
		if err != nil {
			fmt.Println("error initing logger: ", err)
		}
		logger = tmp
	} else {
		tmp, err := zap.NewDevelopment()
		if err != nil {
			fmt.Println("error initing logger: ", err)
		}
		logger = tmp
	}
	defer logger.Sync() // flushes buffer, if any

	// create our device server struct
	devSvr, err := NewDeviceSvr(logger, config.DEVICE_SVR_ENDPOINT, config.CAPACITY, config.BUF_SIZE, config.SVR_MSGBUF_SIZE)
	if err != nil {
		logger.Fatal("fatal error creating device server: %v", zap.Error(err))
	}

	go devSvr.Run()
}
