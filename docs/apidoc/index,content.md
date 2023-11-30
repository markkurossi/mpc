# Multi-Party Computation Language (MPCL)

Multi-Party Computation Language is a high-level programming language
that implements secure two-party computation with [Garbled Circuit
Protocol](https://en.wikipedia.org/wiki/Garbled_circuit). The
[Go](https://golang.org) programming language highly inspires the
MCPL. Therefore, it is easy to use existing Go functions and programs
in MPCL with minor modifications. However, the goal is not to make
MPCL fully compatible with Go.

## Basic Application Invocation

Secure multi-party computation has two parties: garbler and
evaluator. Both parties run the `garbled` application with
appropritate arguments:

`-e`
: specifies circuit _evaluator_ mode. If the `-e` option is not
  specified, the peer runs in the _garbler_ mode.

`-i`
: defines the peer input values. Multiple input values can be
  specified by providing the `-i` option multiple times, or by
  separating input values with comma.

For example, this is how you can run [Yao's Millionaires'
Problem](https://en.wikipedia.org/wiki/Yao%27s_Millionaires%27_Problem). The
evaluator is started with input 800000:

```
$ apps/garbled/garbled -e -i 800000 apps/garbled/examples/millionaire.mpcl
 - In1: a{1,0}i64:int64
 + In2: b{1,0}i64:int64
 - Out: %_{0,1}b1:bool1
 -  In: [800000]
Listening for connections at :8080
```

The garbler is started with input 750000:

```
$ apps/garbled/garbled -i 750000 apps/garbled/examples/millionaire.mpcl
 + In1: a{1,0}i64:int64
 - In2: b{1,0}i64:int64
 - Out: %_{0,1}b1:bool1
 -  In: [75000]
Result[0]: false
```

Both garbler and evaluator return the result `false` since garbler's
input is smaller than evaluator's input.

## Streaming Mode

## Command Line Arguments

The garbled application takes the following command line arguments:

`-O`
: optimization level (default 1 enabling all current optimizations).

`-circ`
: compile inputs to circuit format.

`-cpuprofile`
: write cpu profile to the specified file.

`-d`
: enable diagnostics outputs.

`-dot`
: generate Graphviz DOT output.

`-e`
: specifies circuit _evaluator_ / _garbler_ mode. The circuit
  evaluator creates a TCP listener and waits for garblers to connect
  with computation.

`-format`
: specifies circuit format for the `-circ` output file. Possible
  values are: `mpclc` (default), `bristol`.

`-i`
: specifies comma-separated input values for the circuit.

`-memprofile`
: write memory profile to the specified file.

`-ssa`
: compile MPCL input to SSA assembly.

`-stream`
: streaming mode.

`-svg`
: generate SVG output.

`-v`
: enabled verbose output.
