package hint_compr

import (
	m "github.com/ryanleh/secure-inference/matrix"

    "unsafe"
)

type Client struct {
    *Socket

    batchSize uint64
}

func NewClient(rows, cols, batchSize uint64) *Client {
    client := &Client{ NewSocket(CLIENT_SOCKET), batchSize }
    params := []uint32 { uint32(rows), uint32(cols), uint32(batchSize) }
    client.SendUints(params)
    return client
}

func (c *Client) RecvParams(params []byte) {
    c.SendBytes(params)
}

func (c *Client) Query(keys []*m.Matrix[m.Elem32]) []byte {
    if uint64(len(keys)) != c.batchSize {
        panic("Invalid number of keys")
    }
    for i := range keys {
        ptr := unsafe.Pointer(unsafe.SliceData(keys[i].Data()))
        data32 := unsafe.Slice((*uint32)(ptr), keys[i].Size())
        c.SendUints(data32)
    }

    // Receive back queries
    return c.RecvBytes()
}

func (c *Client) Recover(answer []byte, rows uint64) [][]m.Elem32 {
    // TODO: Copy for now
    c.SendBytes(answer)
   
    results := make([][]m.Elem32, c.batchSize)
    for i := range c.batchSize {
        result := c.RecvUints()
        results[i] = make([]m.Elem32, rows)
        for j := range rows {
            results[i][j] = (m.Elem32)(result[j])
        }
    }

    return results
}
