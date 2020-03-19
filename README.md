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
Circuit: #gates=386, #wires=515 n1=65, n2=64, n3=1
 - N1: a{1,0}i64:int64, %0:uint1
 + N2: b{1,0}i64:int64
 - N3: %_{0,1}b1:bool1
 - In: [800000]
Listening for connections at :8080
```

The evaluator's input is 800000 and it is set to the circuit inputs
`N2`. The evaluator is now waiting for garblers to connect to the TCP
port `:8080`.

Next, let's start the garbler:

```
$ ./garbled -i 750000,0 examples/millionaire.mpcl
Circuit: #gates=386, #wires=515 n1=65, n2=64, n3=1
 + N1: a{1,0}i64:int64, %0:uint1
 - N2: b{1,0}i64:int64
 - N3: %_{0,1}b1:bool1
 - In: [750000 0]
```

The garbler's input is 750000 and it is set to the circuit inputs
`N1`. The garbler connects to the evaluator's TCP port and they run
the garbled circuit protocol. At the end, garbler (and evaluator)
print the result of the circuit, which is this case is single `bool`
value `N3`:

```
Result[0]: 0
Result[0]: 0b0
Result[0]: 0x00
```

In our example, the evaluator's argument N2 is bound to the MPCL
program's `b int64` argument, and garblers' N1 to `a
int64`. Therefore, the result of the computation is `false` because
N1=750000 <= N2=800000. If we increase the garbler's input to 900000,
we see that the result is now `true` since the garbler's input is now
bigger than the evaluator's input:

```
$ ./garbled -i 900000,0 examples/millionaire.mpcl
Circuit: #gates=386, #wires=515 n1=65, n2=64, n3=1
 + N1: a{1,0}i64:int64, %0:uint1
 - N2: b{1,0}i64:int64
 - N3: %_{0,1}b1:bool1
 - In: [900000 0]
Result[0]: 1
Result[0]: 0b1
Result[0]: 0x01
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

| Name   | Size | Signed | Alias  |
|:------:|:----:|:------:|:------:|
| bool   | 1    | no     |        |
| byte   | 8    | no     | uint8  |
| uint   | 32   | no     | uint32 |
| int    | 32   | yes    | int32  |
| uintN  | N    | no     |        |
| intN   | N    | yes    |        |
| floatN | N    | yes    |        |

### Builtin functions

The MPCL runtime defines the following builtin functions:

 - `native(NAME, ARG...)`: calls a builtin function _name_ with
   arguments _arg..._. The _name_ can specify a circuit file (*.circ)
   of one of the following builtin functions:
   - `hamming(a, b uint_)` computes 2the bitwise hamming distance between argument values
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
# main#0:
l0:
        igt    a{1,0}i4 b{1,0}i4 %_{0,0}b1
        jump   l2
l2:
        if     %_{0,0}b1 l3
        jump   l4
l4:
        mov    b{1,0}i4 %ret0{1,2}i4
        jump   l1
# main.ret#0:
l1:
        phi    %_{0,0}b1 %ret0{1,1}i4 %ret0{1,2}i4 %_{0,1}i4
        ret    %_{0,1}i4
l3:
        mov    a{1,0}i4 %ret0{1,1}i4
        jump   l1
```
<img align="center" width="524" height="394" src="ifelse.png">

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

<img align="center" width="500" height="660" src="max.png">

# TODO

 - Compiler:
   - [ ] Constant folding
     - [X] binary expressions
     - [X] if-blocks
     - [X] For-loop unrolling
     - [ ] Function call and return
   - [ ] sort blocks in topological order
     - [ ] peephole optimization over block boundaries
   - [ ] Signed / unsigned arithmetics
   - [ ] unary expressions
     - [ ] logical not
   - [ ] BitShift
 - Circuit & garbling:
   - [ ] identity gate
   - [ ] Row reduction
   - [ ] Half AND
   - [ ] 3-input gates for full-adder carry
   - [ ] Oblivious transfer extensions
 - Packages:
   - [X] MODP circuit
     - [X] variable definition with init value
     - [X] binary modulo operation
     - [X] peephole optimizer for creating bit test instructions
     - [ ] computed types: `multType := make(uint, size(a)*2)`
 - Misc:
   - [ ] TLS for garbler-evaluator protocol
   - [X] Session-specific circuit encryption key

## Free XOR

SHA-256, before: 3.936451044s

```
┏━━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━┓
┃ Op      ┃         Time ┃      % ┃ Xfer ┃
┣━━━━━━━━━╋━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━┫
┃ Garble  ┃ 161.871045ms ┃  4.25% ┃      ┃
┃ Xfer    ┃  65.838224ms ┃  1.73% ┃ 11MB ┃
┃ OT Init ┃  67.703136ms ┃  1.78% ┃ 264B ┃
┃ Eval    ┃ 3.509030323s ┃ 92.24% ┃ 11MB ┃
┃ OT      ┃ 3.479226123s ┃ 99.15% ┃      ┃
┃ Total   ┃ 3.804442728s ┃        ┃      ┃
┗━━━━━━━━━┻━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━┛
```

After: 3.727278622s

```
┏━━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━┓
┃ Op      ┃         Time ┃      % ┃ Xfer ┃
┣━━━━━━━━━╋━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━┫
┃ Garble  ┃  98.590262ms ┃  2.65% ┃      ┃
┃ Xfer    ┃  13.978017ms ┃  0.38% ┃  2MB ┃
┃ OT Init ┃ 101.184332ms ┃  2.71% ┃ 264B ┃
┃ Eval    ┃ 3.513526011s ┃ 94.27% ┃  3MB ┃
┃ OT      ┃ 3.482825234s ┃ 99.13% ┃      ┃
┃ Total   ┃ 3.727278622s ┃        ┃      ┃
┗━━━━━━━━━┻━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━┛
```

## Changing evaluator to keep wires in slice instead of map

Test program:

```go
package main

