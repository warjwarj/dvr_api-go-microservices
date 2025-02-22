package main

import (
	"encoding/json"
	"io"
	"net"
	"strings"
	"time"

	config "dvr_api-go-microservices/pkg/config"
	utils "dvr_api-go-microservices/pkg/utils" // our shared utils package

	amqp "github.com/rabbitmq/amqp091-go" // rabbitmq import
	zap "go.uber.org/zap"
)

type DeviceSvr struct {
	logger                *zap.Logger
	endpoint              string                                 // IP + port, ex: "192.168.1.77:9047"
	capacity              int                                    // num of connections
	sockOpBufSize         int                                    // how much memory do we give each connection to perform send/recv operations
	sockOpBufStack        utils.Stack[*[]byte]                   // memory region we give each conn to so send/recv
	svrMsgBufSize         int                                    // how many messages can we queue on the server at once
	svrMsgBufChan         chan []byte                            // the channel we use to queue the messages
	svrRegisterChangeChan chan utils.DeviceConnectionStateChange // fire a struct down this channel each time a device connects or disconnects
	connIndex             utils.Dictionary[net.Conn]             // index of the connection objects against the ids of the devices represented thusly
	rabbitmqAmqpChannel   *amqp.Channel                          // tcp connection with rabbitmq server
}

func NewDeviceSvr(
	logger *zap.Logger,
	endpoint string,
	capacity int,
	bufSize int,
	svrMsgBufSize int,
	rabbitmqAmqpChannel *amqp.Channel,
) (*DeviceSvr, error) {
	// holder struct
	svr := DeviceSvr{
		logger,
		endpoint,
		capacity,
		bufSize,
		utils.Stack[*[]byte]{},
		svrMsgBufSize,
		make(chan []byte),
		make(chan utils.DeviceConnectionStateChange),
		utils.Dictionary[net.Conn]{},
		rabbitmqAmqpChannel}

	// init the stack we use to store the buffers
	svr.sockOpBufStack.Init()

	// init the Dictionary
	svr.connIndex.Init(capacity)

	// create and store the buffers
	for i := 0; i < svr.capacity; i++ {
		buf := make([]byte, svr.sockOpBufSize)
		svr.sockOpBufStack.Push(&buf)
	}
	return &svr, nil
}

// create and store our buffers
func (s *DeviceSvr) Init() error {
	// init the buffers we use for io socket operations. double capacity - buf for send buf for recv for each socket
	for i := 0; i < s.capacity; i++ {
		buf := make([]byte, s.sockOpBufSize)
		s.sockOpBufStack.Push(&buf)
	}
	return nil
}

