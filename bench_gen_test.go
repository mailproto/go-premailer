package premailer

import (
	"fmt"
	"strings"
)

// GenEmail builds a realistic table-based marketing email of roughly the
// requested byte size, with a <style> block of `nRules` rules.
func GenEmail(nRules, nRows int) string {
	var css strings.Builder
	css.WriteString("body{margin:0;padding:0;background:#f4f4f4;font-family:Helvetica,Arial,sans-serif}\n")
	css.WriteString(".wrapper{width:100%;background-color:#ffffff}\n")
	css.WriteString("table{border-collapse:collapse}\n")
	css.WriteString("a{color:#1a73e8;text-decoration:underline}\n")
	css.WriteString("h1,h2,h3{margin:0 0 12px 0;line-height:1.3}\n")
	for i := 0; i < nRules; i++ {
		fmt.Fprintf(&css, ".c%d{color:#%06x;font-size:%dpx;padding:%dpx}\n", i, i*7919%0xffffff, 10+i%12, i%20)
		fmt.Fprintf(&css, "td.c%d span{font-weight:%d00;letter-spacing:%dpx}\n", i, 1+i%9, i%3)
		fmt.Fprintf(&css, "#id%d > p{margin-bottom:%dpx}\n", i, i%16)
	}
	css.WriteString("@media only screen and (max-width:600px){.wrapper{width:100%!important}.c1{font-size:18px!important}}\n")
	css.WriteString("a:hover{color:#ff0000}\n")

	var body strings.Builder
	body.WriteString(`<table class="wrapper" width="600" cellpadding="0" cellspacing="0" role="presentation"><tbody>`)
	for i := 0; i < nRows; i++ {
		fmt.Fprintf(&body, `<tr id="id%d"><td class="c%d" align="left" valign="top">`+
			`<h2>Section %d heading text</h2>`+
			`<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor `+
			`incididunt ut labore et dolore magna aliqua number %d.</p>`+
			`<span>Highlighted copy %d</span> `+
			`<a href="https://example.com/click/%d?utm_source=email">Read more</a>`+
			`<img src="https://cdn.example.com/img/%d.png" width="120" height="80" alt="promo %d">`+
			`</td></tr>`, i, i%nRules, i, i, i, i, i, i)
	}
	body.WriteString(`</tbody></table>`)

	return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>Benchmark Email</title><style type="text/css">` + css.String() + `</style></head>` +
		`<body>` + body.String() + `</body></html>`
}
