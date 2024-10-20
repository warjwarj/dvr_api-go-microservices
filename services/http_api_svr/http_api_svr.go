package main

import (
	"io"
	"net"
	"net/http"

	"dvr_api-go-microservices/pkg/config"

	zap "go.uber.org/zap"
)

type HttpApiSvr struct {
	logger   *zap.Logger
	endpoint string // IP + port, ex: "192.168.1.77:9047"
}

func NewHttpSvr(logger *zap.Logger, endpoint string) (*HttpApiSvr, error) {
	// create the struct
	svr := HttpApiSvr{
		logger,
		endpoint,
	}
	return &svr, nil
}

// run the server
func (s *HttpApiSvr) Run() {
	// listen tcp
	l, err := net.Listen("tcp", s.endpoint)
	if err != nil {
		s.logger.Fatal("error listening on %v: %v", zap.String("s.endpoint", s.endpoint), zap.Error(err))
	} else {
		s.logger.Info("http server listening on: %v", zap.String("s.endpoint", s.endpoint))
	}

	// accept http on tcp port we've just ran
	HttpApiSvr := &http.Server{
		Handler: s,
	}
	err = HttpApiSvr.Serve(l)
	if err != nil {
		s.logger.Fatal("error serving http api server: %v", zap.Error(err))
	}
}

// serve the http API
func (s *HttpApiSvr) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Set the URL of the destination server
	destinationURL := "http://" + config.DBPROXY_SVR_ENDPOINT // Replace with actual URL

	// Create a new request to forward to the destination server
	req, err := http.NewRequest(http.MethodPost, destinationURL, r.Body)
	if err != nil {
		s.logger.Error("error creeating http request: %v", zap.Error(err))
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy the headers from the original request to the new request
	req.Header = r.Header.Clone()

	// Forward the request to the destination server
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("error forwarding request to dbproxy: %v", zap.Error(err))
		http.Error(w, "Failed to forward request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Set the status code and headers for the response
	w.WriteHeader(resp.StatusCode)
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Set(key, value)
		}
	}

	// Copy the response body from the destination server to the original response
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		s.logger.Error("failed to copy response body: %v", zap.Error(err))
		http.Error(w, "Failed to copy response body", http.StatusInternalServerError)
	}
}
