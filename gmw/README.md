# Performance Evaluation

## AES128-GCM Benchmarks -- 512B payload, 5B AAD

| Version         | 2-Party | 3-Party | 4-Party |
| ------:         | ------: | ------: | ------: |
| Baseline        | 6.784   | 10.080  | 14.046  |
| Batched Triples | 3.202   | 7.331   | 11.248  |
| AND Layering    | 2.142   | 4.515   | 6.574   |

# AES128-GCM -- 256 bytes plaintext, 5 bytes AAD

| Network   | Ping    | Yao Stream | Yao Circ  | GMW-2     |
| :-------- | ------: | ---------: | --------: | --------: |
| localhost | 0.451   | 889.038ms  | 448.613ms | 979.478ms |
| WAN       | 104.957 | 28.013s    | 5.372s    | 4m20.943s |

``` text
AND: #batches=2364, max=3286, AND/batch=288.56
```

2364 batches times 104.956ms roundtrip = 248s = 4m8s

## 2-Party Addition

| Bits | GMW Depth | GMW AND Count | Yao Depth | Yao AND Count |
| ---: | --------: | ------------: | --------: | ------------: |
| 32   | 5         | 249           | 93        | 31            |
| 64   | 6         | 631           | 189       | 63            |
| 128  | 7         | 1525          | 381       | 127           |
| 256  | 8         | 3571          | 765       | 255           |
| 512  | 9         | 8177          | 1533      | 511           |
| 1024 | 10        | 18415         | 3069      | 1023          |

## 2-Party Subtraction

| Bits | GMW Depth | GMW AND Count | Yao Depth | Yao AND Count |
| ---: | --------: | ------------: | --------: | ------------: |
| 32   | 5         | 249           | 96        | 32            |
| 64   | 6         | 631           | 192       | 64            |
| 128  | 7         | 1525          | 384       | 128           |
| 256  | 8         | 3571          | 768       | 256           |
| 512  | 9         | 8177          | 1536      | 512           |
| 1024 | 10        | 18415         | 3072      | 1024          |

## 2-Party Multiplication

| Bits | GMW Depth | GMW AND Count | Yao Depth | Yao AND Count |
| ---: | --------: | ------------: | --------: | ------------: |
| 32   | 14        | 1196          | 121       | 1054          |
| 64   | 17        | 4680          | 218       | 4006          |
| 128  | 19        | 17919         | 411       | 14081         |
| 256  | 22        | 69460         | 796       | 47058         |
| 512  | 25        | 271670        | 1565      | 152028        |
| 1024 | 27        | 1069875       | 3102      | 480149        |

## 2-Party Division

| Bits | GMW Depth | GMW AND Count | Yao Depth | Yao AND Count |
| ---: | --------: | ------------: | --------: | ------------: |
| 16   | 121       | 8927          | 257       | 482           |
| 32   | 189       | 39104         | 1024      | 1986          |
| 64   | 282       | 168708        | 4097      | 8066          |
| 128  | 435       | 730740        | 16385     | 32514         |
| 256  | 693       | 3172070       | 65537     | 130562        |

## 2-Party Comparison of UintN Numbers

| Bits | Depth | AND Count |
| ---: | ----: | --------: |
| 32   | 5     | 31        |
| 64   | 6     | 63        |
| 128  | 7     | 127       |
| 256  | 8     | 255       |
| 512  | 9     | 511       |
| 1024 | 10    | 1023      |

# Implementation Notes

## OR Gate Elimination

| A   | B   | A∨B | A⊕B | A∧B | (A⊕B)⊕(A∧B)  |
| --- | --- | --- | --- | --- | ------------ |
| 0   | 0   | 0   | 0   | 0   | 0            |
| 0   | 1   | 1   | 1   | 0   | 1            |
| 1   | 0   | 1   | 1   | 0   | 1            |
| 1   | 1   | 1   | 0   | 1   | 1            |
