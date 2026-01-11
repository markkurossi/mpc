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

| Algorithm    |      ns/op |   ops/s |
| :----------- | ---------: | ------: |
| RSA-1024     |    1256961 |     796 |
| RSA-2048     |    7785958 |     128 |
| CO-batch-1   |     170791 |    5855 |
| CO-batch-2   |     269399 |    7424 |
| CO-batch-4   |     468161 |    8544 |
| CO-batch-8   |     877664 |    9115 |
| CO-batch-16  |    1706184 |    9378 |
| CO-batch-32  |    3273137 |    9777 |
| CO-batch-64  |    6480310 |    9876 |
| CO-batch-128 |   12845639 |    9964 |

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
