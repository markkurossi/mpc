# mpc
Secure Multi-Party Computation

# SSA (Static single assignment form)

```go
package main

func main(a, b int) int {
    if a > b {
        return a
    }
    return b
}
```

The compiler creates the following SSA form assembly:

```
l0:
	igt    a{1,0}i b{1,0}i %_{0,0}b
	jump   l2
l2:
	if     %_{0,0}b l3
	jump   l4
l4:
	mov    b{1,0}i %ret0{1,2}i
	jump   l1
l1:
	phi    %_{0,0}b %ret0{1,1}i %ret0{1,2}i %_{0,1}i
	ret    %_{0,1}i
l3:
	mov    a{1,0}i %ret0{1,1}i
	jump   l1
```
<img align="center" width="500" height="400" src="ifelse.png">

# Syntax

## Mathematical operations

```go
package main

func main(a, b int) (int, int) {
    q, r := a / b
    return q, r
}
```

## Building blocks

### MUX

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
