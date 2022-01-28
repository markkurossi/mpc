# Benchmarks and tests

## Running benchmark: 32-bit RSA encryption (64-bit modp)

```
Circuit: #gates=7366376 (XOR=3146111 AND=3133757 OR=1032350 INV=54158)
┏━━━━━━━━┳━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op     ┃          Time ┃      % ┃  Xfer ┃
┣━━━━━━━━╋━━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━┫
┃ Wait   ┃  10.72622192s ┃ 75.33% ┃       ┃
┃ Recv   ┃  2.501608027s ┃ 17.57% ┃ 364MB ┃
┃ Inputs ┃  232.257386ms ┃  1.63% ┃  41kB ┃
┃ Eval   ┃  778.170361ms ┃  5.47% ┃       ┃
┃ Result ┃     158.838µs ┃  0.00% ┃   1kB ┃
┃ Total  ┃ 14.238416532s ┃        ┃       ┃
┗━━━━━━━━┻━━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━┛
```

Optimized full-adder:

```
Circuit: #gates=7366376 (XOR=5210811 AND=2101407 OR=0 INV=54158)
┏━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op     ┃         Time ┃      % ┃  Xfer ┃
┣━━━━━━━━╋━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━┫
┃ Wait   ┃   7.9561241s ┃ 76.09% ┃       ┃
┃ Recv   ┃ 1.660386022s ┃ 15.88% ┃ 199MB ┃
┃ Inputs ┃ 232.583145ms ┃  2.22% ┃  41kB ┃
┃ Eval   ┃ 606.479521ms ┃  5.80% ┃       ┃
┃ Result ┃    304.202µs ┃  0.00% ┃   1kB ┃
┃ Total  ┃ 10.45587699s ┃        ┃       ┃
┗━━━━━━━━┻━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━┛
```

Karatsuba multiplication algorithm:

```
Circuit: #gates=6822632 (XOR=4828475 XNOR=58368 AND=1874719 OR=0 INV=61070)
┏━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op     ┃         Time ┃      % ┃  Xfer ┃
┣━━━━━━━━╋━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━┫
┃ Wait   ┃ 6.672590283s ┃ 75.99% ┃       ┃
┃ Recv   ┃ 1.320621876s ┃ 15.04% ┃ 179MB ┃
┃ Inputs ┃ 231.659961ms ┃  2.64% ┃  41kB ┃
┃ Eval   ┃ 555.192105ms ┃  6.32% ┃       ┃
┃ Result ┃    298.855µs ┃  0.00% ┃   1kB ┃
┃ Total  ┃  8.78036308s ┃        ┃       ┃
┗━━━━━━━━┻━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━┛
```

Optimized INV-gates:

```
Circuit: #gates=6769820 (XOR=4836732 XNOR=58368 AND=1874719 OR=0 INV=1)
┏━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op     ┃         Time ┃      % ┃  Xfer ┃
┣━━━━━━━━╋━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━┫
┃ Wait   ┃ 6.338729612s ┃ 75.20% ┃       ┃
┃ Recv   ┃ 1.352880139s ┃ 16.05% ┃ 177MB ┃
┃ Inputs ┃  227.12815ms ┃  2.69% ┃  41kB ┃
┃ Eval   ┃ 509.574258ms ┃  6.05% ┃       ┃
┃ Result ┃    344.425µs ┃  0.00% ┃   1kB ┃
┃ Total  ┃ 8.428656584s ┃        ┃       ┃
┗━━━━━━━━┻━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━┛
```

Labels by value:

```
Circuit: #gates=6717340 (XOR=4787324 XNOR=108545 AND=1821471 OR=0 INV=0)
┏━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op     ┃         Time ┃      % ┃  Xfer ┃
┣━━━━━━━━╋━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━┫
┃ Wait   ┃ 6.117743762s ┃ 77.25% ┃       ┃
┃ Recv   ┃ 1.196140342s ┃ 15.10% ┃ 172MB ┃
┃ Inputs ┃ 236.647371ms ┃  2.99% ┃  41kB ┃
┃ Eval   ┃ 368.944904ms ┃  4.66% ┃       ┃
┃ Result ┃    347.483µs ┃  0.00% ┃   1kB ┃
┃ Total  ┃ 7.919823862s ┃        ┃       ┃
┗━━━━━━━━┻━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━┛
```

Labels by value in protocol, garbler, and evaluator:

