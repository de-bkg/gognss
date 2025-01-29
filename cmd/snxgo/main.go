// Decode SINEX files and development for the sinex new block

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/de-bkg/gognss/pkg/sinex"
)

func main() {
	r, err := os.Open("/home/lwang/sandbox/gognss/pkg/sinex/testdata/igs20P21161.snx")
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
	fmt.Printf("%+v", hdr)

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
