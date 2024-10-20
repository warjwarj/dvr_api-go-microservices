package main

import (
	"dvr_api-go-microservices/pkg/config"
	"fmt"

	zap "go.uber.org/zap"
)

// Simple server. Basically a proxy which we use to forward requests through to the actual database proxy

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

	// create our database proxy server
	htpSvr, err := NewHttpSvr(logger, config.HTTP_SVR_ENDPOINT)
	if err != nil {
		logger.Fatal("fatal error creating database proxy server")
	}

	// run the server
	htpSvr.Run()

	// stop program from exiting
	var forever chan struct{}
	<-forever
}
