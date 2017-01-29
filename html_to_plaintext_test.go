package premailer_test

import (
	"strings"
	"testing"

	premailer "github.com/mailproto/go-premailer"
)

func checkPlaintext(t *testing.T, expect, html string) {
	if plain := premailer.ConvertToText(html, premailer.DefaultLineLength, premailer.DefaultCharset); strings.TrimSpace(plain) != expect {
		t.Errorf("Wrong conversion of `%v`, want: %v got: %v", html, expect, plain)
	}
}

func TestToPlaintextWithFragment(t *testing.T) {
	t.Skip("Requires full premailer pass")
	//     def test_to_plain_text_with_fragment
	//     premailer = Premailer.new('<p>Test</p>', :with_html_string => true)
	//     assert_match /Test/, premailer.to_plain_text
	// end
}

func TestToPlaintextWithBody(t *testing.T) {
	t.Skip("Requires full premailer pass")
	// def test_to_plain_text_with_body
	//     html = <<END_HTML
	//     <html>
	//     <title>Ignore me</title>
	//     <body>
	// 		<p>Test</p>
	// 		</body>
	// 		</html>
	// END_HTML

	//     premailer = Premailer.new(html, :with_html_string => true)
	//     assert_match /Test/, premailer.to_plain_text
	// end
}

func TestToPlaintextWithMalformedBody(t *testing.T) {
	t.Skip("Requires full premailer pass")
	//   def test_to_plain_text_with_malformed_body
	//     html = <<END_HTML
	//     <html>
	//     <title>Ignore me</title>
	//     <body>
	// 		<p>Test
	// END_HTML

	//     premailer = Premailer.new(html, :with_html_string => true)
	//     assert_match /Test/, premailer.to_plain_text
	//   end
}

func TestSpecialChars(t *testing.T) {
	checkPlaintext(t, "cédille garçon & à ñ", "c&eacute;dille gar&#231;on &amp; &agrave; &ntilde;")
}

func TestStrippingWhitespace(t *testing.T) {
	checkPlaintext(t, "text\ntext", "  \ttext\ntext\n")
	checkPlaintext(t, "a\na", "  \na \n a \t")
	checkPlaintext(t, "a\n\na", "  \na \n\t \n \n a \t")
	checkPlaintext(t, "test text", "test text&nbsp;")
	checkPlaintext(t, "test text", "test        text")
}

func TestWrappingSpans(t *testing.T) {
	t.Skip("Requires full premailer pass")
	//        html = <<END_HTML
	//     <html>
	//     <body>
	// 		<p><span>Test</span>
	// 		<span>line 2</span>
	// 		</p>
	// END_HTML

	//     premailer = Premailer.new(html, :with_html_string => true)
	// assert_match /Test line 2/, premailer.to_plain_text
}

func TestLineBreaks(t *testing.T) {
	checkPlaintext(t, "Test text\nTest text", "Test text\r\nTest text")
	checkPlaintext(t, "Test text\nTest text", "Test text\rTest text")
}

func TestLists(t *testing.T) {
	checkPlaintext(t, "* item 1\n* item 2", "<li class='123'>item 1</li> <li>item 2</li>\n")
	checkPlaintext(t, "* item 1\n* item 2\n* item 3", "<li>item 1</li> \t\n <li>item 2</li> <li> item 3</li>\n")
}

func TestStrippingHTML(t *testing.T) {
	checkPlaintext(t, "test text", "<p class=\"123'45 , att\" att=tester>test <span class='te\"st'>text</span>\n")
}

func TestStrippingIgnoredBlocks(t *testing.T) {
	t.Skip("Requires full premailer pass")
	//        html = <<END_HTML
	//     <html>
	//     <body>
	// 		<p><span>Test</span>
	// 		<span>line 2</span>
	// 		</p>
	// END_HTML

	//     premailer = Premailer.new(html, :with_html_string => true)
	// assert_match /Test line 2/, premailer.to_plain_text
}

func TestParagraphsAndBreaks(t *testing.T) {
	checkPlaintext(t, "Test text\n\nTest text", "<p>Test text</p><p>Test text</p>")
	checkPlaintext(t, "Test text\n\nTest text", "\n<p>Test text</p>\n\n\n\t<p>Test text</p>\n")
	checkPlaintext(t, "Test text\nTest text", "\n<p>Test text<br/>Test text</p>\n")
	checkPlaintext(t, "Test text\nTest text", "\n<p>Test text<br> \tTest text<br></p>\n")
	checkPlaintext(t, "Test text\n\nTest text", "Test text<br><BR />Test text")
}

func TestHeadings(t *testing.T) {
	checkPlaintext(t, "****\nTest\n****", "<h1>Test</h1>")
	checkPlaintext(t, "****\nTest\n****", "\t<h1>\nTest</h1>")
	checkPlaintext(t, "***********\nTest line 1\nTest 2\n***********", "\t<h1>\nTest line 1<br>Test 2</h1> ")
	checkPlaintext(t, "****\nTest\n****\n\n****\nTest\n****", "<h1>Test</h1> <h1>Test</h1>")
	checkPlaintext(t, "----\nTest\n----", "<h2>Test</h2>")
	checkPlaintext(t, "Test\n----", "<h3> <span class='a'>Test </span></h3>")
}

