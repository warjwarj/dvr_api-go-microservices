package utils

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go" // rabbitmq import
	bson "go.mongodb.org/mongo-driver/bson"
	mongo "go.mongodb.org/mongo-driver/mongo"
	options "go.mongodb.org/mongo-driver/mongo/options"
	zap "go.uber.org/zap"
)

/*
~~~~~~~~~~~~~~~
STACK
Threadsafe Stack implementation
~~~~~~~~~~~~~~~
*/

type Stack[T any] struct {
	items []T
	lock  sync.Mutex
}

// init
func (s *Stack[T]) Init() {
	s.items = make([]T, 0)
}

// Push adds an item to the top of the stack.
func (s *Stack[T]) Push(item T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.items = append(s.items, item)
}

// Pop removes and returns the top item from the stack.
func (s *Stack[T]) Pop() (T, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.items) == 0 {
		var zVal T
		return zVal, errors.New("stack is empty")
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item, nil
}

// Peek returns the top item from the stack without removing it.
func (s *Stack[T]) Peek() (T, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.items) == 0 {
		var zero T
		return zero, errors.New("stack is empty")
	}
	return s.items[len(s.items)-1], nil
}

// IsEmpty checks if the stack is empty.
func (s *Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}

// Size returns the number of items in the stack.
func (s *Stack[T]) Size() int {
	return len(s.items)
}

/*
~~~~~~~~~~~~~~~
DICTIONARY
Threadsafe dictionary implementation
~~~~~~~~~~~~~~~
*/

// Define a dictionary type
type Dictionary[T any] struct {
	internal map[string]*T
	lock     sync.Mutex
}

func (d *Dictionary[T]) Init(capacity int) {
	d.internal = make(map[string]*T, capacity)
}

// Function to add a key-value pair to the dictionary
func (d *Dictionary[T]) Add(key string, value T) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.internal[key] = &value
}

// Function to get a value from the dictionary by key
func (d *Dictionary[T]) Get(key string) (*T, bool) {
	d.lock.Lock()
	defer d.lock.Unlock()
	value, exists := d.internal[key]
	return value, exists
}

// Function to get a value from the dictionary by key
func (d *Dictionary[T]) GetAllKeys() []string {
	d.lock.Lock()
	defer d.lock.Unlock()
	keys := make([]string, len(d.internal))
	i := 0
	for k := range d.internal {
		keys[i] = k
		i++
	}
	return keys
}

// Function to delete a key-value pair from the dictionary
func (d *Dictionary[T]) Delete(key string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	delete(d.internal, key)
}

