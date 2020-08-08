package client

// ----------------------------------------------------------------------------
// Functions for the BKG NtripCaster only
// ----------------------------------------------------------------------------

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	// Pattern for connection duration string, e.g. '12 days, 12 hours, 56 minutes and 5 seconds'
	connectionTimePattern = regexp.MustCompile(`((\d+) days, )?((\d+) hours, )?((\d+) minutes)?( and )?((\d+) seconds)?`)
)

// CasterStats contains general statistics like number of clients, sources etc.
type CasterStats struct {
	Admins     int           `json:"admins"`
	Sources    int           `json:"sources"`
	Listeners  int           `json:"listeners"`
	Uptime     time.Duration `json:"uptime"`
	LastResync time.Time     `json:"last_resync"`

	// Following fields since last resync
	KBytesRead    int `json:"KBytes_recv"`
	KBytesWritten int `json:"KBytes_sent"`
}

// CasterListener contains the information about an connected listener/client like IP, user agent etc.
type CasterListener struct {
	Host         string        `json:"host"`
	IP           string        `json:"ip"`
	User         string        `json:"username"`
	MP           string        `json:"mountpoint"`
	ID           int           `json:"id"`
	ConnectedFor time.Duration `json:"connected_for"`
	BytesWritten int           `json:"bytes_written"`
	Errors       int           `json:"errors"`
	UserAgent    string        `json:"user_agent"`
	Type         string        `json:"type"`
}

// CasterSource contains information about an active data stream.
type CasterSource struct {
	Host, IP                  string
	MP                        string
	ID                        int
	Agent                     string
	ConnectedFor              time.Duration
	ConnectionTime            time.Time
	Clients                   int
	ClientConnections         int
	KBytesRead, KBytesWritten int
}

// CasterConn defines a clients connection to the caster.
type CasterConn struct {
	MP           string
	Type         string
	ID           int
	UserAgent    string
	IP           string
	User         string
	ConnectedFor time.Duration
}

func (c *Client) sendRequest() (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", c.URL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.Useragent)
	req.Header.Add("Ntrip-Version", "Ntrip/2.0")
	req.SetBasicAuth(c.Username, c.Password)
	//req.Header.Set("Accept-Encoding", `gzip;q=0,bzip2;q=0,compress;q=0,deflate;q=0`)
	req.Header.Set("Accept", `text/html, text/plain, text/*`)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "close")

	// fmt.Println("The request header is:")
	// re, _ := httputil.DumpRequest(req, false)
	// fmt.Println(string(re))

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	ntripcaster := resp.Header.Get("Server")
	log.Printf("Server: %s", ntripcaster)

	if resp.StatusCode != http.StatusOK { // / if resp.Status != "200 OK"
		respi, _ := httputil.DumpResponse(resp, false)
		fmt.Print(string(respi))
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP Request failed: %d (%s)", resp.StatusCode, resp.Status)
	}

	return resp.Body, nil
}

// GetListeners fetches the currently connected listeners.
func (c *Client) GetListeners() ([]CasterListener, error) { // pruefen []*listener
	c.URL.Path = "admin"
	c.URL.RawQuery = "mode=listeners"

	reader, err := c.sendRequest()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	listeners := make([]CasterListener, 0, 1000)

	li := CasterListener{}
	fields := make([]string, 12)
	scanner := bufio.NewScanner(reader)
	ln := ""
	for scanner.Scan() {
		ln = scanner.Text()
		if strings.HasPrefix(ln, "<li>") {
			ln = strings.TrimPrefix(ln, "<li>[")
			ln = strings.TrimSuffix(ln, "]<br>")
			li = CasterListener{}
			fields = strings.Split(ln, "] [")
			for _, v := range fields {
				hlp := strings.Split(v, ":")
				// fixed in caster version 2.0.3?
				if len(hlp) == 1 {
					if strings.HasPrefix(hlp[0], "Mountpoint") {
						li.MP = strings.TrimPrefix(hlp[0], "Mountpoint /")
						continue
					}
				}

				key, val := hlp[0], strings.TrimSpace(hlp[1])
				switch key {
				case "Host":
					li.Host = val
				case "IP":
					li.IP = val
				case "User":
					li.User = val
				case "Mountpoint":
					li.MP = strings.TrimPrefix(val, "/")
				case "Id":
					if i, err := strconv.Atoi(val); err == nil {
						li.ID = i
					} else {
						log.Printf("%v", err)
					}
				case "Connected for":
					if d, err := parseDuration(val); err == nil {
						li.ConnectedFor = d
					} else {
						log.Printf("%v", err)
					}
				case "Bytes written":
					if i, err := strconv.Atoi(val); err == nil {
						li.BytesWritten = i
					} else {
						log.Printf("%v", err)
					}
				case "Errors":
					if i, err := strconv.Atoi(val); err == nil {
						li.Errors = i
					} else {
						log.Printf("%v", err)
					}
				case "User agent":
					li.UserAgent = val
				case "Type":
					li.Type = val // strings.TrimSuffix(val, "]")
				default:
					log.Printf("unknown key: %s", key)
				}

			}

			listeners = append(listeners, li)
		}

	}

	sort.Slice(listeners, func(i, j int) bool {
		return listeners[i].User < listeners[j].User
	})

	return listeners, nil
}

