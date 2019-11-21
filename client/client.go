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
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
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
	//Proxy string

	//request *http.Request

	// UserAgent is the http User Agent, defaults to "InfluxDBClient".
	// For accessing the (BKG) casters websites (not sourcetable), it must not contain "NTRIP",
	// for accesing a stream, it MUST contain "NTRIP"
	UserAgent string

	// Timeout for GET requests as duration (e.g. "1h10m10s"), defaults to 5 seconds.
	Timeout string

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

// Sourcetable holds the streams from an NtripCasters' sourcetable.
type Sourcetable struct {
	Casters  []Caster
	Networks []Network
	Streams  []Stream
}

// Caster specifies a sourcetable record for a caster.
// See http://software.rtcm-ntrip.org/wiki/CAS.
type Caster struct {
	Host         string  // host name or IP address
	Port         int     // numeric port number
	Identifier   string  // Name of Caster or Caster provider
	Operator     string  // Name of institution or company operating the caster
	Nmea         bool    // Caster accepts NMEA input (1, true) or not (0)
	Country      string  // 3 char ISO 3166 country code
	Lat, Lon     float32 // Position, Latitude and Longitude in degree
	FallbackHost string  // Fallback Caster Internet address
	FallbackPort int     // Fallback Caster Port number
	Misc         string  // Miscellaneous information
}

// Network specifies a sourcetable record for a network.
// See http://software.rtcm-ntrip.org/wiki/NET.
type Network struct {
	Identifier string //  Network Identifier
	Operator   string //  Name of institution or company operating the network
	Auth       string // access protection for data streams: None (N), Basic (B) or Digest (D)
	Fee        bool   // User fee for data access: yes (Y) or no (N)
	WebNet     string // URL, Web address for network information
	WebStream  string // URL, Web address for stream information
	WebReg     string // URL or e-mail, Web or mail address for registration
	Misc       string // Miscellaneous information
}

// Stream specifies a sourcetable record for a stream.
// See http://software.rtcm-ntrip.org/wiki/STR.
type Stream struct {
	MP            string   // datastream mountpoint name
	Identifier    string   // Source identifier (most time nearest city)
	Format        string   //  Data format (see http://software.rtcm-ntrip.org/wiki/STR)
	FormatDetails string   // Specifics of data format (see http://software.rtcm-ntrip.org/wiki/STR)
	Carrier       int      // Phase information (see http://software.rtcm-ntrip.org/wiki/STR)
	NavSystem     []string // Navigation System (see http://software.rtcm-ntrip.org/wiki/STR)
	Network       string   // network name (Network.Identifier)
	Country       string   // 3 char ISO 3166 country code
	Lat, Lon      float32  // Position, Latitude and Longitude in degree
	Nmea          bool     // Caster accepts NMEA input (1, true) or not (0)
	Solution      int      // Generated by single base (0) or network (1)
	Generator     string   // Generating soft- or hardware
	Compression   string   // Compression algorithm
	Auth          string   // access protection for data streams: None (N), Basic (B) or Digest (D)
	Fee           bool     // User fee for data access: yes (Y) or no (N)
	Bitrate       int      // Datarate in bits per second
	Misc          string   // Miscellaneous information
}

