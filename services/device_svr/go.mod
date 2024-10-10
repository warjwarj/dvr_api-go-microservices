module dvr_api-go-microservices/services/device_svr

go 1.22.1

require (
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	dvr_api-go-microservices/pkg/utils v0.0.0
	dvr_api-go-microservices/pkg/config v0.0.0
)

replace dvr_api-go-microservices/pkg/utils => ../../pkg/utils
replace dvr_api-go-microservices/pkg/config => ../../pkg/config