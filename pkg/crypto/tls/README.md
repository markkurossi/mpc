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
|         16 |         305721 |            77431 |    451404 |      116390 |
|         32 |         368109 |            96600 |    565962 |      148903 |
|         64 |         492889 |           134939 |    795078 |      213929 |
|        128 |         742453 |           211618 |   1253310 |      343981 |
|        256 |        1241585 |           364977 |   2169774 |      604085 |
|        512 |        2239853 |           671696 |   4002702 |     1124293 |
|       1024 |        4236393 |          1285135 |   7668558 |     2164709 |
