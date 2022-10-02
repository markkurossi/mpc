# BMR Protocol


```ascii
   k_{u}0      +-----\
   k_{u}1 --u--+      \
               |  f   +--w-- k_{w}0
   k_{v}0 --v--+      /      k_{w}1
   k_{v}1      +-----/
```

| u      | v      | w      |
|:------:|:------:|:------:|
| k_{u}0 | k_{v}0 | k_{w}0 |
| k_{u}0 | k_{v}1 | k_{w}0 |
| k_{u}1 | k_{v}0 | k_{w}0 |
| k_{u}1 | k_{v}1 | k_{w}1 |


In particular, all n parties choose their own random 0-labels and
random 1-labels on every wire, and the output wire labels are
encrypted separately under every single party’s input wire labels

Let `k^{i}_{u,b}` input-label that party P_i holds for value b{0,1} on
wire u.

Encryption

``` ascii
for g := 0...numGates {
  for j := 1...n {
    for i := 1...n {
      input ^= F2(k^i_{u,a}, k^i_{u,b}, g|j)
    }
    P_{j} = input|k^j_{w,g(a,b)}
  }
}
```

Offline phase: each party P_{i} locally computes:

``` ascii
F2(k^i_{u,a}, k^i_{v,b}, g|j)
```

for every a,b in {0,1} and j in [1...n]

Observe that this means that each party must carry out 4n double-key
PRF computations per gate.