func main(a, b uint1024) (uint1024, uint1024) {
    return b / a, b % a
}
```

Wires in map:

```
┏━━━━━━━━┳━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op     ┃          Time ┃      % ┃  Xfer ┃
┣━━━━━━━━╋━━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━┫
┃ Wait   ┃ 24.307607142s ┃ 51.43% ┃       ┃
┃ Recv   ┃  5.694420084s ┃ 12.05% ┃ 739MB ┃
┃ Inputs ┃  7.416426898s ┃ 15.69% ┃   1MB ┃
┃ Eval   ┃  9.841078673s ┃ 20.82% ┃       ┃
┃ Result ┃    2.168686ms ┃  0.00% ┃  41kB ┃
┃ Total  ┃ 47.261701483s ┃        ┃       ┃
┗━━━━━━━━┻━━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━┛
```

Wires in slice:

```
┏━━━━━━━━┳━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op     ┃          Time ┃      % ┃  Xfer ┃
┣━━━━━━━━╋━━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━┫
┃ Wait   ┃ 23.478917543s ┃ 61.46% ┃       ┃
┃ Recv   ┃  5.510398508s ┃ 14.42% ┃ 739MB ┃
┃ Inputs ┃  7.235458941s ┃ 18.94% ┃   1MB ┃
┃ Eval   ┃  1.978486396s ┃  5.18% ┃       ┃
┃ Result ┃     1.22263ms ┃  0.00% ┃  41kB ┃
┃ Total  ┃ 38.204484018s ┃        ┃       ┃
┗━━━━━━━━┻━━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━┛
```

## 32-bit RSA encryption (64-bit modp)

```
Circuit: #gates=14707672 (XOR=6292351 AND=6267581 OR=2064702 INV=83038), #wires=14707994 n1=258, n2=64, n3=64
 + N1: mode{1,0}b1:bool1, msg{1,0}u64:uint64, gD{1,0}u64:uint64, pubN{1,0}u64:uint64, pubE{1,0}u64:uint64, %0:uint1
 - N2: eD{1,0}u64:uint64
 - N3: %_{0,1025}u64:uint64
 - In: [1 0x6d7472 0x321af130 0xd60b2b09 0x10001 0]
 - Garbling...
 - Sending garbled circuit...
 - Processing messages...
┏━━━━━━━━━┳━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━┓
┃ Op      ┃          Time ┃      % ┃  Xfer ┃
┣━━━━━━━━━╋━━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━┫
┃ Garble  ┃ 21.301538337s ┃ 71.07% ┃       ┃
┃ Xfer    ┃  6.031901951s ┃ 20.13% ┃ 728MB ┃
┃ OT Init ┃  270.099382ms ┃  0.90% ┃  264B ┃
┃ Eval    ┃  2.367941121s ┃  7.90% ┃ 728MB ┃
┃ OT      ┃  495.455402ms ┃ 20.92% ┃       ┃
┃ Total   ┃ 29.971480791s ┃        ┃       ┃
┗━━━━━━━━━┻━━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━┛
Result[0]: 1643769736
Result[0]: 0b1100001111110011110111110001000
Result[0]: 0x61f9ef88
```

## 64-bit RSA encryption (128-bit modp)

```
Circuit: #gates=58821912 (XOR=25166591 AND=25117309 OR=8323390 INV=214622), #wires=58822233 n1=257, n2=64, n3=128
 + N1: msg{1,0}u64:uint64, gD{1,0}u64:uint64, pubN{1,0}u64:uint64, pubE{1,0}u64:uint64, %0:uint1
 - N2: eD{1,0}u64:uint64
 - N3: %ret0{1,9}u64:uint64, %ret1{1,1}u64:uint64
 - In: [0x6d7472 0x321af130 0xd60b2b09 0x10001 0]
 - Garbling...
 - Sending garbled circuit...
 - Processing messages...
┏━━━━━━━━━┳━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━┓
┃ Op      ┃            Time ┃      % ┃ Xfer ┃
┣━━━━━━━━━╋━━━━━━━━━━━━━━━━━╋━━━━━━━━╋━━━━━━┫
┃ Garble  ┃ 3m38.029198671s ┃ 48.48% ┃      ┃
┃ Xfer    ┃ 2m12.133617292s ┃ 29.38% ┃  2GB ┃
┃ OT Init ┃    170.307866ms ┃  0.04% ┃ 264B ┃
┃ Eval    ┃ 1m39.353739395s ┃ 22.09% ┃  2GB ┃
┃ OT      ┃    1.540024033s ┃  1.55% ┃      ┃
┃ Total   ┃ 7m29.686863224s ┃        ┃      ┃
┗━━━━━━━━━┻━━━━━━━━━━━━━━━━━┻━━━━━━━━┻━━━━━━┛
Result[0]: 1643769736
Result[0]: 0b1100001111110011110111110001000
Result[0]: 0x61f9ef88
Result[1]: 7173234
Result[1]: 0b11011010111010001110010
Result[1]: 0x6d7472
```

# Develoment ideas

## Mathematical operations

```go
package main

func main(a, b int) (int, int) {
    q, r := a / b
    return q, r
}
```
