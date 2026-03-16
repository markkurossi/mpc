# circ-iter

The `circ-iter` binary compiles circuits with different input sizes
and prints the resulting circuit statistics. As an example, let's
iterate the addition circuit sizes for different inputs. The circuit
is:

``` go
package main

func main(g, e uint) uint {
	return g + e
}
```

Now we iterate over power-of-two input sizes from 8 to 4096 bits:

``` shell
$ ./circ-iter -start 8 -end 4096 -step x2 -i i -pi i add.mpcl
Iter,XOR,NonXOR,Cost,Depth
8,27,7,14,21
16,59,15,30,45
32,123,31,62,93
64,251,63,126,189
128,507,127,254,381
256,1019,255,510,765
512,2043,511,1022,1533
1024,4091,1023,2046,3069
2048,8187,2047,4094,6141
4096,16379,4095,8190,12285
```

For linear iteration, change the `-step` argument to a numerical
increment:


``` shell
$ ./circ-iter -start 8 -end 128 -step 8 -i i -pi i add.mpcl
Iter,XOR,NonXOR,Cost,Depth
8,27,7,14,21
16,59,15,30,45
24,91,23,46,69
32,123,31,62,93
40,155,39,78,117
48,187,47,94,141
56,219,55,110,165
64,251,63,126,189
72,283,71,142,213
80,315,79,158,237
88,347,87,174,261
96,379,95,190,285
104,411,103,206,309
112,443,111,222,333
120,475,119,238,357
128,507,127,254,381
```

The iterator operates on bits. If the iterated argument is, for
example, a byte array, you can specify the iterator size with the
`-size` option. As an example, we iterate the AES128-GCM circuit
sizes.

``` go
type Garbler struct {
	key   [16]byte
	nonce [gcm.NonceSize]byte
	plain []byte
	aad   [5]byte
}

type Evaluator struct {
	key [16]byte
}

func main(g Garbler, e Evaluator) []byte {
	var key [16]byte

	for i := 0; i < len(key); i++ {
		key[i] = g.key[i] ^ e.key[i]
	}
	return gcm.SealAES128(key, g.nonce, g.plain, g.aad)
}
```

We iterate the plain-text input sizes from 8 to 128 bytes with 8 byte
increments:

``` shell
$ ./circ-iter -start 8 -end 128 -step 8 -size 8 -i 0,0,i,0 -pi i aesgcm.mpcl
Iter,XOR,NonXOR,Cost,Depth
8,294288,98217,196434,1758
16,295811,98473,196946,1758
24,406695,137130,274260,2270
32,408217,137386,274772,2270
40,519100,176043,352086,2782
48,520623,176299,352598,2782
56,631507,214956,429912,3294
64,633028,215212,430424,3294
72,743912,253869,507738,3806
80,745435,254125,508250,3806
88,856320,292782,585564,4318
96,857842,293038,586076,4318
104,968725,331695,663390,4830
112,970248,331951,663902,4830
120,1081132,370608,741216,5342
128,1082652,370864,741728,5342
```

## Usage

The `circ-iter` program accepts the following arguments:

 - `-workers`: number of worker threads
 - `-start`: iterator start value
 - `-end`: iterator end value (inclusive)
 - `-step`: iterator step: _integer_ or `xNum` for `iter *= Num`
 - `-size`: iterator size multiplier
 - `-i`: comma-separated list of peer-0 inputs
 - `-pi`: comma-separated list of peer-1 inputs
 - `-gmw`: compiler for the GMW protocol (optimizes circuit AND depth)

The `-i` and `-pi` arguments take a comma-separated list of input
sizes. Non-numeric values (any) indicate where the iterator value is
substituted. Currently, only one iterator value is supported.
