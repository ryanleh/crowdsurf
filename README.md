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
git submodule update --init`
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

## License

CrowdSurf is licensed under either of the following licenses, at your discretion.

 * Apache License Version 2.0 ([LICENSE-APACHE](LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
 * MIT license ([LICENSE-MIT](LICENSE-MIT) or http://opensource.org/licenses/MIT)

Unless you explicitly state otherwise, any contribution submitted for inclusion in Muse by you shall be dual licensed as above (as defined in the Apache v2 License), without any additional terms or conditions.
