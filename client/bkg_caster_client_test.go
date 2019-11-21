package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var user, pass string = "", ""

func TestGetStats(t *testing.T) {
	c, err := New(exAddr, Options{Username: user, Password: pass, UserAgent: "BKGGoClient"})
	assert.NoError(t, err)
	defer c.CloseIdleConnections()

	stats, err := c.GetStats()
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.NotZero(t, stats.Sources, "Number of sources")
	assert.NotZero(t, stats.Listeners, "Number of listeners")
	assert.NotZero(t, stats.KBytesRead, "KBytes read")
	assert.NotZero(t, stats.KBytesWritten, "KBytes written")
	assert.NotZero(t, stats.Uptime, "Uptime")
	assert.NotZero(t, stats.LastResync, "last resync")

	fmt.Printf("%+v\n", stats)
}

func TestGetListeners(t *testing.T) {
	c, err := New(exAddr, Options{Username: user, Password: pass, UserAgent: "BKGGoClient"})
	assert.NoError(t, err)
	defer c.CloseIdleConnections()

	listeners, err := c.GetListeners()
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.NotZero(t, len(listeners), "Number of Listeners")
	li := listeners[0]
	assert.NotZero(t, li.Host, "Listener Host")
	assert.NotZero(t, li.IP, "Listener IP")
	assert.NotZero(t, li.MP, "Listener MP")
	assert.NotZero(t, li.ID, "Id")
	assert.NotZero(t, li.User, "User")
	assert.NotZero(t, li.BytesWritten, "Bytes written")
	assert.NotZero(t, li.ConnectedFor, "Connected for")
	assert.NotZero(t, li.UserAgent, "Useragent")

	fmt.Printf("%+v\n", li)
}

func TestGetSources(t *testing.T) {
	c, err := New(exAddr, Options{Username: user, Password: pass, UserAgent: "BKGGoClient"})
	assert.NoError(t, err)
	defer c.CloseIdleConnections()

	sources, err := c.GetSources()
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.NotZero(t, len(sources), "Number of Sources")
	src := sources[0]
	assert.NotZero(t, src.Host, "Source Host")
	assert.NotZero(t, src.IP, "Source IP")
	assert.NotZero(t, src.MP, "Source MP")
	assert.NotZero(t, src.ID, "Source Id")
	assert.NotZero(t, src.Clients, "Source #clients")
	assert.NotZero(t, src.KBytesRead, "Source KBytes read")
	assert.NotZero(t, src.KBytesWritten, "Source KBytes written")
	assert.NotZero(t, src.ConnectionTime, "Source Connection Time")
	assert.NotZero(t, src.Agent, "Source Aagent")

	fmt.Printf("%+v\n", src)
}

func TestGetConnections(t *testing.T) {
	c, err := New(exAddr, Options{Username: user, Password: pass, UserAgent: "BKGGoClient"})
	assert.NoError(t, err)
	defer c.CloseIdleConnections()

	conns, err := c.GetConnections()
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.NotZero(t, len(conns), "Number of Connections")
	conn := conns[0]
	assert.NotZero(t, conn.IP, "IP")
	assert.NotZero(t, conn.MP, "MP")
	assert.NotZero(t, conn.ID, "Id")
	assert.NotZero(t, conn.User, "User")
	assert.NotZero(t, conn.ConnectedFor, "Connected for")
	assert.NotZero(t, conn.UserAgent, "User Agent")
	assert.NotZero(t, conn.Type, "conn type")

	fmt.Printf("%+v\n", conn)
}
