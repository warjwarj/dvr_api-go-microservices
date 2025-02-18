package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

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
	subscriptionsLock   sync.Mutex                   // lock for the subs map
	msgSubscriptions    map[string]map[string]string // device ids against client ids, whose value is the same client it - map it used so we don't have to loop over the inner map.
	vidReqSubscriptions map[string]*string           // req match strings against client ids
}

// constructor
func NewPubSubHandler(logger *zap.Logger, apiSvr *WsApiSvr, rabbitmqAmqpChannel *amqp.Channel) (*PubSubHandler, error) {
	r := &PubSubHandler{
		logger:              logger,
		apiSvr:              apiSvr,
		rabbitmqAmqpChannel: rabbitmqAmqpChannel,
		msgSubscriptions:    make(map[string]map[string]string),
		vidReqSubscriptions: make(map[string]*string),
	}
	return r, nil
}

// runner
func (sh *PubSubHandler) Run() error {

	// start the intake of device messages from the broker
	go sh.pipeMessagesFromBroker(config.MSGS_FROM_DEVICE_SVR)                // <-- pub dev messages
	go sh.pipeVideoDescriptionsFromBroker(config.VIDEO_DESCRIPTION_EXCHANGE) // <-- pub video descriptions

	// handle message subs
	go func() {
		// handle client messages, main program loop
		for {
			// process one sub req received from a websocket
			subReq, ok := <-sh.apiSvr.svrSubReqBufChan
			if ok {
				err := sh.Sub_DeviceMessages(&subReq) // <-- sub device messages
				if err != nil {
					sh.logger.Error("error processing subscription request: %v", zap.Error(err))
				}
			} else {
				sh.logger.Error("Couldn't receive value from svrSubReqBufChan")
			}
		}
	}()

	// handle vidreq subs
	go func() {
		// process one sub req received from a websocket
		vidReq, ok := <-sh.apiSvr.svrVideoReqBufChan
		if ok {
			err := sh.Sub_VideoDescription(&vidReq) // <-- sub video descriptions
			if err != nil {
				sh.logger.Error("error processing subscription request: %v", zap.Error(err))
			}
		} else {
			sh.logger.Error("Couldn't receive value from svrSubReqBufChan")
		}
	}()

	// hang indefinitely
	var forever chan struct{}
	<-forever

	return nil
}

// add the subscription requester's connection onto the list of subscribers for each device
func (sh *PubSubHandler) Sub_DeviceMessages(subReq *utils.SubReqWrapper) error {

	// threadsafety
	sh.subscriptionsLock.Lock()
	defer sh.subscriptionsLock.Unlock()

	// delete old subs
	for _, val := range subReq.OldDevlist {
		// if the map that holds subs isn't initialised then just continue
		if sh.msgSubscriptions[val] == nil {
			continue
		}
		delete(sh.msgSubscriptions[val], *subReq.ClientId)
	}
	// add new subs
	for _, val := range subReq.NewDevlist {
		// if the requested device isn't currently registered as a 'publisher'
		// then register them and add this client as a subscriber in case they register (connect) in the future
		if sh.msgSubscriptions[val] == nil {
			sh.msgSubscriptions[val] = make(map[string]string, 0)
		}
		sh.msgSubscriptions[val][*subReq.ClientId] = *subReq.ClientId
	}
	return nil
}

// publish a device message or a
func (sh *PubSubHandler) Pub_DeviceMessages(msgWrap *utils.MessageWrapper) error {

	// avoid concurrent read/write/iteration errors
	sh.subscriptionsLock.Lock()
	defer sh.subscriptionsLock.Unlock()

	// check if there are even eny susbcribers
	if sh.msgSubscriptions[*msgWrap.ClientId] == nil {
		return nil
	}

	// broadcast message to subscribers - there are no values in the nested map so ignore
	for k, _ := range sh.msgSubscriptions[*msgWrap.ClientId] {

		// get the websocket connection associated with the id stored in the sub list
		conn, ok := sh.apiSvr.connIndex.Get(k)
		if !ok {
			// if client doesn't exist remove its entry in the map and continue
			delete(sh.msgSubscriptions[*msgWrap.ClientId], k)
			continue
		}

		// get date from message
		packTime, err := utils.GetDateFromMessage(msgWrap.Message)
		if err != nil {
			sh.logger.Error("couldn't retreive date from message", zap.Error(err))
		}

		// create our device message struct
		devMsg := &utils.DeviceMessage_Response{msgWrap.RecvdTime, packTime, msgWrap.Message, "from"}

		// send message to this subscriber
		err = wsjson.Write(context.TODO(), conn, devMsg)
		if err != nil {
			delete(sh.msgSubscriptions[*msgWrap.ClientId], k)
			sh.logger.Debug("removed subscriber %v from subscription list because a write operation failed", zap.String("k", k))
			continue
		}
	}
	return nil
}

