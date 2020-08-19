package rinex

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildFilename(t *testing.T) {
	assert := assert.New(t)

	rnx := &RnxFil{StartTime: time.Date(2018, 11, 6, 19, 0, 0, 0, time.UTC),
		DataSource: "R", FilePeriod: "01H", DataFreq: "30S", DataType: "MO", Format: "rnx"}

	assert.NotNil(rnx)
	rnx.SetStationName("WTZR")
	assert.Equal("WTZR", rnx.FourCharID, "FourCharID")

	rnx.SetStationName("BRUX00BEL")
	assert.Equal("BRUX", rnx.FourCharID, "FourCharID")
	assert.Equal(0, rnx.MonumentNumber, "MonumentNumber")
	assert.Equal(0, rnx.ReceiverNumber, "ReceiverNumber")
	assert.Equal("BEL", rnx.CountryCode, "CountryCode")

	t.Logf("RINEX: %+v", rnx)

	fn2, err := rnx.Rnx3Filename()
	if err != nil {
		t.Fatalf("Could not build Rnx filename: %v", err)
	}
	assert.Equal("BRUX00BEL_R_20183101900_01H_30S_MO.rnx", fn2, "Build RINEX3 filename")

	fn3, err := rnx.Rnx2Filename()
	if err != nil {
		t.Fatalf("Could not build Rnx filename: %v", err)
	}
	assert.Equal("brux310t.18o", fn3, "Build RINEX2 filename")
	t.Logf("filename: %s", fn3)

	rnx.Format = "crx"

	fn4, err := rnx.Rnx3Filename()
	if err != nil {
		t.Fatalf("Could not build Rnx filename: %v", err)
	}
	assert.Equal("BRUX00BEL_R_20183101900_01H_30S_MO.crx", fn4, "Build RINEX3 Hatanaka comp filename")
	t.Logf("filename: %s", fn4)

	fn5, err := rnx.Rnx2Filename()
	if err != nil {
		t.Fatalf("Could not build Rnx filename: %v", err)
	}
	assert.Equal("brux310t.18d", fn5, "Build RINEX2 filename")
	t.Logf("filename: %s", fn5)
}

func TestFileNamePattern(t *testing.T) {
	// Rnx2
	res := Rnx2FileNamePattern.FindStringSubmatch("adar335t.18d.Z") // obs hourly
	for k, v := range res {
		fmt.Printf("%d. %s\n", k, v)
	}
	fmt.Println("----------------------------")

	res = Rnx2FileNamePattern.FindStringSubmatch("bcln332d15.18o") // obs highrate
	for k, v := range res {
		fmt.Printf("%d. %s\n", k, v)
	}
	fmt.Println("----------------------------")

	// Rnx3
	res = Rnx3FileNamePattern.FindStringSubmatch("ALGO00CAN_R_20121601000_15M_01S_GO.rnx") // obs highrate
	for k, v := range res {
		fmt.Printf("%d. %s\n", k, v)
	}
	fmt.Println("----------------------------")

	res = Rnx3FileNamePattern.FindStringSubmatch("ALGO00CAN_R_20121600000_01D_MN.rnx.gz") // nav
	for k, v := range res {
		fmt.Printf("%d. %s\n", k, v)
	}
}

// Convert a RINEX 3 filename to a RINEX 2 filename.
func ExampleRnxFil_Rnx2Filename() {
	file, err := NewFile("ALGO00CAN_R_20121601001_15M_01S_GO.rnx")
	if err != nil {
		log.Fatalln(err)
	}

	rnx2name, err := file.Rnx2Filename()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(rnx2name)
	// Output: algo160k00.12o
}

// Convert a RINEX 2 filename to a RINEX 3 filename.
func ExampleRnxFil_Rnx3Filename() {
	file, err := NewFile("testdata/white/brst155h.20o")
	if err != nil {
		log.Fatalln(err)
	}
	file.CountryCode = "FRA"
	file.DataSource = "R"

	rnx3name, err := file.Rnx3Filename()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(rnx3name)
	// Output: BRST00FRA_R_20201550700_01H_30S_MO.rnx
}

