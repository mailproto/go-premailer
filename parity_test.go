package premailer

// Differential test against juice, the reference implementation.
//
// Goldens in testdata/out are produced ONLY by testdata/oracle/generate.mjs.
// There is deliberately no -update flag here: regenerating expectations from
// the code under test would certify this implementation against itself and
// destroy the oracle. Run `make goldens` instead.
//
// The assertion is raw byte equality with no normalization. juice's output is
// what a mail client receives, so differences in entity encoding or element
// structure are real differences, not formatting. Failures run a classifier
// to label the difference, but that only shapes the message.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixtureOptions struct {
	Options map[string]any `json:"options"`
	Client  map[string]any `json:"client"`
}

func TestParity(t *testing.T) {
	skips := loadSkips(t)
	seen := map[string]bool{}
	var pass, skipped int

	root := filepath.Join("testdata", "in")
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".html") {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(strings.TrimSuffix(rel, ".html"))
		seen[name] = true

		t.Run(name, func(t *testing.T) {
			in, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			opts := loadOptions(t, strings.TrimSuffix(p, ".html")+".json")
			want, wantErr := loadGolden(t, name)

			got, gotErr := New(opts...).InlineBytes(in)
			ok := (gotErr != nil) == wantErr && (wantErr || bytes.Equal(got, want))

			reason, isSkipped := skips[name]
			switch {
			case isSkipped && ok:
				t.Errorf("fixture now passes; drop its line from testdata/skip.txt\n  was: %s", reason)
			case isSkipped:
				skipped++
				t.Skipf("known gap: %s", reason)
			case !ok:
				if gotErr != nil {
					t.Fatalf("Inline returned an error, juice did not: %v", gotErr)
				}
				t.Errorf("%s\n%s", classify(got, want), contextDiff(got, want))
			default:
				pass++
			}
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// A skip naming a fixture that no longer exists is stale.
	for name, reason := range skips {
		if !seen[name] {
			t.Errorf("testdata/skip.txt names missing fixture %q (%s)", name, reason)
		}
	}
	total := pass + skipped
	if total > 0 {
		t.Logf("parity: %d/%d fixtures (%.0f%%), %d skipped",
			pass, total, 100*float64(pass)/float64(total), skipped)
	}
}

func loadSkips(t *testing.T) map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "skip.txt"))
	if os.IsNotExist(err) {
		return map[string]string{}
	}
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, reason, _ := strings.Cut(line, " ")
		out[name] = strings.TrimSpace(reason)
	}
	return out
}

// loadOptions maps the fixture's juice options onto Go Options. Only the
// options a fixture actually uses need a case here; an unmapped one is a test
// bug, not a silent pass.
func loadOptions(t *testing.T, path string) []Option {
	t.Helper()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var f fixtureOptions
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("bad options sidecar: %v", err)
	}
	var opts []Option
	for k, v := range f.Options {
		b, _ := v.(bool)
		s, _ := v.(string)
		switch k {
		case "extraCss":
			opts = append(opts, ExtraCSS(s))
		case "applyStyleTags":
			opts = append(opts, ApplyStyleTags(b))
		case "removeStyleTags":
			opts = append(opts, RemoveStyleTags(b))
		case "preserveMediaQueries":
			opts = append(opts, PreserveMediaQueries(b))
		case "preserveFontFaces":
			opts = append(opts, PreserveFontFaces(b))
		case "preserveKeyFrames":
			opts = append(opts, PreserveKeyFrames(b))
		case "preservePseudos":
			opts = append(opts, PreservePseudos(b))
		case "preserveImportant":
			opts = append(opts, PreserveImportant(b))
		case "applyWidthAttributes":
			opts = append(opts, ApplyWidthAttributes(b))
		case "applyHeightAttributes":
			opts = append(opts, ApplyHeightAttributes(b))
		case "applyAttributesTableElements":
			opts = append(opts, ApplyAttributesTableElements(b))
		case "resolveCSSVariables":
			opts = append(opts, ResolveCSSVariables(b))
		case "inlinePseudoElements":
			opts = append(opts, InlinePseudoElements(b))
		case "styleAttributeName":
			opts = append(opts, StyleAttributeName(s))
		default:
			t.Fatalf("options sidecar uses %q, which parity_test.go does not map", k)
		}
	}
	return opts
}

func loadGolden(t *testing.T, name string) (content []byte, isErr bool) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "out", name+".html"))
	if err == nil {
		return b, false
	}
	if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join("testdata", "out", name+".err")); err == nil {
		return nil, true
	}
	t.Fatalf("no golden for %q; run `make goldens`", name)
	return nil, false
}

// classify labels a mismatch for the failure message. It must never decide
// whether a test passes -- a normalization that silences a diff here would
// grow until it hides real bugs.
func classify(got, want []byte) string {
	for _, c := range []struct {
		name string
		norm func([]byte) []byte
	}{
		{"VOID_SLASH", func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("/>"), []byte(">")) }},
		{"EMPTY_ATTR", func(b []byte) []byte { return bytes.ReplaceAll(b, []byte(`=""`), nil) }},
		{"DOC_WRAPPER", stripWrapper},
	} {
		if bytes.Equal(c.norm(got), c.norm(want)) {
			return "DIFF: " + c.name + " only"
		}
	}
	if bytes.Contains(want, []byte("style=")) || bytes.Contains(got, []byte("style=")) {
		return "DIFF: SEMANTIC (style attribute or cascade)"
	}
	return "DIFF: SEMANTIC (structural)"
}

func stripWrapper(b []byte) []byte {
	for _, s := range []string{"<html>", "</html>", "<head>", "</head>", "<body>", "</body>", "<tbody>", "</tbody>"} {
		b = bytes.ReplaceAll(b, []byte(s), nil)
	}
	return b
}

// contextDiff reports the first differing byte with surrounding context.
// Email HTML is one long line, so a line-oriented diff is useless.
func contextDiff(got, want []byte) string {
	i := 0
	for i < len(got) && i < len(want) && got[i] == want[i] {
		i++
	}
	lo := i - 60
	if lo < 0 {
		lo = 0
	}
	return fmt.Sprintf("first difference at byte %d of %d (want %d)\nwant: %s\n      %s^\ngot:  %s",
		i, len(got), len(want),
		clip(want, lo, i+60), strings.Repeat(" ", i-lo), clip(got, lo, i+60))
}

func clip(b []byte, lo, hi int) string {
	if lo > len(b) {
		lo = len(b)
	}
	if hi > len(b) {
		hi = len(b)
	}
	s := string(b[lo:hi])
	if lo > 0 {
		s = "…" + s
	}
	if hi < len(b) {
		s += "…"
	}
	return s
}
