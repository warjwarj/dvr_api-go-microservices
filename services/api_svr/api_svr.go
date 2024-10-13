package main

import (
	"context"
	"net"
	"net/http"
	"time"

	config "dvr_api-go-microservices/pkg/config" // our shared config
	utils "dvr_api-go-microservices/pkg/utils"   // shared utils

	websocket "github.com/coder/websocket"     // websocket utils
	wsjson "github.com/coder/websocket/wsjson" // json utils for sending it over a websocket
	uuid "github.com/google/uuid"              // unique id for clients
	amqp "github.com/rabbitmq/amqp091-go"      // message broker
	zap "go.uber.org/zap"                      // logger
)

type ApiSvr struct {
	logger              *zap.Logger                      // logger
	endpoint            string                           // IP + port, ex: "192.168.1.77:9047"
	capacity            int                              // num of connections
	sockOpBufSize       int                              // how much memory do we give each connection to perform send/recv operations
	sockOpBufStack      utils.Stack[*[]byte]             // memory region we give each conn to so send/recv
	svrMsgBufSize       int                              // how many messages can we queue on the server at once
	svrMsgBufChan       chan utils.MessageWrapper        // channel we use to queue messages
	svrSubReqBufChan    chan utils.SubReqWrapper         // channel we use to queue subscription requests
	connIndex           utils.Dictionary[websocket.Conn] // index the connection objects against the ids of the clients represented thusly
	getConnectedDevices func() []string                  // function to retreive an index of connected devices
	rabbitmqAmqpChannel *amqp.Channel                    // channel connected to the broker
}

func NewApiSvr(
	logger *zap.Logger,
	endpoint string,
	capacity int,
	bufSize int,
	svrMsgBufSize int,
	getConnectedDevices func() []string,
	rabbitmqAmqpChannel *amqp.Channel,
) (*ApiSvr, error) {
	// create the struct
	svr := ApiSvr{
		logger,
		endpoint,
		capacity,
		bufSize,
		utils.Stack[*[]byte]{},
		svrMsgBufSize,
		make(chan utils.MessageWrapper),
		make(chan utils.SubReqWrapper),
		utils.Dictionary[websocket.Conn]{},
		getConnectedDevices,
		rabbitmqAmqpChannel}

	// init things that need initing
	svr.sockOpBufStack.Init()
	svr.connIndex.Init(capacity)

	// create and store the buffers
	for i := 0; i < svr.capacity; i++ {
		buf := make([]byte, svr.sockOpBufSize)
		svr.sockOpBufStack.Push(&buf)
	}
	return &svr, nil
}

// create and store our buffers
func (s *ApiSvr) Init() error {
	for i := 0; i < s.capacity; i++ {
		buf := make([]byte, s.sockOpBufSize)
		s.sockOpBufStack.Push(&buf)
	}
	return nil
}

// run the server
func (s *ApiSvr) Run() {
	// listen tcp
	l, err := net.Listen("tcp", s.endpoint)
	if err != nil {
		s.logger.Fatal("error listening on %v: %v", zap.String("s.endpoint", s.endpoint), zap.Error(err))
	} else {
		s.logger.Info("websocket server listening on: %v", zap.String("s.endpoint", s.endpoint))
	}

	// start the message broker input and output
	go s.pipeToBroker(config.MSGS_FROM_API_SVR)
	go s.pipeFromBroker(config.MSGS_TO_API_SVR)

	// accept http on the port open for tcp above
	httpSvr := &http.Server{
		Handler: s,
	}
	err = httpSvr.Serve(l)
	if err != nil {
		s.logger.Fatal("error serving websocket server: %v", zap.Error(err))
	}
}

