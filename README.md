# go-premailer

A Go CSS inliner for HTML email, matching the behaviour of
[juice](https://github.com/Automattic/juice) v12.

```go
out, err := premailer.Inline(`<style>p{color:red}</style><p>hi</p>`)
// <p style="color: red;">hi</p>
```

## Why another one

Email clients strip `<style>` blocks, so CSS has to be moved into `style`
attributes before sending. juice is the reference implementation the JavaScript
ecosystem uses; this is a port of its semantics to Go.

Output is intended to be **byte-identical to juice's**. That constraint drives
an unusual choice: the HTML parser here deliberately does not implement HTML5
tree construction. juice parses with htmlparser2, which does not insert
`<tbody>`, does not synthesise `<html>`/`<head>`/`<body>`, and does not decode
entities. Those differences are visible to a mail client, so matching them
matters more than spec compliance. See `parse.go`.

## Status

Under active development. `testdata/skip.txt` lists what is not yet
implemented; the test suite prints a parity percentage on every run.

## Testing

juice itself is the oracle. `testdata/in` holds inputs, `testdata/out` holds
juice's output for them, and the suite asserts raw byte equality with no
normalization.

```
make test             # parity suite
make goldens-verify   # committed goldens still match pinned juice 12.1.3
make goldens          # regenerate after a deliberate juice upgrade
```

Goldens are written only by `testdata/oracle/generate.mjs`. There is no
`-update` flag on the Go side: regenerating expectations from the code under
test would certify the implementation against itself.
