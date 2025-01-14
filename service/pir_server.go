package service

import (
	"log"
	"net"
	"net/rpc"
)

import (
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
)

type Server struct {
	lhe.Server[m.Elem32]
	listener     net.Listener

    queries []lhe.Query[m.Elem32]
}

// Create a new RPC server
func StartServer(pirServer lhe.Server[m.Elem32]) *Server {
    server := &Server{pirServer, nil, nil}

	// Start RPC server
    RegisterTypes()
	rpcHandler := rpc.NewServer()
	rpcHandler.Register(server)
	l, err := net.Listen("tcp", ":8728")
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
func (s *Server) StopServer() {
	s.listener.Close()
	s.Free()
}

// RPC called to initiatlize a new client for evaluation
func (s *Server) ClientInitRPC(args PirInitRequest, response *PirInitResponse) error {
	log.Printf("Got ClientInit RPC Call")

	// Generate new client token + set client state
    response.Params = s.Hint()
	return nil
}

// RPC called to register a query
func (s *Server) QueryRPC(args PirQueryRequest, response *PirQueryResponse) error {
	log.Printf("Got Query RPC Call")
    s.queries = args.Queries

    totalSize := uint64(0)
    for _, query := range s.queries {
        totalSize += query.Size()
    }
    log.Printf("Query storage (%d) size: %0.2f KB", len(s.queries), float64(totalSize) / 1024.0)

	return nil
}

// RPC called to answer a query
func (s *Server) AnswerRPC(args PirAnswerRequest, response *PirAnswerResponse) error {
	log.Printf("Got Answer RPC Call")
	response.Answers = s.Answer(s.queries)
	return nil
}

func (s *Server) DBInfo() *lhe.DBInfo {
    return s.DB().Info
}
