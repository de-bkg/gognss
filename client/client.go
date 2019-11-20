// Package client provides functions for the Ntrip protocol
//
// The HTTP-based NtripClient to NtripCaster communication is fully compatible to
// HTTP 1.1, but in this context Ntrip uses only non-persistent connections.
//
// A loss of the TCP connection between communicating system-components (NtripClient to
// NtripCaster, NtripServer to NtripCaster) will be automatically recognized by the TCPsockets.
// This effect can be used to trigger software events such as an automated reconnection.
//
// The chunked transfer encoding is used to transfer a series of chunks, each with its own
// size information. This allows the transfer of streaming data together with information to
// verify the completeness of transfer. Every NTRIP 2.0 component must be able to handle
// this transfer encoding. The basic idea is to send the size of the following data block before
// the block itself as hexadecimal number.
//
// The net/http package automatically uses chunked encoding for request bodies when the content
// length is not known and the application did not explicitly set the transfer encoding to "identity".
package client

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

// yml config: https://github.com/valasek/timesheet/tree/master/server

// Options provides additional information for connecting to a Ntripcaster.
type Options struct {
	// Username is the Caster username
	Username string

	// Password is the Caster password
	Password string

	// Proxy configures the Proxy function on the HTTP client.
	//Proxy func(req *http.Request) (*url.URL, error)
	Proxy string

	//request *http.Request

	// UserAgent is the http User Agent, defaults to "InfluxDBClient".
	// For accessing the (BKG) casters websites (not sourcetable), it must not contain "NTRIP",
	// for accesing a stream, it MUST contain "NTRIP"
	UserAgent string

	// Timeout for influxdb writes, defaults to no timeout.
	//Timeout time.Duration

	// UnsafeSSL gets passed to the http client, if true, it will skip https certificate verification.
	// Defaults to false.
	UnsafeSSL bool

	// TLSConfig allows the user to set their own TLS config for the HTTP
	// Client. If set, this option overrides UnsafeSSL.
	TLSConfig *tls.Config

	// Transfered
	//DataChan chan []byte
	//ErrorChan chan error //  errorChan := make(chan error)
}

// Client is a caster client for http connections.
// The http.Client's Transport typically has internal state (cached TCP connections), so Clients should be reused
// instead of created as needed. Clients are safe for concurrent use by multiple goroutines.
type Client struct {
	*http.Client
	url       url.URL
	username  string
	password  string
	useragent string

	// Quit chan struct{}
	//errorChan chan error
	//dataChan  chan []byte
}

// New returns a new Ntrip Client with the given caster address and additional options.
// The caster addr should have the form "http://host:port".
func New(addr string, opts Options) (*Client, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("Unsupported protocol scheme: %s: your address must start with http:// or https://", u.Scheme)
	}

	if opts.UserAgent == "" {
		opts.UserAgent = "NTRIP Go Client" // Must start with NTRIP !!!!!!!!!!!!
	}

	// Transport see http DefaultTransport settings
	tr := &http.Transport{
		Dial: (&net.Dialer{ // -> establishment of the connection (kann wahrscheinlich weg!!)
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: opts.UnsafeSSL,
		},
		Proxy:                 http.ProxyFromEnvironment, // default
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second, // e.g. for using the wrong proxy!
	}

	if opts.Proxy != "" {
		proxyURL, err := url.Parse(opts.Proxy)
		if err != nil {
			return nil, fmt.Errorf("Could not parse proxy %s", opts.Proxy)
		}
		tr.Proxy = http.ProxyURL(proxyURL)
	}

	// if opts.TLSConfig != nil {
	// 	tr.TLSClientConfig = opts.TLSConfig
	// }

	return &Client{
		Client: &http.Client{
			//Timeout:   opts.Timeout,
			Transport: tr,
		},
		url:       *u,
		username:  opts.Username,
		password:  opts.Password,
		useragent: opts.UserAgent,
	}, nil
}

// IsCasterAlive checks whether the caster is alive.
// This is done by checking if the caster responds with its sourcetable.
func (c *Client) IsCasterAlive() bool {
	if _, err := c.GetSourcetable(); err != nil {
		return false
	}
	return true
}

