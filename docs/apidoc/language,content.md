# MPCL Language Documentation

[MPCL Language Grammar](mpcl.html)

## MPCL Static Single-Assignment (SSA) Assembly

All opcodes operate on argument values of arbitrary size. The argument
types specify the actual amount of bits (wires) in the argument and
result values. The examples use the following format for values:

 - *name*`{0,0}`*typeSize*
 - `$`*const*

where:

*name*
: is the argument name. The special name `%_` specifies an anonymous
  value.

`{0,0}`
: is the argument scope and version information. In the examples
  below, these will always be `{0,0}`.

*type*
: is the argument type specifier:

  - `i` signed integer
  - `u` unsigned integer
  - `arr` array

*size*
: is the argument size in bits

`$`*const*
: is a constant value.

### opcode iadd (0x00)

```
iadd    a{0,0}i32 b{0,0}i32 r{0,0}i32
```

The `iadd` instruction adds the signed integer arguments `a` and `b`
together and sets the result to the result value `r`.

### opcode uadd (0x01)

```
uadd    a{1,0}u32 b{1,0}u32 r{0,0}u32
```

The `uadd` instruction adds the unsigned integer arguments `a` and `b`
together and sets the result to the result value `r`.

### opcode fadd (0x02) ⚠ not implemented yet ⚠

```
fadd    a{1,0}f32 b{1,0}f32 r{0,0}f32
```

The `fadd` instruction adds the floating point arguments `a` and `b`
together and sets the result to the result value `r`.

### opcode isub (0x03)

```
isub    a{0,0}i32 b{0,0}i32 r{0,0}i32
```

The `isub` instruction subtracts the signed integer arguments `a` and
`b` and sets the result to the result value `r`.

### opcode usub (0x04)

```
usub    a{0,0}u32 b{0,0}u32 r{0,0}u32
```

The `usub` instruction subtracts the unsigned integer arguments `a`
and `b` and sets the result to the result value `r`.

### opcode fsub (0x05) ⚠ not implemented yet ⚠

```
fsub    a{0,0}f32 b{0,0}f32 r{0,0}f32
```

The `fsub` instruction subtracts the floating point arguments `a` and
`b` and sets the result to the result value `r`.

### opcode bor (0x06)

```
bor     a{0,0}u32 b{0,0}u32 r{0,0}u32
```

The `bor` instruction compute the binary OR `a|b` and sets the result
to the result value `r`.

### opcode bxor (0x07)

```
bxor    a{0,0}u32 b{0,0}u32 r{0,0}u32
```

The `bxor` instruction compute the binary XOR `a^b` and sets the
result to the result value `r`.

### opcode band (0x08)

```
band    a{0,0}u32 b{0,0}u32 r{0,0}u32
```

The `band` instruction compute the binary AND `a^b` and sets the
result to the result value `r`.

### opcode bclr (0x09)
### opcode bts (0x0a)
### opcode btc (0x0b)
### opcode imult (0x0c)
### opcode umult (0x0d)
### opcode fmult (0x0e) ⚠ not implemented yet ⚠
### opcode idiv (0x0f)
### opcode udiv (0x10)
### opcode fdiv (0x11) ⚠ not implemented yet ⚠
### opcode imod (0x12)
### opcode umod (0x13)
### opcode fmod (0x14) ⚠ not implemented yet ⚠
### opcode concat (0x15)
### opcode lshift (0x16)
### opcode rshift (0x17)
### opcode srshift (0x18)
### opcode slice (0x19)
### opcode index (0x1a)
### opcode ilt (0x1b)
### opcode ult (0x1c)
### opcode flt (0x1d) ⚠ not implemented yet ⚠
### opcode ile (0x1e)
### opcode ule (0x1f)
### opcode fle (0x20) ⚠ not implemented yet ⚠
### opcode igt (0x21)
### opcode ugt (0x22)
### opcode fgt (0x23) ⚠ not implemented yet ⚠
### opcode ige (0x24)
### opcode uge (0x25)
### opcode fge (0x26) ⚠ not implemented yet ⚠
### opcode eq (0x27)
### opcode neq (0x28)
### opcode and (0x29)
### opcode or (0x2a)
### opcode not (0x2b)
### opcode mov (0x2c)
### opcode smov (0x2d)

### opcode amov (0x2e)

```
amov    val{0,0}u8 base{0,0}arr32 $from $to r{0,0}arr32
```

The `amov` instruction overwrite the range `from`-`to` of `base` with
`val` and set the resulting value to `r`. For example, if `base` has
the value `0xaabbccdd` and `val` is `0x22` the following example sets
`r` to `0xaa22ccdd`:

```
amov    $34 0xaabbccdd $8 $16 r{0,0}arr32 ⇒ r{0,0}=0xaa22ccdd
```

In this example we assume that `val` and `base` bit indices are counted
from left.

### opcode phi (0x2f)

```
phi     cond{0,0}b1 t{0,0}i32 f{1,2}i32 r{0,1}i32
```

The `phi` instruction selects true `t` or false `f` value based on the
condition `cond` and sets the selected value into `r`.

### opcode ret (0x30)

```
ret    %ret0{0,0}i32 %ret1{0,0}i32
```

The `ret` instruction returns the function return values to the
caller. The number and types of the return values depend on the
function signature.

### opcode circ (0x31)

```
circ    arg{0,707}u1024 arg{1,0}u512 {G=349617, W=351153} r{0,708}u512
```

### opcode builtin (0x32)
### opcode gc (0x33)
