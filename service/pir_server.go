package service

import (
	"log"
    "math"
	"net"
	"net/rpc"
    "time"
)

import (
	"github.com/ryanleh/secure-inference/crypto"
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
)

var key = rand.PRGKey([16]byte{
	100, 121, 60, 254, 76, 111, 7, 102, 199, 220, 220, 5, 95, 174, 252, 221,
})

type Server struct {
	lhe.Server[m.Elem32]
	listener     net.Listener

    queries []lhe.Query[m.Elem32]

    // Metrics for keeping track of batch capacity
    totalTime time.Duration
    numIters uint64
}

func randInstance(rows, cols, pMod, bitsPer uint64) lhe.Server[m.Elem32] {
	// Generate random matrix
	prg := rand.NewBufPRG(rand.NewPRG(&key))
	numLimbs := uint64(math.Ceil(float64(bitsPer) / 32.0))
	matrix := m.Rand[m.Elem32](prg, rows*numLimbs, cols, 0)

	// Build server objects
	ctx := crypto.NewContext[m.Elem32](m.Elem32(0).Bitlen(), cols, pMod)
	//return lhe.MakeSimpleServer[m.Elem32](matrix, bitsPer, ctx, &key, lhe.Hybrid, true, true)

    // TODO: We don't use hybrid mode for benchmarks here since
    // a) currently our code has the server modulus switch which is wrong (and adds to eval time)
    // b) we aren't benchmarking encryption time here since it's a preprocessing cost
	return lhe.MakeSimpleServer[m.Elem32](matrix, bitsPer, ctx, &key, lhe.None, true, true)
}

// Create a new RPC server
func StartServer() *Server {
    server := &Server{totalTime: time.Duration(0)}

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

    // Configure a new PIR server based on the received params
    s.Server = randInstance(args.Rows, args.Cols, args.PMod, args.BitsPer)
    s.Server.SetBatch(args.BatchSize)
    s.totalTime = time.Duration(0)
    s.numIters = 0
    log.Printf("ClientInit: Configured new PIR server")
	
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

    start := time.Now()
	response.Answers = s.Answer(s.queries)
    s.totalTime += time.Since(start)
    s.numIters += 1

	return nil
}

// RPC called to get the batch capacity of the PIR server
func (s *Server) BatchCapacityRPC(args PirBatchRequest, response *PirBatchResponse) error {
   
    // Compute average communication latency for client
    avgPirComputeTime := float64(s.totalTime.Milliseconds()) / float64(s.numIters)
    pirCommTime := args.PirTimeMs - avgPirComputeTime

    // Our total time to batch responses is the latency from the hint
    // compression - the communication time of PIR  
    allowedTime := args.HintTimeMs - pirCommTime

    // Try batches of increasing size, until the latency for the batch reaches
    // the average time we took to answer

    // Get the batch granularity in 100s, then 10s, etc.
    //
    // TODO: Better search mechanism here
    batchSize := uint64(100)
    bestGuess := batchSize
    hundreds := false
    tens := false

    // Setup PIR client and make a single query that we copy
    pirClient := &lhe.SimpleClient[m.Elem32]{}
    params := s.Hint().(*lhe.SimpleHint[m.Elem32])
    params.Hint = nil // Take out the hint to save memory
    pirClient.Init(params)
    defer pirClient.Free()

    for {
        log.Println("Trying batch size ", batchSize)
        _, queries := pirClient.DummyQuery(batchSize)
        s.SetBatch(batchSize)
        
        start := time.Now()
        iters := 5
        for range iters {
            s.Answer(queries)
        }
        elapsed := float64(time.Since(start).Milliseconds()) / float64(iters)

        if elapsed < allowedTime {
            bestGuess = batchSize
            if !hundreds {
                batchSize += 100
            } else if !tens {
                batchSize += 10
            } else {
                batchSize += 1
            }
        } else {
            if !hundreds {
                batchSize -= 90
                hundreds = true
            } else if !tens {
                batchSize -= 9
                tens = true 
            } else {
                response.BatchCapacity = batchSize
                break
            }
        }
    }
    response.BatchCapacity = bestGuess

    return nil
}


func (s *Server) DBInfo() *lhe.DBInfo {
    return s.DB().Info
}
