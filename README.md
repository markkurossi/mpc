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
   - [ ] For-loop unrolling
   - [ ] Signed / unsigned arithmetics
   - [X] `const (sym=val,sym=val)` syntax
   - [ ] Variable assignment with _n_ values
   - [ ] logical not
   - [ ] binary divide
   - [ ] unary expressions
   - [X] Bit{And,Or,Xor,Clear}
   - [ ] BitShift
 - Garbling:
   - [X] Free XOR
   - [ ] Row reduction
   - [ ] Half AND
 - Packages:
   - [ ] MODP circuit
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

# Develoment ideas

## Mathematical operations

```go
package main

func main(a, b int) (int, int) {
    q, r := a / b
    return q, r
}
```
