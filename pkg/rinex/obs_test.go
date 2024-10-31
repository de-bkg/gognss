package rinex

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
	"github.com/stretchr/testify/assert"
)

func TestObsFile_parseFilename(t *testing.T) {
	assert := assert.New(t)
	rnx, err := NewObsFile("ALGO01CAN_R_20121601000_15M_01S_GO.rnx.gz")
	assert.NoError(err)
	assert.Equal("ALGO", rnx.FourCharID, "FourCharID")
	assert.Equal(0, rnx.MonumentNumber, "MonumentNumber")
	assert.Equal(1, rnx.ReceiverNumber, "ReceiverNumber")
	assert.Equal("CAN", rnx.CountryCode, "CountryCode")
	assert.Equal("R", rnx.DataSource, "DataSource")
	assert.Equal(time.Date(2012, 6, 8, 10, 0, 0, 0, time.UTC), rnx.StartTime, "StartTime")
	assert.Equal(FilePeriod15Min, rnx.FilePeriod, "FilePeriod")
	assert.Equal("01S", rnx.DataFreq, "DataFreq")
	assert.Equal("GO", rnx.DataType, "DataType")
	assert.Equal("rnx", rnx.Format, "Format")
	assert.Equal(false, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("gz", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)

	// Rnx2
	rnx, err = NewObsFile("abmf255u.19d.Z")
	assert.NoError(err)
	assert.Equal("ABMF", rnx.FourCharID, "FourCharID")
	assert.Equal(time.Date(2019, 9, 12, 20, 0, 0, 0, time.UTC), rnx.StartTime, "StartTime")
	assert.Equal(FilePeriodHourly, rnx.FilePeriod, "FilePeriod")
	assert.Equal("crx", rnx.Format, "Format")
	assert.Equal(true, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("Z", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)

	rnx, err = NewObsFile("aggo237j.19n.Z ")
	assert.NoError(err)
	assert.Equal("AGGO", rnx.FourCharID, "FourCharID")
	assert.Equal(time.Date(2019, 8, 25, 9, 0, 0, 0, time.UTC), rnx.StartTime, "StartTime")
	assert.Equal(FilePeriodHourly, rnx.FilePeriod, "FilePeriod")
	assert.Equal("rnx", rnx.Format, "Format")
	assert.Equal(false, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("Z", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)

	// highrates
	rnx, err = NewObsFile("adis240e00.19d.Z ")
	assert.NoError(err)
	assert.Equal("ADIS", rnx.FourCharID, "FourCharID")
	assert.Equal(time.Date(2019, 8, 28, 4, 0, 0, 0, time.UTC), rnx.StartTime, "StartTime")
	assert.Equal(FilePeriod15Min, rnx.FilePeriod, "FilePeriod")
	assert.Equal("crx", rnx.Format, "Format")
	assert.Equal(true, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("Z", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)

	rnx, err = NewObsFile("adis240e15.19d.Z ")
	assert.NoError(err)
	assert.Equal("ADIS", rnx.FourCharID, "FourCharID")
	assert.Equal(time.Date(2019, 8, 28, 4, 15, 0, 0, time.UTC), rnx.StartTime, "StartTime")
	assert.Equal(FilePeriod15Min, rnx.FilePeriod, "FilePeriod")
	assert.Equal("crx", rnx.Format, "Format")
	assert.Equal(true, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("Z", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)
}
func TestBuildFilename(t *testing.T) {
	assert := assert.New(t)

	rnx := &ObsFile{RnxFil: &RnxFil{StartTime: time.Date(2018, 11, 6, 19, 0, 0, 0, time.UTC), DataSource: "R",
		FilePeriod: FilePeriodHourly, DataFreq: "30S", DataType: "MO", Format: "rnx"}}

	assert.NotNil(rnx)
	err := rnx.SetStationName("WTZR")
	assert.NoError(err)
	assert.Equal("WTZR", rnx.FourCharID, "FourCharID")

	err = rnx.SetStationName("BRUX00BEL")
	assert.NoError(err)
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

	rnx.Format = "crx"
	fn4, err := rnx.Rnx3Filename()
	if err != nil {
		t.Fatalf("Could not build Rnx filename: %v", err)
	}
	assert.Equal("BRUX00BEL_R_20183101900_01H_30S_MO.crx", fn4, "Build RINEX3 Hatanaka comp filename")
	t.Logf("filename: %s", fn4)
}

// Convert a filename from RINEX-2 to RINEX-3.
func ExampleObsFile_Rnx3Filename() {
	file, err := NewObsFile("testdata/white/brst155h.20o")
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

func TestObsFile_ComputeObsStats(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/REYK00ISL_R_20192701000_01H_30S_MO.rnx"
	obsFil, err := NewObsFile(filepath)
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.NotNil(obsFil)
	stat, err := obsFil.ComputeObsStats()
	assert.NoError(err)
	//t.Logf("%+v", stat)
	assert.Equal("GR50 V4.31", obsFil.Header.Pgm)
	assert.Equal(120, stat.NumEpochs)
	assert.Equal(49, obsFil.Header.NSatellites, "number of satellites (header)")
	assert.Equal(49, stat.NumSatellites, "number of satellites (data)")
	assert.Equal(time.Second*30, stat.Sampling)
	assert.Equal(time.Date(2019, 9, 27, 10, 0, 0, 0, time.UTC), stat.TimeOfFirstObs)
	assert.Equal(time.Date(2019, 9, 27, 10, 59, 30, 0, time.UTC), stat.TimeOfLastObs)

	// Sort by PRNS
	prns := make([]gnss.PRN, 0, len(stat.ObsPerSat))
	for k := range stat.ObsPerSat {
		prns = append(prns, k)
	}
	sort.Sort(gnss.ByPRN(prns))
	for _, prn := range prns {
		fmt.Printf("%s: %+v\n", prn, stat.ObsPerSat[prn])
	}

	assert.Equal(map[ObsCode]int{"C1C": 7, "C5Q": 7, "C7Q": 7, "C8Q": 7, "D1C": 7, "D5Q": 7, "D7Q": 7, "D8Q": 7, "L1C": 7, "L5Q": 7, "L7Q": 7, "L8Q": 7, "S1C": 7, "S5Q": 7, "S7Q": 7, "S8Q": 7}, stat.ObsPerSat[gnss.PRN{Sys: gnss.ByAbbr["E"], Num: 7}], "obs E07")
	assert.Equal(map[ObsCode]int{"C1C": 120, "C2S": 0, "C2W": 120, "C5Q": 0, "D1C": 120, "D2S": 0, "D2W": 120, "D5Q": 0, "L1C": 120, "L2S": 0, "L2W": 120, "L5Q": 0, "S1C": 120, "S2S": 0, "S2W": 120, "S5Q": 0}, stat.ObsPerSat[gnss.PRN{Sys: gnss.ByAbbr["G"], Num: 11}], "obs G11")
	assert.Equal(map[ObsCode]int{"C5A": 119, "D5A": 119, "L5A": 72, "S5A": 119}, stat.ObsPerSat[gnss.PRN{Sys: gnss.ByAbbr["I"], Num: 6}], "obs I06")
	assert.Equal(map[ObsCode]int{"C1C": 94, "C2C": 94, "C2P": 94, "D1C": 94, "D2C": 94, "D2P": 94, "L1C": 92, "L2C": 92, "L2P": 92, "S1C": 94, "S2C": 94, "S2P": 94}, stat.ObsPerSat[gnss.PRN{Sys: gnss.ByAbbr["R"], Num: 19}], "obs R19")
	assert.Equal(map[ObsCode]int{"C2I": 117, "C7I": 0, "D2I": 117, "D7I": 0, "L2I": 116, "L7I": 0, "S2I": 117, "S7I": 0}, stat.ObsPerSat[gnss.PRN{Sys: gnss.ByAbbr["C"], Num: 22}], "obs C22")
}

func TestObsFile_ComputeObsStatsV211(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/brst155h.20o"
	obsFil, err := NewObsFile(filepath)
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.NotNil(obsFil)
	stat, err := obsFil.ComputeObsStats()
	assert.NoError(err)
	//t.Logf("%+v", stat)
	assert.Equal("teqc  2019Feb25", obsFil.Header.Pgm)
	assert.Equal(120, stat.NumEpochs)
	//assert.Equal(49, stat.NumSatellites, "number of satellites (data)")
	assert.Equal(time.Second*30, stat.Sampling)
	assert.Equal(time.Date(2020, 6, 3, 7, 0, 0, 0, time.UTC), stat.TimeOfFirstObs)
	assert.Equal(time.Date(2020, 6, 3, 7, 59, 30, 0, time.UTC), stat.TimeOfLastObs)

	prns := make([]gnss.PRN, 0, len(stat.ObsPerSat))
	for k := range stat.ObsPerSat {
		prns = append(prns, k)
	}
	sort.Sort(gnss.ByPRN(prns))
	for _, prn := range prns {
		fmt.Printf("%s: %+v\n", prn, stat.ObsPerSat[prn])
	}
	//STP BRST G TYP    C1    C2    C5    D1    D2    D5    L1    L2    L5    P2    S1    S2    S5
	//STO BRST G G02   120     0     0   120   119     0   120   119     0   119   120   119     0

	//C1:120 C2:0 C5:0 C7:0 C8:0 D1:120 D2:119 D5:0 D7:0 D8:0 L1:120 L2:119 L5:0 L7:0 L8:0 P1:0 P2:119 S1:120 S2:119 S5:0 S7:0 S8:0
	//assert.Equal(map[ObsCode]int{"C1C": 7, "C5Q": 7, "C7Q": 7, "C8Q": 7, "D1C": 7, "D5Q": 7, "D7Q": 7, "D8Q": 7, "L1C": 7, "L5Q": 7, "L7Q": 7, "L8Q": 7, "S1C": 7, "S5Q": 7, "S7Q": 7, "S8Q": 7}, stat.Obsstats[PRN{Sys: gnss.GNSSForAbbr["E"], Num: 7}], "obs E07")
}

func TestObsFile_ComputeObsStatsV2(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/wtzs3290.06o"
	obsFil, err := NewObsFile(filepath)
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.NotNil(obsFil)
	stat, err := obsFil.ComputeObsStats()
	assert.NoError(err)
	assert.Equal("CCRINEXO V2.3.3 UX", obsFil.Header.Pgm)
	assert.Equal(2760, stat.NumEpochs)
	assert.Equal(time.Second*30, stat.Sampling)
	assert.Equal(time.Date(2006, 11, 25, 0, 0, 0, 0, time.UTC), stat.TimeOfFirstObs)
	assert.Equal(time.Date(2006, 11, 25, 23, 59, 30, 0, time.UTC), stat.TimeOfLastObs)

	prns := make([]gnss.PRN, 0, len(stat.ObsPerSat))
	for k := range stat.ObsPerSat {
		prns = append(prns, k)
	}
	sort.Sort(gnss.ByPRN(prns))
	for _, prn := range prns {
		fmt.Printf("%s: %+v\n", prn, stat.ObsPerSat[prn])
	}
}

func TestDiff(t *testing.T) {
	assert := assert.New(t)
	//filePath1 := filepath.Join(homeDir, "IGS000USA_R_20192180344_02H_01S_MO.rnx")
	//filePath2 := filepath.Join(homeDir, "TEST07DEU_S_20192180000_01D_01S_MO.rnx")
	filePath1 := "testdata/white/REYK00ISL_R_20192701000_01H_30S_MO.rnx"
	filePath2 := "testdata/white/REYK00ISL_S_20192701000_01H_30S_MO.rnx"
	obs1, err := NewObsFile(filePath1)
	assert.NotNil(obs1)
	assert.NoError(err)
	obs2, err := NewObsFile(filePath2)
	assert.NotNil(obs2)
	assert.NoError(err)

	obs1.Opts.SatSys = "GR"
	err = obs1.Diff(obs2)
	assert.NoError(err)
}

func TestRnx2crx(t *testing.T) {
	t.Cleanup(func() {
		os.Remove("testdata/white/BRST155H.20O")
	})

	if _, err := copyFile("testdata/white/brst155h.20o", "testdata/white/BRST155H.20O"); err != nil {
		t.Fatal(err)
	}

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		rnxFilename string
		want        string
		wantErr     bool
	}{
		{name: "t1-rnx3", rnxFilename: "testdata/white/BRUX00BEL_R_20183101900_01H_30S_MO.rnx", want: "BRUX00BEL_R_20183101900_01H_30S_MO.crx", wantErr: false},
		{name: "t2-rnx2", rnxFilename: "testdata/white/brst155h.20o", want: "brst155h.20d", wantErr: false},
		{name: "t2-rnx2", rnxFilename: "testdata/white/BRST155H.20O", want: "BRST155H.20D", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rnxFilePath, err := copyToTempDir(tt.rnxFilename, tempDir)
			if err != nil {
				t.Fatalf("Could not copy to temp dir: %v", err)
			}

			got, err := Rnx2crx(rnxFilePath)
			if tt.wantErr {
				t.Logf("%s", err)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Crx2rnx() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			filename := got
			if got != "" {
				filename = filepath.Base(got)
			}
			if filename != tt.want {
				t.Errorf("Crx2rnx() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCrx2rnx(t *testing.T) {
	t.Cleanup(func() {
		os.Remove("testdata/white/BRST155H.20D")
	})

	if _, err := copyFile("testdata/white/brst155h.20d", "testdata/white/BRST155H.20D"); err != nil {
		t.Fatal(err)
	}

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		crxFilename string
		want        string
		wantErr     bool
	}{
		{name: "t1-rnx3", crxFilename: "testdata/white/BRUX00BEL_R_20202302000_01H_30S_MO.crx", want: "BRUX00BEL_R_20202302000_01H_30S_MO.rnx", wantErr: false},
		{name: "t2-rnx2", crxFilename: "testdata/white/brst155h.20d", want: "brst155h.20o", wantErr: false},
		{name: "t2-rnx2-uc", crxFilename: "testdata/white/BRST155H.20D", want: "BRST155H.20O", wantErr: false},
		{name: "t2-rnx3-err", crxFilename: "testdata/black/hubu00DEU_R_20230931500_01H_30S_MO.crx", want: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crxFilePath, err := copyToTempDir(tt.crxFilename, tempDir)
			if err != nil {
				t.Fatalf("Could not copy to temp dir: %v", err)
			}

			got, err := Crx2rnx(crxFilePath)
			if tt.wantErr {
				t.Logf("%s", err)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Crx2rnx() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			filename := got
			if got != "" {
				filename = filepath.Base(got)
			}
			if filename != tt.want {
				t.Errorf("Crx2rnx() = %v, want %v", got, tt.want)
			}
		})
	}
}

func copyToTempDir(src, targetDir string) (string, error) {
	_, err := copyFile(src, targetDir)
	if err != nil {
		return "", err
	}
	_, fileName := filepath.Split(src)
	return filepath.Join(targetDir, fileName), nil
}

// copyFile copies a file.
func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	// if dest is a dir, use the src's filename
	if destFileStat, err := os.Stat(dst); !os.IsNotExist(err) {
		if destFileStat.Mode().IsDir() {
			_, srcFileName := filepath.Split(src)
			dst = filepath.Join(dst, srcFileName)
		}
	}

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func TestObsHeader_Write(t *testing.T) {
	const headerW = `     3.03           OBSERVATION DATA    M                   RINEX VERSION / TYPE
sbf2rin-12.3.1                          20181106 200225 UTC PGM / RUN BY / DATE
SEPTENTRIO RECEIVERS OUTPUT ALIGNED CARRIER PHASES.         COMMENT
NO FURTHER PHASE SHIFT APPLIED IN THE RINEX ENCODER.        COMMENT
BRUX                                                        MARKER NAME
13101M010                                                   MARKER NUMBER
GEODETIC                                                    MARKER TYPE
ROB                 ROB                                     OBSERVER / AGENCY
3001376             SEPT POLARX4TR      2.9.6               REC # / TYPE / VERS
00464               JAVRINGANT_DM   NONE                    ANT # / TYPE
  4027881.8478   306998.2610  4919498.6554                  APPROX POSITION XYZ
        0.4689        0.0000        0.0010                  ANTENNA: DELTA H/E/N
    30.000                                                  INTERVAL
  2018    11     6    19     0    0.0000000     GPS         TIME OF FIRST OBS
  2018    11     6    19    59   30.0000000     GPS         TIME OF LAST OBS
 11 R03  5 R04  6 R05  1 R06 -4 R13 -2 R14 -7 R15  0 R16 -1 GLONASS SLOT / FRQ #
    R22 -3 R23  3 R24  2                                    GLONASS SLOT / FRQ #
                                                            END OF HEADER
`
	type fields struct {
		RINEXVersion    float32
		RINEXType       string
		SatSystem       gnss.System
		Pgm             string
		RunBy           string
		Date            time.Time
		Comments        []string
		MarkerName      string
		MarkerNumber    string
		MarkerType      string
		Observer        string
		Agency          string
		ReceiverNumber  string
		ReceiverType    string
		ReceiverVersion string
		AntennaNumber   string
		AntennaType     string
		Position        Coord
		AntennaDelta    CoordNEU
		DOI             string
		License         string
		StationInfos    []string
		//ObsTypes           map[gnss.System][]ObsCode skip test because of sorting. See test below.
		SignalStrengthUnit string
		Interval           float64
		TimeOfFirstObs     time.Time
		TimeOfLastObs      time.Time
		GloSlots           map[gnss.PRN]int
		LeapSeconds        int
		NSatellites        int
		Labels             []string
	}

	tests := []struct {
		name    string
		fields  fields
		wantW   string
		wantErr bool
	}{
		{name: "t1", fields: fields{RINEXVersion: 3.03, SatSystem: gnss.SysMIXED,
			Pgm: "sbf2rin-12.3.1", Date: time.Date(2018, 11, 6, 20, 2, 25, 0, time.UTC),
			Comments:   []string{"SEPTENTRIO RECEIVERS OUTPUT ALIGNED CARRIER PHASES.", "NO FURTHER PHASE SHIFT APPLIED IN THE RINEX ENCODER."},
			MarkerName: "BRUX", MarkerNumber: "13101M010", MarkerType: "GEODETIC",
			Observer: "ROB", Agency: "ROB",
			ReceiverNumber: "3001376", ReceiverType: "SEPT POLARX4TR", ReceiverVersion: "2.9.6",
			AntennaNumber: "00464", AntennaType: "JAVRINGANT_DM   NONE",
			Position:       Coord{X: 4027881.8478, Y: 306998.2610, Z: 4919498.6554},
			AntennaDelta:   CoordNEU{Up: 0.4689, E: 0.0000, N: 0.0010},
			Interval:       30.0,
			TimeOfFirstObs: time.Date(2018, 11, 6, 19, 0, 0, 0, time.UTC), TimeOfLastObs: time.Date(2018, 11, 6, 19, 59, 30, 0, time.UTC),
			GloSlots: map[gnss.PRN]int{{Sys: gnss.SysGLO, Num: 3}: 5, {Sys: gnss.SysGLO, Num: 4}: 6, {Sys: gnss.SysGLO, Num: 5}: 1,
				{Sys: gnss.SysGLO, Num: 6}: -4, {Sys: gnss.SysGLO, Num: 13}: -2, {Sys: gnss.SysGLO, Num: 14}: -7, {Sys: gnss.SysGLO, Num: 15}: 0,
				{Sys: gnss.SysGLO, Num: 16}: -1, {Sys: gnss.SysGLO, Num: 22}: -3, {Sys: gnss.SysGLO, Num: 23}: 3, {Sys: gnss.SysGLO, Num: 24}: 2},
		},
			wantW: headerW, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hdr := &ObsHeader{
				RINEXVersion:    tt.fields.RINEXVersion,
				RINEXType:       tt.fields.RINEXType,
				SatSystem:       tt.fields.SatSystem,
				Pgm:             tt.fields.Pgm,
				RunBy:           tt.fields.RunBy,
				Date:            tt.fields.Date,
				Comments:        tt.fields.Comments,
				MarkerName:      tt.fields.MarkerName,
				MarkerNumber:    tt.fields.MarkerNumber,
				MarkerType:      tt.fields.MarkerType,
				Observer:        tt.fields.Observer,
				Agency:          tt.fields.Agency,
				ReceiverNumber:  tt.fields.ReceiverNumber,
				ReceiverType:    tt.fields.ReceiverType,
				ReceiverVersion: tt.fields.ReceiverVersion,
				AntennaNumber:   tt.fields.AntennaNumber,
				AntennaType:     tt.fields.AntennaType,
				Position:        tt.fields.Position,
				AntennaDelta:    tt.fields.AntennaDelta,
				DOI:             tt.fields.DOI,
				//Licenses:        tt.fields.License,
				StationInfos: tt.fields.StationInfos,
				//ObsTypes:           tt.fields.ObsTypes,
				SignalStrengthUnit: tt.fields.SignalStrengthUnit,
				Interval:           tt.fields.Interval,
				TimeOfFirstObs:     tt.fields.TimeOfFirstObs,
				TimeOfLastObs:      tt.fields.TimeOfLastObs,
				GloSlots:           tt.fields.GloSlots,
				LeapSeconds:        tt.fields.LeapSeconds,
				NSatellites:        tt.fields.NSatellites,
				Labels:             tt.fields.Labels,
			}
			w := &bytes.Buffer{}
			if err := hdr.Write(w); (err != nil) != tt.wantErr {
				t.Errorf("ObsHeader.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("ObsHeader.Write() = %q, want %q", gotW, tt.wantW)
			}
		})
	}
}

func TestObsHeader_writeObsCodes(t *testing.T) {
	tests := []struct {
		name     string
		obsTypes map[gnss.System][]ObsCode
		wantW    string
	}{
		{name: "t1-Gal", obsTypes: map[gnss.System][]ObsCode{
			gnss.SysGAL: {"C1C", "L1C", "S1C", "C5Q", "L5Q", "S5Q", "C7Q", "L7Q", "S7Q", "C8Q", "L8Q", "S8Q"},
		}, wantW: "E   12 C1C L1C S1C C5Q L5Q S5Q C7Q L7Q S7Q C8Q L8Q S8Q      SYS / # / OBS TYPES\n"},
		{name: "t1-GPS", obsTypes: map[gnss.System][]ObsCode{
			gnss.SysGPS: {"C1C", "L1C", "S1C", "C1W", "S1W", "C2W", "L2W", "S2W", "C2L", "L2L", "S2L", "C5Q", "L5Q", "S5Q"},
		}, wantW: "G   14 C1C L1C S1C C1W S1W C2W L2W S2W C2L L2L S2L C5Q L5Q  SYS / # / OBS TYPES\n       S5Q                                                  SYS / # / OBS TYPES\n"},
		{name: "t1-Glo", obsTypes: map[gnss.System][]ObsCode{
			gnss.SysGLO: {"C1C", "L1C", "S1C", "C2P", "L2P", "S2P", "C2C", "L2C", "S2C", "C3Q", "L3Q", "S3Q"},
		}, wantW: "R   12 C1C L1C S1C C2P L2P S2P C2C L2C S2C C3Q L3Q S3Q      SYS / # / OBS TYPES\n"},
		{name: "t1-BDS", obsTypes: map[gnss.System][]ObsCode{
			gnss.SysBDS: {"C2I", "L2I", "S2I", "C7I", "L7I", "S7I"},
		}, wantW: "C    6 C2I L2I S2I C7I L7I S7I                              SYS / # / OBS TYPES\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			hdr := &ObsHeader{
				ObsTypes: tt.obsTypes,
			}
			hdr.writeObsCodes(w)
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("ObsHeader.writeObsCodes() = %q, want %q", gotW, tt.wantW)
			}
		})
	}
}

func TestObsHeader_formatFirstObsTime(t *testing.T) {
	tests := []struct {
		name string
		ti   time.Time
		want string
	}{
		{name: "t1", ti: time.Date(2018, 11, 6, 19, 0, 0, 0, time.UTC), want: "  2018    11     6    19     0    0.0000000"},
		{name: "t2", ti: time.Date(2018, 11, 6, 19, 59, 30, 0, time.UTC), want: "  2018    11     6    19    59   30.0000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hdr := &ObsHeader{}
			if got := hdr.formatFirstObsTime(tt.ti); got != tt.want {
				t.Errorf("ObsHeader.formatObsTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestObsHeader_writeGloSlotsAndFreqs(t *testing.T) {
	tests := []struct {
		name     string
		gloSlots map[gnss.PRN]int
		wantW    string
	}{
		{name: "t1", gloSlots: map[gnss.PRN]int{{Sys: gnss.SysGLO, Num: 3}: 5, {Sys: gnss.SysGLO, Num: 4}: 6, {Sys: gnss.SysGLO, Num: 5}: 1,
			{Sys: gnss.SysGLO, Num: 6}: -4, {Sys: gnss.SysGLO, Num: 13}: -2, {Sys: gnss.SysGLO, Num: 14}: -7, {Sys: gnss.SysGLO, Num: 15}: 0,
			{Sys: gnss.SysGLO, Num: 16}: -1, {Sys: gnss.SysGLO, Num: 22}: -3, {Sys: gnss.SysGLO, Num: 23}: 3, {Sys: gnss.SysGLO, Num: 24}: 2},
			wantW: " 11 R03  5 R04  6 R05  1 R06 -4 R13 -2 R14 -7 R15  0 R16 -1 GLONASS SLOT / FRQ #\n    R22 -3 R23  3 R24  2                                    GLONASS SLOT / FRQ #\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hdr := &ObsHeader{
				GloSlots: tt.gloSlots,
			}
			w := &bytes.Buffer{}
			hdr.writeGloSlotsAndFreqs(w)
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("ObsHeader.writeGloSlotsAndFreqs() = %q, want %q", gotW, tt.wantW)
			}
		})
	}
}
