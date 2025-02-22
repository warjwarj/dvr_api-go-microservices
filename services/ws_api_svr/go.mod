module dvr_api-go-microservices/services/ws_api_svr

go 1.23

require (
	dvr_api-go-microservices/pkg/config v0.0.0
	dvr_api-go-microservices/pkg/utils v0.0.0
	github.com/coder/websocket v1.8.12
	github.com/google/uuid v1.6.0
	github.com/rabbitmq/amqp091-go v1.10.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.mongodb.org/mongo-driver v1.17.1 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/text v0.17.0 // indirect
)

replace dvr_api-go-microservices/pkg/utils => ../../pkg/utils

replace dvr_api-go-microservices/pkg/config => ../../pkg/config
