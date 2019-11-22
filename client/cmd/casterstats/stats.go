// BKG NtripCaster client program for collection caster statistics like uptime, number of sources etc.
//
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	cli "github.com/erwiese/ntrip/client"
)

const (
	version = "0.1"
)

func main() {
	opts := cli.Options{}
	opts.UserAgent = "BKGStatsClient/" + version
	fs := flag.NewFlagSet("BKGStatsClient/"+version, flag.ExitOnError)
	fs.StringVar(&opts.Username, "username", "", "Operator username to connect to the caster.")
	fs.StringVar(&opts.Password, "pw", "", "Operator Password.")
	fs.BoolVar(&opts.UnsafeSSL, "unsafeSSL", false, "If true, it will skip https certificate verification. Defaults to false.")
	printListeners := fs.Bool("li", false, "Print the currently connected listeners.")
	outpFormat := fs.String("format", "column", "Format specifies the format of the output: column, json.")
	//fs.StringVar(&conf.Proxy, "proxy", "", "the http proxy to use. Default: read the proxy settings from your environment.")

	fs.Usage = func() {
		fmt.Println(`casterstats - a tool to collect statistics from a BKG NtripCaster instance
		
Usage:
    casterstats [flags] <casterURL>
		
Flags:`)
		fs.PrintDefaults()
		fmt.Println(`
Examples:
	# Get general statistics
	$ casterstats -username=xxx -pw=xxx http://www.igs-ip.net:2101`)
		fmt.Printf("\nVersion: casterstats %s\n", version)
		fmt.Printf("Author : %s\n", "Erwin Wiesensarter, BKG Frankfurt")
	}

	fs.Parse(os.Args[1:])
	argsNotParsed := fs.Args()
	if len(argsNotParsed) > 1 {
		fmt.Fprintf(os.Stderr, "unknown arguments: %s\n", strings.Join(argsNotParsed, " "))
		fs.Usage()
		os.Exit(1)
	} else if len(argsNotParsed) < 1 {
		fmt.Fprintf(os.Stderr, "No caster given as argument\n")
		fs.Usage()
		os.Exit(1)
	}

	casterAddr := argsNotParsed[0]
	c, err := cli.New(casterAddr, opts)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer c.CloseIdleConnections()

	if *printListeners {
		listeners, err := c.GetListeners()
		if err != nil {
			log.Fatalf("%v", err)
		}
		if *outpFormat == "json" {
			lisJSON, _ := json.Marshal(listeners)
			fmt.Println(string(lisJSON))
		} else {
			fmt.Printf("%-17s %-20s %-12s %-10s %-13s %-14s %-30s %-12s %s\n",
				"IP", "Username", "MP", "ID", "ConnectedFor", "BytesWritten", "UserAgent", "Type", "Errors")
			for _, li := range listeners[:30] {
				fmt.Printf("%-17s %-20s %-12s %-10d %-13s %-14d %-30s %-12s %d\n",
					li.IP, li.User, li.MP, li.ID, li.ConnectedFor, li.BytesWritten, li.UserAgent, li.Type, li.Errors)
			}

		}

		/*  connsPerUser := make(map[string]int)
		   	for _, v := range listeners {
		   		//fmt.Printf("%-15s %-10s %-15s %-10s\n", v.User, v.MP, v.IP, v.UserAgent)
		   		connsPerUser[v.User]++
		   	}

		for user, nofConns := range connsPerUser {
			fmt.Printf("%-15s: %d\n", user, nofConns)
		} */

		os.Exit(0)
	}

	// default action
	stats, err := c.GetStats()
	if err != nil {
		log.Fatalf("%v", err)
	}

	if *outpFormat == "json" {
		statsJSON, _ := json.Marshal(stats)
		fmt.Println(string(statsJSON))
	} else {
		fmt.Printf("%v", stats)
	}

}

func checkError(errStr string, err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, errStr, ": ", err.Error())
		//log.Exit(errStr+":",  err.Error())
		_, file, line, ok := runtime.Caller(1)
		if ok {
			fmt.Println("Line number", file, line)
		}
		os.Exit(1)
		/*
			fmt.Println("Fatal error ", err.Error())
			os.Exit(1)
		*/
	}
}
