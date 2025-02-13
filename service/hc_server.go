package service

import (
//	"errors"
	"log"
	"net"
	"net/rpc"
    "time"
)

import (
	"github.com/ryanleh/secure-inference/crypto/hint_compr"
)

type HCServer struct {
	*hint_compr.Server
	listener     net.Listener
}

// Create a new RPC server
func StartHCServer() *HCServer {
    server := &HCServer{}

	// Start RPC server
    RegisterTypes()
	rpcHandler := rpc.NewServer()
	rpcHandler.Register(server)
	l, err := net.Listen("tcp", ":8729")
	if err != nil {
		panic("Failed to start listener")
	}
	server.listener = l
	go func() {
		for {
			conn, err := server.listener.Accept()
			if err != nil {
				return
			}
			go rpcHandler.ServeConn(conn)
		}
	}()
	
    return server
}

// Shutdown the RPC server
func (s *HCServer) StopServer() {
	s.listener.Close()
}

// RPC called to initiatlize a new client for evaluation
func (s *HCServer) ClientInitRPC(args HintInitRequest, response *HintInitResponse) error {
	log.Printf("Got ClientInit RPC Call")

    // Start a new hint compression server
    // 
    // TODO: Kill old server if it exists
    s.Server = hint_compr.NewServer(args.Hint.Rows(), args.Hint.Cols())

	// Generate new client token + set client state
    start := time.Now()
    response.Params = s.SetHint(args.Hint.Data())
    log.Printf("Setting hint took: %0.2f", time.Since(start).Seconds())
	return nil
}

// RPC called to setup queries
func (s *HCServer) QueryRPC(args HintQueryRequest, response *HintQueryResponse) error {
	log.Printf("Got Query RPC Call")
    s.SetQuery(args.Queries)
	return nil
}

// RPC called to answer a batch of queries
func (s *HCServer) AnswerRPC(args HintAnswerRequest, response *HintAnswerResponse) error {
	log.Printf("Got Answer RPC Call")
	response.Answers = s.Answer()
	return nil
}
