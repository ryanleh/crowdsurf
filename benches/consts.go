package main

// Throughput cross-client batch sizes
var ks = []uint64{1, 8, 32, 64, 128, 256, 512, 1024, 2048}

// Batching parameters from paper (4GB database)
var bs = []uint64{8, 16, 24, 32, 40, 48, 56, 64}
var sizes = []uint64{68951263, 77048711, 72970119, 69291990, 66555742, 64504344, 62683952, 61071845}
var bits = []uint64{498, 446, 471, 496, 516, 533, 548, 563}

// dPIR parameters TODO: (need to be adjusted per-distribution)
//var avgCaseErrs = []float64{0.05, 0.1, 0.15, 0.2, 0.25, 0.3}
var avgCaseErrs = []float64{0.2}

// These are cutoffs for worst-case correctness error 0.99
var alpha = 0.99
var cutoffs = [][]uint64{
//    []uint64{46233834, 24472103, 13113389, 5963025,  1914628, 463198},
//    []uint64{43821335, 20930098, 8959012,  2774289,  575218,  150303},
//    []uint64{37161222, 16121708, 5998546,  1343345,  244341,  65709},
//    []uint64{44519049, 24807448, 13857931, 6879535,  2731351, 788018},
//    []uint64{44417597, 25862890, 15485749, 8756458,  4226279, 1545457},
//    []uint64{43838133, 26225191, 16305518, 9814302,  5244358, 2283905},
//    []uint64{43179877, 26324012, 16773949, 10482433, 6004522, 2905959},
//    []uint64{42464887, 26291277, 17106710, 11031655, 6683515, 3538762},

    // Avg-case = 0.2
    []uint64{5963025},
    []uint64{2774289},
    []uint64{1343345},
    []uint64{6879535},
    []uint64{8756458},
    []uint64{9814302},
    []uint64{10482433},
    []uint64{11031655},
}

// Instance parameters for LWE batching experiment to get 200MB state size
var batchRows = []uint64{25575, 25578, 25584, 25596, 25593, 25578, 25560, 25544}
var batchCols = []uint64{148283, 147603, 148314, 146186, 148232, 146269, 147146, 148233}
var batchPMods = []uint64{593, 593, 593, 593, 593, 593, 593, 593}