// this function is used to publish a video description received from the camera server
// the req match string we try and look up in the map should have been put there on the websocket server
func (s *PubSubHandler) Pub_VideoDescription(msgWrap *utils.MessageWrapper) {

	// avoid concurrent read/write/iteration errors
	s.subscriptionsLock.Lock()
	defer s.subscriptionsLock.Unlock()

	// nil check
	if msgWrap == nil || msgWrap.VideoDescription == nil {
		s.logger.Error("cannot publish video description as its' field in the msgWrap struct is nil.")
	}

	// get the req match string to look up client id in the map
	reqMatchString, err := utils.ParseReqMatchStringFromVideoDescription(msgWrap.VideoDescription)
	if err != nil {
		s.logger.Error("couldn't parse req match string from video description")
	}

	// retreive the connection from the map - there are no values in the nested map so ignore
	apiClientId, ok := s.vidReqSubscriptions[reqMatchString]
	if !ok {
		s.logger.Error("Received video description that we haven't got a request for")
		return
	}

	// get the connection
	conn, ok := s.apiSvr.connIndex.Get(*apiClientId)
	if !ok {
		fmt.Errorf("couldn't retreive connection from index", zap.Error(err))
	}

	fmt.Println("Vid desc: ", msgWrap.VideoDescription)

	// write the video description struct to the connection
	err = wsjson.Write(context.TODO(), conn, msgWrap.VideoDescription)
	if err != nil {
		s.logger.Debug("websocket write operation failed", zap.Error(err))
	}
}

// this function is used to publish a video description received from the camera server
// the req match string we try and look up in the map should have been put there on the websocket server
func (sh *PubSubHandler) Sub_VideoDescription(vidReq *utils.SubReqWrapper) error {

	// avoid concurrent read/write/iteration errors
	sh.subscriptionsLock.Lock()
	defer sh.subscriptionsLock.Unlock()

	// subscribe and exit.
	sh.vidReqSubscriptions[vidReq.ReqMatchString] = vidReq.ClientId
	return nil
}

// Take messages from the broker and send them to clients. Messages from devices
func (s *PubSubHandler) pipeMessagesFromBroker(exchangeName string) {

	// setup delivery pipe from broker
	err, deliveryChan := utils.SetupPipeFromBroker(exchangeName, s.rabbitmqAmqpChannel)
	if err != nil {
		s.logger.Error("error setting up pipe from broker", zap.Error(err))
	}

	// loop var, reused
	var msgWrap utils.MessageWrapper

	// connection loop
	for d := range deliveryChan {

		// this is the message revceived from broker
		err := json.Unmarshal(d.Body, &msgWrap)
		if err != nil {
			s.logger.Error("received erroneous value from message broker, couldn't unmarshal into message wrapper")
		}

		// publish message to all of our clients who have subscribed to it
		s.Pub_DeviceMessages(&msgWrap)
	}
}

func (s *PubSubHandler) pipeVideoDescriptionsFromBroker(exchangeName string) {

	// setup delivery pipe from broker
	err, deliveryChan := utils.SetupPipeFromBroker(exchangeName, s.rabbitmqAmqpChannel)
	if err != nil {
		s.logger.Error("error setting up pipe from broker", zap.Error(err))
	}

	// reuse it in loop
	var vidDesc utils.VideoDescription

	// connection loop
	for d := range deliveryChan {

		// this is the message revceived from broker
		err := json.Unmarshal(d.Body, &vidDesc)
		if err != nil {
			s.logger.Error("received erroneous value from message broker, couldn't unmarshal into message wrapper")
		}

		// publish the video we've ben notified of
		s.Pub_VideoDescription(&utils.MessageWrapper{VideoDescription: &vidDesc})
	}
}
