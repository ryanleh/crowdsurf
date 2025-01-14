<h1 align="center">CrowdSurf</h1>

__CrowdSurf__ is a Go library for **distributional private information retrieval**.

This library was initially developed as part of the paper *"Distributional Private Information Retrieval"*, and is released under the MIT License and the Apache v2 License (see [License](#license)).

**WARNING:** This is an academic proof-of-concept prototype, and in particular has not received careful code review. It should NOT be used for production use.

## Directory structure

This repository contains several folders that implement the different building blocks of CrowdSurf. The high-level structure of the repository is as follows.

* [`batching`](batching): Implements batch codes for handling batch PIR queries.

* [`benches`](benches): Implements benchmarks used in the paper

* [`crypto`](crypto): Implements core cryptographic components 

* [`external`](external): Contains external libraries needed for building and
  benchmarking

* [`lhe`](lhe): Implements a linearly homomorphic encryption scheme with preprocessing

* [`matrix`](matrix): Implements code for fast matrix multiplication

* [`service`](service): Implements code for running end-to-end experiments

## Build guide

* Pull all the git submodules
* Build Matrix library for GPU (see `matrix/gpu/gpu.go`)
* Build SEAL (see `crypto/rlwe/rlwe.go/`)
* Build GPU library (see `matrix/gpu/gpu.go`)

## License

CrowdSurf is licensed under either of the following licenses, at your discretion.

 * Apache License Version 2.0 ([LICENSE-APACHE](LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
 * MIT license ([LICENSE-MIT](LICENSE-MIT) or http://opensource.org/licenses/MIT)

Unless you explicitly state otherwise, any contribution submitted for inclusion in Muse by you shall be dual licensed as above (as defined in the Apache v2 License), without any additional terms or conditions.
