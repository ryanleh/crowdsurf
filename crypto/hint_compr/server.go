package hint_compr

import (
	m "github.com/ryanleh/secure-inference/matrix"

    "unsafe"
)


type Server struct {
    *Socket
}

func NewServer(rows, cols uint64) *Server {
    server := &Server{ NewSocket(SERVER_SOCKET) }

    params := []uint32 { uint32(rows), uint32(cols) }
    server.SendUints(params)

    return server
}

func (s *Server) SetHint(data []m.Elem32) []byte {
    // Send the hint to the server
    ptr := unsafe.Pointer(unsafe.SliceData(data))
    data32 := unsafe.Slice((*uint32)(ptr), len(data))
    s.SendUints(data32)

    // Receive back public params
    return s.RecvBytes()
}

func (s *Server) SetQuery(queries []byte) {
    s.SendBytes(queries)
}

func (s *Server) Answer() []byte {
    s.SendBytes([]byte{1})
    return s.RecvBytes()
}

func (s *Server) Reset(rows, cols uint64) {
    // Send reset signal
    s.SendBytes([]byte{0, 1, 2, 3, 4, 5})
    
    params := []uint32 { uint32(rows), uint32(cols) }
    s.SendUints(params)
}
