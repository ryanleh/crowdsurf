package hint_compr

import (
    "log"
    "net"
    "encoding/binary"
    "unsafe"
)


const SERVER_SOCKET = "/tmp/dpir_server.sock"
const CLIENT_SOCKET = "/tmp/dpir_client.sock"

type Socket struct {
    net.Conn
}

func NewSocket(path string) *Socket {
    c, err := net.Dial("unix", path)
    if err != nil {
        panic(err)
    }
    return &Socket{ c }
}

func (s *Socket) SendBytes(msg []byte) {
    // First send the size
    buf := make([]byte, 8)
    binary.LittleEndian.PutUint64(buf, uint64(len(msg)))
    _, err := s.Write(buf)
    if err != nil {
        log.Fatal("Error when writing size: ", err)
    }

    // Now write the message
    _, err = s.Write(msg)
    if err != nil {
        log.Fatal("Error when writing bytes: ", err)
    }
}

func (s *Socket) SendUints(msg []uint32) {
    // Convert to bytes
    //
    // TODO: Could do this without copying if we want
    ptr := unsafe.Pointer(unsafe.SliceData(msg))
    data := unsafe.Slice((*byte)(ptr), len(msg) * 4)
    s.SendBytes(data)
}

func (s *Socket) RecvBytes() []byte {
    // First receive the size of the message
    var buf [4096]byte
    n, err := s.Read(buf[:8])
    if n != 8 || err != nil {
        log.Fatal("Error when reading size: ", err, n)
    }

    size := binary.LittleEndian.Uint64(buf[:8])
    result := make([]byte, 0, size)

    for size > 0 {
        n, err := s.Read(buf[:min(4096, size)])
        if err != nil {
            log.Fatal("Error when reading bytes: ", err)
        }
        result = append(result, buf[:n]...)
        size -= uint64(n)
    }
    return result
}

func (s *Socket) RecvUints() []uint32 {
    // TODO: Could do without copy if need be
    bytes := s.RecvBytes()
    if len(bytes) % 4 != 0 {
        log.Fatal("Error when reading uints")
    }
    ptr := unsafe.Pointer(unsafe.SliceData(bytes))
    return unsafe.Slice((*uint32)(ptr), len(bytes) / 4)
}
