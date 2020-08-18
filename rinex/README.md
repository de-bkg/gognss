# rinex

A RINEX Version 3 library.

## Installation

Make sure you have a working Go environment.  Go version 1.14+ is supported.  [See
the install instructions for Go](http://golang.org/doc/install.html).

To install, simply run:
```
$ go get -u github.com/erwiese/rinex
```

## Examples

``` go
package main

import (
	"log"

	"github.com/erwiese/gnss/rinex"
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
