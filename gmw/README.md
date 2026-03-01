# OR Gates

| A   | B   | A∨B | A⊕B | A∧B | (A⊕B)⊕(A∧B)  |
| --- | --- | --- | --- | --- | ------------ |
| 0   | 0   | 0   | 0   | 0   | 0            |
| 0   | 1   | 1   | 1   | 0   | 1            |
| 1   | 0   | 1   | 1   | 0   | 1            |
| 1   | 1   | 1   | 0   | 1   | 1            |

# AES128-GCM Benchmarks -- 512B payload, 5B AAD

| Version         | 2-Party | 3-Party | 4-Party |
| ------:         | ------: | ------: | ------: |
| Baseline        | 6.784   | 10.080  | 14.046  |
| Batched Triples | 3.202   | 7.331   | 11.248  |

``` text
┌─────────┬──────────────┬────────┬────────┬───────┐
│ Op      │         Time │      % │    Onl │  Offl │
├─────────┼──────────────┼────────┼────────┼───────┤
│ Network │              │        │   124B │  57kB │
│ Compile │ 1.056761733s │  9.87% │        │       │
│ Eval    │ 9.649947523s │ 90.13% │   89MB │  74kB │
│ Total   │ 10.70672872s │        │   89MB │ 132kB │
│ ├╴Sent  │              │ 50.00% │   44MB │  66kB │
│ ├╴Rcvd  │              │ 50.00% │   44MB │  66kB │
│ ╰╴Flcd  │              │        │ 105436 │ 66313 │
└─────────┴──────────────┴────────┴────────┴───────┘
Result[0]: db434ff14860cba2400c033baea7928570240b72ea3e5b968acf19ae477ec4ee70e069129d4d51d562a47c984509ba90474b0e6cc81bd5682d2bd682886286c124a31597ab4ec73aef9f3db90102a3e05cd0ab58878a6adf75c3a89dc679285d84fa5ee618fb2fbb5ab968f9914eacbbee9a7100a0241acc724b04409bfed6843bb65dc837a91971213bb516e63f0811520a1fb3e6c581110ac34ece0d85fc1b536022b644555072757074f3bf62300af9aa4d8c18d477e77c349549f84f9e700ca8366c48a6e87b42036b5725d3f8f02912d74c425c11d0b3530a34c9595ccaefe6142a5ca559f20954b72c483e378f45e2482d1247379eb3a6e07751d9fbb4f8b64351db2aa15f65ff12801d55a6f698ebb54be831389957841f59d9212666bc4871b293becf23a2c7d21737a2d658356dc2110ab711de2a45b683662ab4ffb39d2b9e116d3ac65843f38a6065dd557a6b0bec8b50872caba5f7a487f1ca0cdbcfe7ef754aca7b5b618222986e2846d69def254c30ee6ceeab404ce488fd70b139789845af73463d3b30c8a3fb28e7e0843b6ef549ec54a279f8f62b32a336036e38a43fd99bfa3b1efc710ac37785699c0b93ea32233946f8203f7f15135ef89aab98ee28a33fcf4b6710041ec4ebde3d0854e5741b704dd9e17c2b810857dd99550ca0147c668ab5c8665b72b3bd65d200d01d08a1de41852a475fa2f7f6631dd7a554ff2b63c35f8c72e7ccd64e
```

Batched beaver triple generation and AND:

``` text
┌─────────┬──────────────┬────────┬───────┬──────────┐
│ Op      │         Time │      % │   Onl │     Offl │
├─────────┼──────────────┼────────┼───────┼──────────┤
│ Network │              │        │   95B │     57kB │
│ Compile │ 817.564571ms │ 12.10% │       │          │
│ Eval    │  5.93943032s │ 87.90% │   3MB │    122MB │
│ Total   │ 6.757006354s │        │   3MB │    122MB │
│ ├╴Sent  │              │ 49.99% │   1MB │     61MB │
│ ├╴Rcvd  │              │ 50.01% │   1MB │     61MB │
│ ╰╴Flcd  │              │        │ 70294 │ 61030791 │
└─────────┴──────────────┴────────┴───────┴──────────┘
tripleBatch: read tcp 127.0.0.1:61490->127.0.0.1:8082: use of closed network connection
Result[0]: db434ff14860cba2400c033baea7928570240b72ea3e5b968acf19ae477ec4ee70e069129d4d51d562a47c984509ba90474b0e6cc81bd5682d2bd682886286c124a31597ab4ec73aef9f3db90102a3e05cd0ab58878a6adf75c3a89dc679285d84fa5ee618fb2fbb5ab968f9914eacbbee9a7100a0241acc724b04409bfed6843bb65dc837a91971213bb516e63f0811520a1fb3e6c581110ac34ece0d85fc1b536022b644555072757074f3bf62300af9aa4d8c18d477e77c349549f84f9e700ca8366c48a6e87b42036b5725d3f8f02912d74c425c11d0b3530a34c9595ccaefe6142a5ca559f20954b72c483e378f45e2482d1247379eb3a6e07751d9fbb4f8b64351db2aa15f65ff12801d55a6f698ebb54be831389957841f59d9212666bc4871b293becf23a2c7d21737a2d658356dc2110ab711de2a45b683662ab4ffb39d2b9e116d3ac65843f38a6065dd557a6b0bec8b50872caba5f7a487f1ca0cdbcfe7ef754aca7b5b618222986e2846d69def254c30ee6ceeab404ce488fd70b139789845af73463d3b30c8a3fb28e7e0843b6ef549ec54a279f8f62b32a336036e38a43fd99bfa3b1efc710ac37785699c0b93ea32233946f8203f7f15135ef89aab98ee28a33fcf4b6710041ec4ebde3d0854e5741b704dd9e17c2b810857dd99550ca0147c668ab5c8665b72b3bd65d200d01d08a1de41852a475fa2f7f6631dd7a554ff2b63c35f8c72e7ccd64e
```
