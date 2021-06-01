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
program's `b int64` argument, and garblers' In1 to `a
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

# Multi-Party Computation Language (MPCL)

The multi-party computation language is heavily inspired by the Go
programming language, however it is not using the Go's compiler or any
other related components. The compiler is an independent
implementation of the relevant parts of the Go syntax.

## Syntax and parser

The parser parses the MPCL input files, including any referenced
packages, and creates an abstract syntax tree (AST).

The AST is then converted into _Static Single Assignment_ form (SSA)
where each variable is defined and assigned only once. The SSA
transformation does also type checks so that all variable assignments
and function call arguments and return values are checked (or
converted) to be or correct type.

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

 - `len(VALUE)`: returns the length of the value as integer:
   - array: returns the number of array elements
   - string: returns the number of bytes in the string
 - `make(TYPE, SIZE)`: creates an instance of the type _type_ with _size_ bits.
 - `native(NAME, ARG...)`: calls a builtin function _name_ with
   arguments _arg..._. The _name_ can specify a circuit file (*.circ)
   or one of the following builtin functions:
   - `hamming(a, b uint)` computes the bitwise hamming distance between argument values
 - `size(VARIABLE)`: returns the bit size of the argument _variable_.

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

 - [X] Phase 0
   - [X] Oblivious Transfer
   - [X] Garbled circuit garbling and evaluation
   - [X] MPCL compiler for basic arithmetics
 - [ ] Phase 1
   - [X] Compiler
     - [X] Conditionals
     - [X] Struct input types
     - [X] Binary serialization format for circuits
     - [X] RSA 126-bit signature
   - Circuit & garbling
     - [X] RSA 126-bit signature
     - [ ] BMR multi-party protocol
 - [ ] Phase 2
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
       - [ ] variable liveness analysis for templates
     - [ ] Signed / unsigned arithmetics
     - [ ] unary expressions
       - [ ] logical not
     - [ ] BitShift
   - Circuit & garbling:
     - [X] Incremental (streaming) garbling and evaluation
     - [ ] Row reduction
     - [ ] Half AND
     - [ ] Oblivious transfer extensions
   - Misc:
     - [ ] TLS for garbler-evaluator protocol

# Running benchmark: 32-bit RSA encryption (64-bit modp)

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

## RSA signature computation

| Input | MODP |     Gates | Non-XOR  | Stream Gates | Stream Non-XOR |
|------:|-----:|----------:|---------:|-------------:|---------------:|
|     2 |    4 |       708 |      201 |          730 |            205 |
|     4 |    8 |      5596 |     1571 |         5640 |           1579 |
|     8 |   16 |     44796 |    12423 |        44884 |          12439 |
|    16 |   32 |    374844 |   102255 |       375052 |         102287 |
|    32 |   64 |   2986556 |   801887 |      2986972 |         801951 |
|    64 |  128 |  23171068 |  6137023 |     23171900 |        6137151 |
|   128 |  256 | 177580028 | 46495359 |    177581692 |       46495615 |
|   256 |  512 |           |          |   1356768508 |      351848191 |

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
