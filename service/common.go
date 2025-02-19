package service

import (
	m "github.com/ryanleh/secure-inference/matrix"
	"github.com/ryanleh/secure-inference/lhe"
    "encoding/gob"
)

// PIR Structs
type PirInitRequest struct{
    Rows       uint64 
    Cols       uint64 
    PMod       uint64 
    BitsPer    uint64 
    BatchSize  uint64 
}

type PirInitResponse struct {
    Params   lhe.Hint[m.Elem32] 
}

type PirQueryRequest struct {
	Queries []lhe.Query[m.Elem32]
}

type PirQueryResponse struct {}

type PirAnswerRequest struct{}

type PirAnswerResponse struct {
	Answers []lhe.Answer[m.Elem32]
}

type PirBatchRequest struct {
    HintTimeMs float64
    PirTimeMs float64
}

type PirBatchResponse struct {
    BatchCapacity uint64
}

// Hint Compression structs
type HintInitRequest struct{
    Hint *m.Matrix[m.Elem32]
}

type HintInitResponse struct {
    Params   []byte
}

type HintQueryRequest struct{
    Queries []byte
}

type HintQueryResponse struct {}

type HintAnswerRequest struct{}

type HintAnswerResponse struct {
    Answers []byte
}

// Register interface types
func RegisterTypes() {
    gob.Register(&lhe.LocalHint[m.Elem32]{})
    gob.Register(&lhe.Empty[m.Elem32]{})
    gob.Register(&lhe.SimpleHint[m.Elem32]{})
    gob.Register(&lhe.SimpleQuery[m.Elem32]{})
    gob.Register(&lhe.SimpleAnswer[m.Elem32]{})
}
