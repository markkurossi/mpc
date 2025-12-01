# VOLE Summary  
(Current Implemented State)

This document describes the VOLE (Vector Oblivious Linear Evaluation) subsystem as it currently exists in the repository after stopping the Tiny-VOLE experiment and reverting to the original, proven implementation.

## ✔ Features Currently Implemented

### 1. **Packed / IKNP-compatible VOLE (non-silent)**
- Implements a vectorized VOLE using OT and PRG label expansion.
- Uses the standard IKNP extension if a real base OT instance is provided.
- If `oti == nil`, VOLE falls back to a **channel-based shim**, ensuring tests work without OT.
- Supports multi-element batches efficiently (`MulSender`, `MulReceiver`).

### 2. **Correctness Verified**
The following are tested and passing:
- `TestVOLEBasic` – core relation `u − r = x * y (mod p)`
- `TestPackedPathFallback` – correct behavior when IKNP is not active
- All triple generator integration tests

### 3. **Backwards Compatible**
- The API (`NewExt`, `Setup`, `MulSender`, `MulReceiver`) remains unchanged.
- Triple generation and SPDZ integration continue functioning normally.

### 4. **Performance Characteristics**
- This implementation is *not silent*, but fast enough for current needs.
- Requires a single small Y-vector communication per VOLE batch.
- Ideal for correctness and stability before moving into silent VOLE.

---

## ❌ Features *Not Yet Included* (rolled back)

### 1. **Silent VOLE / Tiny-VOLE**
- Fully removed pending future design.
- No codepaths rely on binary linear codes or expanded Δ-bit matrices.
- No untested or partially-correct VOLE logic is included.

### 2. **Large GF(2) matrix layer**
- Removed in favor of stable, originally-working implementation.

---

## ✔ Conclusion

The current VOLE code is stable, correct, test-covered, and ready for production usage with standard IKNP or shim networks. All unfinished experimental code was removed for safety.
