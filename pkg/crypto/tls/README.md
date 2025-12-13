# Circuit sizes

The columns are as follows:
 - Plain Size: the length of the plaintext data
 - ChaCha20 Gates: total number of gates in ChaCha20Poly1305 circuit
 - ChaCha20 Non-XOR: number of non-XOR gates (AND, OR, INV) in
   ChaCha20Poly1305 circuit
 - AES Gates: total number of gates in AES128-GCM circuit
 - AES non-XOR: number of non-XOR gates (AND, OR, INV) in AES128-GCM
   circuit

All tests had also 5 bytes of additional data (AAD).

| Plain Size | ChaCha20 Gates | ChaCha20 Non-XOR | AES Gates | AES Non-XOR |
|-----------:|---------------:|-----------------:|----------:|------------:|
|         16 |         410485 |           107056 |    451404 |      116390 |
|         32 |         509125 |           136413 |    565962 |      148903 |
|         64 |         706409 |           195128 |    795078 |      213929 |
|        128 |        1100981 |           312559 |   1253310 |      343981 |
|        256 |        1890129 |           547422 |   2169774 |      604085 |
|        512 |        3468429 |          1017149 |   4002702 |     1124293 |
|       1024 |        6625033 |          1956604 |   7668558 |     2164709 |
