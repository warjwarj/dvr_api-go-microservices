package main

import (
	"context"
	"encoding/json"
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

type WsApiSvr struct {
	logger              *zap.Logger                      // logger
	endpoint            string                           // IP + port, ex: "192.168.1.77:9047"
	svrMsgBufChan       chan utils.MessageWrapper        // channel we use to queue messages
	svrSubReqBufChan    chan utils.SubReqWrapper         // channel we use to queue subscription requests
	connIndex           utils.Dictionary[websocket.Conn] // index the connection objects against the ids of the clients represented thusly
	rabbitmqAmqpChannel *amqp.Channel                    // channel connected to the broker
	capacity            int                              // amount of devices that can be connected
}

func NewWsApiSvr(
	logger *zap.Logger,
	endpoint string,
	rabbitmqAmqpChannel *amqp.Channel,
	capacity int,
) (*WsApiSvr, error) {
	// create the struct
	svr := WsApiSvr{
		logger,
		endpoint,
		make(chan utils.MessageWrapper),
		make(chan utils.SubReqWrapper),
		utils.Dictionary[websocket.Conn]{},
		rabbitmqAmqpChannel,
		capacity}

	// init the connIndex
	svr.connIndex.Init(capacity)

	// return the server struct
	return &svr, nil
}

// run the server
func (s *WsApiSvr) Run() {
	// listen tcp
	l, err := net.Listen("tcp", s.endpoint)
	if err != nil {
		s.logger.Fatal("error listening on %v: %v", zap.String("s.endpoint", s.endpoint), zap.Error(err))
	} else {
		s.logger.Info("api server listening on: %v", zap.String("s.endpoint", s.endpoint))
	}

	// start the output to message broker
	go s.pipeMessagesToBroker(config.MSGS_FROM_API_SVR)

	// accept http on the port open for tcp above
	httpSvr := &http.Server{
		Handler: s,
	}
	err = httpSvr.Serve(l)
	if err != nil {
		s.logger.Fatal("error serving api server: %v", zap.Error(err))
	}
}

// func called for each connection to handle the websocket connection request, calls and blocks on connHandler
func (s *WsApiSvr) ServeHTTP(w http.ResponseWriter, r *http.Request) {

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
func (s *WsApiSvr) connHandler(conn *websocket.Conn) error {

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

		// register the subscription request
		s.svrSubReqBufChan <- utils.SubReqWrapper{ClientId: &id, NewDevlist: req.Subscriptions, OldDevlist: subscriptions}
		subscriptions = make([]string, len(req.Subscriptions))
		copy(subscriptions, req.Subscriptions)

		// send the message to the function which handles sending to message broker
		for _, val := range req.Messages {
			s.svrMsgBufChan <- utils.MessageWrapper{val, &id, time.Now()}
		}
		req = utils.ApiReq_WS{}
	}
}

// take messages from devices and publish them to broker.
func (s *WsApiSvr) pipeMessagesToBroker(exchangeName string) {

	// set up the pipeline to send messages to the broker
	if err := utils.SetupPipeToBroker(exchangeName, s.rabbitmqAmqpChannel); err != nil {
		s.logger.Error("Error setting up pipe to broker", zap.Error(err))
	}

	// loop over channel and pipe messages into the rabbitmq exchange.
	for msgWrap := range s.svrMsgBufChan {

		// get json bytes so we can send them through the message broker
		bytes, err := json.Marshal(msgWrap)
		if err != nil {
			s.logger.Error("error getting bytes from message wrapper struct: %v", zap.Error(err))
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
			s.logger.Error("error piping messages into message broker %v", zap.Error(err))
		}
	}
}