/*
~~~~~~~~~~~~~~~
GENERIC UTILS
~~~~~~~~~~~~~~~
*/

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomString := make([]byte, length)
	for i := range randomString {
		randomString[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(randomString)
}

/*
~~~~~~~~~~~~~~~
MDVR PROTOCOL HELPERS
~~~~~~~~~~~~~~~
*/

// Assumption: that the date is formatted like: 20060102-150405
func GetDateFromMessage(message string) (time.Time, error) {
	msgArr := strings.Split(message, ";")
	// loop until we find string that we can gte date from
	for _, elem := range msgArr {
		// a bit hacky but should work fine
		if len(elem) == 15 && elem[8] == '-' {
			if len(elem) != 15 {
				return time.Time{}, fmt.Errorf("input string length should be 15 characters")
			}
			return ParseDVRFormatDateTime(elem)
		}
	}
	return time.Time{}, fmt.Errorf("couldn't find datetime string in: %v", message)
}

func ParseDVRFormatDateTime(str string) (time.Time, error) {
	// Parse the input string as a time.Time object
	parsedTime, err := time.Parse("20060102-150405", str)
	if err != nil {
		return time.Time{}, err
	}
	return parsedTime, nil
}

// get id from msg format: IDENTIFIER;1234;<-That's the id;xXxXxXxXxX;<CR>
func GetIdFromMessage(message *string) (*string, error) {
	msgSlice := strings.Split(*message, ";")
	for _, v := range msgSlice {
		_, err := strconv.Atoi(v)
		if err == nil {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("couldn't extract id from: %v", message)
}

// $FILE;V03;1221223799;1;1;20241114-170853;240;20241114-170810;60;4742224
// [0 $FILE];[1 protocol];[2 DeviceID];[3 SN];[4 camera];[5 RStart];[6 Rlen];[7 VStart];[8 Vlen];[9 filelen]<CR>[bytes]

// This is how we name the video, we can read file name in bash script and concat correctly
// 1221223799_20241114-170853_24_8_2 <-- example
func VideoDescriptorStr(strct FilePacketHeader) string {
	return strct.DeviceId + "_" + strct.RequestStartTime + "_" + strct.VideoLength + "_" + strct.SerialNum + "_" + strct.Channel
}

// $FILE;V03;1221223799;1;1;20241114-170853;240;20241114-170810;60;4742224
// [0 $FILE];[1 protocol];[2 DeviceID];[3 SN];[4 camera];[5 RStart];[6 Rlen];[7 VStart];[8 Vlen];[9 filelen]<CR>[bytes]

// parse a $FILE packet header
func ParseFilePacketHeader(str string, strct *FilePacketHeader) error {
	if str == "" || strct == nil {
		fmt.Errorf("string or struct passed to this function were nil")
	}
	splitStr := strings.Split(str, ";")
	if splitStr[0] != "$FILE" {
		fmt.Errorf("required $FILE packets")
	}
	strct.DeviceId = splitStr[2]
	strct.Channel = splitStr[4]
	strct.SerialNum = splitStr[3]
	strct.RequestStartTime = splitStr[5]
	strct.RequestLength = splitStr[6]
	strct.VideoStartTime = splitStr[7]
	strct.VideoLength = splitStr[8]
	return nil
}

// $VIDEO;85000;all;0;20181127-091000;86400<CR>
// [0 $VIDEO];[1 DeviceID];[2 type];[3 camera];[4 start];[5 time length]<CR>

// parse a $VIDEO message
func ParseVideoPacketHeader(str string, strct *VideoPacketHeader) error {
	if str == "" || strct == nil {
		fmt.Errorf("string or struct passed to this function were nil")
	}
	splitStr := strings.Split(str, ";")
	if splitStr[0] != "$VIDEO" {
		fmt.Errorf("required $VIDEO packets")
	}
	strct.DeviceId = splitStr[1]
	strct.RequestStartTime = splitStr[4]
	strct.RequestLength = splitStr[5]
	return nil
}

// helper to get req match string
func ParseReqMatchStringFromVideoPacketHeader(strct *VideoPacketHeader) (string, error) {
	if strct == nil {
		return "", fmt.Errorf("video packet hader was nil")
	}
	return strct.DeviceId + strct.RequestLength + strct.RequestStartTime, nil
}

// helper to get req match string
func ParseReqMatchStringFromVideoDescription(strct *VideoDescription) (string, error) {
	if strct == nil {
		return "", fmt.Errorf("video description was nil")
	}
	return strct.DeviceId + strct.RequestLength + strct.RequestStartTime, nil
}

/*
~~~~~~~~~~~~~~~
MONGODB HELPERS
~~~~~~~~~~~~~~~
*/

// connection to the mongodb instance
type MongoDBConnection struct {
	Logger *zap.Logger
	Client *mongo.Client // connection to the db
	Uri    string        // endpoint
	DbName string        // name of the database inside mongo
	Lock   sync.Mutex    // lock for the db connection
}

// constructor
func NewDBConnection(logger *zap.Logger, uri string, dbName string) (*MongoDBConnection, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	var result bson.M
	// ping db to check the connection
	err = client.Database(dbName).RunCommand(context.TODO(), bson.D{{"ping", 1}}).Decode(&result)
	if err != nil {
		return nil, err
	}
	logger.Info("database %v connected successfully", zap.String("dbName", dbName))
	return &MongoDBConnection{
		Logger: logger,
		Client: client,
		Uri:    uri,
		DbName: dbName,
	}, nil
}

/*
~~~~~~~~~~~~~~~
RABBITMQ HELPERS
~~~~~~~~~~~~~~~
*/

func SetupPipeFromBroker(exchangeName string, rabMqChan *amqp.Channel) (error, <-chan amqp.Delivery) {
	// declare exchange we are taking messages from.
	err := rabMqChan.ExchangeDeclare(
		exchangeName, // name
		"fanout",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("error piping messages from message broker %v", err), nil
	}

	// declare a queue onto the exchange above
	q, err := rabMqChan.QueueDeclare(
		"",    // name -
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("error piping messages from message broker %v", err), nil
	}

	// bind the queue onto the exchange
	err = rabMqChan.QueueBind(
		q.Name,       // queue name
		"",           // routing key
		exchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("error binding queue %v", err), nil
	}

	// get a golang channel we can read messages from.
	msgs, err := rabMqChan.Consume(
		q.Name, // name - WE DEPEND ON THE QUEUE BEING NAMED THE SAME AS THE EXCHANGE - ONLY ONE Q CONSUMING FROM THIS EXCHANGE
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return fmt.Errorf("error consuming from queue %v", err), nil
	}
	return nil, msgs
}

func SetupPipeToBroker(exchangeName string, rabMqChan *amqp.Channel) error {
	// declare the device message output exchange
	err := rabMqChan.ExchangeDeclare(
		exchangeName, // name
		"fanout",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("error declaring exchange  %v", err)
	}
	return nil
}
