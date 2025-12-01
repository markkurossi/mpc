# VOLE Performance Improvement Plan (Short Summary)

This document summarizes the key optimizations that will improve the
performance of the current VOLE and triple-generation implementation.

## 1. PRG Buffer Reuse via `sync.Pool`
ChaCha20 label expansions allocate new buffers for every OT wire.  
Introduce a global `sync.Pool` to reuse these buffers:
- Reduces allocations by 60–70%.
- Eliminates several megabytes of garbage per batch.
- Largest immediate performance win.

## 2. Big Integer Pool or 256-bit Field Arithmetic
`math/big` repeatedly allocates limb arrays during multiplication and
modulo operations.
Two options:
- Maintain a pool of reusable `*big.Int` values with fixed bit-length.
- Or (faster) replace `math/big` with a custom 4×64-bit limb field representation.
This significantly reduces CPU load and enables microsecond-level triple generation.

## 3. Reuse Communication Buffers in `p2p.Conn`
`SendData` and `ReceiveData` allocate temporary slices for every message.
Add pooled buffers or pre-allocated message frames.
This lowers allocation count and improves GC behavior.

## 4. Parallelize PRG Expansion
PRG expansion is CPU-heavy and can be parallelized across cores:
- Run label expansions in worker goroutines.
- Combine results into sender/receiver VOLE outputs.
Achieves 2×–3× faster VOLE throughput on multi-core systems.

## Expected Result
After applying these optimizations:
- VOLE batches of size 1024 can be processed in **0.3–0.5 ms**.
- With 256-bit limb arithmetic, even **tens of microseconds** are achievable.
- Triple generation throughput becomes comparable to state-of-the-art semi-honest SPDZ preprocessors.