// GetSources fetches the current sources.
func (c *Client) GetSources() ([]CasterSource, error) {
	c.URL.Path = "admin"
	c.URL.RawQuery = "mode=sources"

	reader, err := c.sendRequest()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	sources := make([]CasterSource, 0, 200)

	fields := make([]string, 12)
	headline := make([]string, 12)
	src := CasterSource{}
	scanner := bufio.NewScanner(reader)
	ln := ""
line:
	for scanner.Scan() {
		ln = scanner.Text()
		if strings.HasPrefix(ln, "<tr>") {
			ln = strings.TrimPrefix(ln, "<tr><td>")
			ln = strings.TrimSuffix(ln, "</td></tr>")

			src = CasterSource{}

			fields = strings.Split(ln, "</td><td>")
			for i, val := range fields {
				val = strings.TrimSpace(val)
				if val == "Mountpoint" { // Headline
					headline = fields
					continue line
				}

				switch headline[i] {
				case "Mountpoint":
					src.MP = strings.TrimPrefix(val, "/")
				case "Host":
					src.Host = val
				case "IP":
					src.IP = val
				case "Id":
					if i, err := strconv.Atoi(val); err == nil {
						src.ID = i
					} else {
						log.Printf("%v", err)
					}
				case "Connected for":
					if d, err := parseDuration(val); err == nil {
						src.ConnectedFor = d
					} else {
						log.Printf("%v", err)
					}
				case "Time of connect":
					if t, err := time.Parse("02/Jan/2006:15:04:05", val); err == nil {
						src.ConnectionTime = t
					} else {
						log.Printf("could not parse time of connect: %v", err)
					}
				case "KBytes read":
					if i, err := strconv.Atoi(val); err == nil {
						src.KBytesRead = i
					} else {
						log.Printf("%v", err)
					}
				case "KBytes written":
					if i, err := strconv.Atoi(val); err == nil {
						src.KBytesWritten = i
					} else {
						log.Printf("%v", err)
					}
				case "Clients":
					if i, err := strconv.Atoi(val); err == nil {
						src.Clients = i
					} else {
						log.Printf("%v", err)
					}
				case "Client connections":
					if i, err := strconv.Atoi(val); err == nil {
						src.ClientConnections = i
					} else {
						log.Printf("%v", err)
					}
				case "Source Agent":
					src.Agent = val
				default:
					log.Printf("Unknown key \"%s\"", headline[i])
				}

			}

			sources = append(sources, src)
		}
	}

	sort.Slice(sources, func(i, j int) bool {
		return sources[i].MP < sources[j].MP
	})

	return sources, nil
}

// GetConnections fetches the current client connections.
func (c *Client) GetConnections() ([]CasterConn, error) {
	c.URL.Path = "admin"
	c.URL.RawQuery = "mode=connections"

	reader, err := c.sendRequest()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	conns := make([]CasterConn, 0, 1000)

	fields := make([]string, 8)
	header := [7]string{"Mountpoint", "Type", "Id", "Agent", "IP", "User", "Connected for"}

	// <a href="/admin?mode=kick&amp;argument=915">915</a>
	idRegEx := regexp.MustCompile(`<a href="\/admin\?mode=kick&amp;argument=\d+">(\d+)<\/a>`)

	conn := CasterConn{}
	scanner := bufio.NewScanner(reader)
	ln := ""
line:
	for scanner.Scan() {
		ln = scanner.Text()
		if strings.HasPrefix(ln, "<tr>") {
			ln = strings.TrimPrefix(ln, "<tr><td>")
			ln = strings.TrimSuffix(ln, "</td></tr>")

			conn = CasterConn{} // reset

			fields = strings.Split(ln, "</td><td>")
			for i, val := range fields {
				if val == "Mountpoint" { // Headline
					continue line
				}

				switch header[i] {
				case "Mountpoint":
					conn.MP = strings.TrimPrefix(val, "/")
				case "Type":
					conn.Type = val
				case "IP":
					conn.IP = val
				case "User":
					conn.User = val
				case "Id":
					res := idRegEx.FindStringSubmatch(val)
					if len(res) > 0 {
						if i, err := strconv.Atoi(res[1]); err == nil {
							conn.ID = i
						} else {
							log.Printf("%v", err)
						}
					} else {
						log.Printf("RegEx for \"Id\" did not match")
					}
				case "Connected for":
					if d, err := parseDuration(val); err == nil {
						conn.ConnectedFor = d
					} else {
						log.Printf("%v", err)
					}
				case "Agent":
					conn.UserAgent = val
				default:
					log.Printf("Unknown key \"%s\"", header[i])
				}
			}
			conns = append(conns, conn)
		}
	}

	sort.Slice(conns, func(i, j int) bool {
		return conns[i].MP < conns[j].MP
	})

	return conns, nil
}