// func called for each connection to handle the websocket connection request, calls and blocks on connHandler
func (s *ApiSvr) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// accept wenbsocket connection
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:       []string{"dvr_api"},
		OriginPatterns:     []string{"*"}, // Accept all origins for simplicity; customize as needed
		InsecureSkipVerify: true,          // Not recommended for production, remove this line in a real application
	})

	// handle err
	if err != nil {
		s.logger.Info("error accepting websocket connection: %v", zap.Error(err))
		return
	}
	s.logger.Info("connection accepted on api svr...")

	// don't really need this but why not
	if c.Subprotocol() != "dvr_api" {
		s.logger.Debug("declined connection because subprotocol != dvr_api")
		c.Close(websocket.StatusPolicyViolation, "client must speak the dvr_api subprotocol")
		return
	}

	// handle connection
	err = s.connHandler(c)
	if err != nil {
		s.logger.Error("error in connection handler func: %v", zap.Error(err))
	}
	c.CloseNow()
}

/*
 *  %x0 denotes a continuation frame
 *  %x1 denotes a text frame
 *  %x2 denotes a binary frame
 *  %x3-7 are reserved for further non-control frames
 *  %x8 denotes a connection close
 *  %x9 denotes a ping
 *  %xA denotes a pong
 *  %xB-F are reserved for further control frames
 */

// handle one connection.
func (s *ApiSvr) connHandler(conn *websocket.Conn) error {

	// req = reusable holder for string, gen id as an arbitrary number
	var req utils.ApiReq_WS
	// var res utils.ApiRes_WS
	var id string = uuid.New().String()
	var subscriptions []string

	// add to connection index, defer the removal from the connection index
	s.connIndex.Add(id, *conn)
	defer s.connIndex.Delete(id)

	// connection loop
	for {
		// read one websocket message frame as json, unmarshal into a struct
		err := wsjson.Read(context.TODO(), conn, &req)
		if err != nil {
			// don't realistically need to know why but might be useful for debug.
			s.logger.Debug("websocket connection closed, status: %v, websocket.CloseStatus: %v", zap.Error(err), zap.String("websocket.CloseStatus(err)", websocket.CloseStatus(err).String()))
			return nil
		}

		// // if they have send a request for the connected devices list then oblige
		// if req.GetConnectedDevices {
		// 	res = utils.ApiRes_WS{s.getConnectedDevices()}
		// 	wsjson.Write(context.TODO(), conn, &res)
		// }

		// register the subscription request
		s.svrSubReqBufChan <- utils.SubReqWrapper{ClientId: &id, NewDevlist: req.Subscriptions, OldDevlist: subscriptions}
		subscriptions = make([]string, len(req.Subscriptions))
		copy(subscriptions, req.Subscriptions)

		// todo pass the array instead of the induvidual message
		for _, val := range req.Messages {
			s.svrMsgBufChan <- utils.MessageWrapper{val, &id, time.Now()}
		}
		req = utils.ApiReq_WS{}
	}
}

// take messages from devices and publish them to broker.
func (s *ApiSvr) pipeToBroker(exchangeName string) {

	// declare the device message output exchange
	err := s.rabbitmqAmqpChannel.ExchangeDeclare(
		exchangeName, // name
		"fanout",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		s.logger.Fatal("error piping messages from message broker %v", zap.Error(err))
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

// Take messages from the broker and send them to clients.
func (s *ApiSvr) pipeFromBroker(exchangeName string) {

	// declare exchange we are taking messages from.
	err := s.rabbitmqAmqpChannel.ExchangeDeclare(
		exchangeName, // name
		"fanout",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		s.logger.Fatal("error piping messages from message broker %v", zap.Error(err))
	}

	// declare a queue onto the exchange above
	q, err := s.rabbitmqAmqpChannel.QueueDeclare(
		"",    // name -
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		s.logger.Fatal("error declaring queue %v", zap.Error(err))
	}

	// bind the queue onto the exchange
	err = s.rabbitmqAmqpChannel.QueueBind(
		q.Name,       // queue name
		"",           // routing key
		exchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		s.logger.Error("error binding queue %v", zap.Error(err))
	}

	// get a golang channel we can read messages from.
	msgs, err := s.rabbitmqAmqpChannel.Consume(
		q.Name, // name - WE DEPEND ON THE QUEUE BEING NAMED THE SAME AS THE EXCHANGE - ONLY ONE Q CONSUMING FROM THIS EXCHANGE
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		s.logger.Error("error consuming from queue %v", zap.Error(err))
	}

	// connection loop
	for d := range msgs {
		s.svrMsgBufChan <- d.Body
	}
}
