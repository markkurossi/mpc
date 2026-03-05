# Performance Evaluation

## AES128-GCM Benchmarks -- 512B payload, 5B AAD

| Version         | 2-Party | 3-Party | 4-Party |
| ------:         | ------: | ------: | ------: |
| Baseline        | 6.784   | 10.080  | 14.046  |
| Batched Triples | 3.202   | 7.331   | 11.248  |
| AND Layering    | 2.142   | 4.515   | 6.574   |

## 2-Party Addition

| Bits | Depth   | AND Count |
| ---: | ------: | --------: |
| 32   | 5       | 249       |
| 64   | 6       | 631       |
| 128  | 7       | 1525      |
| 256  | 8       | 3571      |
| 512  | 9       | 8177      |
| 1024 | 10      | 18415     |

## 2-Party Subtraction

| Bits | Depth   | AND Count |
| ---: | ------: | --------: |
| 32   | 5       | 249       |
| 64   | 6       | 631       |
| 128  | 7       | 1525      |
| 256  | 8       | 3571      |
| 512  | 9       | 8177      |
| 1024 | 10      | 18415     |

## 2-Party Multiplication

| Bits | Depth   | AND Count |
| ---: | ------: | --------: |
| 32   | 14      | 1196      |
| 64   | 17      | 4680      |
| 128  | 19      | 17919     |
| 256  | 22      | 69460     |
| 512  | 25      | 271670    |
| 1024 | 27      | 1069875   |

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