// New returns a new Ntrip Client with the given caster address and additional options.
// The caster addr should have the form "http://host:port". It uses HTTP proxies
// as directed by the $HTTP_PROXY and $NO_PROXY (or $http_proxy and
// $no_proxy) environment variables.
func New(addr string, opts Options) (*Client, error) {
	casterURL, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if casterURL.Scheme != "http" && casterURL.Scheme != "https" {
		return nil, fmt.Errorf("Unsupported protocol scheme: %s: your address must start with http:// or https://", casterURL.Scheme)
	}

	if opts.UserAgent == "" {
		opts.UserAgent = "NTRIP Go Client" // Must start with NTRIP !!!!!!!!!!!!
	}

	// Transport see http DefaultTransport settings
	/* 	tr := &http.Transport{
	Proxy: ProxyFromEnvironment,
	TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.UnsafeSSL},
	} */

	// Set proxy
	/* 	if opts.Proxy != "" {
		proxyURL, err := url.Parse(opts.Proxy)
		if err != nil {
			return nil, fmt.Errorf("Could not parse proxy %s", opts.Proxy)
		}
		tr.Proxy = http.ProxyURL(proxyURL)
	} */

	// if opts.TLSConfig != nil {
	// 	tr.TLSClientConfig = opts.TLSConfig
	// }
	var timeout time.Duration = 5 * time.Second // default
	if opts.Timeout != "" {
		timeout, err = time.ParseDuration(opts.Timeout)
		if err != nil {
			return nil, fmt.Errorf("Could not parse timeout: %s: %v", opts.Timeout, err)
		}
	}

	return &Client{
		Client: &http.Client{
			Timeout: timeout,
			//Transport: tr, // see http DefaultTransport settings
		},
		url:       *casterURL,
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
func (c *Client) GetSourcetable() (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", c.url.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Ntrip-Version", "Ntrip/2.0")
	req.Header.Set("User-Agent", c.useragent)
	req.Header.Set("Connection", "close")

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	// Debug
	/* 	fmt.Println("The request header is:")
	   	re, _ := httputil.DumpRequest(req, false)
	   	fmt.Println(string(re))

	   	fmt.Println("The response header is:")
	   	respi, _ := httputil.DumpResponse(resp, false)
		   fmt.Print(string(respi)) */
	servername := resp.Header.Get("Server")
	fmt.Printf("Server: %s\n", servername)

	if resp.StatusCode != http.StatusOK { // / if resp.Status != "200 OK"
		resp.Body.Close()
		return nil, fmt.Errorf("GET failed: %d (%s)", resp.StatusCode, resp.Status)
	}

	if resp.Header.Get("Content-Type") != "gnss/sourcetable" {
		resp.Body.Close()
		return nil, fmt.Errorf("sourcetable %s with content-type %s", c.url.String(), resp.Header.Get("Content-Type"))
	}

	return resp.Body, nil
}

// ParseSourcetable downloads the sourcetable and parses its contents.
func (c *Client) ParseSourcetable() (*Sourcetable, error) {
	reader, err := c.GetSourcetable()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	st := &Sourcetable{}
	st.Casters = make([]Caster, 0, 5)
	st.Networks = make([]Network, 0, 5)
	st.Streams = make([]Stream, 0, 200)
	scanner := bufio.NewScanner(reader)
	ln := ""
	for scanner.Scan() {
		ln = scanner.Text()
		if strings.HasPrefix(ln, "#") { // comment
			continue
		}
		fields := strings.Split(ln, ";")
		switch fields[0] {
		case "CAS":
			if ca, err := parseCAS(ln); err == nil {
				st.Casters = append(st.Casters, ca)
			}
		case "NET":
			if netw, err := parseNET(ln); err == nil {
				st.Networks = append(st.Networks, netw)
			}
		case "STR":
			if str, err := parseSTR(ln); err == nil {
				st.Streams = append(st.Streams, str)
			}
		case "ENDSOURCETABLE":
			break
		default:
			log.Printf("illegal sourcetable line: %s", ln)
		}

		//fmt.Println(ln)
	}

	if ln != "ENDSOURCETABLE" {
		return nil, fmt.Errorf("invalid sourcetable: missing string \"ENDSOURCETABLE\"")
	}

	return st, nil
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
	if err != nil {
		return nil, err
	}

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
	if err != nil {
		return nil, err
	}

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

// parseCAS parses a CAS sourcetable line.
func parseCAS(line string) (Caster, error) {
	fields := strings.Split(line, ";")
	if len(fields) < 12 {
		return Caster{}, fmt.Errorf("missing fields at caster line: %s", line)
	}
	port, err := strconv.Atoi(fields[2])
	if err != nil {
		log.Printf("could not parse the casters port in line: %s", line)
	}
	fbPort, err := strconv.Atoi(fields[10])
	if err != nil {
		log.Printf("could not parse the casters fallback-port in line: %s", line)
	}
	nmea, err := strconv.ParseBool(fields[5])
	if err != nil {
		log.Printf("could not parse the casters nmea in line: %s", line)
	}
	lat, err := strconv.ParseFloat(fields[7], 32)
	if err != nil {
		log.Printf("could not parse the casters latitude in line: %s", line)
	}
	lon, err := strconv.ParseFloat(fields[8], 32)
	if err != nil {
		log.Printf("could not parse the casters longitude in line: %s", line)
	}

	return Caster{Host: fields[1], Port: port, Identifier: fields[3], Operator: fields[4],
		Nmea: nmea, Country: fields[6], Lat: float32(lat), Lon: float32(lon), FallbackHost: fields[9],
		FallbackPort: fbPort, Misc: fields[11]}, nil
}

// parseNET parses a NET sourcetable line.
func parseNET(line string) (Network, error) {
	fields := strings.Split(line, ";")
	if len(fields) < 9 {
		return Network{}, fmt.Errorf("missing fields at network line: %s", line)
	}

	fee := false
	if fields[4] == "Y" {
		fee = true
	}

	return Network{Identifier: fields[1], Operator: fields[2], Auth: fields[3], Fee: fee,
		WebNet: fields[5], WebStream: fields[6], WebReg: fields[7], Misc: fields[8]}, nil
}

// parseSTR parses a STR sourcetable line.
func parseSTR(line string) (Stream, error) {
	fields := strings.Split(line, ";")
	if len(fields) < 19 {
		return Stream{}, fmt.Errorf("missing fields at stream line: %s", line)
	}

	carrier, err := strconv.Atoi(fields[5])
	if err != nil {
		log.Printf("could not parse the streams carrier in line: %s", line)
	}

	navSystems := strings.Split(fields[6], "+")

	lat, err := strconv.ParseFloat(fields[9], 32)
	if err != nil {
		log.Printf("could not parse the streams latitude in line: %s", line)
	}
	lon, err := strconv.ParseFloat(fields[10], 32)
	if err != nil {
		log.Printf("could not parse the streams longitude in line: %s", line)
	}

	nmea, err := strconv.ParseBool(fields[11])
	if err != nil {
		log.Printf("could not parse the streams nmea in line: %s", line)
	}

	sol, err := strconv.Atoi(fields[12])
	if err != nil {
		log.Printf("could not parse the streams solution in line: %s", line)
	}

	fee := false
	if fields[4] == "Y" {
		fee = true
	}

	bitrate, err := strconv.Atoi(fields[17])
	if err != nil {
		log.Printf("could not parse the streams carrier in line: %s", line)
	}

	return Stream{MP: fields[1], Identifier: fields[2], Format: fields[3], FormatDetails: fields[4],
		Carrier: carrier, NavSystem: navSystems, Network: fields[7], Country: fields[8],
		Lat: float32(lat), Lon: float32(lon), Nmea: nmea, Solution: sol, Generator: fields[13],
		Compression: fields[14], Auth: fields[15], Fee: fee, Bitrate: bitrate, Misc: fields[18]}, nil
}
