# rinex

Package rinex contains functions for reading RINEX Version 3 files.

## Examples

``` go
package main

import (
	"log"

	"github.com/de-bkg/gognss/rinex"
)

func main() {
	r, _ := os.Open("testdata/white/REYK00ISL_R_20192701000_01H_30S_MO.rnx")
	defer r.Close()

	dec, _ := rinex.NewObsDecoder(r)
	for dec.NextEpoch() {
		epoch := dec.Epoch()
		// Do something with epoch
	}
	if err := dec.Err(); err != nil {
		log.Printf("read epochs: %v", err)
	}
}
```


## Links
Fromats see https://kb.igs.org/hc/en-us/articles/201096516-IGS-Formats
