package nzbparser

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestParseString(t *testing.T) {
	// Test valid NZB
	validNZB := Header + `<nzb xmlns="http://www.newzbin.com/DTD/2003/nzb">
  <head>
    <meta type="title">Test Title</meta>
    <meta type="category">Test Category</meta>
  </head>
  <file poster="test@example.com" date="1234567890" subject="[1/2] Test Subject - &quot;test.txt&quot; yEnc (1/2)">
    <groups>
      <group>alt.test</group>
    </groups>
    <segments>
      <segment bytes="1234" number="1">test-segment-1</segment>
      <segment bytes="5678" number="2">test-segment-2</segment>
    </segments>
  </file>
</nzb>`

	nzb, err := ParseString(validNZB)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	// Verify metadata
	if nzb.Meta["title"] != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", nzb.Meta["title"])
	}

	if nzb.Meta["category"] != "Test Category" {
		t.Errorf("Expected category 'Test Category', got '%s'", nzb.Meta["category"])
	}

	// Verify file data
	if len(nzb.Files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(nzb.Files))
	}

	file := nzb.Files[0]
	if file.Poster != "test@example.com" {
		t.Errorf("Expected poster 'test@example.com', got '%s'", file.Poster)
	}

	if file.Date != 1234567890 {
		t.Errorf("Expected date 1234567890, got %d", file.Date)
	}

	if file.Subject != "[1/2] Test Subject - \"test.txt\" yEnc (1/2)" {
		t.Errorf("Expected subject '[1/2] Test Subject - \"test.txt\" yEnc (1/2)', got '%s'", file.Subject)
	}

	// Verify groups
	if len(file.Groups) != 1 || file.Groups[0] != "alt.test" {
		t.Errorf("Expected group 'alt.test', got %v", file.Groups)
	}

	// Verify segments
	if len(file.Segments) != 2 {
		t.Fatalf("Expected 2 segments, got %d", len(file.Segments))
	}

	if file.Segments[0].Bytes != 1234 || file.Segments[0].Number != 1 || file.Segments[0].ID != "test-segment-1" {
		t.Errorf("Segment 1 data incorrect: %+v", file.Segments[0])
	}

	if file.Segments[1].Bytes != 5678 || file.Segments[1].Number != 2 || file.Segments[1].ID != "test-segment-2" {
		t.Errorf("Segment 2 data incorrect: %+v", file.Segments[1])
	}

	// Verify additional parsed information
	if file.Filename != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got '%s'", file.Filename)
	}

	if file.Number != 1 {
		t.Errorf("Expected file number 1, got %d", file.Number)
	}

	// Test invalid NZB
	invalidNZB := "This is not a valid NZB file"

	_, err = ParseString(invalidNZB)
	if err == nil {
		t.Error("Expected error for invalid NZB, got nil")
	}
}

func TestParseSubjectVariants(t *testing.T) {
	testCases := []struct {
		subject string
		exp     Subject
	}{
		{
			subject: `[04/23] "Lili.en.Marleen.S03E07.FLEMISH.1080p.WEB.h264-TRIPEL" - "lili.en.marleen.s03e07.flemish.1080p.web.h264-tripel.r00" - yEnc(1/140)`,
			exp: Subject{
				Header:        "Lili.en.Marleen.S03E07.FLEMISH.1080p.WEB.h264-TRIPEL",
				Filename:      "lili.en.marleen.s03e07.flemish.1080p.web.h264-tripel.r00",
				Basefilename:  "lili.en.marleen.s03e07.flemish.1080p.web.h264-tripel",
				File:          4,
				TotalFiles:    23,
				Segment:       1,
				TotalSegments: 140,
			},
		},
		{
			subject: `[1/2] Test Subject - "test.txt" yEnc (1/2)`,
			exp: Subject{
				Header:        "Test Subject",
				Filename:      "test.txt",
				Basefilename:  "test",
				File:          1,
				TotalFiles:    2,
				Segment:       1,
				TotalSegments: 2,
			},
		},
		{
			subject: `"singlefile.nfo" yEnc (1/1)`,
			exp: Subject{
				Header:        "singlefile",
				Filename:      "singlefile.nfo",
				Basefilename:  "singlefile",
				File:          1,
				TotalFiles:    1,
				Segment:       1,
				TotalSegments: 1,
			},
		},
		{
			subject: `Some Header - "archive.part01.rar" yEnc (12/120)`,
			exp: Subject{
				Header:        "Some Header",
				Filename:      "archive.part01.rar",
				Basefilename:  "archive",
				File:          1, // assumed single file when not present
				TotalFiles:    1,
				Segment:       12,
				TotalSegments: 120,
			},
		},
	}

	for _, tc := range testCases {
		parsed, err := ParseSubject(tc.subject)
		if err != nil {
			t.Fatalf("ParseSubject returned error for %q: %v", tc.subject, err)
		}
		if parsed.Header != tc.exp.Header {
			t.Errorf("Header mismatch for %q: got %q expected %q", tc.subject, parsed.Header, tc.exp.Header)
		}
		if parsed.Filename != tc.exp.Filename {
			t.Errorf("Filename mismatch for %q: got %q expected %q", tc.subject, parsed.Filename, tc.exp.Filename)
		}
		if parsed.Basefilename != tc.exp.Basefilename {
			t.Errorf("Basefilename mismatch for %q: got %q expected %q", tc.subject, parsed.Basefilename, tc.exp.Basefilename)
		}
		if parsed.File != tc.exp.File || parsed.TotalFiles != tc.exp.TotalFiles {
			t.Errorf("File numbers mismatch for %q: got %d/%d expected %d/%d", tc.subject, parsed.File, parsed.TotalFiles, tc.exp.File, tc.exp.TotalFiles)
		}
		if parsed.Segment != tc.exp.Segment || parsed.TotalSegments != tc.exp.TotalSegments {
			t.Errorf("Segment numbers mismatch for %q: got %d/%d expected %d/%d", tc.subject, parsed.Segment, parsed.TotalSegments, tc.exp.Segment, tc.exp.TotalSegments)
		}
	}
}

