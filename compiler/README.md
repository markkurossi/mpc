# Typecast

Typecast is implemented for [static values](ast/eval.go) `Call.Eval`
and [dynamic values](ast/ssagen.go) `Call.SSA`.

## Static Cast

Value identity for const values is computed from the value's `Name`,
`Scope`, and `Version`. These values remain the same for all constant
value instances so the static cast can creates a new constant instance
with the casted `Type`.
