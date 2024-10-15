package main

import (
	"context"
	"encoding/json"

	config "dvr_api-go-microservices/pkg/config"
	utils "dvr_api-go-microservices/pkg/utils"

	wsjson "github.com/coder/websocket/wsjson" // websocket json utils
	amqp "github.com/rabbitmq/amqp091-go"      // message broker
	zap "go.uber.org/zap"                      // logger
)

// record and index connected devices and clients
type PubSubHandler struct {
	logger              *zap.Logger                  //logger
	apiSvr              *WsApiSvr                    // api svr
	rabbitmqAmqpChannel *amqp.Channel                // channel to msg broker
	subscriptions       map[string]map[string]string // device ids against connections. Use internal map just for indexing the keys
}

// constructor
func NewPubSubHandler(logger *zap.Logger, apiSvr *WsApiSvr, rabbitmqAmqpChannel *amqp.Channel) (*PubSubHandler, error) {
	r := &PubSubHandler{
		logger:              logger,
		apiSvr:              apiSvr,
		rabbitmqAmqpChannel: rabbitmqAmqpChannel,
		subscriptions:       make(map[string]map[string]string),
	}
	return r, nil
}

// runner
func (sh *PubSubHandler) Run() error {

	// start the intake of device messages from the broker
	go sh.pipeMessagesFromBroker(config.MSGS_FROM_DEVICE_SVR)

	// handle client messages, main program loop
	for i := 0; ; i++ {
		// process one message received from the API server
		subReq, ok := <-sh.apiSvr.svrSubReqBufChan
		if ok {
			err := sh.Subscribe(&subReq)
			if err != nil {
				sh.logger.Error("error processing subscription request: %v", zap.Error(err))
			}
		} else {
			sh.logger.Error("Couldn't receive value from svrSubReqBufChan")
		}
	}
}

// add the subscription requester's connection onto the list of subscribers for each device
func (sh *PubSubHandler) Subscribe(subReq *utils.SubReqWrapper) error {
	// delete old subs
	for _, val := range subReq.OldDevlist {
		// if the map that holds subs isn't inited then just continue
		if sh.subscriptions[val] == nil {
			continue
		}
		delete(sh.subscriptions[val], *subReq.ClientId)
	}
	// add new subs
	for _, val := range subReq.NewDevlist {
		// if the requested device isn't currently registered as a 'publisher'
		// then register them and add this client as a subscriber in case they register, connect, in the future
		if sh.subscriptions[val] == nil {
			sh.subscriptions[val] = make(map[string]string, 0)
		}
		sh.subscriptions[val][*subReq.ClientId] = *subReq.ClientId
	}
	return nil
}

// publish a message. This function works
func (sh *PubSubHandler) Publish(msgWrap *utils.MessageWrapper) error {
	// check if there are even eny susbcribers
	if sh.subscriptions[*msgWrap.ClientId] == nil {
		return nil
	}
	// broadcast message to subscribers
	for k, _ := range sh.subscriptions[*msgWrap.ClientId] {
		// get the websocket connection associated with the id stored in the sub list
		conn, ok := sh.apiSvr.connIndex.Get(k)
		if !ok {
			// if client doesn't exist remove its entry in the map and continue
			delete(sh.subscriptions[*msgWrap.ClientId], k)
			continue
		}
		packTime, err := utils.GetDateFromMessage(msgWrap.Message)
		devMsg := &utils.DeviceMessage_Response{msgWrap.RecvdTime, packTime, msgWrap.Message, "from"}
		// send message to this subscriber
		err = wsjson.Write(context.TODO(), conn, devMsg)
		if err != nil {
			delete(sh.subscriptions[*msgWrap.ClientId], k)
			sh.logger.Debug("removed subscriber %v from subscription list because a write operation failed", zap.String("k", k))
			continue
		}
	}
	// no err
	return nil
}

// Take messages from the broker and send them to clients. Messages from devices
func (s *PubSubHandler) pipeMessagesFromBroker(exchangeName string) {

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

	// reuse it in loop
	var msgWrap utils.MessageWrapper

	// connection loop
	for d := range msgs {

		// this is the message revceived from broker
		err := json.Unmarshal(d.Body, &msgWrap)
		if err != nil {
			s.logger.Error("received erroneous value from message broker, couldn't unmarshal into message wrapper")
		}

		// publish message to all of our clients who have subscribed to it
		s.Publish(&msgWrap)
	}
}