func TestParse(t *testing.T) {
	// Test valid NZB
	validNZB := Header + `<nzb xmlns="http://www.newzbin.com/DTD/2003/nzb">
  <file poster="test@example.com" date="1234567890" subject="Test Subject">
    <groups>
      <group>alt.test</group>
    </groups>
    <segments>
      <segment bytes="1234" number="1">test-segment-1</segment>
    </segments>
  </file>
</nzb>`

	nzb, err := Parse(bytes.NewBufferString(validNZB))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(nzb.Files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(nzb.Files))
	}

	// Test invalid NZB
	invalidNZB := "This is not a valid NZB file"

	_, err = Parse(bytes.NewBufferString(invalidNZB))
	if err == nil {
		t.Error("Expected error for invalid NZB, got nil")
	}
}

func TestWriteString(t *testing.T) {
	nzb := &Nzb{
		Comment: "Test Comment",
		Meta: map[string]string{
			"title":    "Test Title",
			"category": "Test Category",
		},
		Files: []NzbFile{
			{
				Poster:  "test@example.com",
				Date:    1234567890,
				Subject: "Test Subject",
				Groups:  []string{"alt.test"},
				Segments: []NzbSegment{
					{
						Bytes:  1234,
						Number: 1,
						ID:     "test-segment-1",
					},
				},
			},
		},
	}

	output, err := WriteString(nzb)
	if err != nil {
		t.Fatalf("WriteString failed: %v", err)
	}

	// Check that output contains expected elements
	if !strings.Contains(output, "Test Comment") {
		t.Error("Output doesn't contain comment")
	}

	if !strings.Contains(output, "<meta type=\"title\">Test Title</meta>") {
		t.Error("Output doesn't contain title metadata")
	}

	if !strings.Contains(output, "<meta type=\"category\">Test Category</meta>") {
		t.Error("Output doesn't contain category metadata")
	}

	if !strings.Contains(output, "poster=\"test@example.com\"") {
		t.Error("Output doesn't contain poster attribute")
	}

	if !strings.Contains(output, "<group>alt.test</group>") {
		t.Error("Output doesn't contain group")
	}

	if !strings.Contains(output, "<segment bytes=\"1234\" number=\"1\">test-segment-1</segment>") {
		t.Error("Output doesn't contain segment")
	}
}

func TestWrite(t *testing.T) {
	nzb := &Nzb{
		Meta: map[string]string{
			"title": "Test Title",
		},
		Files: []NzbFile{
			{
				Subject: "Test Subject",
				Groups:  []string{"alt.test"},
				Segments: []NzbSegment{
					{
						Bytes:  1234,
						Number: 1,
						ID:     "test-segment-1",
					},
				},
			},
		},
	}

	output, err := Write(nzb)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Check that output is a valid byte slice
	if len(output) == 0 {
		t.Error("Output is empty")
	}

	// Parse the output to ensure it's valid
	parsedNzb, err := Parse(bytes.NewReader(output))
	if err != nil {
		t.Fatalf("Parsing output failed: %v", err)
	}

	if parsedNzb.Meta["title"] != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", parsedNzb.Meta["title"])
	}
}