func TestWrappingLines(t *testing.T) {
	txt := premailer.ConvertToText(strings.Repeat("test ", 100), 20, "UTF-8")

	var offendingLines []int
	for i, line := range strings.Split(txt, "\n") {
		if len(line) > 20 {
			offendingLines = append(offendingLines, i)
		}
	}

	if len(offendingLines) > 0 {
		t.Errorf("Found lines longer than 20 chars: %v", offendingLines)
	}
}

func TestWrappingLinesWithSpaces(t *testing.T) {
	raw := "Long     line new line"
	expect := "Long line\nnew line"
	if plain := premailer.ConvertToText(raw, 10, premailer.DefaultCharset); strings.TrimSpace(plain) != expect {
		t.Errorf("Wrong plaintext content, want: %v got: %v", expect, plain)
	}
}

func TestImgAltTags(t *testing.T) {

	//  ensure html imag tags that aren't self-closed are parsed,
	//  along with accepting both '' and "" as attribute quotes

	//  <img alt="" /> closed
	checkPlaintext(t, "Example ( http://example.com/ )", `<a href="http://example.com/"><img src="http://example.ru/hello.jpg" alt="Example"/></a>`)

	//  <img alt=""> not closed
	checkPlaintext(t, "Example ( http://example.com/ )", `<a href="http://example.com/"><img src="http://example.ru/hello.jpg" alt="Example"></a>`)

	//  <img alt='' />
	checkPlaintext(t, "Example ( http://example.com/ )", `<a href='http://example.com/'><img src='http://example.ru/hello.jpg' alt='Example'/></a>`)

	//  <img alt=''>
	checkPlaintext(t, "Example ( http://example.com/ )", `<a href='http://example.com/'><img src='http://example.ru/hello.jpg' alt='Example'></a>`)

}

func TestLinks(t *testing.T) {

	// basic
	checkPlaintext(t, `Link ( http://example.com/ )`, `<a href="http://example.com/">Link</a>`)

	// nested html
	checkPlaintext(t, `Link ( http://example.com/ )`, `<a href="http://example.com/"><span class="a">Link</span></a>`)

	// nested html with new line
	checkPlaintext(t, `Link ( http://example.com/ )`, "<a href='http://example.com/'>\n\t<span class='a'>Link</span>\n\t</a>")

	// mailto
	checkPlaintext(t, `Contact Us ( contact@example.org )`, `<a href='mailto:contact@example.org'>Contact Us</a>`)

	// complex link
	checkPlaintext(t, `Link ( http://example.com:80/~user?aaa=bb&c=d,e,f#foo )`, `<a href="http://example.com:80/~user?aaa=bb&amp;c=d,e,f#foo">Link</a>`)

	// attributes
	checkPlaintext(t, `Link ( http://example.com/ )`, `<a title='title' href="http://example.com/">Link</a>`)

	// spacing
	checkPlaintext(t, `Link ( http://example.com/ )`, `<a href="   http://example.com/ "> Link </a>`)

	// multiple
	checkPlaintext(t, `Link A ( http://example.com/a/ ) Link B ( http://example.com/b/ )`, `<a href="http://example.com/a/">Link A</a> <a href="http://example.com/b/">Link B</a>`)

	// merge links
	checkPlaintext(t, `Link ( %%LINK%% )`, `<a href="%%LINK%%">Link</a>`)
	checkPlaintext(t, `Link ( [LINK] )`, `<a href="[LINK]">Link</a>`)
	checkPlaintext(t, `Link ( {LINK} )`, `<a href="{LINK}">Link</a>`)

	// unsubscribe
	checkPlaintext(t, `Link ( [[!unsubscribe]] )`, `<a href="[[!unsubscribe]]">Link</a>`)

	// empty link gets dropped, and shouldn`t run forever
	content := strings.Repeat("\n<p>This is some more text</p>", 15)
	checkPlaintext(t, strings.Repeat("This is some more text\n\n", 14)+"This is some more text", "<a href=\"test\"></a>"+content)

	// links that go outside of line should wrap nicely
	checkPlaintext(t, "Long text before the actual link and then LINK TEXT \n( http://www.long.link ) and then more text that does not wrap", `Long text before the actual link and then <a href="http://www.long.link"/>LINK TEXT</a> and then more text that does not wrap`)

	// same text and link
	checkPlaintext(t, `http://example.com`, `<a href="http://example.com">http://example.com</a>`)

}

// see https://github.com/alexdunae/premailer/issues/72
func TestMultipleLinksPerLine(t *testing.T) {
	html := `<p>This is <a href="http://www.google.com" >link1</a> and <a href="http://www.google.com" >link2 </a> is next.</p>`
	expect := `This is link1 ( http://www.google.com ) and link2 ( http://www.google.com ) is next.`

	if plain := premailer.ConvertToText(html, 10000, premailer.DefaultCharset); strings.TrimSpace(plain) != expect {
		t.Errorf("Wrong conversion of `%v`, want: %v got: %v", html, expect, plain)
	}
}

// see https://github.com/alexdunae/premailer/issues/72
func TestLinksWithinHeadings(t *testing.T) {
	checkPlaintext(t, "****************************\nTest ( http://example.com/ )\n****************************", "<h1><a href='http://example.com/'>Test</a></h1>")
}

//   def assert_plaintext(out, raw, msg = nil, line_length = 65)
//     assert_equal out, convert_to_text(raw, line_length), msg
//   end
// end
