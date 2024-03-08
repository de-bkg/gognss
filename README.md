# GNSS
[![PkgGoDev](https://pkg.go.dev/badge/de-bkg/gognss)](https://pkg.go.dev/github.com/de-bkg/gognss)

Please note that all packages are not stable yet and can change any time!

Golang packages for 
* **ntrip**: connect to an NtripCaster, get status information from a BKG NtripCaster, run commands against a BKG NtripCaster
* **rinex**: read RINEX3 files
* **sinex**: read SINEX files
* **site**: handle metadata for a GNSS site/station, read and write IGS sitelog files
  * generate a Bernese Station Information (STA) file from IGS sitelog files



## Installation

Make sure you have a working Go environment.  Go version 1.15+ is supported.  [See
the install instructions for Go](http://golang.org/doc/install.html).

To install, simply run:
```go
$ go get -u github.com/de-bkg/gognss
```
