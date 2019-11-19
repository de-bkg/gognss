package client

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

// yml config: https://github.com/valasek/timesheet/tree/master/server

// HTTPConfig provides information for connecting to a Ntripcaster
type HTTPConfig struct {
	//Caster *url.URL

	// Addr should be of the form "http://host:port"
	Addr string

	MP string

	// Username is the Caster username
	Username string

	// Password is the Caster password
	Password string

	Proxy string

	// Proxy configures the Proxy function on the HTTP client.
	//Proxy func(req *http.Request) (*url.URL, error)

	//request *http.Request

	// UserAgent is the http User Agent, defaults to "InfluxDBClient".
	// For accessing the casters websites, it must not contain "NTRIP"
	// For accesing a stream, it MUST contain "NTRIP"
	UserAgent string

	// Timeout for influxdb writes, defaults to no timeout.
	//Timeout time.Duration

	// UnsafeSSL gets passed to the http client, if true, it will
	// skip https certificate verification. Defaults to false.
	UnsafeSSL bool

	// TLSConfig allows the user to set their own TLS config for the HTTP
	// Client. If set, this option overrides UnsafeSSL.
	TLSConfig *tls.Config

	// Transfered
	//DataChan chan []byte

	//ErrorChan chan error //  errorChan := make(chan error)
}

// The http.Client's Transport typically has internal state (cached TCP connections), so Clients should be reused instead of created as needed. Clients are safe for concurrent use by multiple goroutines.
type client struct {
	// N.B - if url.UserInfo is accessed in future modifications to the
	// methods on client, you will need to synchronize access to url.
	url        url.URL
	username   string
	password   string
	useragent  string
	httpClient *http.Client
	// Quit chan struct{}
	//errorChan chan error
	//dataChan  chan []byte
}

// NewNtripClient returns a new Client from the provided config.
// Client is safe for concurrent use by multiple goroutines.
func NewNtripClient(conf HTTPConfig) (Client, error) {
	if conf.UserAgent == "" {
		conf.UserAgent = "NTRIP Go Client" // Must start with NTRIP !!!!!!!!!!!!
	}

	u, err := url.Parse(conf.Addr)
	if err != nil {
		return nil, err
	} else if u.Scheme != "http" && u.Scheme != "https" {
		m := fmt.Sprintf("Unsupported protocol scheme: %s, your address"+
			" must start with http:// or https://", u.Scheme)
		return nil, errors.New(m)
	}
	// Set mountpoint
	if conf.MP != "" {
		u.Path = conf.MP
	}

	// Transport see http DefaultTransport settings
	tr := &http.Transport{
		Dial: (&net.Dialer{ // -> establishment of the connection (kann wahrscheinlich weg!!)
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: conf.UnsafeSSL,
		},
		Proxy:                 http.ProxyFromEnvironment, // default
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second, // e.g. for using the wrong proxy!
	}

	if conf.Proxy != "" {
		proxyURL, err := url.Parse(conf.Proxy)
		if err != nil {
			return nil, fmt.Errorf("Could not parse proxy %s", conf.Proxy)
		}
		tr.Proxy = http.ProxyURL(proxyURL)
	}

	// if conf.TLSConfig != nil {
	// 	tr.TLSClientConfig = conf.TLSConfig
	// }

	return &client{
		url:       *u,
		username:  conf.Username,
		password:  conf.Password,
		useragent: conf.UserAgent,
		httpClient: &http.Client{
			//Timeout:   conf.Timeout,
			Transport: tr,
		},
	}, nil
}

// DownloadSourcetable downloads the sourcetable named by st and returns the contents.
func (c *client) DownloadSourcetable() ([]byte, error) {
	req, err := http.NewRequest("GET", c.url.String(), nil)
	checkError(err)

	req.Header.Set("User-Agent", c.useragent)
	req.Header.Set("Connection", "close")
	req.Header.Add("Ntrip-Version", "Ntrip/2.0")

	resp, err := c.httpClient.Do(req)
	checkError(err)
	if resp.StatusCode != 200 {
		fmt.Println(resp.Status)
		os.Exit(2)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "gnss/sourcetable" {
		return nil, fmt.Errorf("sourcetable %s with content-type %s", c.url.String(), resp.Header.Get("Content-Type"))
	}

	ntripcaster := resp.Header.Get("Server")
	fmt.Printf("Server: %s\n", ntripcaster)

	fmt.Println("The request header is:")
	re, _ := httputil.DumpRequest(req, false)
	fmt.Println(string(re))

	fmt.Println("The response header is:")
	respi, _ := httputil.DumpResponse(resp, false)
	fmt.Print(string(respi))

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

/*
func TestDownloadSourcetable(t *testing.T) {
	addr := "http://www.igs-ip.net:2101"
	proxy := "http://"

	config := HTTPConfig{Addr: addr, Proxy: proxy}
	c, err := NewNtripClient(config)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer c.Close()

	_, err = c.DownloadSourcetable()
	if err != nil {
		t.Fatalf("%v", err)
	}
	//t.Logf("sourcetable: \n%s", st)
} */
