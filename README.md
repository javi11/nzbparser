# NZBParser

A Go module for parsing NZB files.

## About

This is a fork of [Tensai75's nzbparser](https://github.com/Tensai75/nzbparser), a library for parsing NZB files in Go.

NZB files are XML-based files that contain information necessary for downloading binary files from Usenet servers. This parser allows you to extract and work with that information programmatically.

## Installation

```bash
go get github.com/javi11/nzbparser
```

## Usage

```go
package main

import (
	"fmt"
	"github.com/javi11/nzbparser"
)

func main() {
	// Parse an NZB file
	nzb, err := nzbparser.ParseNZBFile("path/to/file.nzb")
	if err != nil {
		panic(err)
	}

	// Access NZB metadata
	fmt.Println("NZB File:", nzb.Filename)
	fmt.Println("Total Size:", nzb.Size)
	fmt.Println("Number of files:", len(nzb.Files))

	// Iterate through files in the NZB
	for _, file := range nzb.Files {
		fmt.Println("File:", file.Filename)
		fmt.Println("Size:", file.Size)
	}
}
```

## License

This project is licensed under the MIT License - see the original repository for details.

## Acknowledgments

- Original implementation by [Tensai75](https://github.com/Tensai75/nzbparser)
