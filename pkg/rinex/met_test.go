package rinex

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMeteoFile_ReadHeader(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/BAUT00DEU_R_20223131300_01H_10S_MM.rnx"
	metFil, err := NewMeteoFile(filepath)
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.NotNil(metFil)
	hdr, err := metFil.ReadHeader()
	assert.NoError(err)
	t.Logf("Header: %+v", hdr)
	t.Logf("1st sensor: %+v", hdr.Sensors[0])

	assert.Equal(float32(3.05), hdr.RINEXVersion)
	assert.Equal("M", hdr.RINEXType)
	assert.Equal("MAKERINEX 2.0.56659", hdr.Pgm)
	assert.Equal("NTRIPS20_7B76B7", hdr.RunBy)
	assert.Equal(time.Date(2022, 11, 9, 14, 1, 0, 0, time.UTC), hdr.Date)
	assert.Equal("BAUT", hdr.MarkerName)
	assert.Equal("14102M001", hdr.MarkerNumber)
	assert.Equal([]MeteoObsType{"PR", "TD", "HR", "WD", "WS", "RI"}, hdr.ObsTypes)
	assert.Equal(7, len(hdr.Sensors))
	firstSens := hdr.Sensors[0]
	assert.Equal(MeteoObsType("PR"), firstSens.ObservationType)
	assert.Equal("M3910031", firstSens.Model)
	assert.Equal("WXTPTU", firstSens.Type)
	assert.Equal(float64(1), firstSens.Accuracy)
	assert.Equal(3877548.3, firstSens.Position.X)
	assert.Equal(1004400.3, firstSens.Position.Y)
	assert.Equal(4947140.2, firstSens.Position.Z)
	assert.Equal(211.9, firstSens.Height)
}

func TestMeteoFile_ReadHeaderV2(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/func3060.19m"
	metFil, err := NewMeteoFile(filepath)
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.NotNil(metFil)
	hdr, err := metFil.ReadHeader()
	assert.NoError(err)
	t.Logf("Header: %+v", hdr)
	t.Logf("1st sensor: %+v", hdr.Sensors[0])

	assert.Equal(float32(2.11), hdr.RINEXVersion)
	assert.Equal("M", hdr.RINEXType)
	assert.Equal("Spider V7.1.1.7438", hdr.Pgm)
	assert.Equal("DGT", hdr.RunBy)
	assert.Equal(time.Date(2019, 11, 3, 0, 7, 0, 0, time.UTC), hdr.Date)
	assert.Equal("FUNC", hdr.MarkerName)
	assert.Equal("13911S001", hdr.MarkerNumber)
	assert.Equal([]MeteoObsType{"PR", "TD", "HR"}, hdr.ObsTypes)
}

func TestMeteoFile_Rnx3Filename(t *testing.T) {
	assert := assert.New(t)

	rnx := &MeteoFile{RnxFil: &RnxFil{StartTime: time.Date(2018, 11, 6, 19, 0, 0, 0, time.UTC), DataSource: "R",
		FilePeriod: FilePeriodHourly, DataFreq: "10S", Format: "rnx"}}
	assert.NotNil(rnx)

	rnx.SetStationName("BRUX00BEL")
	assert.Equal("BRUX", rnx.FourCharID, "FourCharID")
	assert.Equal(0, rnx.MonumentNumber, "MonumentNumber")
	assert.Equal(0, rnx.ReceiverNumber, "ReceiverNumber")
	assert.Equal("BEL", rnx.CountryCode, "CountryCode")

	fn, err := rnx.Rnx3Filename()
	if err != nil {
		t.Fatalf("Could not build Rnx filename: %v", err)
	}
	assert.Equal("BRUX00BEL_R_20183101900_01H_10S_MM.rnx", fn, "Build RINEX3 filename")
}

func TestMetDecoder_NextEpoch(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/BAUT00DEU_R_20223131300_01H_10S_MM.rnx"
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewMetDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)

	firstEpo := &MeteoEpoch{}
	numOfEpochs := 0
	for dec.NextEpoch() {
		numOfEpochs++
		epo := dec.Epoch()
		//fmt.Printf("%v\n", epo)
		if numOfEpochs == 1 {
			firstEpo = epo
		}
	}
	assert.NoError(dec.Err())
	t.Logf("1st epoch: %+v", firstEpo)

	assert.Equal(360, numOfEpochs, "#epochs")
	assert.Equal(380, dec.lineNum, "#lines")
}

func TestMeteoFile_ComputeObsStats(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/BAUT00DEU_R_20223131300_01H_10S_MM.rnx"
	metFil, err := NewMeteoFile(filepath)
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.NotNil(metFil)
	stat, err := metFil.ComputeObsStats()
	assert.NoError(err)
	t.Logf("%+v", stat)
	assert.Equal(360, stat.NumEpochs)
	assert.Equal(time.Second*10, stat.Sampling)
	assert.Equal(time.Date(2022, 11, 9, 13, 0, 1, 0, time.UTC), stat.TimeOfFirstObs)
	assert.Equal(time.Date(2022, 11, 9, 13, 59, 51, 0, time.UTC), stat.TimeOfLastObs)
}

func TestMeteoFile_ComputeObsStatsV2(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/func3060.19m"
	metFil, err := NewMeteoFile(filepath)
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.NotNil(metFil)
	stat, err := metFil.ComputeObsStats()
	assert.NoError(err)
	t.Logf("%+v", stat)
	assert.Equal(95, stat.NumEpochs)
	assert.Equal(time.Minute*15, stat.Sampling)
	assert.Equal(time.Date(2019, 11, 2, 0, 0, 3, 0, time.UTC), stat.TimeOfFirstObs)
	assert.Equal(time.Date(2019, 11, 2, 23, 45, 3, 0, time.UTC), stat.TimeOfLastObs)
}

func Test_decodeMeteoLineHlp(t *testing.T) {
	// Helper test. Go slices are inclusive-exclusive.
	// Rnx3
	line := " 2022 11  9 13  0  1  993.4   12.1   63.5  214.0    1.1    0.0"
	assert.Equal(t, "2022 11  9 13  0  1", line[1:20], "datetime")
	assert.Equal(t, "  993.4", line[20:27], "1st obs")
	assert.Equal(t, "   12.1", line[27:34], "2st obs")
}
