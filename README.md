# mpc

Secure Multi-Party Computation with Go. This project implements secure
two-party computation with [Garbled circuit](https://en.wikipedia.org/wiki/Garbled_circuit) protocol. The main components are:
 - [ot](ot/): **oblivious transfer** library
 - [circuit](circuit/): **garbled circuit** parser, garbler, and evaluator
 - [compiler](compiler/): **Multi-Party Computation Language (MPCL)** compiler

## Getting started

The easiest way to experiment with the system is to compile the
[garbled](apps/garbled/) application and use it to evaluate
programs. The `garbled` application takes the following command line
options:

 - `-e`: specifies circuit _evaluator_ / _garbler_ mode. The circuit evaluator creates a TCP listener and waits for garblers to connect with computation.
 - `-i`: specifies comma-separated input values for the circuit.
 - `-v`: enabled verbose output.

The [examples](apps/garbled/examples/) directory contains various MPCL
example programs which can be executed with the `garbled`
application. For example, here's how you can run the [Yao's
Millionaires'
Problem](https://en.wikipedia.org/wiki/Yao%27s_Millionaires%27_Problem)
which can be found from the
[millionaire.mpcl](apps/garbled/examples/millionaire.mpcl) file:

```go
package main

func main(a, b int64) bool {
    if a > b {
        return true
    } else {
        return false
    }
}
```

First, start the evaluator (these examples are run in the
`apps/garbled` directory):

```
$ ./garbled -e -i 800000 examples/millionaire.mpcl
 - In1: a{1,0}i64:int64
 + In2: b{1,0}i64:int64
 - Out: %_{0,1}b1:bool1
 -  In: [800000]
Listening for connections at :8080
```

The evaluator's input is 800000 and it is set to the circuit inputs
`In2`. The evaluator is now waiting for garblers to connect to the TCP
port `:8080`.

Next, let's start the garbler:

```
$ ./garbled -i 750000 examples/millionaire.mpcl
 + In1: a{1,0}i64:int64
 - In2: b{1,0}i64:int64
 - Out: %_{0,1}b1:bool1
 -  In: [750000]
Result[0]: false
```

The garbler's input is 750000 and it is set to the circuit inputs
`In1`. The garbler connects to the evaluator's TCP port and they run
the garbled circuit protocol. At the end, garbler (and evaluator)
print the result of the circuit, which is this case is single `bool`
value `Result[0]`:

```
Result[0]: false
```

In our example, the evaluator's argument In2 is bound to the MPCL
program's `b int64` argument, and garbler's In1 to `a
int64`. Therefore, the result of the computation is `false` because
In1=750000 <= In2=800000. If we increase the garbler's input to 900000,
we see that the result is now `true` since the garbler's input is now
bigger than the evaluator's input:

```
$ ./garbled -i 900000 examples/millionaire.mpcl
 + In1: a{1,0}i64:int64
 - In2: b{1,0}i64:int64
 - Out: %_{0,1}b1:bool1
 -  In: [900000]
Result[0]: true
```

## Ed25519 Key Generation and Signature Computation

The [ed25519](apps/garbled/examples/ed25519/) directory contains
Ed25519 [key generation](apps/garbled/examples/ed25519/keygen.mpcl)
and [signature computation](apps/garbled/examples/ed25519/sign.mpcl)
examples.

### Key Generation

```
$ ./garbled -stream -e -v -i 0x784db0ec4ca0cf5338249e6a09139109366dca1fac2838e5f0e5a46f0e191bae,0xd0da45d3c99e756da831d1e7d696eae3fa9fe39d3b1b2618c7ff997d17777989b5cf415b114298c8b10bed0f0eff118e43ab606ab01143151dff89171307dffa,0x44bf09357e19b1f96f9cf6d9e7d25a0e8dd62d6e0d4bba2bec4c59983c7dc84d1486677b6d8837746cd948c881913c36faeaee08e8309afac58be4757a1c544e
```

```
$ ./garbled -stream -v -i 0x57c0e59c20ac7d75ef7e3188fdd7f5876abee1cab394af8125acaca9760bb54c,0x76b42e6292f4a3dc339d208481abeb9a24e08127c7cd8dbde62abcddc0c0e6f7a0f740e756b44dae137f0e7ff8eae0ceb1a962c130fdcbe8cbee3e31ab55b8dc,0xeb83eb1f5203f5b752c96264a21ff4a27fa60cf2313f5f53c3fa96e0b52a2814b786e43a3af64b66291b5b29f432cb8d5a930e31f4e6f072a6d33b861b5b5f13 examples/ed25519/keygen.mpcl
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━┓
┃ Op          ┃            Time ┃      % ┃ Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━┩
│ Init        │       258.059µs │  0.00% │ 20kB │
│ OT Init     │    140.256553ms │  0.09% │ 264B │
│ Peer Inputs │   10.292906559s │  6.76% │  1MB │
│ Eval        │ 2m21.830984013s │ 93.15% │ 25GB │
│ Total       │ 2m32.264405184s │        │ 25GB │
└─────────────┴─────────────────┴────────┴──────┘
Max permanent wires: 51993714, cached circuits: 23
#gates=925736899 (XOR=607899819 XNOR=29055825 AND=288660077 OR=116232 INV=4946) #w=948767997
Result[0]: 8ae64963506002e267a59665e9a2e6f9348cc159be53747894478e182ece9fcb
Result[1]: 4ded80ae09692306c9659307f522f5dba1d96e48cde9f4f6e22fb340629db76aa2bee5867d009e008b6fb85902273acda8910c9a740a788f70c28ca0a3093835
Result[2]: cd5c37f4497fd56e236aa858442b3ff90f7a6401ee2186ea18d074fe93d8f9d18b582fa47a1ee0f0a9083ddd9e262b8f3c642dfad68f667f87dddd4bec80aca3
```

### Signature Computation

```
$ ./garbled -stream -e -v -i 0x46eb82a021d88960fb13388b0e76ba13b84524ffe114d7f3a728b39efc185eeaa7137132182bab7504daf200d882b787ee8b9b1c9f41be9c38fb4e0ba1aff326
```

```
$ ./garbled -stream -v -i 0x4d61726b6b7520526f737369203c6d747240696b692e66693e2068747470733a2f2f7777772e6d61726b6b75726f7373692e636f6d2f,0x5e768ad83640b43d93d6c26b34021d0a0cda6bf5eb962970554d7ab074e2f4cd49bc6fef2fa4dc2f763c1f70b751b7f03d398e8930d837130426454ea52d4449 examples/ed25519/sign.mpcl
┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op          ┃            Time ┃      % ┃  Xfer ┃
┡━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━━┩
│ Init        │       244.421µs │  0.00% │  16kB │
│ OT Init     │    144.588209ms │  0.09% │  264B │
│ Peer Inputs │    4.113222291s │  2.63% │ 667kB │
│ Eval        │  2m32.10905711s │ 97.28% │  25GB │
│ Total       │ 2m36.367112031s │        │  25GB │
└─────────────┴─────────────────┴────────┴───────┘
Max permanent wires: 53770978, cached circuits: 26
#gates=932823577 (XOR=612421393 XNOR=29253505 AND=290634679 OR=494216 INV=19784) #w=956830643
Result[0]: b71a55aece64574bedd94729a9ca95a87b5fe0a587fecf50ff0238805132c1291e08cb871016cb4f3935bd45423626f61dc648a91affda3671b19d7b28e03505
```

# Multi-Party Computation Language (MPCL)

The multi-party computation language is heavily inspired by the Go
programming language, however it is not using the Go's compiler or any
other related components. The compiler is an independent
implementation of the relevant parts of the Go syntax.

## Compiler

<img align="center" src="docs/mpcl-compiler.png">

The parser parses the MPCL input files, including any referenced
packages, and creates an abstract syntax tree (AST).

The AST is then converted into _Static Single Assignment_ form (SSA)
where each variable is defined and assigned only once. The SSA
transformation does also type checks so that all variable assignments
and function call arguments and return values are checked (or
converted) to be or correct type.

<img aling="center" src="docs/mpcc.png">

### Types

| Name    | Size          | Signed |
|:-------:|:-------------:|:------:|
| bool    | 1             | no     |
| uint    | unspecified   | no     |
| int     | unspecified   | yes    |
| uintN   | N    	  | no     |
| intN    | N    	  | yes    |
| floatN  | N    	  | yes    |
| stringN | N    	  | no     |

The unsized `uint` and `int` types can be used as function arguments
and return values. Their sizes are resolved during compilation
time. The only exception is the `main` function. Its arguments must
use fixed type sizes. The following example shows a `MinMax` function
that returns the minimum and maximum arguments. This function works
for all argument sizes.

```go
func MinMax(a, b int) (int, int) {
    if a < b {
        return a, b
    } else {
        return b, a
    }
}
```

### Builtin functions

The MPCL runtime defines the following builtin functions:

 - `copy(dst, src)`: copies the content of the array _src_ to
   _dst_. The function returns the number of elements copied, which is
   the minimum of len(src) and len(dst).
 - `len(value)`: returns the length of the value as integer:
   - array: returns the number of array elements
   - string: returns the number of bytes in the string
 - `make(type, size)`: creates an instance of the type _type_ with _size_ bits.
 - `native(name, arg...)`: calls a builtin function _name_ with
   arguments _arg..._. The _name_ can specify a circuit file (*.circ)
   or one of the following builtin functions:
   - `hamming(a, b uint)` computes the bitwise hamming distance between argument values
 - `size(variable)`: returns the bit size of the argument _variable_.

## SSA (Static single assignment form)

```go
package main

func main(a, b int4) int4 {
    if a > b {
        return a
    }
    return b
}
```

The compiler creates the following SSA form assembly:

```
# Input0: a{1,0}i4:int4
# Input1: b{1,0}i4:int4
# Output0: %_{0,1}i4:int4
# main#0:
	igt     a{1,0}i4 b{1,0}i4 %_{0,0}b1
	mov     a{1,0}i4 %ret0{1,1}i4
	mov     b{1,0}i4 %ret0{1,2}i4
# main.ret#0:
	phi     %_{0,0}b1 %ret0{1,1}i4 %ret0{1,2}i4 %_{0,1}i4
	gc      %_{0,0}b1
	gc      %ret0{1,1}i4
	gc      %ret0{1,2}i4
	gc      a{1,0}i4
	gc      b{1,0}i4
	ret     %_{0,1}i4
```
<img align="center" width="476" height="284" src="ifelse.png">

The SSA assembly (and logical circuit) form a Directed Acyclic Graph
(DAG) without any mutable storage locations. This means that all
alternative execution paths must be evaluate and when the program is
returning its computation results, any conflicting values from
different execution paths must be resolved with the branching
condition. This value resolution is implemented as the `phi` assembly
instruction, which effectively implements a MUX logical circuit:

    O=(D0 XOR D1)C XOR D0

| D0  | D1  | C   | D0 XOR D1 | AND C | XOR D0 |
|:---:|:---:|:---:|:---------:|:-----:|:------:|
| 0   | 0   | 0   |     0     |   0   |   0    |
| 0   | 1   | 0   |     1     |   0   |   0    |
| 1   | 0   | 0   |     1     |   0   |   1    |
| 1   | 1   | 0   |     0     |   0   |   1    |
| 0   | 0   | 1   |     0     |   0   |   0    |
| 0   | 1   | 1   |     1     |   1   |   1    |
| 1   | 0   | 1   |     1     |   1   |   0    |
| 1   | 1   | 1   |     0     |   0   |   1    |


## Circuit generation

The 3rd compiler phase converts SSA form assembly into logic gate
circuit. The following circuit was generated from the previous SSA
form assembly:

<img align="center" width="454" height="1047" src="max.png">

# TODO

 - [ ] Compiler
   - [ ] Incremental compiler
     - [ ] Constant folding
       - [ ] Implement using AST rewrite
       - [ ] binary expressions
       - [ ] if-blocks
       - [ ] For-loop unrolling
       - [ ] Function call and return
     - [ ] peephole optimization
       - [X] sort blocks in topological order
       - [X] peephole optimization over block boundaries
       - [ ] SSA aliasing is 1:1 but `amov` has 2:1 relation
       - [ ] variable liveness analysis for templates
     - [ ] BitShift
   - [ ] Circuit & garbling:
     - [ ] Row reduction
     - [ ] Oblivious transfer extensions
   - [ ] Misc:
     - [ ] TLS for garbler-evaluator protocol
 - [ ] BMR multi-party protocol
 - [ ] Ed25519
   - [X] Parsing Ed25519 MPCL files
     - [ ] for-range statements
     - [ ] local variables in for-loop unrolling
   - [X] Pointer handling
     - [X] Pointer to struct field
     - [ ] Cleanup pointer r-value handling
     - [ ] Slices are passed by value instead of by reference
     - [ ] Selecting struct members from struct pointer value
   - [ ] Compound init values must be zero-padded to full size
   - [ ] Circuit generation:
     - [ ] SSA variable liveness analysis must be optimized
   - [ ] SHA-512 message digest
     - [ ] Empty arrays should be allowed, now unspecified length
     - [ ] `block = 0` sets block's type to int32
   - [ ] `copy()` does not work on arrays which have been `make()`:ed
   - [ ] `&base[pos][i]` returns the address of the first element
   - [ ] reading from `*[32]int32` returns invalid values

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

## RSA signature computation

| Input | MODP |     Gates | Non-XOR  | Stream Gates | Stream !XOR | Stream   |
|------:|-----:|----------:|---------:|-------------:|------------:|---------:|
|     2 |    4 |       708 |      201 |          730 |         205 | 125.56ms |
|     4 |    8 |      5596 |     1571 |         5640 |        1579 | 144.60ms |
|     8 |   16 |     44796 |    12423 |        44884 |       12439 | 250.11ms |
|    16 |   32 |    374844 |   102255 |       375052 |      102287 | 296.88ms |
|    32 |   64 |   2986556 |   801887 |      2986972 |      801951 | 662.76ms |
|    64 |  128 |  23171068 |  6137023 |     23171900 |     6137151 |  2.6671s |
|   128 |  256 | 177580028 | 46495359 |    177630716 |    46528255 | 16.3596s |
|   256 |  512 |           |          |   1356964860 |   351979007 |  1m58.8s |
|   512 | 1024 |           |          |  10391387132 |  2673970175 | 15m11.6s |

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
| sub64.circ     |       442 |        63 |           |
| mul64.circ     |      9707 |      4033 |           |
| div64.circ     |     25328 |      4664 |           |
| MPCL a+b       |       316 |        63 |       100 |
| MPCL a-b       |       319 |        63 |       100 |
| MPCL a*b       |      9304 |      4242 |     105.2 |
| MPCL a/b       |     24833 |      8192 |     175.6 |

# Develoment ideas

## Mathematical operations

```go
package main

func main(a, b int) (int, int) {
    q, r := a / b
    return q, r
}
```