```
Circuit: #gates=6717340 (XOR=4787324 XNOR=108545 AND=1821471 OR=0 INV=0)
┏━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op     ┃         Time ┃      % ┃  Xfer ┃
┣━━━━━━━━╋━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━┫
┃ Wait   ┃ 5.076677475s ┃ 78.67% ┃       ┃
┃ Recv   ┃ 940.758975ms ┃ 14.58% ┃ 143MB ┃
┃ Inputs ┃ 229.741398ms ┃  3.56% ┃  41kB ┃
┃ Eval   ┃ 205.513944ms ┃  3.18% ┃       ┃
┃ Result ┃    185.197µs ┃  0.00% ┃   1kB ┃
┃ Total  ┃ 6.452876989s ┃        ┃       ┃
┗━━━━━━━━┻━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━┛
```

Gate wires by value in garbler:

```
Circuit: #gates=6717340 (XOR=4787324 XNOR=108545 AND=1821471 OR=0 INV=0)
┏━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op     ┃         Time ┃      % ┃  Xfer ┃
┣━━━━━━━━╋━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━┫
┃ Wait   ┃  4.37061338s ┃ 76.48% ┃       ┃
┃ Recv   ┃  962.20669ms ┃ 16.84% ┃ 143MB ┃
┃ Inputs ┃ 229.360283ms ┃  4.01% ┃  41kB ┃
┃ Eval   ┃ 152.258636ms ┃  2.66% ┃       ┃
┃ Result ┃    162.316µs ┃  0.00% ┃   1kB ┃
┃ Total  ┃ 5.714601305s ┃        ┃       ┃
┗━━━━━━━━┻━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━┛
```

Garbler keeping wires in an array instead of map:

```
Circuit: #gates=6717340 (XOR=4787324 XNOR=108545 AND=1821471 OR=0 INV=0)
┏━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op     ┃         Time ┃      % ┃  Xfer ┃
┣━━━━━━━━╋━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━┫
┃ Wait   ┃ 1.710655914s ┃ 53.63% ┃       ┃
┃ Recv   ┃ 1.088403425s ┃ 34.12% ┃ 143MB ┃
┃ Inputs ┃ 243.393526ms ┃  7.63% ┃  41kB ┃
┃ Eval   ┃ 146.879726ms ┃  4.61% ┃       ┃
┃ Result ┃      222.8µs ┃  0.01% ┃   1kB ┃
┃ Total  ┃ 3.189555391s ┃        ┃       ┃
┗━━━━━━━━┻━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━┛
```

Pruning dead gates:

```
Circuit: #gates=5972956 (XOR=4315452 XNOR=53761 AND=1603743 OR=0 INV=0)
┏━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op     ┃         Time ┃      % ┃  Xfer ┃
┣━━━━━━━━╋━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━┫
┃ Wait   ┃  1.28140619s ┃ 55.30% ┃       ┃
┃ Recv   ┃ 676.432166ms ┃ 29.19% ┃ 126MB ┃
┃ Inputs ┃ 229.527559ms ┃  9.91% ┃  41kB ┃
┃ Eval   ┃ 129.623668ms ┃  5.59% ┃       ┃
┃ Result ┃    203.248µs ┃  0.01% ┃   1kB ┃
┃ Total  ┃ 2.317192831s ┃        ┃       ┃
┗━━━━━━━━┻━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━┛
```

Optimized garbling:

```
Circuit: #gates=5972956 (XOR=4315452 XNOR=53761 AND=1603743 OR=0 INV=0)
┏━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op     ┃         Time ┃      % ┃  Xfer ┃
┣━━━━━━━━╋━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━┫
┃ Wait   ┃ 700.031233ms ┃ 38.57% ┃       ┃
┃ Recv   ┃ 706.339086ms ┃ 38.92% ┃ 126MB ┃
┃ Inputs ┃ 233.615365ms ┃ 12.87% ┃  41kB ┃
┃ Eval   ┃  174.84741ms ┃  9.63% ┃       ┃
┃ Result ┃    215.733µs ┃  0.01% ┃   1kB ┃
┃ Total  ┃ 1.815048827s ┃        ┃       ┃
┗━━━━━━━━┻━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━┛
```

Optimized dynamic memory allocations from garbling:

