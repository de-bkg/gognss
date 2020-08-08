# ntrip client

with functions for
- parsing the caster sourcetable
- checking if caster is alive
- ...

Connecting to the caster is done by HTTP only, i.e. no UDP, RTSP etc.


## Installation

Make sure you have a working Go environment.  Go version 1.13+ is supported.  [See
the install instructions for Go](http://golang.org/doc/install.html).

To install, simply run:
```
$ go get github.com/erwiese/gnss
```

## Examples

``` go
package main

import (
	"log"

	"github.com/erwiese/gnss/ntrip"
)

func main() {
    casterAddr := "http://www.igs-ip.net:2101"
    c, err := ntrip.NewClient(casterAddr, Options{})
	if err != nil {
		log.Fatal(err)
	}
    defer c.CloseIdleConnections()

    if !c.IsCasterAlive() {
        log.Printf("caster %s seems to be down", casterAddr)
    }
    
    st, err := c.ParseSourcetable()
    if err != nil {
		log.Printf(err)
    }

}
```

## Links
- Open Source Software development for Ntrip: http://software.rtcm-ntrip.org/wiki
- NtripCaster sourcetable format: http://software.rtcm-ntrip.org/wiki/Sourcetable
- NtripCaster SW packages: https://igs.bkg.bund.de/ntrip/download