// KickConnection stops an active client connection.
func (c *Client) KickConnection(id int) error {
	if id < 1 {
		return fmt.Errorf("Invalid id: %d", id) // kann das ueberhaupt vorkommen?
	}

	c.URL.Path = "admin"
	c.URL.RawQuery = "mode=kick&argument=" + strconv.Itoa(id)

	reader, err := c.sendRequest()
	if err != nil {
		return err
	}
	defer reader.Close()

	return nil
}

// GetStats requests some general statistics from the caster like number of clients, connections etc.
func (c *Client) GetStats() (*CasterStats, error) {
	c.URL.Path = "admin"
	c.URL.RawQuery = "mode=stats"

	stats := &CasterStats{}

	reader, err := c.sendRequest()
	if err != nil {
		return stats, err
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	ln := ""
	isBody := false
	fields := make([]string, 12)
	i := 0
	if err != nil {
		return nil, fmt.Errorf("Could not parse the number of sources in line: %s", ln)
	}
	for scanner.Scan() {
		ln = scanner.Text()
		if isBody == false && strings.HasPrefix(ln, "<body>") {
			isBody = true
			continue
		}
		if isBody == false {
			continue
		}
		if strings.HasPrefix(ln, "<div") {
			continue
		}

		ln = strings.TrimSuffix(ln, "<br>")

		fields = strings.Split(ln, ":")
		if len(fields) < 2 {
			//log.Printf(">>>>>>>>>>>>>>>>>> %s", ln)
			continue
		}
		key, val := fields[0], strings.TrimSpace(fields[1])

		if key == "Admins" {
			i, err = strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("Could not parse the number of Admins in line: %s", ln)
			}
			stats.Admins = i
		} else if key == "Sources" {
			i, err = strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("Could not parse the number of Sources in line: %s", ln)
			}
			stats.Sources = i
		} else if key == "Listeners" {
			i, err = strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("Could not parse the number of Listeners in line: %s", ln)
			}
			stats.Listeners = i
		} else if strings.Contains(key, "uptime") {
			if d, err := parseDuration(val); err == nil {
				stats.Uptime = d
			} else {
				log.Printf("%v", err)
			}
		} else if strings.Contains(key, "last resync") {
			if t, err := time.Parse("02/Jan/2006:150405", val+":"+fields[2]+fields[3]+fields[4]); err == nil {
				stats.LastResync = t
			} else {
				log.Printf("could not parse time of last resync: %v", err)
			}
		} else if strings.Contains(key, "KBytes read") {
			if i, err := strconv.Atoi(val); err == nil {
				stats.KBytesRead = i
			} else {
				log.Printf("%v", err)
			}
		} else if strings.Contains(key, "KBytes written") {
			if i, err := strconv.Atoi(val); err == nil {
				stats.KBytesWritten = i
			} else {
				log.Printf("%v", err)
			}
		}

	}

	return stats, nil
}

func parseDuration(dur string) (time.Duration, error) {
	res := connectionTimePattern.FindStringSubmatch(dur)
	for k, v := range res {
		//fmt.Printf("%d. %s\n", k, v)
		if v == "" {
			res[k] = "0"
		}
	}
	if len(res) == 10 {
		days, _ := strconv.Atoi(res[2])
		hours, _ := strconv.Atoi(res[4])
		min, _ := strconv.Atoi(res[6])
		secs, _ := strconv.Atoi(res[9])
		return time.Duration(days)*time.Hour*24 + time.Duration(hours)*time.Hour +
			time.Duration(min)*time.Minute + time.Duration(secs)*time.Second, nil
	}

	return 0, fmt.Errorf("Could not parse duration from: %s (%+v)", dur, res)
}
