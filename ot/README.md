# Oblivious Transfer

This module implements the Oblivous Transfer with the following
algorithms:

 - RSA: simple RSA encryption based OT. Each transfer requires one RSA
   operation.
 - Chou Orlandi OT: Diffie-Hellman - like fast OT algorithm.

## Pure CO helpers

The CO implementation now exposes pure helper functions so applications can
prepare, encrypt, and decrypt OT payloads without depending on the streaming
`IO` API.  Use:

1. `GenerateCOSenderSetup` to sample the sender's randomness and public point.
2. `EncryptCOCiphertexts` to turn `Wire` labels and evaluator points into OT ciphertexts.
3. `BuildCOChoices` to build evaluator curve points deterministically from the sender's public key.
4. `DecryptCOCiphertexts` to decode the evaluator's chosen labels.

These helpers accept any `io.Reader` for randomness, which makes deterministic
testing straightforward.  The streaming `CO` type is now a small wrapper around
the same helpers, so both styles always share the exact same cryptographic core.

## Performance

``` text
goarch: amd64
pkg: github.com/markkurossi/mpc/ot
cpu: Intel(R) Core(TM) i5-8257U CPU @ 1.40GHz
```

| Algorithm       | ns/op      | ops/s   |
| :-----------    | ---------: | ------: |
| RSA-2048        | 8,086,829  | 124     |
| CO              | 152,745    | 6,545   |
| COT semi-honest | 3,007      | 3,007   |
| COT malicious   | 3,703      | 270,000 |

# IKNP OT Extension

The `COTSender` and `COTReceiver` implement correlated OT with the
`IKNP` protocol.

## TODO

 - [x] malicious IKNP
 - [x] COT send / receive
 - [ ] random send / receive
 - [x] rename `COTSender` and `COTReceiver` to `IKNPSender` and `IKNPReceiver`
 - [x] new `COTSender` and `COTReceiver` for correlated OT send and receive
 - [ ] new `ROTSender` and `ROTReceiver` for random OT send and receive
 - [ ] [SoftSpokenOT](https://eprint.iacr.org/2022/192)
