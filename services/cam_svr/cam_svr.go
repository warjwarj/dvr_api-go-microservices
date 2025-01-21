package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	utils "dvr_api-go-microservices/pkg/utils" // our shared utils package

	amqp "github.com/rabbitmq/amqp091-go" // rabbitmq import
	zap "go.uber.org/zap"
)

type CamSvr struct {
	logger              *zap.Logger
	endpoint            string               // IP + port, ex: "192.168.1.77:9047"
	sockOpBufSize       int                  // how much memory do we give each connection to perform send/recv operations
	sockOpBufNum        int                  // how many buffers should we provision for the server (a lot)
	sockOpBufStack      utils.Stack[*[]byte] // memory region we give each conn to so send/recv
	rabbitmqAmqpChannel *amqp.Channel        // tcp connection with rabbitmq server
}

func NewCamSvr(
	logger *zap.Logger,
	endpoint string,
	bufSize int,
	bufNum int,
	rabbitmqAmqpChannel *amqp.Channel,
) (*CamSvr, error) {
	svr := CamSvr{
		logger,
		endpoint,
		bufSize,
		bufNum,
		utils.Stack[*[]byte]{},
		rabbitmqAmqpChannel}

	// init the stack we use to store buffers
	svr.sockOpBufStack.Init()

	// create and store the buffers
	for i := 0; i < svr.sockOpBufNum; i++ {
		buf := make([]byte, svr.sockOpBufSize)
		svr.sockOpBufStack.Push(&buf)
	}
	return &svr, nil
}

// run the server, blocking
func (s *CamSvr) Run() {
	// start server listening for connections
	ln, err := net.Listen("tcp", s.endpoint)
	if err != nil {
		s.logger.Fatal("error listening on %v: %v", zap.String("s.endpoint", s.endpoint), zap.Error(err))
	} else {
		s.logger.Info("cam server listening on: %v", zap.String("s.endpoint", s.endpoint))
	}

	// // start the output and input from the message broker
	// go s.pipeMessagesToBroker(config.MSGS_FROM_DEVICE_SVR)
	// go s.pipeMessagesFromBroker(config.MSGS_FROM_API_SVR)

	// // start piping device conneciton state changes to server
	// go s.pipeConnectedDevicesToBroker(config.DEVICE_CONNECTION_STATE_CHANGES)

	// start accepting device connections
	for {
		c, err := ln.Accept()
		if err != nil {
			s.logger.Info("error accepting websocket connection: %v", zap.Error(err))
			continue
		}
		s.logger.Info("connection accepted on cam svr...")
		go func() {
			err := s.deviceConnLoop(c)
			if err != nil {
				s.logger.Error("error in cam server connection loop: %v", zap.Error(err))
			}
			c.Close()
		}()
	}
}

// TODO implement io.Reader
// receive video
func (s *CamSvr) deviceConnLoop(conn net.Conn) error {
	// close connection upon exit
	defer conn.Close()

	// video file with random temp name
	randFileName := utils.GenerateRandomString(10)
	f, err := os.OpenFile(randFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("error creating scratch file to save video: %v", err)
	}
	defer f.Close()

	// Buffers
	frameHeaderBuf := make([]byte, 32)
	framePayloadHeaderBuf := make([]byte, 128)
	framePayloadBodyBuf := make([]byte, 4194304)

	// alter this object as needed
	//var videoDescription utils.VideoDescription

	for {
		// Read the frame header
		if _, err := io.ReadFull(conn, frameHeaderBuf); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("Failed to read raw header: %v", err)
		}

		// Check for $FILEEND signal
		if strings.HasPrefix(string(frameHeaderBuf), "$FILEEND") {
			break
		}

		// get payload header length
		payloadHeaderLenBuf := frameHeaderBuf[14:20]
		payloadHeaderLen, err := strconv.ParseInt(strings.TrimPrefix(string(payloadHeaderLenBuf), "0x"), 16, 64)
		if err != nil {
			return fmt.Errorf("failed to parse payload header length: %v", err)
		}

		// get payload body length
		payloadBodyLenBuf := frameHeaderBuf[21:31]
		payloadBodyLen, err := strconv.ParseInt(strings.TrimPrefix(string(payloadBodyLenBuf), "0x"), 16, 64)
		if err != nil {
			return fmt.Errorf("failed to parse payload body length: %v", err)
		}

		// read payload header
		framePayloadHeaderBuf = make([]byte, payloadHeaderLen)
		if _, err = io.ReadFull(conn, framePayloadHeaderBuf); err != nil {
			return fmt.Errorf("failed to read payload header: %v", err)
		}

		// read payload body
		framePayloadBodyBuf = make([]byte, payloadBodyLen)
		if _, err = io.ReadFull(conn, framePayloadBodyBuf); err != nil {
			return fmt.Errorf("failed to read payload body: %v", err)
		}

		// write vid data to file
		if _, err := f.Write(framePayloadBodyBuf); err != nil {
			return fmt.Errorf("failed to write to file: %v", err)
		}
	}
	fmt.Println("file writen successfully")
	return nil
}

// // publish device messages to broker
// func (s *CamSvr) pipeMessagesToBroker(exchangeName string) {