func TestScanNzbFile(t *testing.T) {
	nzb := &Nzb{
		Files: []NzbFile{
			{
				Subject: "[1/2] Test Subject - \"test.txt\" yEnc (1/2)",
				Segments: []NzbSegment{
					{
						Bytes:  1000,
						Number: 1,
					},
					{
						Bytes:  2000,
						Number: 2,
					},
				},
			},
			{
				Subject: "[2/2] Test Subject - \"test.txt\" yEnc (1/1)",
				Segments: []NzbSegment{
					{
						Bytes:  3000,
						Number: 1,
					},
				},
			},
		},
	}

	ScanNzbFile(nzb)

	// Verify total counts
	if nzb.TotalFiles != 2 {
		t.Errorf("Expected TotalFiles 2, got %d", nzb.TotalFiles)
	}

	if nzb.Segments != 3 {
		t.Errorf("Expected Segments 3, got %d", nzb.Segments)
	}

	if nzb.Bytes != 6000 {
		t.Errorf("Expected Bytes 6000, got %d", nzb.Bytes)
	}

	// Verify file information
	if nzb.Files[0].Filename != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got '%s'", nzb.Files[0].Filename)
	}

	if nzb.Files[0].Number != 1 {
		t.Errorf("Expected file number 1, got %d", nzb.Files[0].Number)
	}

	if nzb.Files[0].Bytes != 3000 {
		t.Errorf("Expected file bytes 3000, got %d", nzb.Files[0].Bytes)
	}
}

func TestScanNzbFileEdgeCases(t *testing.T) {
	// Test with empty file list
	emptyNzb := &Nzb{
		Files: []NzbFile{},
	}
	ScanNzbFile(emptyNzb)

	if emptyNzb.TotalFiles != 0 {
		t.Errorf("Expected TotalFiles 0 for empty nzb, got %d", emptyNzb.TotalFiles)
	}

	// Test with invalid subject format that can't be parsed
	invalidSubjectNzb := &Nzb{
		Files: []NzbFile{
			{
				Subject: "This is not a parseable subject",
				Segments: []NzbSegment{
					{
						Bytes:  1000,
						Number: 1,
					},
				},
			},
		},
	}
	ScanNzbFile(invalidSubjectNzb)

	// Test with more total files from subject than actual files
	moreFilesNzb := &Nzb{
		Files: []NzbFile{
			{
				Subject: "[1/10] Test Subject - \"test.txt\" yEnc (1/1)",
				Segments: []NzbSegment{
					{
						Bytes:  1000,
						Number: 1,
					},
				},
			},
		},
	}
	ScanNzbFile(moreFilesNzb)

	if moreFilesNzb.TotalFiles != 10 {
		t.Errorf("Expected TotalFiles 10, got %d", moreFilesNzb.TotalFiles)
	}
}

func TestMakeUnique(t *testing.T) {
	// Create NZB with duplicate file entries and segments
	nzb := &Nzb{
		Files: []NzbFile{
			{
				Subject: "Test Subject",
				Segments: []NzbSegment{
					{
						ID:     "segment-1",
						Number: 1,
						Bytes:  1000,
					},
					{
						ID:     "segment-2",
						Number: 2,
						Bytes:  2000,
					},
				},
			},
			{
				Subject: "Test Subject", // Same subject, should be merged
				Segments: []NzbSegment{
					{
						ID:     "segment-2", // Duplicate segment, should be removed
						Number: 2,
						Bytes:  2000,
					},
					{
						ID:     "segment-3",
						Number: 3,
						Bytes:  3000,
					},
				},
			},
			{
				Subject: "Different Subject", // Different subject, should be kept
				Segments: []NzbSegment{
					{
						ID:     "segment-4",
						Number: 1,
						Bytes:  4000,
					},
				},
			},
		},
	}

	MakeUnique(nzb)

	// Should now have 2 files
	if len(nzb.Files) != 2 {
		t.Fatalf("Expected 2 files after MakeUnique, got %d", len(nzb.Files))
	}

	// First file should have 3 unique segments
	if len(nzb.Files[0].Segments) != 3 {
		t.Errorf("Expected 3 segments in first file, got %d", len(nzb.Files[0].Segments))
	}

	// Check that segments are correct
	segmentIDs := make(map[string]bool)
	for _, seg := range nzb.Files[0].Segments {
		segmentIDs[seg.ID] = true
	}

	if !segmentIDs["segment-1"] || !segmentIDs["segment-2"] || !segmentIDs["segment-3"] {
		t.Errorf("Missing expected segments, got: %v", segmentIDs)
	}

	// Second file should have 1 segment
	if len(nzb.Files[1].Segments) != 1 {
		t.Errorf("Expected 1 segment in second file, got %d", len(nzb.Files[1].Segments))
	}
}

