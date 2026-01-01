# References

IKNP / ALSZ — Semi-honest OT extension
 - [Extending Oblivious Transfers Efficiently](https://www.iacr.org/archive/crypto2003/27290145/27290145.pdf)

ALSZ — Malicious-secure OT extension
 - [More Efficient Oblivious Transfer Extensions with Security for Malicious Adversaries](https://eprint.iacr.org/2015/061.pdf)

# Pseudocode

## Conventions / helper primitives

 - `k` = symmetric security parameter (e.g. 128).

 - `m` = number of extended OTs required.

 - `PRG(seed, outlen)` — expands `seed` (k-bit) to `outlen` bits (`prgAESCTR`)

 - `H(index, input)` — correlation-robust hash / random oracle used to
   mask sender messages.

 - `BASE_OT_Send(wires)` and `BASE_OT_Receive(choices)` — perform `k`
   base OTs, returning seeds/labels.

 - Pack rows as bit-rows of length `m`. Use `rowBytes =
   ceil(m/8)`. Pack `k` rows → `k × rowBytes` bytes.

 - For label construction: treat 128-bit column bits as 16-byte label blocks.

## IKNP / ALSZ — Semi-honest OT extension

``` text
PROTOCOL IKNP_Extend(m, k):

// Roles:
// Sender S has inputs { (x0_j, x1_j) } for j=1..m
// Receiver R has choice bits r = (r1..rm)

// Parameters:
// k: security parameter (number of base OTs / PRG seeds)
// rowBytes = ceil(m/8)

// 1) Base OTs (k OTs)
// S (receiver in base OTs) chooses random choice bits s[1..k]
// S runs baseOT.Receive(s) and obtains k seeds seedS[i] for i=1..k
// R (sender in base OTs) constructs pairs (seed0[i], seed1[i]) for i=1..k
// R runs baseOT.Send() for paris (seed0[i], seed1[i]) for i=1..k

// 2) Receiver expands seeds => T0 and T1 columns
for i in 1..k:
    T0[i] = PRG(seed0[i], rowBytes)        // rowBytes bytes => column i
    T1[i] = PRG(seed1[i], rowBytes)

// 3) Receiver computes U = T0 XOR T1 (k × rowBytes) and sends U to Sender
U = concat_i (T0[i] XOR T1[i])   // send as k rows
send U to Sender

// 4) Sender reconstructs Q columns using its choice bits s[i]
for i in 1..k:
    ti = PRG(seedR[i], rowBytes)      // PRG of seed received in base OT
    if s[i] == 1:
        qi = ti XOR U_row[i]          // U_row[i] is the i-th row block of U
    else:
        qi = ti                       // if s[i]==0

// Now think of the k columns qi as matrix Q (m rows, k columns).
// For each j in 1..m the j-th row q_j is a k-bit vector (packed in rowBytes).

// 5) Sender masks messages and sends to Receiver
for j in 1..m:
    qj = j-th row of Q   // k bits packed
    y0_j = x0_j XOR H(j, qj)
    y1_j = x1_j XOR H(j, qj XOR s_vector)
send all (y0_j, y1_j) to Receiver

// 6) Receiver recovers chosen messages
for j in 1..m:
    tj = j-th row of T0   // note: receiver stored T0 and T1
    // Because qj XOR s = tj (analysis), receiver can compute:
    x_rj = y_{rj}_j XOR H(j, tj)
return x_rj for j=1..m
```

## ALSZ — Malicious-secure OT extension

``` text
PROTOCOL ALSZ_Malicious_Extend(m, κ, ρ):
// κ = symmetric security param, ℓ = κ + ρ
// rowBytes = ceil(m/8)
// Notation: for i in 1..ℓ

// 0) Base OTs: perform ℓ base OTs
// Receiver (PR) chooses choices s[1..ℓ] and base-OT seeds k0_i,k1_i are delivered accordingly.
// After base OTs:
// - Sender PS holds k_si for each i (i.e., G(k_si) when expanded)
// - Receiver PR holds both k0_i and k1_i locally (they were sender in base-OT)

// 1) PR computes for each i:
//    ti = G(k0_i)      // expand
//    ui = ti XOR G(k1_i) XOR r0
// where r0 = r || τ  (append random τ ∈ {0,1}^κ to choice vector r)
// PR sends all ui to PS

// 2) Consistency check (main addition):
// For every unordered pair (α, β) ⊆ [ℓ], PR computes four hashes:
//    h_{0,0}^{α,β} = H( G(k0_α) XOR G(k0_β) )
//    h_{0,1}^{α,β} = H( G(k0_α) XOR G(k1_β) )
//    h_{1,0}^{α,β} = H( G(k1_α) XOR G(k0_β) )
//    h_{1,1}^{α,β} = H( G(k1_α) XOR G(k1_β) )
// PR sends the collection { H^{α,β} } to PS

// PS (sender) knows s_α, s_β and the corresponding G(k_{s_α}α), G(k_{s_β}β).
// PS checks for every pair (α,β):
//   (i)  h_{s_α,s_β}^{α,β} == H( G(k_{s_α}α) XOR G(k_{s_β}β) )
//   (ii) h_{s_α,s_β}^{α,β} == H( G(k_{s_α}α) XOR G(k_{s_β}β) XOR u_α XOR u_β )
//   (iii) u_α != u_β
// Abort if any check fails.

// Intuition: these checks force PR to use the same r0 across most columns; if PR cheats
// for many indices it will be detected except with probability ~2^{-ρ}.

// 3) If checks pass, continue with IKNP-style extension:
// For i in 1..ℓ:
//    PS computes q_i = (s_i · u_i) XOR G(k_{s_i}^i)   // as IKNP step
// Form matrix Q; for each row j (1..m):
//    PS sends y0_j = x0_j XOR H(j, q_j)
//    PS sends y1_j = x1_j XOR H(j, q_j XOR s_vector)
// PR recovers x_{rj} as y_{rj}_j XOR H(j, t_j) where t_j is PR's corresponding row

// 4) Output: PR obtains its chosen messages.

// Notes:
// - Use ℓ = κ + ρ so that even if PR learns up to ρ bits of s by cheating, plenty of entropy remains.
// - ALSZ proves that with these checks the protocol is secure against malicious receivers and
//   has a tight failure probability bounded by 2^{-ρ} (plus standard crypto assumptions).
```
