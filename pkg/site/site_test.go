package site

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSite_cleanAntennatypes(t *testing.T) {
	antennas := []*Antenna{{Type: "ASH701945E_M NONE", Radome: "NONE", RadomeSerialNum: "", DateInstalled: time.Date(2006, 07, 07, 0, 0, 0, 0, time.UTC),
		DateRemoved: time.Date(2008, 03, 19, 8, 45, 0, 0, time.UTC)},
		{Type: "LEIAR25.R3", Radome: "LEIT", RadomeSerialNum: "", DateInstalled: time.Date(2008, 3, 19, 9, 0, 0, 0, time.UTC),
			DateRemoved: time.Date(2008, 05, 20, 10, 0, 0, 0, time.UTC)}}
	s := Site{Antennas: antennas}
	err := s.cleanAntennas(false)
	assert.NoError(t, err)
	assert.Equal(t, "ASH701945E_M    NONE", s.Antennas[0].Type, "ANT TYPE string length")
	assert.Equal(t, "LEIAR25.R3      LEIT", s.Antennas[1].Type, "ANT TYPE string length")
}

func TestSite_cleanAntennaDates(t *testing.T) {
	antennas := []*Antenna{{Type: "ASH701945E_M NONE", Radome: "NONE", RadomeSerialNum: "",
		SerialNum: "CR620023301", ReferencePoint: "BPA", EccUp: 0.1266, EccNorth: 0.001, EccEast: 0, AlignmentFromTrueNorth: 0,
		CableType: "ANDREW heliax LDF2-50A", CableLength: 60, DateInstalled: time.Date(2006, 07, 07, 0, 0, 0, 0, time.UTC),
		DateRemoved: time.Date(2008, 03, 19, 8, 45, 0, 0, time.UTC), Notes: ""},
		{Type: "LEIAR25.R3", Radome: "LEIT", RadomeSerialNum: "",
			SerialNum: "CR620023301", ReferencePoint: "BPA", EccUp: 0.1266, EccNorth: 0.001, EccEast: 0, AlignmentFromTrueNorth: 0,
			CableType: "ANDREW heliax LDF2-50A", CableLength: 60, DateInstalled: time.Date(2008, 3, 19, 8, 30, 0, 0, time.UTC),
			DateRemoved: time.Date(2008, 05, 20, 10, 0, 0, 0, time.UTC), Notes: ""},
	}
	s := Site{Antennas: antennas}
	err := s.cleanAntennas(true)
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2008, 3, 19, 8, 30, 0, 0, time.UTC).Add(timeShift*-1), s.Antennas[0].DateRemoved, "adjust antenna dates")
}

// ReadFromGAJSON parses the GA formated JSON site description file.
func TestReadFromGAJSON(t *testing.T) {
	assert := assert.New(t)

	f, err := os.Open("testdata/ALIC.json")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer f.Close()

	site := new(Site)
	err = json.NewDecoder(f).Decode(site)
	assert.NoError(err)
	t.Logf("%+v", site)
}

func TestSite_StationInfo(t *testing.T) {
	assert := assert.New(t)
	f, err := os.Open("testdata/brux_20200225.log")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer f.Close()

	site, err := DecodeSitelog(f)
	assert.NoError(err)

	err = site.ValidateAndClean(false)
	assert.NoError(err)

	staInfo, err := site.StationInfo()
	assert.NoError(err)
	assert.Len(staInfo, 19, "number of station changes")
	//t.Logf("%+v", staInfo)
	for _, sta := range staInfo {
		fmt.Println(sta)
	}

	// check some dates
	assert.Equal(time.Date(2006, 7, 7, 0, 0, 0, 0, time.UTC), staInfo[0].From, "First From Date")
	assert.Equal(time.Date(2008, 2, 14, 9, 0, 0, 0, time.UTC), staInfo[0].To, "First To Date")
	assert.Equal(time.Date(2008, 2, 15, 8, 0, 0, 0, time.UTC), staInfo[1].From, "Second From Date")
	assert.Equal(time.Date(2008, 3, 19, 8, 45, 0, 0, time.UTC), staInfo[1].To, "Second To Date")
	assert.Equal(time.Date(2008, 9, 2, 13, 0, 0, 0, time.UTC), staInfo[4].From, "5 From Date")
	assert.Equal(time.Date(2008, 9, 22, 8, 59, 59, 0, time.UTC), staInfo[4].To, "5 To Date")
	assert.Equal(time.Date(2020, 2, 25, 13, 30, 0, 0, time.UTC), staInfo[len(staInfo)-1].From, "Last From Date")
	assert.Equal(time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC), staInfo[len(staInfo)-1].To, "Last To Date")
}

func TestSites_WriteBerneseSTA(t *testing.T) {
	assert := assert.New(t)

	fmtvers := "1.03"

	var sites Sites
	sitelogs := []string{"testdata/brux_20200225.log", "testdata/WTZR00DEU_20200602.log"}
	for _, slPath := range sitelogs {
		f, err := os.Open(slPath)
		if err != nil {
			t.Fatalf("%v", err)
		}
		defer f.Close()

		site, err := DecodeSitelog(f)
		assert.NoError(err)

		err = site.ValidateAndClean(false)
		assert.NoError(err)

		sites = append(sites, site)
	}

	//w := &bytes.Buffer{}
	w := os.Stdout

	err := sites.WriteBerneseSTA(w, fmtvers)
	assert.NoError(err)
}