```
Circuit: #gates=5972956 (XOR=4315452 XNOR=53761 AND=1603743 OR=0 INV=0) #w=5973116
┏━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op     ┃         Time ┃      % ┃  Xfer ┃
┡━━━━━━━━╇━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━━┩
│ Wait   │ 574.859679ms │ 35.11% │       │
│ Recv   │ 678.520719ms │ 41.44% │ 126MB │
│ Inputs │  274.49709ms │ 16.77% │  41kB │
│ Eval   │   109.1673ms │  6.67% │       │
│ Result │    158.416µs │  0.01% │   1kB │
│ Total  │ 1.637203204s │        │ 126MB │
└────────┴──────────────┴────────┴───────┘
```

Half And optimization:

```
Circuit: #gates=5972956 (XOR=4315452 XNOR=53761 AND=1603743 OR=0 INV=0) #w=5973116
┏━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━┓
┃ Op     ┃         Time ┃      % ┃ Xfer ┃
┡━━━━━━━━╇━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━┩
│ Wait   │ 653.959737ms │ 50.33% │      │
│ Recv   │ 218.966485ms │ 16.85% │ 75MB │
│ Inputs │  255.00393ms │ 19.63% │ 41kB │
│ Eval   │ 171.306681ms │ 13.18% │      │
│ Result │    131.077µs │  0.01% │  1kB │
│ Total  │  1.29936791s │        │ 75MB │
└────────┴──────────────┴────────┴──────┘
```

Circuit constant propagation:

```
Circuit: #gates=5539148 (XOR=3996414 XNOR=48825 AND=1493909 OR=0 INV=0) #w=5539308
┏━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━┓
┃ Op     ┃         Time ┃      % ┃ Xfer ┃
┡━━━━━━━━╇━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━┩
│ Wait   │ 672.421083ms │ 52.54% │      │
│ Recv   │ 218.976021ms │ 17.11% │ 69MB │
│ Inputs │ 258.380313ms │ 20.19% │ 41kB │
│ Eval   │ 129.760304ms │ 10.14% │      │
│ Result │    182.268µs │  0.01% │  1kB │
│ Total  │ 1.279719989s │        │ 70MB │
└────────┴──────────────┴────────┴──────┘
```

## Ed25519 signature computation

The first signature computation without SHA-512:

```
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━┓
┃ Op          ┃            Time ┃      % ┃ Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━┩
│ Init        │       141.651µs │  0.00% │ 276B │
│ OT Init     │    243.714662ms │  0.06% │ 264B │
│ Peer Inputs │     69.857046ms │  0.02% │ 10kB │
│ Eval        │ 6m55.975310366s │ 99.92% │ 25GB │
│ Total       │ 6m56.289023725s │        │ 25GB │
└─────────────┴─────────────────┴────────┴──────┘
Max permanent wires: 43786395, cached circuits: 26
#gates=935552365, #non-XOR=291882258
```

Optimized p2p.Conn.SendUint{16,32}() not to allocate memory:

```
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━┓
┃ Op          ┃            Time ┃      % ┃ Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━┩
│ Init        │        41.043µs │  0.00% │ 276B │
│ OT Init     │     166.70528ms │  0.07% │ 264B │
│ Peer Inputs │     66.153689ms │  0.03% │ 10kB │
│ Eval        │ 3m48.820978578s │ 99.90% │ 25GB │
│ Total       │  3m49.05387859s │        │ 25GB │
└─────────────┴─────────────────┴────────┴──────┘
Max permanent wires: 43786395, cached circuits: 26
#gates=935552365, #non-XOR=291882258
```

Added one missing ed25519.ScReduce() (+1220540 gates, +381217 non-XOR gates):

```
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━┓
┃ Op          ┃            Time ┃      % ┃ Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━┩
│ Init        │        64.159µs │  0.00% │ 276B │
│ OT Init     │    317.026065ms │  0.14% │ 264B │
│ Peer Inputs │     66.088379ms │  0.03% │ 10kB │
│ Eval        │ 3m46.188650399s │ 99.83% │ 25GB │
│ Total       │ 3m46.571829002s │        │ 25GB │
└─────────────┴─────────────────┴────────┴──────┘
Max permanent wires: 43892147, cached circuits: 26
#gates=936772905, #non-XOR=292263475
```

Optimizing p2p.Conn.SendLabel() not to allocate memory:

```
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━┓
┃ Op          ┃            Time ┃      % ┃ Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━┩
│ Init        │        73.349µs │  0.00% │ 276B │
│ OT Init     │    231.818904ms │  0.13% │ 264B │
│ Peer Inputs │     67.641816ms │  0.04% │ 10kB │
│ Eval        │ 2m57.583645976s │ 99.83% │ 25GB │
│ Total       │ 2m57.883180045s │        │ 25GB │
└─────────────┴─────────────────┴────────┴──────┘
Max permanent wires: 43892147, cached circuits: 26
#gates=936772905, #non-XOR=292263475
```

