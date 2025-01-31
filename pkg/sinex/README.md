# Decode SINEX files

SINEX - Solution (Software/technique) INdependent EXchange Format

The format description is available at https://www.iers.org/IERS/EN/Organization/AnalysisCoordinator/SinexFormat/sinex.html.

So far decoding is implemented for the following blocks:
- FILE/REFERENCE
- SITE/ID
- SITE/RECEIVER
- SITE/ANTENNA
- SOLUTION/ESTIMATE

### Install
```go
$ go get -u github.com/de-bkg/gognss
```

### Decode estimates
```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/de-bkg/gognss/pkg/sinex"
)

func main() {
	r, err := os.Open("path/to/sinexfile")
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	dec, err := sinex.NewDecoder(r)
	if err != nil {
		log.Fatal(err)
	}

	// Get the header
	hdr := dec.Header

	// Iterate over blocks.
	for dec.NextBlock() {
		name := dec.CurrentBlock()

		// Decode all SOLUTION/ESTIMATE records.
		if name == sinex.BlockSolEstimate {
			for dec.NextBlockLine() {
				var est sinex.Estimate
				if err := dec.Decode(&est); err != nil {
					log.Fatal(err)
				}

				// Do something with est.
				fmt.Printf("%s: %.5f\n", est.SiteCode, est.Value)
			}
		}
	}
}
```