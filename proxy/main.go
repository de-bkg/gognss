package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Attempts stores the numner of attempts for the same request
	Attempts int = iota
	Retry
)

// Caster holds the data about a NtripCaster
type Caster struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

// SetAlive for this caster
func (ca *Caster) SetAlive(alive bool) {
	ca.mux.Lock()
	ca.Alive = alive
	ca.mux.Unlock()
}

// IsAlive returns true when caster is alive
func (ca *Caster) IsAlive() (alive bool) {
	ca.mux.RLock()
	alive = ca.Alive
	ca.mux.RUnlock()
	return
}

// CasterPool holds information about reachable casters
type CasterPool struct {
	casters []*Caster
	current uint64
}

// AddCaster to the server pool
func (s *CasterPool) AddCaster(caster *Caster) {
	s.casters = append(s.casters, caster)
}

// NextIndex atomically increase the counter and return an index
func (s *CasterPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.casters)))
}

// MarkCasterStatus changes a status of a caster
func (s *CasterPool) MarkCasterStatus(casterURL *url.URL, alive bool) {
	for _, b := range s.casters {
		if b.URL.String() == casterURL.String() {
			b.SetAlive(alive)
			break
		}
	}
}

// GetNextPeer returns next active peer to take a connection
func (s *CasterPool) GetNextPeer() *Caster {
	// loop entire casters to find out an Alive caster
	next := s.NextIndex()
	l := len(s.casters) + next // start from next and move a full cycle
	for i := next; i < l; i++ {
		idx := i % len(s.casters)     // take an index by modding
		if s.casters[idx].IsAlive() { // if we have an alive caster, use it and store if its not the original one
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.casters[idx]
		}
	}
	return nil
}

// HealthCheck pings the casters and update the status
func (s *CasterPool) HealthCheck() {
	for _, b := range s.casters {
		status := "up"
		alive := isCasterAlive(b.URL)
		b.SetAlive(alive)
		if !alive {
			status = "down"
		}
		log.Printf("%s [%s]\n", b.URL, status)
	}
}

// GetAttemptsFromContext returns the attempts for request
func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}
	return 1
}

// GetRetryFromContext returns the retries for request
func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}

// lb load balances the incoming request
func lb(w http.ResponseWriter, r *http.Request) {
	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}

	peer := casterPool.GetNextPeer()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

// isAlive checks whether a caster is Alive by establishing a TCP connection
func isCasterAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		log.Println("Caster unreachable, error: ", err)
		return false
	}
	defer conn.Close()
	return true
}

// healthCheck runs a routine for check status of the casters every 2 mins
func healthCheck() {
	t := time.NewTicker(time.Second * 20)
	for {
		select {
		case <-t.C:
			log.Println("Starting health check...")
			casterPool.HealthCheck()
			log.Println("Health check completed")
		}
	}
}

var casterPool CasterPool

func main() {
	var serverList string
	var port int
	flag.StringVar(&serverList, "casters", "", "Load balanced casters, use commas to separate")
	flag.IntVar(&port, "port", 3030, "Port to serve")
	flag.Parse()

	if len(serverList) == 0 {
		log.Fatal("Please provide one or more casters to load balance")
	}

	// parse servers
	tokens := strings.Split(serverList, ",")
	for _, tok := range tokens {
		serverURL, err := url.Parse(tok)
		if err != nil {
			log.Fatal(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(serverURL)
		proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
			log.Printf("[%s] %s\n", serverURL.Host, e.Error())
			retries := GetRetryFromContext(request)
			if retries < 3 {
				select {
				case <-time.After(10 * time.Millisecond):
					ctx := context.WithValue(request.Context(), Retry, retries+1)
					proxy.ServeHTTP(writer, request.WithContext(ctx))
				}
				return
			}

			// after 3 retries, mark this caster as down
			casterPool.MarkCasterStatus(serverURL, false)

			// if the same request routing for few attempts with different casters, increase the count
			attempts := GetAttemptsFromContext(request)
			log.Printf("%s(%s) Attempting retry %d\n", request.RemoteAddr, request.URL.Path, attempts)
			ctx := context.WithValue(request.Context(), Attempts, attempts+1)
			lb(writer, request.WithContext(ctx))
		}

		casterPool.AddCaster(&Caster{
			URL:          serverURL,
			Alive:        true,
			ReverseProxy: proxy,
		})
		log.Printf("Configured caster: %s\n", serverURL)
	}

	// create http server
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(lb),
	}

	// start health checking
	go healthCheck()

	log.Printf("ntripproxy started at :%d\n", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
