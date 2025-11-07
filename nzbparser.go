package nzbparser

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"sort"

	"golang.org/x/net/html/charset"
)

const (
	// xml header for nzb files
	Header = `<?xml version="1.0" encoding="utf-8" ?>
<!DOCTYPE nzb PUBLIC "-//newzBin//DTD NZB 1.1//EN" "http://www.newzbin.com/DTD/nzb/nzb-1.1.dtd">` + "\n"
	// xml namespace for nzb files
	Xmlns = "http://www.newzbin.com/DTD/2003/nzb"
)

// ParseOptions allows configuration of the NZB parsing behavior
type ParseOptions struct {
	RemoveDuplicates bool // whether to remove duplicate files and segments
}

// nzb file structure with additional information
type Nzb struct {
	Comment       string            // comment tag
	Meta          map[string]string // meta data as map
	Files         NzbFiles          // files structure
	TotalFiles    int               // number of total files
	Segments      int               // number of available segments
	TotalSegments int               // number of total segments
	Bytes         int64             // total size of all files
}

// a slice of NzbFiles extended to allow sorting
type NzbFiles []NzbFile

func (s NzbFiles) Len() int           { return len(s) }
func (s NzbFiles) Less(i, j int) bool { return s[i].Number < s[j].Number }
func (s NzbFiles) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// individual file structure with additional information
type NzbFile struct {
	Groups        []string    `xml:"groups>group"`
	Segments      NzbSegments `xml:"segments>segment"`
	Poster        string      `xml:"poster,attr"`
	Date          int         `xml:"date,attr"`
	Subject       string      `xml:"subject,attr"`
	Bytes         int64       `xml:"bytes,attr"`    // total size of the file
	FileHash      string      `xml:"filehash,attr"` // hash of the file
	Number        int         `xml:"-"`             // number of the file (if indicated in the subject)
	Filename      string      `xml:"-"`             // filename of the file (if indicated in the subject)
	Basefilename  string      `xml:"-"`             // basefilename of the file (if indicated in the subject)
	TotalSegments int         `xml:"-"`             // number of total segments
}

// a slice of NzbSegments extended to allow sorting
type NzbSegments []NzbSegment

func (s NzbSegments) Len() int           { return len(s) }
func (s NzbSegments) Less(i, j int) bool { return s[i].Number < s[j].Number }
func (s NzbSegments) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// individual segment structure
type NzbSegment struct {
	Bytes  int    `xml:"bytes,attr"`
	Number int    `xml:"number,attr"`
	ID     string `xml:",innerxml"`
}

// parse nzb file provided as string
func ParseString(data string) (*Nzb, error) {
	return Parse(bytes.NewBufferString(data))
}

// parse nzb file provided as string with custom options
func ParseStringWithOptions(data string, opts ParseOptions) (*Nzb, error) {
	return ParseWithOptions(bytes.NewBufferString(data), opts)
}

// parse nzb file provided as io.Reader buffer
func Parse(buf io.Reader) (*Nzb, error) {
	return ParseWithOptions(buf, ParseOptions{RemoveDuplicates: true})
}

// parse nzb file provided as io.Reader buffer with custom options
func ParseWithOptions(buf io.Reader, opts ParseOptions) (*Nzb, error) {
	// parse nzb file into temp structure
	xnzb := new(xNzb)
	decoder := xml.NewDecoder(buf)
	decoder.CharsetReader = charset.NewReaderLabel
	decoder.Strict = false // ignore unknown or malformed character entities

	if err := decoder.Decode(xnzb); err != nil {
		return nil, fmt.Errorf("unable to parse NZB file: %s", err.Error())
	}

	// convert to nicer format
	nzb := new(Nzb)

	// copy elements
	nzb.Comment = xnzb.Comment
	nzb.Files = xnzb.Files

	// convert metadata
	nzb.Meta = make(map[string]string)
	for _, md := range xnzb.Metadata {
		nzb.Meta[md.Type] = md.Value
	}

	// conditionally remove duplicate entries
	if opts.RemoveDuplicates {
		MakeUnique(nzb)
	}

	// scan the nzb for the additional information
	ScanNzbFile(nzb)

	// sort the files and segments
	sort.Sort(nzb.Files)

	for id := range nzb.Files {
		sort.Sort(nzb.Files[id].Segments)
	}

	return nzb, nil
}

