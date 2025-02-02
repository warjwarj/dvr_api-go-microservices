package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	config "dvr_api-go-microservices/pkg/config"
	utils "dvr_api-go-microservices/pkg/utils" // our shared utils package

	amqp "github.com/rabbitmq/amqp091-go" // rabbitmq import
	zap "go.uber.org/zap"
)

type CamSvr struct {
	logger              *zap.Logger
	endpoint            string               // IP + port, ex: "192.168.1.77:9047"
	sockOpBufSize       int                  // how much memory do we give each connection to perform send/recv operations
	sockOpBufNum        int                  // how many buffers should we provision for the server (a lot)
	sockOpBufStack      utils.Stack[*[]byte] // memory region we give each conn to so send/recv
	rabbitmqAmqpChannel *amqp.Channel        // tcp connection with rabbitmq server
	svrMsgBufChan       chan utils.VideoDescription
}

func NewCamSvr(
	logger *zap.Logger,
	endpoint string,
	bufSize int,
	bufNum int,
	rabbitmqAmqpChannel *amqp.Channel,
) (*CamSvr, error) {
	svr := CamSvr{
		logger,
		endpoint,
		bufSize,
		bufNum,
		utils.Stack[*[]byte]{},
		rabbitmqAmqpChannel,
		make(chan utils.VideoDescription)}

	// init the stack we use to store buffers
	svr.sockOpBufStack.Init()

	// create and store the buffers
	for i := 0; i < svr.sockOpBufNum; i++ {
		buf := make([]byte, svr.sockOpBufSize)
		svr.sockOpBufStack.Push(&buf)
	}
	return &svr, nil
}

// run the server, blocking
func (s *CamSvr) Run() {
	// start server listening for connections
	ln, err := net.Listen("tcp", s.endpoint)
	if err != nil {
		s.logger.Fatal("error listening on %v: %v", zap.String("s.endpoint", s.endpoint), zap.Error(err))
	} else {
		s.logger.Info("cam server listening on: %v", zap.String("s.endpoint", s.endpoint))
	}

	// start the output and input from the message broker
	go s.pipeVideoDescriptionsToBroker(config.VIDEO_DESCRIPTION_EXCHANGE)

	// start accepting device connections
	for {
		c, err := ln.Accept()
		if err != nil {
			s.logger.Info("error accepting websocket connection: %v", zap.Error(err))
			continue
		}
		s.logger.Info("connection accepted on cam svr...")
		go func() {
			err := s.deviceConnLoop(c)
			if err != nil {
				s.logger.Error("error in cam server connection loop: %v", zap.Error(err))
			}
			c.Close()
		}()
	}
}

// TODO implement io.Reader
// receive video
func (s *CamSvr) deviceConnLoop(conn net.Conn) error {
	// close connection upon exit
	defer conn.Close()

	// Create the folder in the current directory
	saveLocationRoot := "/var/tmp/"
	// Create the folder in the current directory
	folderPath := saveLocationRoot + utils.GenerateRandomString(10)
	err := os.Mkdir(folderPath, 0777) // 0755 gives read, write, and execute permissions for the owner, and read and execute for others.
	if err != nil {
		return fmt.Errorf("Error creating folder:", err)
	}

	// Buffers
	frameHeaderBuf := make([]byte, 32)
	framePayloadHeaderBuf := make([]byte, 128)
	framePayloadBodyBuf := make([]byte, 4194304)

	receivedVideoLog := make(map[string]utils.VideoDescription)

	for {

		// loop variable to kep track of packet metadata
		var filePacketHeader utils.FilePacketHeader

		// Read the frame header
		if _, err := io.ReadFull(conn, frameHeaderBuf); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("Failed to read raw header: %v", err)
		}

		// Check for $FILEEND signal
		if strings.HasPrefix(string(frameHeaderBuf), "$FILEEND") {
			break
		}

		// get payload header length
		payloadHeaderLenBuf := frameHeaderBuf[14:20]
		payloadHeaderLen, err := strconv.ParseInt(strings.TrimPrefix(string(payloadHeaderLenBuf), "0x"), 16, 64)
		if err != nil {
			return fmt.Errorf("failed to parse payload header length: %v", err)
		}

		// get payload body length
		payloadBodyLenBuf := frameHeaderBuf[21:31]
		payloadBodyLen, err := strconv.ParseInt(strings.TrimPrefix(string(payloadBodyLenBuf), "0x"), 16, 64)
		if err != nil {
			return fmt.Errorf("failed to parse payload body length: %v", err)
		}

		// read payload header
		framePayloadHeaderBuf = make([]byte, payloadHeaderLen)
		if _, err = io.ReadFull(conn, framePayloadHeaderBuf); err != nil {
			return fmt.Errorf("failed to read payload header: %v", err)
		}
		// read vals from header into vid desc
		if utils.ParseFilePacketHeader(string(framePayloadHeaderBuf), &filePacketHeader); err != nil {
			return err
		}
		// create our video descriptor string
		videoDescriptorStr := utils.VideoDescriptorStr(filePacketHeader)

		// read payload body
		framePayloadBodyBuf = make([]byte, payloadBodyLen)
		if _, err = io.ReadFull(conn, framePayloadBodyBuf); err != nil {
			return fmt.Errorf("failed to read payload body: %v", err)
		}

		// extract video length as an int
		vidLengthInt, err := strconv.Atoi(filePacketHeader.VideoLength)
		if err != nil {
			return fmt.Errorf("Couldn't extract video length from file packet header")
		}

		// if we've not received video from this channel before then instantiate an entry for it in the map
		if vidDesc, exists := receivedVideoLog[filePacketHeader.Channel]; !exists {
			// init struct describing concatatated video
			receivedVideoLog[filePacketHeader.Channel] = utils.VideoDescription{
				DeviceId:         filePacketHeader.DeviceId,
				Channel:          filePacketHeader.Channel,
				RequestStartTime: filePacketHeader.RequestStartTime,
				VideoLength:      vidLengthInt,
			}
		} else {
			// else we have received video from this channel before, so just increase the video length tally
			vidDesc.VideoLength += vidLengthInt
			receivedVideoLog[filePacketHeader.Channel] = vidDesc
		}

		// write vid data to file named for packet index
		filepath := folderPath + "/" + videoDescriptorStr
		if err = os.WriteFile(filepath, framePayloadBodyBuf, 0777); err != nil {
			return fmt.Errorf("failed to write to file")
		}
		fmt.Println("Read and saved file: ", string(framePayloadHeaderBuf))
	}

	// define the file concat command
	convertCmd := exec.Command("./convert.sh", folderPath)
	concatCmd := exec.Command("./concat.sh", folderPath)

	// Set the script's output to be shown in the Go program's output
	convertCmd.Stdout = os.Stdout
	concatCmd.Stdout = os.Stdout
	convertCmd.Stderr = os.Stderr
	concatCmd.Stderr = os.Stderr

	// Run the shell scripts
	if err := convertCmd.Run(); err != nil {
		return fmt.Errorf("error running shell script convert.sh: %v", err)
	}
	if err := concatCmd.Run(); err != nil {
		return fmt.Errorf("error running shell script concat.sh: %v", err)
	}
	return nil
}

// publish device messages to broker
func (s *CamSvr) pipeVideoDescriptionsToBroker(exchangeName string) {

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