// run the server, blocking
func (s *DeviceSvr) Run() {
	// start server listening for connections
	ln, err := net.Listen("tcp", s.endpoint)
	if err != nil {
		s.logger.Fatal("error listening on %v: %v", zap.String("s.endpoint", s.endpoint), zap.Error(err))
	} else {
		s.logger.Info("device server listening on: %v", zap.String("s.endpoint", s.endpoint))
	}

	// start the output and input from the message broker
	go s.pipeMessagesToBroker(config.MSGS_FROM_DEVICE_SVR)
	go s.pipeMessagesFromBroker(config.MSGS_FROM_API_SVR)

	// start piping device conneciton state changes to server
	go s.pipeConnectedDevicesToBroker(config.DEVICE_CONNECTION_STATE_CHANGES)

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

// receive connections and messages. Input from devices
func (s *DeviceSvr) deviceConnLoop(conn net.Conn) error {

	// get buffer for read operations
	buf, err := s.sockOpBufStack.Pop()
	if err != nil {
		s.logger.Error("error retreiving buffer from stack: %v", zap.Error(err))
	}

	// loop variables
	var msg string = ""
	var id *string = nil
	var recvd int = 0

	// connection loop
	for {
		// read from wherever we finished last time
		tmp, err := conn.Read((*buf)[recvd:])
		recvd += tmp

		// if error is just disconnection then return nil else return the error
		if err != nil {
			s.logger.Debug("connection closed on device svr...")
			if err == io.EOF {
				return nil
			}
			return err
		}

		// check if we've recvd complete message.
		if (*buf)[recvd-1] != '\r' {
			continue
		}

		// get complete message, reset byte counter
		msg = string((*buf)[:recvd])
		recvd = 0

		// set id if not already set
		if id == nil {

			// parse the id from the message
			id, err = utils.GetIdFromMessage(&msg)
			if err != nil {
				s.logger.Debug("error getting id from msg %v", zap.String("msg", msg))
				continue
			}

			// add to conn index and notify of connection
			s.connIndex.Add(*id, conn)
			s.svrRegisterChangeChan <- utils.DeviceConnectionStateChange{strings.Clone(*id), true}

			// remove from index and notify of disconnection
			defer func() {
				s.connIndex.Delete(*id)
				s.sockOpBufStack.Push(buf)
				s.svrRegisterChangeChan <- utils.DeviceConnectionStateChange{strings.Clone(*id), false}
			}()
		}

		// marshal the struct into json then convert into byte array
		bytes, err := json.Marshal(utils.MessageWrapper{Message: msg, ClientId: id, RecvdTime: time.Now()})
		if err != nil {
			return err
		}

		// send msg to channel
		s.svrMsgBufChan <- bytes
	}
}

// publish device messages to broker
func (s *DeviceSvr) pipeMessagesToBroker(exchangeName string) {

	// set up the pipeline to send messages to the broker
	if err := utils.SetupPipeToBroker(exchangeName, s.rabbitmqAmqpChannel); err != nil {
		s.logger.Error("Error setting up pipe to broker", zap.Error(err))
	}

	// loop over channel and pipe messages into the rabbitmq exchange.
	for bytes := range s.svrMsgBufChan {
		// publish our JSON message to the exchange which handles messages from devices
		err := s.rabbitmqAmqpChannel.Publish(
			exchangeName, // exchange
			"",           // routing key
			false,        // mandatory
			false,        // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(bytes),
			})
		if err != nil {
			s.logger.Error("error piping messages into message broker %v", zap.Error(err))
		}
	}
}

// publish device messages to broker
func (s *DeviceSvr) pipeConnectedDevicesToBroker(exchangeName string) {

	// set up the pipeline to send messages to the broker
	if err := utils.SetupPipeToBroker(exchangeName, s.rabbitmqAmqpChannel); err != nil {
		s.logger.Error("Error setting up pipe to broker", zap.Error(err))
	}

	// this loop fires each time a device connects or disconnects
	for i := range s.svrRegisterChangeChan {

		// get bytes
		bytes, err := json.Marshal(i)
		if err != nil {
			s.logger.Error("error marshalling into json %v", zap.Error(err))
		}

		// publish our JSON message to the exchange which handles messages from devices
		err = s.rabbitmqAmqpChannel.Publish(
			exchangeName, // exchange
			"",           // routing key
			false,        // mandatory
			false,        // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(bytes),
			})
		if err != nil {
			s.logger.Error("error piping device connection event into message broker %v", zap.Error(err))
		}
	}
}

// Take messages from the broker and send them to devices.
func (s *DeviceSvr) pipeMessagesFromBroker(exchangeName string) {

	// setup delivery pipe from broker
	err, deliveryChan := utils.SetupPipeFromBroker(exchangeName, s.rabbitmqAmqpChannel)
	if err != nil {
		s.logger.Error("error setting up pipe from broker", zap.Error(err))
	}

	// connection loop
	for d := range deliveryChan {

		// this is the message revceived from broker
		var msgWrap utils.MessageWrapper
		err := json.Unmarshal(d.Body, &msgWrap)
		if err != nil {
			s.logger.Error("received erroneous value from message broker, couldn't unmarshal into message wrapper")
		}

		// extract the device the message pertains to
		dev_id, err := utils.GetIdFromMessage(&msgWrap.Message)
		if err != nil {
			s.logger.Error("error processing message from device", zap.Error(err))
			continue
		}

		// verify device connection, get the connection object
		devConn, devConnOk := s.connIndex.Get(*dev_id)
		if !devConnOk {
			s.logger.Warn("message sent for device not connected")
			continue
		}

		// send the requested message to the device
		_, err = (*devConn).Write([]byte(msgWrap.Message))
		if err != nil {
			s.logger.Error("received message destined for device not connected")
		}
	}
}