// 	// declare the device message output exchange
// 	err := s.rabbitmqAmqpChannel.ExchangeDeclare(
// 		exchangeName, // name
// 		"fanout",     // type
// 		true,         // durable
// 		false,        // auto-deleted
// 		false,        // internal
// 		false,        // no-wait
// 		nil,          // arguments
// 	)
// 	if err != nil {
// 		s.logger.Fatal("error piping messages from message broker %v", zap.Error(err))
// 	}

// 	// loop over channel and pipe messages into the rabbitmq exchange.
// 	for bytes := range s.svrMsgBufChan {
// 		// publish our JSON message to the exchange which handles messages from devices
// 		err := s.rabbitmqAmqpChannel.Publish(
// 			exchangeName, // exchange
// 			"",           // routing key
// 			false,        // mandatory
// 			false,        // immediate
// 			amqp.Publishing{
// 				ContentType: "text/plain",
// 				Body:        []byte(bytes),
// 			})
// 		if err != nil {
// 			s.logger.Error("error piping messages into message broker %v", zap.Error(err))
// 		}
// 	}
// }

// // publish device messages to broker
// func (s *CamSvr) pipeConnectedDevicesToBroker(exchangeName string) {

// 	// declare the device message output exchange
// 	err := s.rabbitmqAmqpChannel.ExchangeDeclare(
// 		exchangeName, // name
// 		"fanout",     // type
// 		true,         // durable
// 		false,        // auto-deleted
// 		false,        // internal
// 		false,        // no-wait
// 		nil,          // arguments
// 	)
// 	if err != nil {
// 		s.logger.Fatal("error piping messages from message broker %v", zap.Error(err))
// 	}

// 	// this loop fires each time a device connects or disconnects
// 	for i := range s.svrRegisterChangeChan {

// 		// get bytes
// 		bytes, err := json.Marshal(i)
// 		if err != nil {
// 			s.logger.Error("error marshalling into json %v", zap.Error(err))
// 		}

// 		// publish our JSON message to the exchange which handles messages from devices
// 		err = s.rabbitmqAmqpChannel.Publish(
// 			exchangeName, // exchange
// 			"",           // routing key
// 			false,        // mandatory
// 			false,        // immediate
// 			amqp.Publishing{
// 				ContentType: "text/plain",
// 				Body:        []byte(bytes),
// 			})
// 		if err != nil {
// 			s.logger.Error("error piping device connection event into message broker %v", zap.Error(err))
// 		}
// 	}
// }

// // Take messages from the broker and send them to devices.
// func (s *CamSvr) pipeMessagesFromBroker(exchangeName string) {

// 	// declare exchange we are taking messages from.
// 	err := s.rabbitmqAmqpChannel.ExchangeDeclare(
// 		exchangeName, // name
// 		"fanout",     // type
// 		true,         // durable
// 		false,        // auto-deleted
// 		false,        // internal
// 		false,        // no-wait
// 		nil,          // arguments
// 	)
// 	if err != nil {
// 		s.logger.Fatal("error piping messages from message broker %v", zap.Error(err))
// 	}

// 	// declare a queue onto the exchange above
// 	q, err := s.rabbitmqAmqpChannel.QueueDeclare(
// 		"",    // name -
// 		false, // durable
// 		false, // delete when unused
// 		true,  // exclusive
// 		false, // no-wait
// 		nil,   // arguments
// 	)
// 	if err != nil {
// 		s.logger.Fatal("error declaring queue %v", zap.Error(err))
// 	}

// 	// bind the queue onto the exchange
// 	err = s.rabbitmqAmqpChannel.QueueBind(
// 		q.Name,       // queue name
// 		"",           // routing key
// 		exchangeName, // exchange
// 		false,
// 		nil,
// 	)
// 	if err != nil {
// 		s.logger.Error("error binding queue %v", zap.Error(err))
// 	}

// 	// get a golang channel we can read messages from.
// 	msgs, err := s.rabbitmqAmqpChannel.Consume(
// 		q.Name, // name - WE DEPEND ON THE QUEUE BEING NAMED THE SAME AS THE EXCHANGE - ONLY ONE Q CONSUMING FROM THIS EXCHANGE
// 		"",     // consumer
// 		true,   // auto-ack
// 		false,  // exclusive
// 		false,  // no-local
// 		false,  // no-wait
// 		nil,    // args
// 	)
// 	if err != nil {
// 		s.logger.Error("error consuming from queue %v", zap.Error(err))
// 	}

// 	// connection loop
// 	for d := range msgs {

// 		// this is the message revceived from broker
// 		var msgWrap utils.MessageWrapper
// 		err := json.Unmarshal(d.Body, &msgWrap)
// 		if err != nil {
// 			s.logger.Error("received erroneous value from message broker, couldn't unmarshal into message wrapper")
// 		}

// 		// extract the device the message pertains to
// 		dev_id, err := utils.GetIdFromMessage(&msgWrap.Message)
// 		if err != nil {
// 			s.logger.Error("error processing message from device", zap.Error(err))
// 			continue
// 		}

// 		// verify device connection, get the connection object
// 		devConn, devConnOk := s.connIndex.Get(*dev_id)
// 		if !devConnOk {
// 			s.logger.Warn("message sent for device not connected")
// 			continue
// 		}

// 		// send the requested message to the device
// 		_, err = (*devConn).Write([]byte(msgWrap.Message))
// 		if err != nil {
// 			s.logger.Error("received message destined for device not connected")
// 		}
// 	}
// }
