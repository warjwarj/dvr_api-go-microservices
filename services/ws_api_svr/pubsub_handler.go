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
	logger              *zap.Logger                       //logger
	apiSvr              *WsApiSvr                         // api svr
	rabbitmqAmqpChannel *amqp.Channel                     // channel to msg broker
	subscriptionsLock   sync.Mutex                        // lock for the subs map
	subscriptions       map[string]map[string]string      // device ids against client ids, whose value is the same client it - map it used so we don't have to loop over the inner map.
	videoRequsts        map[string]utils.VideoDescription // video description string
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
	go sh.pipeVideoDescriptionsFromBroker(config.VIDEO_DESCRIPTION_EXCHANGE)

	// handle client messages, main program loop
	for i := 0; ; i++ {
		// process one message received from the API server
		subReq, ok := <-sh.apiSvr.svrSubReqBufChan
		if ok {
			err := sh.SubscribeToMessages(&subReq)
			if err != nil {
				sh.logger.Error("error processing subscription request: %v", zap.Error(err))
			}
		} else {
			sh.logger.Error("Couldn't receive value from svrSubReqBufChan")
		}
	}
}

// add the subscription requester's connection onto the list of subscribers for each device
func (sh *PubSubHandler) SubscribeToMessages(subReq *utils.SubReqWrapper) error {

	// threadsafety
	sh.subscriptionsLock.Lock()
	defer sh.subscriptionsLock.Unlock()

	// delete old subs
	for _, val := range subReq.OldDevlist {
		// if the map that holds subs isn't initialised then just continue
		if sh.subscriptions[val] == nil {
			continue
		}
		delete(sh.subscriptions[val], *subReq.ClientId)
	}
	// add new subs
	for _, val := range subReq.NewDevlist {
		// if the requested device isn't currently registered as a 'publisher'
		// then register them and add this client as a subscriber in case they register (connect) in the future
		if sh.subscriptions[val] == nil {
			sh.subscriptions[val] = make(map[string]string, 0)
		}
		sh.subscriptions[val][*subReq.ClientId] = *subReq.ClientId
	}
	return nil
}

// publish a device message or a
func (sh *PubSubHandler) PublishDeviceMessage(msgWrap *utils.MessageWrapper) error {

	// avoid concurrent read/write/iteration errors
	sh.subscriptionsLock.Lock()
	defer sh.subscriptionsLock.Unlock()

	// check if there are even eny susbcribers
	if sh.subscriptions[*msgWrap.ClientId] == nil {
		return nil
	}

	// broadcast message to subscribers - there are no values in the nested map so ignore
	for k, _ := range sh.subscriptions[*msgWrap.ClientId] {

		// get the websocket connection associated with the id stored in the sub list
		conn, ok := sh.apiSvr.connIndex.Get(k)
		if !ok {
			// if client doesn't exist remove its entry in the map and continue
			delete(sh.subscriptions[*msgWrap.ClientId], k)
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
			delete(sh.subscriptions[*msgWrap.ClientId], k)
			sh.logger.Debug("removed subscriber %v from subscription list because a write operation failed", zap.String("k", k))
			continue
		}
	}
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
		s.PublishDeviceMessage(&msgWrap)
	}
}

// this function is used to publish a video description received from the camera server
func (sh *PubSubHandler) PublishVideoDescription(msgWrap *utils.MessageWrapper) error {

	fmt.Println(msgWrap)

	// avoid concurrent read/write/iteration errors
	sh.subscriptionsLock.Lock()
	defer sh.subscriptionsLock.Unlock()

	// get the req match string to look up client id in the map
	reqMatchString, err := utils.ParseReqMatchStringFromVideoDescription(msgWrap.VideoDescription)
	if err != nil {
		sh.logger.Error("couldn't parse req match string from video description")
	}

	// retreive the connection from the map - there are no values in the nested map so ignore
	for apiClientId, _ := range sh.subscriptions[reqMatchString] {

		// get the connection
		conn, ok := sh.apiSvr.connIndex.Get(apiClientId)
		if !ok {
			fmt.Errorf("couldn't retreive connection from index", zap.Error(err))
		}

		// write the video description struct to the connection
		err = wsjson.Write(context.TODO(), conn, msgWrap.VideoDescription)
		if err != nil {
			sh.logger.Debug("websocket write operation failed", zap.Error(err))
		}
	}
	return nil
}

func (s *PubSubHandler) pipeVideoDescriptionsFromBroker(exchangeName string) {

	// setup delivery pipe from broker
	err, deliveryChan := utils.SetupPipeFromBroker(exchangeName, s.rabbitmqAmqpChannel)
	if err != nil {
		s.logger.Error("error setting up pipe from broker", zap.Error(err))
	}

	// reuse it in loop
	var msgWrap utils.VideoDescription

	// connection loop
	for d := range deliveryChan {

		// this is the message revceived from broker
		err := json.Unmarshal(d.Body, &msgWrap)
		if err != nil {
			s.logger.Error("received erroneous value from message broker, couldn't unmarshal into message wrapper")
		}

		// publish the video we've ben notified of
		s.PublishVideoDescription(&utils.MessageWrapper{VideoDescription: &msgWrap})
	}
}