func TestRnxFil_parseFilename(t *testing.T) {
	assert := assert.New(t)
	rnx, err := NewObsFil("ALGO01CAN_R_20121601000_15M_01S_GO.rnx.gz")
	assert.NoError(err)
	assert.Equal("ALGO", rnx.FourCharID, "FourCharID")
	assert.Equal(0, rnx.MonumentNumber, "MonumentNumber")
	assert.Equal(1, rnx.ReceiverNumber, "ReceiverNumber")
	assert.Equal("CAN", rnx.CountryCode, "CountryCode")
	assert.Equal("R", rnx.DataSource, "DataSource")
	assert.Equal(time.Date(2012, 6, 8, 10, 0, 0, 0, time.UTC), rnx.StartTime, "StartTime")
	assert.Equal("15M", rnx.FilePeriod, "FilePeriod")
	assert.Equal("01S", rnx.DataFreq, "DataFreq")
	assert.Equal("GO", rnx.DataType, "DataType")
	assert.Equal("rnx", rnx.Format, "Format")
	assert.Equal(false, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("gz", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)

	// Rnx2
	rnx, err = NewObsFil("abmf255u.19d.Z")
	assert.NoError(err)
	assert.Equal("ABMF", rnx.FourCharID, "FourCharID")
	assert.Equal(time.Date(2019, 9, 12, 20, 0, 0, 0, time.UTC), rnx.StartTime, "StartTime")
	assert.Equal("01H", rnx.FilePeriod, "FilePeriod")
	assert.Equal("crx", rnx.Format, "Format")
	assert.Equal(true, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("Z", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)

	rnx, err = NewObsFil("aggo237j.19n.Z ")
	assert.NoError(err)
	assert.Equal("AGGO", rnx.FourCharID, "FourCharID")
	assert.Equal(time.Date(2019, 8, 25, 9, 0, 0, 0, time.UTC), rnx.StartTime, "StartTime")
	assert.Equal("01H", rnx.FilePeriod, "FilePeriod")
	assert.Equal("rnx", rnx.Format, "Format")
	assert.Equal(false, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("Z", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)

	// highrates
	rnx, err = NewObsFil("adis240e00.19d.Z ")
	assert.NoError(err)
	assert.Equal("ADIS", rnx.FourCharID, "FourCharID")
	assert.Equal(time.Date(2019, 8, 28, 4, 0, 0, 0, time.UTC), rnx.StartTime, "StartTime")
	assert.Equal("15M", rnx.FilePeriod, "FilePeriod")
	assert.Equal("crx", rnx.Format, "Format")
	assert.Equal(true, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("Z", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)

	rnx, err = NewObsFil("adis240e15.19d.Z ")
	assert.NoError(err)
	assert.Equal("ADIS", rnx.FourCharID, "FourCharID")
	assert.Equal(time.Date(2019, 8, 28, 4, 15, 0, 0, time.UTC), rnx.StartTime, "StartTime")
	assert.Equal("15M", rnx.FilePeriod, "FilePeriod")
	assert.Equal("crx", rnx.Format, "Format")
	assert.Equal(true, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("Z", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)
}

func TestParseDoy(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(time.Date(2001, 12, 31, 0, 0, 0, 0, time.UTC), ParseDoy(2001, 365))
	assert.Equal(time.Date(2018, 12, 5, 0, 0, 0, 0, time.UTC), ParseDoy(2018, 339))
	assert.Equal(time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC), ParseDoy(2017, 1))
	assert.Equal(time.Date(2016, 12, 31, 0, 0, 0, 0, time.UTC), ParseDoy(2016, 366))
	assert.Equal(time.Date(2016, 12, 31, 0, 0, 0, 0, time.UTC), ParseDoy(16, 366))
	assert.Equal(time.Date(1998, 1, 2, 0, 0, 0, 0, time.UTC), ParseDoy(98, 2))

	// parse Rnx3 starttime
	tests := map[string]time.Time{
		"20121601000": time.Date(2012, 6, 8, 10, 0, 0, 0, time.UTC),
		"20192681900": time.Date(2019, 9, 25, 19, 0, 0, 0, time.UTC),
		"20192660415": time.Date(2019, 9, 23, 4, 15, 0, 0, time.UTC),
	}

	for k, v := range tests {
		t, err := time.Parse(rnx3StartTimeFormat, k) // or "2006__2"
		assert.NoError(err)
		assert.Equal(t, v)
		fmt.Printf("epoch: %s\n", t)
	}
}
