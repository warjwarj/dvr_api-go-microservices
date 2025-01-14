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
		s.logger.Info("device server listening on: %v", zap.String("s.endpoint", s.endpoint))
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
		s.logger.Info("connection accepted on device svr...")
		go func() {
			err := s.deviceConnLoop(c)
			if err != nil {
				s.logger.Error("error in device connection loop: %v", zap.Error(err))
			}
			c.Close()
		}()
	}
}

// better way to do this might be to write to disk buffer by buffer
// receive connections and messages. Input from devices
func (s *CamSvr) deviceConnLoop(conn net.Conn) error {

	defer conn.Close()

	// Buffers
	rawHeader := make([]byte, 32)
	payloadHeaderLenBuffer := make([]byte, 6)
	payloadBodyLenBuffer := make([]byte, 10)

	// loop vars
	firstPacket := true

	for {
		// Read the raw header
		_, err := io.ReadFull(conn, rawHeader)
		if err != nil {
			return fmt.Errorf("Failed to read raw header: %v", err)
		}

		// Check for $FILEEND signal
		if string(rawHeader[:7]) == "$FILEEND" {
			break
		}

		// Parse lengths
		copy(payloadHeaderLenBuffer, rawHeader[14:20])
		copy(payloadBodyLenBuffer, rawHeader[21:31])

		fmt.Println(string(payloadHeaderLenBuffer))

		// parse header length
		payloadHeaderLen, err := strconv.ParseInt(strings.TrimPrefix(string(payloadHeaderLenBuffer), "0x"), 16, 64)
		if err != nil {
			return fmt.Errorf("failed to parse payload header length: %v", err)
		}

		// parse body length
		payloadBodyLen, err := strconv.ParseInt(strings.TrimPrefix(string(payloadBodyLenBuffer), "0x"), 16, 64)
		if err != nil {
			return fmt.Errorf("failed to parse payload body length: %v", err)
		}

		// Read payload header
		payloadHeader := make([]byte, payloadHeaderLen)
		_, err = io.ReadFull(conn, payloadHeader)
		if err != nil {
			return fmt.Errorf("Failed to read payload header: %v", err)
		}
		payloadHeaderString := string(payloadHeader)

		if firstPacket {
			if !strings.HasPrefix(payloadHeaderString, "$FILE") {
				return fmt.Errorf("Expected $FILE packets, not %s", payloadHeaderString)
			}
			firstPacket = false
		}

		// Read payload body
		payloadBody := make([]byte, payloadBodyLen-payloadHeaderLen)
		_, err = io.ReadFull(conn, payloadBody)
		if err != nil {
			return fmt.Errorf("Failed to read payload body: %v", err)
		}

		// Save payload body to file
		filePath := "./output.avi"
		err = os.WriteFile(filePath, payloadBody, 0644)
		if err != nil {
			return fmt.Errorf("Failed to write to file: %v", err)
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
