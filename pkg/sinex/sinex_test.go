package sinex

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/de-bkg/gognss/pkg/site"
	"github.com/stretchr/testify/assert"
)

func TestDecoder_readHeader(t *testing.T) {
	assert := assert.New(t)
	snx, err := Decode(strings.NewReader("%=SNX 2.02 IGN 20:225:43202 IGN 20:208:75600 20:210:43200 C  1577 2 S E"))
	assert.NoError(err)
	assert.NotNil(snx)

	assert.Equal("2.02", snx.Header.Version, "Format Version")
	assert.Equal("IGN", snx.Header.Agency, "Agency")
	assert.Equal(time.Date(2020, 8, 12, 12, 0, 2, 0, time.UTC), snx.Header.CreationTime, "File Creation Time")
	assert.Equal("IGN", snx.Header.AgencyDataProvider, "Agency Data Provider")
	assert.Equal(time.Date(2020, 7, 26, 21, 0, 0, 0, time.UTC), snx.Header.StartTime, "Start Time")
	assert.Equal(time.Date(2020, 7, 28, 12, 0, 0, 0, time.UTC), snx.Header.EndTime, "End Time")
	assert.Equal(ObsTechCombined, snx.Header.ObsTech, "Obs Techn")
	assert.Equal(1577, snx.Header.NumEstimates, "Number of Estimates")
	assert.Equal(2, snx.Header.ConstraintCode, "Constraint Code")
	assert.Equal([]string{"S", "E"}, snx.Header.SolutionTypes, "Solution Types")
	t.Logf("Header: %+v\n", snx)
}

func TestNewDecoder(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/igs20P21161.snx"
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	snx, err := Decode(r)
	assert.NoError(err)
	assert.NotNil(snx)

	assert.Equal("2.02", snx.Header.Version, "Format Version")
	assert.Equal("IGN", snx.Header.Agency, "Agency")
	t.Logf("Header: %+v\n", snx)
	abmf := snx.Sites["ABMF"]

	recvFirst := &site.Receiver{Type: "SEPT POLARX5", SerialNum: "45014", Firmware: "5.3.2", ElevationCutoff: 0,
		TemperatureStabiliz: "", DateInstalled: time.Date(2020, 2, 7, 10, 0, 0, 0, time.UTC), DateRemoved: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)}
	assert.Equal(recvFirst, abmf.Receivers[0], "ABMF first receiver")

	//t.Logf("Header: %+v\n", snx.Sites["ABMF"])
}

func Test_parseTime(t *testing.T) {
	tests := map[string]time.Time{
		"95:120:86399": time.Date(1995, 4, 30, 23, 59, 59, 0, time.UTC),
		"20:038:00000": time.Date(2020, 2, 7, 0, 0, 0, 0, time.UTC),
		"20:038:36000": time.Date(2020, 2, 7, 10, 0, 0, 0, time.UTC),
		"20:211:43184": time.Date(2020, 7, 29, 11, 59, 44, 0, time.UTC),
		"00:000:00000": time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	for k, v := range tests {
		ti, err := parseTime(k) // or "2006__2"
		assert.NoError(t, err)
		assert.Equal(t, ti, v)
		fmt.Printf("epoch: %s\n", ti)
	}
}
