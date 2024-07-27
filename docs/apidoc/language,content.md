# MPCL Language Documentation

[MPCL Language Grammar](mpcl.html)

## MPCL Static Single-Assignment (SSA) Assembly

### opcode iadd (0x00)

```
iadd left right dst
```

### opcode uadd (0x01)
### opcode fadd (0x02)
### opcode isub (0x03)
### opcode usub (0x04)
### opcode fsub (0x05)
### opcode bor (0x06)
### opcode bxor (0x07)
### opcode band (0x08)
### opcode bclr (0x09)
### opcode bts (0x0a)
### opcode btc (0x0b)
### opcode imult (0x0c)
### opcode umult (0x0d)
### opcode fmult (0x0e)
### opcode idiv (0x0f)
### opcode udiv (0x10)
### opcode fdiv (0x11)
### opcode imod (0x12)
### opcode umod (0x13)
### opcode fmod (0x14)
### opcode concat (0x15)
### opcode lshift (0x16)
### opcode rshift (0x17)
### opcode srshift (0x18)
### opcode slice (0x19)
### opcode index (0x1a)
### opcode ilt (0x1b)
### opcode ult (0x1c)
### opcode flt (0x1d)
### opcode ile (0x1e)
### opcode ule (0x1f)
### opcode fle (0x20)
### opcode igt (0x21)
### opcode ugt (0x22)
### opcode fgt (0x23)
### opcode ige (0x24)
### opcode uge (0x25)
### opcode fge (0x26)
### opcode eq (0x27)
### opcode neq (0x28)
### opcode and (0x29)
### opcode or (0x2a)
### opcode not (0x2b)
### opcode mov (0x2c)
### opcode smov (0x2d)

### opcode amov (0x2e)

```
amov src arr from to dst
```

The `amov` instructions overwrite the range `from`-`to` of `arr` with
`src` and set the resulting value to `dst`. For example, if `arr` has
the value `0xaabbccdd` and `src` is `0x22` the following instruction
sets `dst` to `0xaa22ccdd`:

```
amov src arr 8 16 dst
```

In this example we assume that `src` and `arr` bit indices are counted
from left.

### opcode phi (0x2f)
### opcode ret (0x30)
### opcode circ (0x31)
### opcode builtin (0x32)
### opcode gc (0x33)