Optimizing circuit.decrypt() not to allocate memory:

```
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━┓
┃ Op          ┃            Time ┃      % ┃ Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━┩
│ Init        │        72.412µs │  0.00% │ 276B │
│ OT Init     │     201.50279ms │  0.12% │ 264B │
│ Peer Inputs │     66.133977ms │  0.04% │ 10kB │
│ Eval        │ 2m43.437916872s │ 99.84% │ 25GB │
│ Total       │ 2m43.705626051s │        │ 25GB │
└─────────────┴─────────────────┴────────┴──────┘
Max permanent wires: 43893683, cached circuits: 26
#gates=936737641, #non-XOR=292263475
```

Added SHA-512 computation:

```
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op          ┃            Time ┃      % ┃  Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━━┩
│ Init        │       209.487µs │  0.00% │  16kB │
│ OT Init     │     97.139359ms │  0.06% │  264B │
│ Peer Inputs │    4.195529436s │  2.56% │ 667kB │
│ Eval        │ 2m39.404971518s │ 97.38% │  26GB │
│ Total       │   2m43.6978498s │        │  26GB │
└─────────────┴─────────────────┴────────┴───────┘
Max permanent wires: 45442203, cached circuits: 29
#gates=938232660 (XOR=616081537 XNOR=29253505 AND=292519636 OR=363144 INV=14838) #w=968139480
```

The first correct Ed25519 signature:

```
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op          ┃            Time ┃      % ┃  Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━━┩
│ Init        │      1.204858ms │  0.00% │  20kB │
│ OT Init     │    171.049292ms │  0.10% │  264B │
│ Peer Inputs │    4.186015741s │  2.37% │ 667kB │
│ Eval        │ 2m52.585569971s │ 97.54% │  26GB │
│ Total       │ 2m56.943839862s │        │  26GB │
└─────────────┴─────────────────┴────────┴───────┘
Max permanent wires: 53771683, cached circuits: 29
#gates=938713349 (XOR=616368261 XNOR=29253505 AND=292577583 OR=494216 INV=19784) #w=968883849
```

Optimized `smov` and `srshift` instructions:

```
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op          ┃            Time ┃      % ┃  Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━━┩
│ Init        │       213.214µs │  0.00% │  16kB │
│ OT Init     │    188.070588ms │  0.12% │  264B │
│ Peer Inputs │    4.129665419s │  2.60% │ 667kB │
│ Eval        │ 2m34.291433121s │ 97.28% │  25GB │
│ Total       │ 2m38.609382342s │        │  25GB │
└─────────────┴─────────────────┴────────┴───────┘
Max permanent wires: 53770978, cached circuits: 26
#gates=932823577 (XOR=612421393 XNOR=29253505 AND=290634679 OR=494216 INV=19784) #w=956830643
```

Increased p2p bufio buffer size to `1024 * 1024` and optimized wire
slice allocations in `compiler/ssa/streamer.go`:

```
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op          ┃           Time ┃      % ┃  Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━━┩
│ Init        │      175.567µs │  0.00% │  16kB │
│ OT Init     │    82.231558ms │  0.06% │  264B │
│ Peer Inputs │   4.117307505s │  3.23% │ 667kB │
│ Eval        │ 2m3.276233546s │ 96.71% │  25GB │
│ Total       │ 2m7.475948176s │        │  25GB │
└─────────────┴────────────────┴────────┴───────┘
Max permanent wires: 53770978, cached circuits: 26
#gates=932823577 (XOR=612421393 XNOR=29253505 AND=290634679 OR=494216 INV=19784) #w=956830643
```

Optimized p2p writer and reader:

```
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op          ┃            Time ┃      % ┃  Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━━┩
│ Init        │        166.41µs │  0.00% │    0B │
│ OT Init     │    409.906284ms │  0.39% │  16kB │
│ Peer Inputs │    4.137928557s │  3.91% │ 667kB │
│ Eval        │ 1m41.187033817s │ 95.70% │  25GB │
│ Total       │ 1m45.735035068s │        │  25GB │
└─────────────┴─────────────────┴────────┴───────┘
Max permanent wires: 53770978, cached circuits: 26
#gates=932823577 (XOR=612421393 XNOR=29253505 AND=290634679 OR=494216 INV=19784) #w=956830643
```

