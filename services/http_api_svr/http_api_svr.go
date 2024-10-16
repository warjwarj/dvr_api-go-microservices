package main

import (
	"encoding/json"
	"io"
	"net"
	"net/http"

	"go.uber.org/zap"
)

type HttpApiSvr struct {
	logger   *zap.Logger
	endpoint string // IP + port, ex: "192.168.1.77:9047"
	dbc      *DBConnection
}

func NewHttpSvr(logger *zap.Logger, endpoint string, dbc *DBConnection) (*HttpApiSvr, error) {
	// create the struct
	svr := HttpApiSvr{
		logger,
		endpoint,
		dbc,
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
	// error handling
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !PROD {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		s.logger.Fatal("cors enabled on http server, disable in prod")
	}

	// read the bytes
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Unable to read request body", zap.Error(err))
		return
	}

	// unmarshal bytes into a struct we can work with
	var req ApiRequest_HTTP
	err = json.Unmarshal(body, &req)
	if err != nil {
		s.logger.Warn("failed to unmarshal json: \n%v", zap.String("body", string(body)))
	}

	// query the database
	res, err := s.dbc.QueryMsgHistory(req.Devices, req.Before, req.After)
	if err != nil {
		s.logger.Error("failed to query msg history: %v", zap.Error(err))
	}

	// marshal back into json
	bytes, err := json.Marshal(res)
	if err != nil {
		s.logger.Error("failed to marshal golang struct into json bytes: %v", zap.Error(err))
		return
	}

	// set the response header Content-Type to application/json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
