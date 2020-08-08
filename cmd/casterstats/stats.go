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
	"time"

	"github.com/erwiese/gnss/ntrip"
)

const (
	version = "0.1"
)

func main() {
	opts := ntrip.Options{}
	opts.UserAgent = "BKGStatsClient/" + version
	fs := flag.NewFlagSet("BKGStatsClient/"+version, flag.ExitOnError)
	fs.StringVar(&opts.Username, "username", "", "Operator username to connect to the caster.")
	fs.StringVar(&opts.Password, "pw", "", "Operator Password.")
	fs.BoolVar(&opts.UnsafeSSL, "unsafeSSL", false, "If true, it will skip https certificate verification. Defaults to false.")
	printListeners := fs.Bool("listeners", false, "Print the currently connected listeners.")
	printSources := fs.Bool("sources", false, "Print the currently available Ntrip sources.")
	outpFormat := fs.String("format", "column", "Format specifies the format of the output: column, json.")
	printHeader := fs.Bool("H", false, "Print the header line. Defaults to false.")
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
    $ casterstats -username=xxx -pw=xxx http://www.igs-ip.net:2101
	
    # Get sources in JSON format
    $ casterstats -username=xxx -pw=xxx -sources -format=json http://www.igs-ip.net:2101`)
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
	c, err := ntrip.NewClient(casterAddr, opts)
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
			json.NewEncoder(os.Stdout).Encode(listeners)
		} else {
			if *printHeader {
				fmt.Printf("%-17s %-20s %-12s %-10s %-13s %-14s %-30s %-12s %s\n",
					"# IP", "Username", "MP", "ID", "ConnectedFor", "BytesWritten", "UserAgent", "Type", "Errors")
			}
			for _, li := range listeners {
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

	if *printSources {
		sources, err := c.GetSources()
		if err != nil {
			log.Fatalf("%v", err)
		}
		if *outpFormat == "json" {
			json.NewEncoder(os.Stdout).Encode(sources)
		} else {
			if *printHeader {
				fmt.Printf("%-17s %-12s %-9s %-45s %-13s %-21s %-8s %-12s %-14s %-14s\n",
					"# IP", "MP", "ID", "Agent", "ConnectedFor", "ConnectTime", "Clients", "ClientConns", "KBytesRead", "KBytesWritten")
			}
			for _, s := range sources {
				fmt.Printf("%-17s %-12s %-9d %-45s %-13s %-21s %-8d %-12d %-14d %-14d\n",
					s.IP, s.MP, s.ID, s.Agent, s.ConnectedFor, s.ConnectionTime.Format(time.RFC3339), s.Clients, s.ClientConnections, s.KBytesRead, s.KBytesWritten)
			}
		}
		os.Exit(0)
	}

	// default action
	stats, err := c.GetStats()
	if err != nil {
		log.Fatalf("%v", err)
	}

	if *outpFormat == "json" {
		json.NewEncoder(os.Stdout).Encode(stats)
	} else {
		if *printHeader {
			fmt.Printf("%-8s %-8s %-10s %-14s %-21s %-15s %-15s\n",
				"# Admins", "Sources", "Listeners", "Uptime", "LastResync", "KBytesRead", "KBytesWritten")
		}
		fmt.Printf("%-8d %-8d %-10d %-14s %-21s %-15d %-15d\n",
			stats.Admins, stats.Sources, stats.Listeners, stats.Uptime, stats.LastResync.Format(time.RFC3339), stats.KBytesRead, stats.KBytesWritten)
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
