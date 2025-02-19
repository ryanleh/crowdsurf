package service

import (
	"log"
	"net"
	"net/rpc"
    "sync"
    "time"
)

import (
	m "github.com/ryanleh/secure-inference/matrix"
	"github.com/ryanleh/secure-inference/lhe"
	"github.com/ryanleh/secure-inference/crypto/hint_compr"
)

// TODO: Probably update this interface to have a single preprocess function
type Client struct {
    // PIR Client
	pirConn      *CountingIO
	pirRpcClient *rpc.Client
	pirClient    lhe.Client[m.Elem32]
    pirType      lhe.LHEType

    // Hint compression client
	hcConn      *CountingIO
	hcRpcClient *rpc.Client
    hcClient    *hint_compr.Client
}

func MakeClient(
    pirAddr, hcAddr string,
    lheType lhe.LHEType,
    rows, cols, pMod, bitsPer, batchSize uint64,
) *Client {
    RegisterTypes()
	
    // Connect to PIR server
	socket, err := net.Dial("tcp", pirAddr+":8728")
	if err != nil {
		log.Println("Error connecting to server")
		panic(err)
	}
    pirConn := NewCountingIO(socket)
    pirRpcClient := rpc.NewClient(pirConn)

	// Fetch hints / DB info / seeds from server
    request := PirInitRequest {
        Rows: rows,
        Cols: cols,
        PMod: pMod,
        BitsPer: bitsPer,
        BatchSize: batchSize,
    }
	var reply PirInitResponse
	err = pirRpcClient.Call("Server.ClientInitRPC", request, &reply)
	if err != nil {
		log.Println("Error initializing client")
		panic(err)
	}

    // Setup hint compression server if needed
    switch lheType {
    case lhe.Local:
        pirClient := &lhe.LocalClient[m.Elem32]{}
        pirClient.Init(reply.Params)
        return &Client{nil, nil, pirClient, lheType, nil, nil, nil}

    case lhe.Simple, lhe.SimpleHybrid:
        // Setup PIR client (take out the hint)
        pirClient := &lhe.SimpleClient[m.Elem32]{}
	    params := reply.Params.(*lhe.SimpleHint[m.Elem32])
        hint := params.Hint
        params.Hint = nil
        pirClient.Init(params)

        // Hint compression server if needed
        if hcAddr != "" {
            hcClient := hint_compr.NewClient(params.DBInfo.L, 2048, batchSize)

            // Connect to HC server
            socket, err := net.Dial("tcp", hcAddr+":8729")
            if err != nil {
                log.Println("Error connecting to HC server")
                panic(err)
            }
            hcConn := NewCountingIO(socket)
            hcRpcClient := rpc.NewClient(hcConn)

            // Send hint to server and receive back public params
            var reply HintInitResponse 
            err = hcRpcClient.Call("HCServer.ClientInitRPC", &HintInitRequest{hint}, &reply)
            if err != nil {
                log.Println("Error initializing client")
                panic(err)
            }
            hcClient.RecvParams(reply.Params)
            return &Client{pirConn, pirRpcClient, pirClient, lheType, hcConn, hcRpcClient, hcClient}
        } else {
            return &Client{pirConn, pirRpcClient, pirClient, lheType, nil, nil, nil}
        }
    
    default:
        panic("Unreachable")
    }
}

// Must call to free C++ memory
func (c *Client) Free() {
	c.pirClient.Free()
}


// Make a query
func (c *Client) Query(inputs []*m.Matrix[m.Elem32]) []lhe.Secret[m.Elem32] {
    keys, queries := c.pirClient.Query(inputs)

    switch c.pirType {
    case lhe.Local:
        return keys
    case lhe.Simple, lhe.SimpleHybrid:
        // Register the query on the PIR server
        err := c.pirRpcClient.Call("Server.QueryRPC", &PirQueryRequest{queries}, &PirQueryResponse{})
        if err != nil {
            log.Printf("Error making query")
            panic(err)
	    }

        // Register rotation keys on the HC server
        if c.hcConn != nil {
            secrets := make([]*m.Matrix[m.Elem32], len(keys))
            for i := range keys {
                key := keys[i].(*lhe.SimpleSecret[m.Elem32])
                secrets[i] = key.GetInner()
            }
            
            args := HintQueryRequest{
                Queries: c.hcClient.Query(secrets),
            }
            err = c.hcRpcClient.Call("HCServer.QueryRPC", &args, &HintQueryResponse{})
            if err != nil {
                log.Printf("Error making query")
                panic(err)
            }
        }
        return keys

    default:
        panic("Unreachable")
    }
}

// Get an answer
func (c *Client) Answer(keys []lhe.Secret[m.Elem32]) (float64, float64, []*m.Matrix[m.Elem32]) {
    switch c.pirType {
    case lhe.Local:
        return 0.0, 0.0, c.pirClient.Recover(keys, nil)
    case lhe.Simple, lhe.SimpleHybrid:
        // Fetch hint and PIR response in parallel
        var pTime, hTime float64
        var wg sync.WaitGroup
        wg.Add(2)

        // Get hint
        go func() {
            defer wg.Done()
            if c.hcConn != nil {
                start := time.Now()
                var reply HintAnswerResponse
                err := c.hcRpcClient.Call("HCServer.AnswerRPC", &HintAnswerRequest{}, &reply)
                if err != nil {
                    log.Printf("Error making query")
                    panic(err)
                }
                results := c.hcClient.Recover(reply.Answers, c.DBInfo().L)
                hTime = float64(time.Since(start).Milliseconds())

                // TODO: Better conversion
                for i := range keys {
                    key := keys[i].(*lhe.SimpleSecret[m.Elem32])
                    key.SetInner(m.NewFromRaw(results[i], uint64(len(results[i])), 1))
                }
            }
        }()

        // Get PIR answer
        var reply PirAnswerResponse
        go func() {
            defer wg.Done()

            start := time.Now()
            err := c.pirRpcClient.Call("Server.AnswerRPC", &PirAnswerRequest{}, &reply)
            if err != nil {
                log.Printf("Error making query")
                panic(err)
            }
            pTime = float64(time.Since(start).Milliseconds())
        }()
        wg.Wait()
       
        var result []*m.Matrix[m.Elem32]
        if c.hcConn != nil {
           result = c.pirClient.Recover(keys, reply.Answers)
        }
        return pTime, hTime, result
    default:
        panic("Unreachable")
    }
}

func (c *Client) GetBatchCapacity(hintTimeMs, pirTimeMs float64) uint64 {
    request := PirBatchRequest{hintTimeMs, pirTimeMs}
    var reply PirBatchResponse
    err := c.pirRpcClient.Call("Server.BatchCapacityRPC", request, &reply)
    if err != nil {
        log.Printf("Error asking for batch capacity")
        panic(err)
    }
    return reply.BatchCapacity
}


func (c *Client) GetConns() (*CountingIO, *CountingIO) {
	return c.pirConn, c.hcConn
}

func (c *Client) DBInfo() *lhe.DBInfo {
    return c.pirClient.DBInfo()
}
