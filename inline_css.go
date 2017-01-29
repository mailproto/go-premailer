package premailer

import (
	"bytes"
	"crypto/md5"
	"fmt"

	"golang.org/x/net/html"
)

func (p *Premailer) ToInlineCSS() (string, error) {

	inlined, err := Douceur(string(p.orig))
	if err != nil {
		return "", err
	}

	if p.RemoveScripts {
		removeAllElement(inlined, "script")
	}

	if p.RemoveClasses || p.RemoveComments {
		eachElement(inlined, func(n *html.Node) bool {
			if p.RemoveComments && n.Type == html.CommentNode {
				if n.Parent != nil {
					n.Parent.RemoveChild(n)
				}
			} else if p.RemoveClasses && n.Type == html.ElementNode {
				var attrs []html.Attribute
				for _, attr := range n.Attr {
					if attr.Key == "class" {
						continue
					}
					attrs = append(attrs, attr)
				}
				n.Attr = attrs
			}
			return true
		})
	}

	if p.RemoveIDs {
		targets := make(map[string]struct{})

		// find all anchor's targets and hash them
		eachElement(inlined, func(n *html.Node) bool {
			if n.Type == html.ElementNode && n.Data == "a" {
				var attrs []html.Attribute
				for _, attr := range n.Attr {
					if attr.Key == "href" && attr.Val[0] == '#' {
						targets[attr.Val[1:]] = struct{}{}
						attr.Val = fmt.Sprintf("#%x", md5.Sum([]byte(attr.Val[1:])))
					}
					attrs = append(attrs, attr)
				}
				n.Attr = attrs
			}
			return true
		})

		// hash ids that are links target, delete others
		eachElement(inlined, func(n *html.Node) bool {
			var attrs []html.Attribute
			for _, attr := range n.Attr {
				if attr.Key == "id" {
					if _, ok := targets[attr.Val]; ok {
						attr.Val = fmt.Sprintf("%x", md5.Sum([]byte(attr.Val)))
					} else {
						continue
					}
				}
				attrs = append(attrs, attr)
			}
			n.Attr = attrs
			return true
		})
	}

	if p.ResetContentEditable {
		eachElement(inlined, func(n *html.Node) bool {
			removeAttribute(n, "contenteditable")
			return true
		})
	}

	//         @processed_doc = doc
	//         if is_xhtml?
	//           # we don't want to encode carriage returns
	//           @processed_doc.to_xhtml(:encoding => @options[:output_encoding]).gsub(/&\#(xD|13);/i, "\r")
	//         else
	//           @processed_doc.to_html(:encoding => @options[:output_encoding])
	//         end
	//       end

	p.processed = inlined

	var buf bytes.Buffer
	err = html.Render(&buf, p.processed)
	if err != nil {
		return "", err
	}

	return string(buf.Bytes()), nil

}

//       # Create a <tt>style</tt> element with un-mergable rules (e.g. <tt>:hover</tt>)
//       # and write it into the <tt>body</tt>.
//       #
//       # <tt>doc</tt> is an Nokogiri document and <tt>unmergable_css_rules</tt> is a Css::RuleSet.
//       #
//       # @return [::Nokogiri::XML] a document.
//       def write_unmergable_css_rules(doc, unmergable_rules) # :nodoc:
//         styles = unmergable_rules.to_s

//         unless styles.empty?
//           style_tag = "<style type=\"text/css\">\n#{styles}</style>"
//           unless (body = doc.search('body')).empty?
//             if doc.at_css('body').children && !doc.at_css('body').children.empty?
//               doc.at_css('body').children.before(::Nokogiri::XML.fragment(style_tag))
//             else
//               doc.at_css('body').add_child(::Nokogiri::XML.fragment(style_tag))
//             end
//           else
//             doc.inner_html = style_tag += doc.inner_html
//           end
//         end
//         doc
//       end

//       # Converts the HTML document to a format suitable for plain-text e-mail.
//       #
//       # If present, uses the <body> element as its base; otherwise uses the whole document.
//       #
//       # @return [String] a plain text.
//       def to_plain_text
//         html_src = ''
//         begin
//           html_src = @doc.at("body").inner_html
//         rescue;
//         end

//         html_src = @doc.to_html unless html_src and not html_src.empty?
//         convert_to_text(html_src, @options[:line_length], @html_encoding)
//       end

//       # Gets the original HTML as a string.
//       # @return [String] HTML.
//       def to_s
//         if is_xhtml?
//           @doc.to_xhtml(:encoding => nil)
//         else
//           @doc.to_html(:encoding => nil)
//         end
//       end

//       # Load the HTML file and convert it into an Nokogiri document.
//       #
//       # @return [::Nokogiri::XML] a document.
//       def load_html(input) # :nodoc:
//         thing = nil

//         # TODO: duplicate options
//         if @options[:with_html_string] or @options[:inline] or input.respond_to?(:read)
//           thing = input
//         elsif @is_local_file
//           @base_dir = File.dirname(input)
//           thing = File.open(input, 'r')
//         else
//           thing = open(input)
//         end

//         if thing.respond_to?(:read)
//           thing = thing.read
//         end

//         return nil unless thing
//         doc = nil

//         # Handle HTML entities
//         if @options[:replace_html_entities] == true and thing.is_a?(String)
//           HTML_ENTITIES.map do |entity, replacement|
//             thing.gsub! entity, replacement
//           end
//         end
//         # Default encoding is ASCII-8BIT (binary) per http://groups.google.com/group/nokogiri-talk/msg/0b81ef0dc180dc74
//         # However, we really don't want to hardcode this. ASCII-8BIT should be the default, but not the only option.
//         if thing.is_a?(String) and RUBY_VERSION =~ /1.9/
//           thing = thing.force_encoding(@options[:input_encoding]).encode!
//           doc = ::Nokogiri::HTML(thing, nil, @options[:input_encoding]) { |c| c.recover }
//         else
//           default_encoding = RUBY_PLATFORM == 'java' ? nil : 'BINARY'
//           doc = ::Nokogiri::HTML(thing, nil, @options[:input_encoding] || default_encoding) { |c| c.recover }
//         end

//         # Fix for removing any CDATA tags from both style and script tags inserted per
//         # https://github.com/sparklemotion/nokogiri/issues/311 and
//         # https://github.com/premailer/premailer/issues/199
//         %w(style script).each do |tag|
//           doc.search(tag).children.each do |child|
//             child.swap(child.text()) if child.cdata?
//           end
//         end

//         doc
//       end

//     end
//   end
// end