// GetSourcetable downloads the sourcetable and returns the contents.
func (c *Client) GetSourcetable() (*http.Response, error) {
	req, err := http.NewRequest("GET", c.url.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.useragent)
	req.Header.Set("Connection", "close")
	req.Header.Add("Ntrip-Version", "Ntrip/2.0")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK { // / if resp.Status != "200 OK"
		resp.Body.Close()
		return nil, fmt.Errorf("GET failed: %d (%s)", resp.StatusCode, resp.Status)
	}

	if resp.Header.Get("Content-Type") != "gnss/sourcetable" {
		resp.Body.Close()
		return nil, fmt.Errorf("sourcetable %s with content-type %s", c.url.String(), resp.Header.Get("Content-Type"))
	}

	return resp, nil
}

// ParseSourcetable downloads the sourcetable and parses its contents.
func (c *Client) ParseSourcetable() ([]byte, error) {
	resp, err := c.GetSourcetable()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	servername := resp.Header.Get("Server")
	fmt.Printf("Server: %s\n", servername)

	// Debug
	/* 	fmt.Println("The request header is:")
	   	re, _ := httputil.DumpRequest(req, false)
	   	fmt.Println(string(re))

	   	fmt.Println("The response header is:")
	   	respi, _ := httputil.DumpResponse(resp, false)
	   	fmt.Print(string(respi)) */

	scanner := bufio.NewScanner(resp.Body)
	ln := ""
	for scanner.Scan() {
		ln = scanner.Text()
		fields := strings.Split(ln, ";")
		if fields[0] == "STR" || fields[0] == "CAS" || fields[0] == "NET" {
			// do something
		}

		fmt.Println(ln)
	}

	if ln != "ENDSOURCETABLE" {
		return nil, fmt.Errorf("invalid sourcetable: could not find string \"ENDSOURCETABLE\"")
	}

	return nil, nil
}

// ConnectStream requests GNSS data from the NtripCaster.
func (c *Client) ConnectStream(mp string) (io.ReadCloser, error) {

	// // Transport see http DefaultTransport settings
	// tr := &http.Transport{
	// 	Dial: (&net.Dialer{ // -> establishment of the connection
	// 		Timeout:   30 * time.Second,
	// 		KeepAlive: 30 * time.Second,
	// 	}).Dial,
	// 	Proxy:                 http.ProxyFromEnvironment,
	// 	IdleConnTimeout:       30 * time.Second,
	// 	TLSHandshakeTimeout:   10 * time.Second,
	// 	ResponseHeaderTimeout: 10 * time.Second, // e.g. for using the wrong proxy!
	// 	//TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	// }
	// if cl.Proxy != nil {
	// 	tr.Proxy = http.ProxyURL(cl.Proxy)
	// }
	// httpClient := &http.Client{Transport: tr}

	c.url.Path = mp // Set mountpoint
	req, err := http.NewRequest("GET", c.url.String(), nil)
	checkError(err)

	req.Header.Set("User-Agent", c.useragent)
	req.Header.Add("Ntrip-Version", "Ntrip/2.0")
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "close")

	//cl.request = req

	fmt.Println("The request header is:")
	re, _ := httputil.DumpRequest(req, false)
	fmt.Print(string(re))

	// Send request
	log.Printf("Pull stream %s", c.url.String())
	resp, err := c.Do(req)
	checkError(err)

	fmt.Println("The response header is:")
	respi, _ := httputil.DumpResponse(resp, false)
	fmt.Print(string(respi))

	if resp.StatusCode != http.StatusOK { // / if resp.Status != "200 OK"
		return nil, fmt.Errorf("GET failed: %d (%s)", resp.StatusCode, resp.Status)
	}

	if resp.Header.Get("Content-Type") != "gnss/data" { // Ntrip 2.0
		return nil, fmt.Errorf("sourcetable %s has content-type %s", c.url.String(), resp.Header.Get("Content-Type"))
	}

	return resp.Body, nil
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
