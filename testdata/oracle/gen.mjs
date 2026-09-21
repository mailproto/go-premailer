// mirror of gen.go
function genEmail(nRules, nRows) {
  let css = "body{margin:0;padding:0;background:#f4f4f4;font-family:Helvetica,Arial,sans-serif}\n" +
    ".wrapper{width:100%;background-color:#ffffff}\ntable{border-collapse:collapse}\n" +
    "a{color:#1a73e8;text-decoration:underline}\nh1,h2,h3{margin:0 0 12px 0;line-height:1.3}\n";
  for (let i = 0; i < nRules; i++) {
    css += `.c${i}{color:#${((i*7919)%0xffffff).toString(16).padStart(6,'0')};font-size:${10+i%12}px;padding:${i%20}px}\n`;
    css += `td.c${i} span{font-weight:${1+i%9}00;letter-spacing:${i%3}px}\n`;
    css += `#id${i} > p{margin-bottom:${i%16}px}\n`;
  }
  css += "@media only screen and (max-width:600px){.wrapper{width:100%!important}.c1{font-size:18px!important}}\na:hover{color:#ff0000}\n";
  let body = `<table class="wrapper" width="600" cellpadding="0" cellspacing="0" role="presentation"><tbody>`;
  for (let i = 0; i < nRows; i++) {
    body += `<tr id="id${i}"><td class="c${i%nRules}" align="left" valign="top">` +
      `<h2>Section ${i} heading text</h2>` +
      `<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua number ${i}.</p>` +
      `<span>Highlighted copy ${i}</span> ` +
      `<a href="https://example.com/click/${i}?utm_source=email">Read more</a>` +
      `<img src="https://cdn.example.com/img/${i}.png" width="120" height="80" alt="promo ${i}">` +
      `</td></tr>`;
  }
  body += `</tbody></table>`;
  return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">` +
    `<meta name="viewport" content="width=device-width,initial-scale=1">` +
    `<title>Benchmark Email</title><style type="text/css">` + css + `</style></head><body>` + body + `</body></html>`;
}
export { genEmail };
