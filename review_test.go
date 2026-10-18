package nzbparser

import (
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
)

func reviewDoc(head, files string) string {
	return Header + `<nzb xmlns="http://www.newzbin.com/DTD/2003/nzb">` + head + files + `</nzb>`
}

func reviewFile(subject, id string) string {
	return `<file poster="p" date="1" subject="` + subject + `"><groups><group>alt.test</group></groups>` +
		`<segments><segment bytes="10" number="1">` + id + `</segment></segments></file>`
}

// A parsed NZB written back is well-formed XML with the same ids
// (ids were unescaped after parsing, then written raw).
func TestWriteAfterParseIsWellFormed(t *testing.T) {
	nzb, err := ParseString(reviewDoc("", reviewFile("&quot;a.bin&quot; yEnc (1/1)", "part1&amp;x&lt;y&gt;@example.test")))
	if err != nil {
		t.Fatal(err)
	}
	out, err := WriteString(nzb)
	if err != nil {
		t.Fatal(err)
	}
	if err := xml.Unmarshal([]byte(out[len(Header):]), new(xNzb)); err != nil {
		t.Fatalf("Write output is not well-formed: %v\n%s", err, out)
	}
	again, err := ParseString(out)
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Files[0].Segments[0].ID; got != "part1&x<y>@example.test" {
		t.Fatalf("id after a round trip: %q", got)
	}
}

// Pretty-printed NZBs put whitespace around ids, groups and meta values.
func TestWhitespaceIsTrimmed(t *testing.T) {
	doc := reviewDoc("<head><meta type=\"title\">\n  A Title\n</meta></head>",
		"<file poster=\"p\" date=\"1\" subject=\"&quot;a.bin&quot; (1/1)\"><groups><group>\n alt.test \n</group></groups>"+
			"<segments><segment bytes=\"10\" number=\"1\">\n      p@example.test\n    </segment></segments></file>")
	nzb, err := ParseString(doc)
	if err != nil {
		t.Fatal(err)
	}
	if id := nzb.Files[0].Segments[0].ID; id != "p@example.test" {
		t.Errorf("id %q", id)
	}
	if g := nzb.Files[0].Groups[0]; g != "alt.test" {
		t.Errorf("group %q", g)
	}
	if m := nzb.Meta["title"]; m != "A Title" {
		t.Errorf("meta %q", m)
	}
}

// A segment count too big for an int counts as missing; the real segments decide.
func TestOverflowingSubjectNumbers(t *testing.T) {
	nzb, err := ParseString(reviewDoc("", reviewFile("&quot;b.bin&quot; yEnc (1/99999999999999999999)", "b@example.test")))
	if err != nil {
		t.Fatal(err)
	}
	if nzb.TotalSegments != 1 || nzb.Files[0].TotalSegments != 1 {
		t.Fatalf("TotalSegments = %d / %d, want 1", nzb.TotalSegments, nzb.Files[0].TotalSegments)
	}
}

func TestBasefilenameIsSet(t *testing.T) {
	nzb, err := ParseString(reviewDoc("", reviewFile("[1/2] &quot;release.part01.rar&quot; yEnc (1/1)", "c@example.test")))
	if err != nil {
		t.Fatal(err)
	}
	if f := nzb.Files[0]; f.Filename != "release.part01.rar" || f.Basefilename != "release" {
		t.Fatalf("Filename %q Basefilename %q", f.Filename, f.Basefilename)
	}
}

// Parse → Write → Parse keeps the comment and the output is the same every time.
func TestWriteIsStable(t *testing.T) {
	doc := reviewDoc("<!-- hi --><head><meta type=\"b\">2</meta><meta type=\"a\">1</meta><meta type=\"c\">3</meta></head>",
		reviewFile("&quot;d.bin&quot; (1/1)", "d@example.test"))
	nzb, err := ParseString(doc)
	if err != nil {
		t.Fatal(err)
	}
	first, _ := WriteString(nzb)
	for i := 0; i < 20; i++ {
		again, err := ParseString(first)
		if err != nil {
			t.Fatal(err)
		}
		if again.Comment != "hi" {
			t.Fatalf("comment after %d round trips: %q", i+1, again.Comment)
		}
		out, _ := WriteString(again)
		if out != first {
			t.Fatalf("Write output changed on round trip %d:\n%s\n---\n%s", i+1, first, out)
		}
	}
}

// Files without a number in the subject keep the NZB's order (sort.Sort is not stable).
func TestFilesWithoutNumbersKeepOrder(t *testing.T) {
	var files []string
	for i := 0; i < 200; i++ {
		files = append(files, reviewFile(fmt.Sprintf("post &quot;f%03d.bin&quot; yEnc", i), fmt.Sprintf("m%03d@example.test", i)))
	}
	nzb, err := ParseString(reviewDoc("", strings.Join(files, "")))
	if err != nil {
		t.Fatal(err)
	}
	for i, f := range nzb.Files {
		if want := fmt.Sprintf("f%03d.bin", i); f.Filename != want {
			t.Fatalf("file %d is %q, want %q", i, f.Filename, want)
		}
	}
}

func BenchmarkParseSubject(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := ParseSubject(`[1/7] "ubuntu-22.04-live-server-amd64.iso" yEnc (1/2047)`); err != nil {
			b.Fatal(err)
		}
	}
}
