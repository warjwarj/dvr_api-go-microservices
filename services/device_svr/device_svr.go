package main

import (
	"io"
	"net"

	"dvr_api-go-microservices/pkg/config" // our shared config package
	"dvr_api-go-microservices/pkg/utils"  // our shared utils package

	amqp "github.com/rabbitmq/amqp091-go" // rabbitmq import
	"go.uber.org/zap"
)

type DeviceSvr struct {
	logger         *zap.Logger
	endpoint       string                     // IP + port, ex: "192.168.1.77:9047"
	capacity       int                        // num of connections
	sockOpBufSize  int                        // how much memory do we give each connection to perform send/recv operations
	sockOpBufStack utils.Stack[*[]byte]       // memory region we give each conn to so send/recv
	svrMsgBufSize  int                        // how many messages can we queue on the server at once
	svrMsgBufChan  chan utils.MessageWrapper  // the channel we use to queue the messages
	connIndex      utils.Dictionary[net.Conn] // index of the connection objects against the ids of the devices represented thusly
	msgChannel     *amqp.Channel
}

func NewDeviceSvr(logger *zap.Logger, endpoint string, capacity int, bufSize int, svrMsgBufSize int) (*DeviceSvr, error) {
	// holder struct
	svr := DeviceSvr{
		logger,
		endpoint,
		capacity,
		bufSize,
		utils.Stack[*[]byte]{},
		svrMsgBufSize,
		make(chan utils.MessageWrapper),
		utils.Dictionary[net.Conn]{}}

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
	// init the buffers we use for io socket operations
	for i := 0; i < s.capacity; i++ {
		buf := make([]byte, s.sockOpBufSize)
		s.sockOpBufStack.Push(&buf)
	}

	// dial the rabbitmq server
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return err
	}
	defer conn.Close()

	// open the channel
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// declare the exchange to which we're publishing to
	err = ch.ExchangeDeclare(
		config.DEVICEMSG_OUTPUT_CHANNEL, // name
		"fanout",                        // type
		true,                            // durable
		false,                           // auto-deleted
		false,                           // internal
		false,                           // no-wait
		nil,                             // arguments
	)
	if err != nil {
		return err
	}

	// put it in the server struct and return
	s.msgChannel = ch
	return nil
}

// run the server, blocking
func (s *DeviceSvr) Run() {
	ln, err := net.Listen("tcp", s.endpoint)
	if err != nil {
		s.logger.Fatal("error listening on %v: %v", zap.String("s.endpoint", s.endpoint), zap.Error(err))
	} else {
		s.logger.Info("device server listening on: %v", zap.String("s.endpoint", s.endpoint))
	}
	for {
		c, err := ln.Accept()
		if err != nil {
			s.logger.Info("error accepting websocket connection: %v", zap.Error(err))
			continue
		}
		s.logger.Info("connection accepted on device svr...")
		go func() {
			err := s.intake(c)
			if err != nil {
				s.logger.Error("error in device connection loop: %v", zap.Error(err))
			}
			c.Close()
		}()
	}
}

// handle each connection
func (s *DeviceSvr) intake(conn net.Conn) error {

	// get buffer for read operations
	buf, err := s.sockOpBufStack.Pop()
	if err != nil {
		s.logger.Error("error retreiving buffer from stack: %v", zap.Error(err))
	}

	// loop variables
	var msg string = ""
	var id string = ""
	var recvd int = 0

	// connection loop
	for {
		// read from wherever we finished last time
		tmp, err := conn.Read((*buf)[recvd:])
		recvd += tmp

		// if error is just disconnection then return nil else return the error
		if err != nil {
			s.logger.Debug("connection closed on device svr...")
			s.sockOpBufStack.Push(buf)
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
		if id == "" {
			err = utils.GetIdFromMessage(&msg, &id)
			if err != nil {
				s.logger.Debug("error getting id from msg %v", zap.String("msg", msg))
				continue
			}
			s.connIndex.Add(id, conn)
			defer s.connIndex.Delete(id)
		}

		// send the messages to the relay
		//s.svrMsgBufChan <- utils.MessageWrapper{Message: msg, ClientId: &id, RecvdTime: time.Now()}
		err = s.msgChannel.Publish(
			"logs", // exchange
			"",     // routing key
			false,  // mandatory
			false,  // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(body),
			})
	}
}
