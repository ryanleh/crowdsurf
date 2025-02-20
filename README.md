<h1 align="center">CrowdSurf</h1>

__CrowdSurf__ is a Go library for **distributional private information retrieval**.

This library was initially developed as part of the paper *"Distributional Private Information Retrieval"*, and is released under the MIT License and the Apache v2 License (see [License](#license)).

**WARNING:** This is an academic proof-of-concept prototype, and in particular has not received careful code review. It should NOT be used for production use.

## Directory structure

This repository contains several folders that implement the different building blocks of CrowdSurf. The high-level structure of the repository is as follows.

* [`batching`](batching): Batch codes for handling batch PIR queries.

* [`benches`](benches): Benchmarks used in the paper

* [`crypto`](crypto): Core cryptographic components 

* [`external`](external): External libraries for building and benchmarking

* [`lhe`](lhe): A linearly homomorphic encryption scheme with preprocessing

* [`matrix`](matrix): Code for fast matrix multiplication

* [`service`](service): Scaffolding for running end-to-end experiments

## Build guide

First, install Go following the directions [here](https://go.dev/doc/install).

Next, pull in external submodules:
```
git submodule update --init
```
Install the required C++ libraries:
```
cmake -S external/SEAL/ -B external/SEAL/build -DCMAKE_INSTALL_PREFIX=$PWD/external/SEAL/build -DSEAL_USE_INTEL_HEXL=ON
cmake --build external/SEAL/build/ -j 8
cmake --install external/SEAL/build
cmake -S matrix/gpu/cuda -B matrix/gpu/cuda/build
cmake --build matrix/gpu/cuda/build/ -j 8
```

Finally, test that everything is working:
```
go test ./...
```

## Experiments

Below we outline how to recreate several of the key figures in our paper

### Figures 3, 4, and 9

The numbers for all of these figures can be obtained by running the `benches/run_benches.py`. In particular, to recreate Figure 3 run:
```
python3 run_benches.py -q
```
to recreate Figure 4 run:
```
python3 run_benches.py -p
```
(Note that if you add the `-r` flag, the benchmark is slightly more accurate
but will take a few hours to run).

Finally, to recreate Figure 9 run:
```
python3 run_benches.py -b
```

### Table 12

To get end-to-end estimates for the costs of CrowdSurf, you will need access to three machines: a client, two CPU-based machines, and a GPU-based machine.

One of the CPU-based machines will be hint compression. On this machine, run the following two commands in different shells:
```
cd external/hintless_pir/dpir
bazel run -c opt //dpir:dpir-server
```
and
```
cd service/bin/server
go run . hint
```

The remaining CPU + GPU machines will be for PIR. On these machines run:
```
cd service/bin/server
go run . "pir"
```
Finally, make sure that all machines accept traffic on ports 8728/8729 and run the following command:
```
cd service
python3 run_e2e.py --pir_gpu $GPU_IP --pir_cpu $CPU1_IP --hint $CPU2_IP
```
which will output the costs displayed in Table 12.

## License

CrowdSurf is licensed under either of the following licenses, at your discretion.

 * Apache License Version 2.0 ([LICENSE-APACHE](LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
 * MIT license ([LICENSE-MIT](LICENSE-MIT) or http://opensource.org/licenses/MIT)

Unless you explicitly state otherwise, any contribution submitted for inclusion in CrowdSurf by you shall be dual licensed as above (as defined in the Apache v2 License), without any additional terms or conditions.
