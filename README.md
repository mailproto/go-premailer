# go-premailer

A Go CSS inliner for HTML email, matching the behaviour of
[juice](https://github.com/Automattic/juice) v12.

```go
out, err := premailer.Inline(`<style>p{color:red}</style><p>hi</p>`)
// <p style="color: red;">hi</p>
```

Email clients strip `<style>` blocks, so CSS has to be moved into `style`
attributes before sending. juice is the reference implementation the
JavaScript ecosystem uses; this ports its semantics to Go.

## Performance

On a 99 KB table-based marketing email (1,500 elements, 185 selectors),
Apple M4 Max, Go 1.26 / Node 22.22:

| | ms/doc | |
|---|---|---|
| **this package** | **2.0** | |
| Rust `css-inline` 0.21 | 4.8 | measured previously, same document |
| `vanng822/go-premailer` | 7.3 | |
| Node `juice` 12.1.3 | 28.4 | |

Roughly 14x faster than juice on identical bytes. Most of the gap is selector
matching: juice runs one full document traversal per rule, which is
O(rules x nodes). Here rules are bucketed by the id, class or tag of their
rightmost compound, so a node only tests rules it can actually reach — the
index alone took the document from 3.3 ms to 1.9 ms.

`make bench` runs the Go side, `make bench-node` runs juice over the same
input, and `TestGeneratorsAgree` asserts the two corpora are byte-identical
so the comparison cannot silently drift.

## Correctness

juice itself is the oracle. `testdata/in` holds inputs, `testdata/out` holds
juice's output for them, and the suite asserts **raw byte equality with no
normalization**. 118 of 119 fixtures match exactly; the suite prints a parity
percentage on every run and `testdata/skip.txt` lists anything that does not.

```
make test             # parity suite, race detector on
make goldens-verify   # committed goldens still match pinned juice 12.1.3
make goldens          # regenerate after a deliberate juice upgrade
```

Goldens are written only by `testdata/oracle/generate.mjs`. There is no
`-update` flag on the Go side: regenerating expectations from the code under
test would certify the implementation against itself.

`FuzzCSS` checks that inlining never panics and is deterministic. Determinism
is the one that earns its keep — the cascade depends on insertion order, so
any accidental reliance on Go map iteration shows up there and nowhere else.

### Why the HTML parser is not `html.Parse`

Byte-exactness rules out `x/net/html`'s `Parse` and `Render`. juice parses
with htmlparser2 in non-XML mode, which performs no HTML5 tree construction
and leaves entities undecoded, so a spec-compliant round trip diverges on
almost every real document:

| input | juice | `x/net/html` |
|---|---|---|
| `<table><tr><td>x` | unchanged | inserts `<tbody>` |
| `<p>x</p>` | unchanged | adds `<html><head></head><body>` |
| `<br>` | unchanged | `<br/>` |
| `a &nbsp; &copy;` | unchanged | `&nbsp;` becomes U+00A0 |
| `href="?a=1&b=2"` | unchanged | `&` becomes `&amp;` |
| `<p class="">` | `<p class>` | `class=""` |

A mail client can see all of those. `parse.go` builds an htmlparser2-shaped
tree over `html.NewTokenizer` instead, reading raw token bytes rather than the
decoding accessors. It is also the faster path: the tokenizer is about twice
as quick as the full parser, and nothing is escaped on the way out.

## Deliberate divergences from juice

Three cases where matching juice would mean matching a bug. Each is covered by
a test in `divergence_test.go`.

1. **Liquid templates.** juice's default `codeBlocks` covers Handlebars and
   EJS but not Liquid's `{% %}`, so it corrupts
   `<td {% if x %}class="a"{% endif %}>` into `class="a" endif %}`. Liquid is
   in the default set here.
2. **Cyclic custom properties.** `--a:var(--b);--b:var(--a)` recurses until
   juice's stack overflows, which makes a cyclic stylesheet a denial of
   service. Resolution here is depth-capped.
3. **Encoded quotes in a style attribute.** juice throws a `CssSyntaxError` on
   `style="font-family:&quot;A B&quot;"`, because `decodeStyleAttributes`
   defaults off and the raw entity reaches postcss. This package inlines it.

## Status

The cascade, at-rule preservation, attribute promotion, CSS variables,
Selectors L4 specificity and `:is()`/`:where()` are implemented. Not yet:
CSS nesting flattening, `::before`/`::after` materialization (off by default
in juice too), and `/* juice ignore */` directives.

External resources are out of scope. This package does no network I/O; resolve
`<link rel=stylesheet>` yourself and pass the CSS via `ExtraCSS`.
