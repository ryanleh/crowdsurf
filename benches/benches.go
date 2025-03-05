package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"testing"

	"github.com/ryanleh/secure-inference/batching"
	"github.com/ryanleh/secure-inference/batching/pbc"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
)

// TODO: Make this separate binaries
var rows *uint64
var cols *uint64
var pMod *uint64
var bitsPer *uint64
var batchSize *uint64
var cutoff *uint64
var mode lhe.Mode
var hashMode pbc.Mode
var packing batching.Packing

//// Plain LWE experiments
//var bs = []uint64{1}
//var sizes = []uint64{68951263}
//var avgCase = 1.0
//var cutoffs = [][]uint64{{0, 68951262}}
//var bits = []uint64{498}
//var loads = [][]uint64{{1}}
//var types = []dpir.PirType{dpir.Simple}

func runBench[T m.Elem](benchType string) {
	// Run benchmark
	switch benchType {
	case "query":
		result := benchmarkQuery[T]()

		// Print Results
		avgTimeSec := float64(result.T.Microseconds()) / float64(result.N)
		fmt.Printf("%d iters\n", result.N)
		fmt.Printf(
			"Avg. Latency(%d x %d, %d, %b): %0.2fµs\n",
			*rows, *cols, *pMod, mode, avgTimeSec,
		)

	case "throughput":
		benchmarkThroughput[T]()
	case "preprocessing":
		result := benchmarkPreprocessing[T]()

		// Print Results
		avgTimeSec := result.T.Seconds() / float64(result.N)
		fmt.Printf("%d iters\n", result.N)
		fmt.Printf(
			"Avg. Latency(%d x %d, %d, %b): %0.2fs\n",
			*rows, *cols, *pMod, mode, avgTimeSec,
		)
	case "pbc":
		benchmarkPBC[T]()

	case "dpir":
		benchmarkDPIR[T]()
	
    case "size":
        computeSizes()

	default:
        panic(fmt.Sprintf("Invalid bench name %s", benchType))
	}
}

func main() {
	// Required to call this before flag.Parse for testing flags to work
	testing.Init()
	rows = flag.Uint64("rows", 4096, "# of rows")
	cols = flag.Uint64("cols", 4096, "# of cols")
	logQ := flag.Int("q", 32, "log q (32 / 64)")
	pMod = flag.Uint64("p", 1<<9, "plaintext modulus")
	bitsPer = flag.Uint64("bits", 9, "bits per DB element")
	benchType := flag.String("bench", "throughput", "(throughput/preprocessing/pbc/dpir)")
	modeType := flag.String("mode", "hybrid", "(hybrid/none)")
	hashModeType := flag.String("hash", "cuckoo", "(cuckoo/hash)")
	packingType := flag.String("packing", "balanced", "(balanced/comm/storage)")
	batchSize = flag.Uint64("batch", 1, "batch size")
	cutoff = flag.Uint64("cutoff", 0, "dPIR cutoff")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")
	flag.Parse()

	if *modeType == "none" {
		mode = lhe.None
	} else {
		mode = lhe.Hybrid
	}

	if *hashModeType == "hash" {
		hashMode = pbc.Hash
	} else {
		hashMode = pbc.Cuckoo
	}

	if *packingType == "comm" {
		packing = batching.Comm
	} else if *packingType == "storage" {
		packing = batching.Storage
	} else {
		packing = batching.Balanced
	}

	switch *logQ {
	case 32:
		runBench[m.Elem32](*benchType)
	case 64:
		runBench[m.Elem64](*benchType)
	default:
		panic("q must be in (32 / 64)")
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}
