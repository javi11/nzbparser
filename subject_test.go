package nzbparser

import "testing"

// Dedicated tests for the Subject parser
func TestParseSubject(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		header string
		fname  string
		base   string
		file   int
		totalF int
		seg    int
		totalS int
	}{
		{
			name:   "dual quoted release + r00",
			input:  `[04/23] "Lili.en.Marleen.S03E07.FLEMISH.1080p.WEB.h264-TRIPEL" - "lili.en.marleen.s03e07.flemish.1080p.web.h264-tripel.r00" - yEnc(1/140)`,
			header: "Lili.en.Marleen.S03E07.FLEMISH.1080p.WEB.h264-TRIPEL",
			fname:  "lili.en.marleen.s03e07.flemish.1080p.web.h264-tripel.r00",
			base:   "lili.en.marleen.s03e07.flemish.1080p.web.h264-tripel",
			file:   4, totalF: 23, seg: 1, totalS: 140,
		},
		{
			name:   "simple with brackets and paren",
			input:  `[1/2] Test Subject - "test.txt" yEnc (1/2)`,
			header: "Test Subject",
			fname:  "test.txt",
			base:   "test",
			file:   1, totalF: 2, seg: 1, totalS: 2,
		},
		{
			name:   "single file no numbers",
			input:  `"singlefile.nfo" yEnc (1/1)`,
			header: "singlefile",
			fname:  "singlefile.nfo",
			base:   "singlefile",
			file:   1, totalF: 1, seg: 1, totalS: 1,
		},
		{
			name:   "archive part with segment only",
			input:  `Some Header - "archive.part01.rar" yEnc (12/120)`,
			header: "Some Header",
			fname:  "archive.part01.rar",
			base:   "archive",
			file:   1, totalF: 1, seg: 12, totalS: 120,
		},
		{
			name:   "of pattern for files",
			input:  `[ 5 of 12 ] "Example.txt" yEnc (1/1)`,
			header: "Example",
			fname:  "Example.txt",
			base:   "Example",
			file:   5, totalF: 12, seg: 1, totalS: 1,
		},
		{
			name:   "both bracket pairs used (second are segments)",
			input:  `[003/120] [03/140] "Release.Name.r03" yEnc`,
			header: "Release.Name",
			fname:  "Release.Name.r03",
			base:   "Release.Name",
			file:   3, totalF: 120, seg: 3, totalS: 140,
		},
	}

	for _, c := range cases {
		parsed, err := ParseSubject(c.input)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.name, err)
		}
		if parsed.Header != c.header {
			t.Errorf("%s: header got %q want %q", c.name, parsed.Header, c.header)
		}
		if parsed.Filename != c.fname {
			t.Errorf("%s: filename got %q want %q", c.name, parsed.Filename, c.fname)
		}
		if parsed.Basefilename != c.base {
			t.Errorf("%s: basefilename got %q want %q", c.name, parsed.Basefilename, c.base)
		}
		if parsed.File != c.file || parsed.TotalFiles != c.totalF {
			t.Errorf("%s: file numbers got %d/%d want %d/%d", c.name, parsed.File, parsed.TotalFiles, c.file, c.totalF)
		}
		if parsed.Segment != c.seg || parsed.TotalSegments != c.totalS {
			t.Errorf("%s: segment numbers got %d/%d want %d/%d", c.name, parsed.Segment, parsed.TotalSegments, c.seg, c.totalS)
		}
	}
}
