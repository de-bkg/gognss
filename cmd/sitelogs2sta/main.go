// sitelog2sta reads IGS sitelog files and uses them to generate a Bernese
// Station Information (STA) file.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/de-bkg/gognss/pkg/site"
)

const (
	version = "0.0.3"
)

func main() {
	fs := flag.NewFlagSet("sitelog2sta/"+version, flag.ExitOnError)
	fs.Usage = func() {
		fmt.Println("sitelog2sta - create a Bernese STA-File based on IGS sitelog formated files")
		fmt.Println("")
		fmt.Println("USAGE: sitelog2sta [OPTIONS] FILE...")
		//fmt.Printf("\nFLAGS:\n")
		//fs.PrintDefaults()

		fmt.Println(`
EXAMPLES:
    $ sitelogs2sta ~/sitelogs/*.log >out.sta 2>out.sta.err
        `)

		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Source: %s\n", "https://github.com/de-bkg/gognss/tree/master/cmd/sitelog2sta")
		fmt.Println("BKG Frankfurt 2020, G2")
	}
	fs.Parse(os.Args[1:])

	var sites []*site.Site
	for _, slPath := range fs.Args() {
		f, err := os.Open(slPath)
		if err != nil {
			log.Fatalf("%v", err)
		}
		defer f.Close()

		s, err := site.DecodeSitelog(f)
		if err != nil {
			log.Printf("decoding sitelog %s: %v", slPath, err)
			continue
		}
		for _, warn := range s.Warnings {
			log.Printf("WARN: %s: %v", slPath, warn)
		}

		err = s.ValidateAndClean(false)
		if err != nil {
			log.Printf("validate sitelog %s: %v", slPath, err)
			continue
		}

		sites = append(sites, s)
	}

	err := site.EncodeSTAfile(os.Stdout, sites)
	if err != nil {
		log.Fatalf("encode STA-File: %v", err)
	}
}
