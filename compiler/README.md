# Compiler

<img align="center" src="../docs/mpcl-compiler.png">

The parser parses the MPCL input files, including any referenced
packages, and creates an abstract syntax tree (AST).

The AST is then converted into _Static Single Assignment_ form (SSA)
where each variable is defined and assigned only once. The SSA
transformation does also type checks so that all variable assignments
and function call arguments and return values are checked (or
converted) to be or correct type.

<img align="center" src="../docs/mpcc.png">

## Types

| Name    | Size          | Signed |
|:-------:|:-------------:|:------:|
| bool    | 1             | no     |
| uint    | unspecified   | no     |
| int     | unspecified   | yes    |
| uintN   | N             | no     |
| intN    | N             | yes    |
| floatN  | N             | yes    |
| stringN | N             | no     |

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


# Typecast

Typecast is implemented for [static values](ast/eval.go) `Call.Eval`
and [dynamic values](ast/ssagen.go) `Call.SSA`.

## Static Cast

Value identity for const values is computed from the value's `Name`,
`Scope`, and `Version`. These values remain the same for all constant
value instances so the static cast can creates a new constant instance
with the casted `Type`.

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

<img align="center" width="454" height="1047" src="../docs/max.png">

## Develoment ideas

### Mathematical operations

```go
package main

func main(a, b int) (int, int) {
    q, r := a / b
    return q, r
}
```
