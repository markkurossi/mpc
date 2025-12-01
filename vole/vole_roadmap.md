# VOLE Development Roadmap

This roadmap outlines the prioritized tasks and architectural improvements for VOLE and SPDZ-related cryptographic components. It focuses on correctness, performance, and future upgrades.

---

# Phase 0 — Current (Stable) Implementation

- Correct packed/IKNP VOLE  
- VOLE integrated with triple generator  
- Fallback shim ensures easy testing  
- No silent VOLE code active  
- All VOLE tests pass

---

# Phase 1 — Performance & Engineering Improvements  
*(Low Risk, High Immediate Value)*

### 1. **Batch Optimizations**
- Reuse PRG buffers and `[]big.Int` scratch space  
- Pool `[]byte` buffers for label expansion  
- Minimize per-element modular multiplications  
- Combine multiple VOLE calls in triple generation for better cache locality

### 2. **Parallelization**
- Allow `MulSender` / `MulReceiver` to run in goroutine pools  
- Overlap PRG expansion with OT communication

### 3. **Benchmark Suite**
Add:
```
go test -bench=. ./vole
go test -bench=. ./triplegen
```
Track:
- VOLE throughput (triples/sec)
- OT bandwidth usage
- IKNP expansion cost

---

# Phase 2 — Security Hardening  
*(Medium Risk, Medium Cost)*

### 1. **Malicious Security Extensions**
- KOS-style consistency checks for IKNP  
- VOLE correlation consistency proofs  
- Commitment to Δ-value

### 2. **Field Arithmetic Validation**
- Revisit conversion helpers between `Label` <-> `big.Int`  
- Ensure constant-time operations where required

### 3. **Deterministic PRG audit**
- Verify correct, secure use of ChaCha20 as PRG  
- Consider domain separation (e.g., label||wireIndex)

---

# Phase 3 — Silent VOLE (Revisit with Proper Foundations)  
*(High Risk, High Value — Requires Correct Design)*

Silent VOLE should only be implemented when the design is fully validated. Steps:

### 1. **Literature Review / Parameter Validation**
Choose an established, validated silent VOLE construction:
- Tiny-VOLE (Boyle–Kolesnikov–Peikert series)
- Silent OT extension (BCG+ 2019)
- KKRT-style correlated OT + lift

Pick recommended parameters (matrix dimensions, code structure).

### 2. **Introduce Binary Linear Code Layer**
- Add stable GF(2) matrices `H` (and possibly `Hᵀ`)
- Define field lift vector `G`
- Build deterministic generator for test reproducibility

### 3. **Sender/Receiver Mappings**
Implement:
```
r = Lift(H * s0)
u = Lift(H * s_sel)
```
with correct encoding of choices → y-values.

### 4. **Full Proof Test Suite**
- Property tests: `u − r == x*y (mod p)`  
- Δ-linearity tests  
- Correlation checks  
- Statistical testing of correlation quality

### 5. **Performance Benchmark**
Target:
- < 1ms for 10k VOLEs  
- Triples/sec competitive with modern Tiny-VOLE systems

---

# Phase 4 — SPDZ Integration Enhancements  
*(Medium Risk)*

### 1. **Full Beaver Triple Factory**
- Precompute large batches
- Integrate VOLE batching with triple batching

### 2. **Streaming Triples**
- Continuous pipeline for triples  
- Backpressure and rate control between peers

---

# Phase 5 — Optional Future Improvements  
*(Optional, depending on project direction)*

### 1. **Replace IKNP with KKRT or newer OT ext**
- Dramatically improves OT throughput  
- Reduces PRG cost  
- Built-in silent OTs possible

### 2. **Accelerate GF(p) arithmetic**
- Use 4×64 limb arithmetic  
- Integrate AVX2/AVX-512 SIMD acceleration  
- Precompute modular reduction constants

### 3. **WebAssembly support**
- If used in browser contexts  
- Needs portable big.Int and PRG operations

---

# Summary

The VOLE system is back to a clean, correct, stable foundation.  
The roadmap guides development toward:

- performance  
- security  
- correctness  
- future silent VOLE support  

This ensures future steps are built on stable ground without risk of introducing unsound cryptography.
