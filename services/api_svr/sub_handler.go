package main

import (
	"context"
	utils "dvr_api-go-microservices/pkg/utils"
	"sync"

	// our shared config

	wsjson "github.com/coder/websocket/wsjson" // message broker
	zap "go.uber.org/zap"                      // logger
)

// record and index connected devices and clients
type SubscriptionHandler struct {

	// internal
	subscriptions map[string]map[string]string // device ids against connections. Use internal map just for indexing the keys
	lock          sync.Mutex                   // might be uneccessary

	// injected
	logger  *zap.Logger
	clients *ApiSvr // api svr
}

// constructor
func NewSubscriptionHandler(logger *zap.Logger, clients *ApiSvr) (*SubscriptionHandler, error) {
	r := &SubscriptionHandler{
		logger:        logger,
		clients:       clients,
		subscriptions: make(map[string]map[string]string),
	}
	return r, nil
}

func (sh *SubscriptionHandler) SubIntake() error {
	// handle messages, main program loop
	for i := 0; ; i++ {
		// process one message received from the API server
		subReq, ok := <-sh.clients.svrSubReqBufChan
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
func (sh *SubscriptionHandler) Subscribe(subReq *utils.SubReqWrapper) error {
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
func (sh *SubscriptionHandler) Publish(msgWrap *utils.MessageWrapper) error {
	// check if there are even eny susbcribers
	if sh.subscriptions[*msgWrap.ClientId] == nil {
		return nil
	}
	// broadcast message to subscribers
	for k, _ := range sh.subscriptions[*msgWrap.ClientId] {
		// get the websocket connection associated with the id stored in the sub list
		conn, ok := sh.clients.connIndex.Get(k)
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

// // publish a message. O(n) where n is the number of API clients connected to the server.
// func (sh *SubscriptionHandler) PublishConnectedDevices(msgWrap *utils.MessageWrapper) error {
// 	// do this once at the start and reuse to avoid locking.
// 	connectedDevices := sh.clients.connIndex.GetAllKeys()

// 	// the json we'll send to the API clients
// 	var connectedDevList utils.ApiRes_WS = utils.ApiRes_WS{ConnectedDevicesList: connectedDevices}

// 	// broadcast message to subscribers
// 	for _, k := range connectedDevices {
// 		// get the websocket connection associated with the id stored in the sub list
// 		conn, ok := sh.clients.connIndex.Get(k)
// 		if !ok {
// 			continue
// 		}
// 		// send message to this subscriber
// 		err := wsjson.Write(context.TODO(), conn, connectedDevList)
// 		if err != nil {
// 			delete(sh.subscriptions[*msgWrap.ClientId], k)
// 			sh.logger.Debug("removed subscriber %v from subscription list because a write operation failed", zap.String("k", k))
// 			continue
// 		}
// 	}
// 	// no err
// 	return nil
// }