// write nzb struct to nzb xml as string
func WriteString(nzb *Nzb) (string, error) {
	file, err := Write(nzb)
	return string(file), err
}

// write nzb struct to nzb xml as byte slice
func Write(nzb *Nzb) ([]byte, error) {
	// create temp structure
	xnzb := new(xNzb)

	// copy elements
	if nzb.Comment != "" {
		xnzb.Comment = " " + nzb.Comment + " "
	}

	xnzb.Files = nzb.Files

	// add namespace
	xnzb.Xmlns = Xmlns

	// add metadata
	for t, v := range nzb.Meta {
		xnzb.Metadata = append(xnzb.Metadata, xNzbMeta{Type: t, Value: v})
	}

	// Marshal and prepend header
	file, err := xml.MarshalIndent(xnzb, "", "  ")
	if err != nil {
		return []byte(""), err
	}

	return append([]byte(Header), file...), nil
}

// scan the nzb struct for additional information
func ScanNzbFile(nzb *Nzb) {
	var segments int // total amount of available segments

	var totalSegments int // theoretical total amount of segments based on the subject count

	var totalBytes int64 // total size of all available segments

	var totalFiles int // theoretical total amount of files based on the subject count

	for id, file := range nzb.Files {
		var totalFileSegments int // theoretical total amount of segments of this file based on the subject count

		var totalFileBytes int64 // total size of all available segments of this file

		if subject, err := ParseSubject(file.Subject); err == nil {
			nzb.Files[id].Number = subject.File

			if subject.Filename != "" {
				nzb.Files[id].Filename = subject.Filename
			} else {
				nzb.Files[id].Filename = subject.Header
			}

			totalFileSegments = subject.TotalSegments

			if subject.TotalFiles > totalFiles {
				totalFiles = subject.TotalFiles
			}
		}

		for i, segment := range file.Segments {
			if segment.Number > totalFileSegments {
				totalFileSegments = segment.Number
			}

			totalBytes = totalBytes + int64(segment.Bytes)
			totalFileBytes = totalFileBytes + int64(segment.Bytes)
			nzb.Files[id].Segments[i].ID = html.UnescapeString(segment.ID)
		}

		segments = segments + file.Segments.Len()
		totalSegments = totalSegments + totalFileSegments
		nzb.Files[id].TotalSegments = totalFileSegments
		nzb.Files[id].Bytes = totalFileBytes
	}

	if totalFiles < nzb.Files.Len() {
		nzb.TotalFiles = nzb.Files.Len()
	} else {
		nzb.TotalFiles = totalFiles
	}

	nzb.Segments = segments
	nzb.TotalSegments = totalSegments
	nzb.Bytes = totalBytes
}

// clean up nzb files by keeping only the first occurrence of duplicate file entries and removing duplicate segments
func MakeUnique(nzb *Nzb) {
	// check for duplicate file entries and keep only the first occurrence
	var uniqueFiles []NzbFile

	fileKeys := make(map[string]int) // helper map for unique keys
	for _, file := range nzb.Files {
		if _, ok := fileKeys[file.Subject]; ok {
			// file already found, skip it (discard duplicate and its segments)
			continue
		}
		// Unique file found. Record position and collect in result.
		fileKeys[file.Subject] = len(uniqueFiles)
		uniqueFiles = append(uniqueFiles, file)
	}

	nzb.Files = uniqueFiles

	// remove duplicate segments
	for i, file := range nzb.Files {
		var uniqueSegments []NzbSegment

		segmentKeys := make(map[string]int) // helper map for unique keys
		for _, segment := range file.Segments {
			if _, ok := segmentKeys[segment.ID]; !ok {
				// Unique key found. Record position and collect in result.
				segmentKeys[segment.ID] = len(uniqueSegments)
				uniqueSegments = append(uniqueSegments, segment)
			}
		}

		nzb.Files[i].Segments = uniqueSegments
	}
}

// temp nzb file struct for (un)marshalling
type xNzb struct {
	Comment  string     `xml:",comment"`
	XMLName  xml.Name   `xml:"nzb"`
	Xmlns    string     `xml:"xmlns,attr"`
	Metadata []xNzbMeta `xml:"head>meta"`
	Files    NzbFiles   `xml:"file"`
}

// temp raw meta data for (un)marshalling
type xNzbMeta struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",innerxml"`
}
