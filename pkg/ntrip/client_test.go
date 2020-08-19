package ntrip

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var exAddr string = "http://www.euref-ip.net:2101"

func TestIsCasterAlive(t *testing.T) {
	c, err := NewClient(exAddr, Options{})
	assert.NoError(t, err)
	defer c.CloseIdleConnections()
	_ = c.IsCasterAlive()
}

func TestDownloadSourcetable(t *testing.T) {
	c, err := NewClient(exAddr, Options{})
	assert.NoError(t, err)
	defer c.CloseIdleConnections()

	st, err := c.GetSourcetable()
	assert.NoError(t, err)
	if _, err := io.Copy(os.Stdout, st); err != nil {
		t.Fatal(err)
	}
}

func TestParseSourcetable(t *testing.T) {
	c, err := NewClient(exAddr, Options{})
	assert.NoError(t, err)
	defer c.CloseIdleConnections()

	st, err := c.ParseSourcetable()
	assert.NoError(t, err)
	t.Logf("%+v", st)
}

func TestSourcetable_Write(t *testing.T) {
	c, err := NewClient(exAddr, Options{})
	assert.NoError(t, err)
	defer c.CloseIdleConnections()

	st, err := c.ParseSourcetable()
	assert.NoError(t, err)

	err = st.Write(os.Stdout)
	assert.NoError(t, err)
}

func TestSourcetable_HasStream(t *testing.T) {
	c, err := NewClient(exAddr, Options{})
	assert.NoError(t, err)
	defer c.CloseIdleConnections()

	st, err := c.ParseSourcetable()
	assert.NoError(t, err)

	searchStream := "JaGeLaeckMiDoch"
	_, found := st.HasStream(searchStream)
	assert.False(t, found, "search for stream in sourcetable")

	searchStream = "VALA00ESP0"
	if str, found := st.HasStream(searchStream); found {
		t.Logf("stream was found in st: %v", str)
	} else {
		t.Logf("stream was not found in st: %s", searchStream)
	}
}

func TestMergeSourcetables(t *testing.T) {
	c, err := NewClient(exAddr, Options{})
	assert.NoError(t, err)
	defer c.CloseIdleConnections()

	st1, err := c.ParseSourcetable()
	assert.NoError(t, err)

	exAddr2, err := url.Parse("http://igs-ip.net:2101")
	assert.NoError(t, err)
	c.URL = exAddr2
	st2, err := c.ParseSourcetable()
	assert.NoError(t, err)

	combinedST, err := MergeSourcetables(st1, st2)
	assert.NoError(t, err)
	t.Logf("%+v", combinedST)
}

func TestPullStream(t *testing.T) {
	user, pass := "", ""
	c, err := NewClient("http://www.euref-ip.net:2101", Options{Username: user, Password: pass, Timeout: 10})
	assert.NoError(t, err)
	defer c.CloseIdleConnections()

	r, err := c.GetStream("WARN00DEU0")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer r.Close()

	var buf [1024]byte
	for {
		n, err := r.Read(buf[0:])
		if err != nil {
			t.Logf("read error: %v", err)
			break
		}
		fmt.Print(string(buf[0:n]))
	}

	// type Message struct{}
	// var m Message
	// dec := NewRTCM3Decoder(r)
	// dec.Decode(&m)
}

/*
func TestRawFil(t *testing.T) {
	r, err := os.Open("testdata/YELL7_171207")
	if err != nil {
		t.Fatal(err)
	}

	type Message struct{}
	var m Message

	dec := NewRTCM3Decoder(r)
	dec.Decode(&m)

	// for {
	// 	if err := dec.Decode(&m); err == io.EOF {
	// 		break
	// 	} else if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	//fmt.Printf("%s: %s\n", m.Name, m.Text)
	// }
}

func TestDecoder(t *testing.T) {
	// RTCMv3 1005 test message
	var m1005 = []byte{0xD3, 0x00, 0x13, 0x3E, 0xD7, 0xD3, 0x02, 0x02, 0x98, 0x0E, 0xDE, 0xEF, 0x34, 0xB4, 0xBD, 0x62, 0xAC, 0x09, 0x41, 0x98, 0x6F, 0x33, 0x36, 0x0B, 0x98}

	// RTCMv3 1029 test message
	var m1029 = []byte{0xD3, 0x00, 0x27, 0x40, 0x50, 0x17, 0x00, 0x84, 0x73, 0x6E, 0x15, 0x1E, 0x55, 0x54, 0x46, 0x2D, 0x38, 0x20, 0xD0, 0xBF, 0xD1, 0x80, 0xD0, 0xBE, 0xD0, 0xB2, 0xD0, 0xB5, 0xD1, 0x80, 0xD0, 0xBA, 0xD0, 0xB0, 0x20, 0x77, 0xC3, 0xB6, 0x72, 0x74, 0x65, 0x72, 0xED, 0xA3, 0x3B}

	inp := append(m1005, m1029...)
	r := bytes.NewReader(inp)

	type Message struct{}
	var m Message

	dec := NewRTCM3Decoder(r)
	dec.Decode(&m)
} */