func TestNzbFilesSorting(t *testing.T) {
	files := NzbFiles{
		{Number: 3},
		{Number: 1},
		{Number: 2},
	}

	// Test sort interface
	if files.Len() != 3 {
		t.Errorf("Expected length 3, got %d", files.Len())
	}

	if !files.Less(1, 0) {
		t.Errorf("Expected files[1] < files[0]")
	}

	// Test actual sorting
	files.Swap(0, 1)

	if files[0].Number != 1 || files[1].Number != 3 {
		t.Errorf("Swap didn't work correctly")
	}

	// Reset and sort
	files = NzbFiles{
		{Number: 3},
		{Number: 1},
		{Number: 2},
	}

	sort.Sort(files)

	if files[0].Number != 1 || files[1].Number != 2 || files[2].Number != 3 {
		t.Errorf("Sort didn't work correctly, got: %v", files)
	}
}

func TestNzbSegmentsSorting(t *testing.T) {
	segments := NzbSegments{
		{Number: 3},
		{Number: 1},
		{Number: 2},
	}

	// Test sort interface
	if segments.Len() != 3 {
		t.Errorf("Expected length 3, got %d", segments.Len())
	}

	if !segments.Less(1, 0) {
		t.Errorf("Expected segments[1] < segments[0]")
	}

	// Test actual sorting
	segments.Swap(0, 1)

	if segments[0].Number != 1 || segments[1].Number != 3 {
		t.Errorf("Swap didn't work correctly")
	}

	// Reset and sort
	segments = NzbSegments{
		{Number: 3},
		{Number: 1},
		{Number: 2},
	}

	sort.Sort(segments)

	if segments[0].Number != 1 || segments[1].Number != 2 || segments[2].Number != 3 {
		t.Errorf("Sort didn't work correctly, got: %v", segments)
	}
}

func TestXNzbStructs(t *testing.T) {
	// Test that our internal structs can be marshalled and unmarshalled correctly
	xnzb := xNzb{
		Comment: " Test Comment ",
		Xmlns:   Xmlns,
		Metadata: []xNzbMeta{
			{Type: "title", Value: "Test Title"},
			{Type: "category", Value: "Test Category"},
		},
		Files: []NzbFile{
			{
				Poster:  "test@example.com",
				Date:    1234567890,
				Subject: "Test Subject",
				Groups:  []string{"alt.test"},
				Segments: []NzbSegment{
					{
						Bytes:  1234,
						Number: 1,
						ID:     "test-segment-1",
					},
				},
			},
		},
	}

	// Marshal to XML
	data, err := xml.MarshalIndent(xnzb, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal xNzb: %v", err)
	}

	// Unmarshal back
	var newXnzb xNzb

	err = xml.Unmarshal(data, &newXnzb)
	if err != nil {
		t.Fatalf("Failed to unmarshal xNzb: %v", err)
	}

	// Verify data
	if newXnzb.Comment != " Test Comment " {
		t.Errorf("Expected comment ' Test Comment ', got '%s'", newXnzb.Comment)
	}

	if len(newXnzb.Metadata) != 2 {
		t.Fatalf("Expected 2 metadata entries, got %d", len(newXnzb.Metadata))
	}

	if newXnzb.Metadata[0].Type != "title" || newXnzb.Metadata[0].Value != "Test Title" {
		t.Errorf("Metadata[0] incorrect: %+v", newXnzb.Metadata[0])
	}

	if len(newXnzb.Files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(newXnzb.Files))
	}

	if newXnzb.Files[0].Subject != "Test Subject" {
		t.Errorf("Expected subject 'Test Subject', got '%s'", newXnzb.Files[0].Subject)
	}
}

// ErrorReader is a mock reader that always returns an error
type ErrorReader struct{}

func (er ErrorReader) Read(_ []byte) (int, error) {
	return 0, fmt.Errorf("mock read error")
}

func TestParseWithReaderError(t *testing.T) {
	// Test with a reader that returns an error
	_, err := Parse(ErrorReader{})
	if err == nil {
		t.Error("Expected error from reader, got nil")
	}
}

func TestWriteWithEmptyComment(t *testing.T) {
	// Test Write with empty comment
	nzb := &Nzb{
		Comment: "",
		Meta: map[string]string{
			"title": "Test Title",
		},
		Files: []NzbFile{
			{
				Subject: "Test Subject",
				Groups:  []string{"alt.test"},
				Segments: []NzbSegment{
					{
						Bytes:  1234,
						Number: 1,
						ID:     "test-segment-1",
					},
				},
			},
		},
	}

	output, err := Write(nzb)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Parse the output to ensure it's valid
	parsedNzb, err := Parse(bytes.NewReader(output))
	if err != nil {
		t.Fatalf("Parsing output failed: %v", err)
	}

	if parsedNzb.Comment != "" {
		t.Errorf("Expected empty comment, got '%s'", parsedNzb.Comment)
	}
}