Half AND:

```
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op          ┃            Time ┃      % ┃  Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━━┩
│ Init        │       165.871µs │  0.00% │    0B │
│ OT Init     │    219.979136ms │  0.21% │  16kB │
│ Peer Inputs │    4.091995195s │  3.94% │ 667kB │
│ Eval        │ 1m39.488931885s │ 95.85% │  16GB │
│ Total       │ 1m43.801072087s │        │  16GB │
└─────────────┴─────────────────┴────────┴───────┘
Max permanent wires: 53770978, cached circuits: 26
#gates=932823577 (XOR=612421393 XNOR=29253505 AND=290634679 OR=494216 INV=19784) #w=956830643
```

Fastpath:

```
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op          ┃            Time ┃      % ┃  Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━━┩
│ Init        │       173.549µs │  0.00% │    0B │
│ OT Init     │    104.099457ms │  0.11% │  16kB │
│ Peer Inputs │    4.086109869s │  4.38% │ 667kB │
│ Eval        │ 1m29.111191227s │ 95.51% │  16GB │
│ Total       │ 1m33.301574102s │        │  16GB │
└─────────────┴─────────────────┴────────┴───────┘
Max permanent wires: 53770978, cached circuits: 26
#gates=932823577 (XOR=612421393 XNOR=29253505 AND=290634679 OR=494216 INV=19784) #w=956830643
```

Circuit constant propagation:

```
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op          ┃            Time ┃      % ┃  Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━━┩
│ Init        │      1.272423ms │  0.00% │    0B │
│ OT Init     │    176.148754ms │  0.20% │  16kB │
│ Peer Inputs │     4.10966228s │  4.57% │ 667kB │
│ Eval        │ 1m25.673832556s │ 95.23% │  15GB │
│ Total       │ 1m29.960916013s │        │  15GB │
└─────────────┴─────────────────┴────────┴───────┘
Max permanent wires: 53768930, cached circuits: 26
#gates=844095703 (XOR=542214087 XNOR=29105707 AND=272261909 OR=494216 INV=19784) #w=868102769
```


## RSA signature computation

| Input | MODP |     Gates | Non-XOR  | Stream Gates | Stream !XOR | Stream   |
|------:|-----:|----------:|---------:|-------------:|------------:|---------:|
|     2 |    4 |       708 |      201 |          740 |         271 | 367.66ms |
|     4 |    8 |      5596 |     1571 |         5548 |        1719 | 115.51ms |
|     8 |   16 |     44796 |    12423 |        46252 |       13199 | 218.26ms |
|    16 |   32 |    374844 |   102255 |       364892 |      101535 | 245.80ms |
|    32 |   64 |   2986556 |   801887 |      2895932 |      788799 | 563.39ms |
|    64 |  128 |  23171068 |  6137023 |     22494524 |     6029311 |  2.4991s |
|   128 |  256 | 177580028 | 46495359 |    172945532 |    45732095 | 14.2368s |
|   256 |  512 |           |          |   1326461180 |   346797567 |  1m40.9s |
|   512 | 1024 |           |          |  10197960188 |  2641252351 | 13m3.86s |

## Multiplication

| Input Size | Total gates | Non-XOR gates |
|-----------:|------------:|--------------:|
|          2 |           7 |             3 |
|          4 |          29 |            13 |
|          8 |         145 |            57 |
|         16 |         655 |           241 |
|         32 |        3347 |          1100 |
|         64 |       13546 |          4242 |
|        128 |       49249 |         14986 |
|        256 |      167977 |         50167 |
|        512 |      549965 |        162147 |
|       1024 |     1752826 |        512099 |
|       2048 |     5485700 |       1592234 |
|       4096 |    16954032 |       4897756 |
|       8192 |    51940803 |      14953708 |

## Mathematic operations with compiler and optimized circuits

Optimized circuits from [pkg/math/](pkg/math/):

| Implementation | XOR gates | AND gates | % of circ |
|:---------------| ---------:|----------:|----------:|
| add64.circ     |       313 |        63 |           |
| sub64.circ     |       313 |        63 |           |
| mul64.circ     |      9707 |      4033 |           |
| div64.circ     |     24817 |      4664 |           |
| MPCL a+b       |       316 |        63 |       100 |
| MPCL a-b       |       317 |        63 |       100 |
| MPCL a*b       |      8294 |      4005 |      99.3 |
| MPCL a/b       |     24581 |      8192 |     175.6 |