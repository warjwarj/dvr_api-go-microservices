package utils

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

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
