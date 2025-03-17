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
	var estimates []sinex.Estimate
	for name, err := range dec.Blocks() {
		if err != nil {
			log.Fatal(err)
		}

  	// Decode all SOLUTION/ESTIMATE records.
		if name == sinex.BlockSolEstimate {
			for _, err := range dec.BlockLines() { // Reads the next line into buffer.
				if err != nil {
					log.Fatal(err)
				}

				var est sinex.Estimate
				err := dec.Decode(&est)
				if err != nil {
					log.Fatal(err)
				}

				// Do something with est.
				fmt.Printf("%s: %.5f\n", est.SiteCode, est.Value)
				estimates = append(estimates, est)
			}
		}
	}

	// Iterate over coordinates for each station and epoch, based on estimates.
	for rec := range AllStationCoordinates(estimates) {
		fmt.Printf("%v\n", rec)
	}
}
```