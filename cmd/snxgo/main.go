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
	// r, err := os.Open("/home/lwang/sandbox/gognss/pkg/sinex/testdata/soln.snx")

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

		// Decode all SOLUTION/ESTIMATE records.
		if name == sinex.BlockSolDiscon {
			for dec.NextBlockLine() {
				var disc sinex.Discon
				if err := dec.Decode(&disc); err != nil {
					log.Fatal(err)
				}

				layout := "2006-01-02 15:04:05"
				// Do something with est.
				fmt.Printf("%s; %s; %s; %s; %v\n", disc.SiteCode, disc.StartTime.Format(layout), disc.EndTime.Format(layout), disc.DisType, disc.Event)
			}
		}
	}
}
