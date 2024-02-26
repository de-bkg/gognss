package rinex

import (
	"log"
	"testing"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
	"github.com/stretchr/testify/assert"
)

func TestNavFile_Rnx3Filename(t *testing.T) {
	file, err := NewNavFile("testdata/white/brst155h.20n")
	if err != nil {
		log.Fatalln(err)
	}
	file.CountryCode = "FRA"
	file.DataSource = "R"

	rnx3name, err := file.Rnx3Filename()
	if err != nil {
		log.Fatalln(err)
	}
	assert.Equal(t, "BRST00FRA_R_20201550700_01H_GN.rnx", rnx3name)
}

func TestNavFile_GetStats(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/AREG00PER_R_20201690000_01D_MN.rnx"
	fil, err := NewNavFile(filepath)
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.NotNil(fil)
	stats, err := fil.GetStats()
	assert.NoError(err)
	t.Logf("%+v", stats)
	assert.Equal(3612, stats.NumEphemeris)
	assert.ElementsMatch([]gnss.System{gnss.SysGPS, gnss.SysGLO, gnss.SysGAL, gnss.SysBDS, gnss.SysSBAS}, stats.SatSystems, "satellite systems")
	assert.Equal(105, len(stats.Satellites), "number of satellites")
	assert.Equal(time.Date(2020, 6, 16, 20, 10, 0, 0, time.UTC), stats.EarliestEphTime)
	assert.Equal(time.Date(2020, 6, 18, 00, 00, 0, 0, time.UTC), stats.LatestEphTime)
}
