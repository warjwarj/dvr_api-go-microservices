module dvr_api-go-microservices/services/ws_api_svr

go 1.22.1

require (
	dvr_api-go-microservices/pkg/config v0.0.0
	dvr_api-go-microservices/pkg/utils v0.0.0
)

require (
	github.com/coder/websocket v1.8.12 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	nhooyr.io/websocket v1.8.17 // indirect
)

replace dvr_api-go-microservices/pkg/utils => ../../pkg/utils

replace dvr_api-go-microservices/pkg/config => ../../pkg/config
