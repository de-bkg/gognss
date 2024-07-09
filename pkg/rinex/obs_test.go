package rinex

import (
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

	assert.Equal(map[ObsCode]int{"C1C": 7, "C5Q": 7, "C7Q": 7, "C8Q": 7, "D1C": 7, "D5Q": 7, "D7Q": 7, "D8Q": 7, "L1C": 7, "L5Q": 7, "L7Q": 7, "L8Q": 7, "S1C": 7, "S5Q": 7, "S7Q": 7, "S8Q": 7}, stat.ObsPerSat[gnss.PRN{Sys: sysPerAbbr["E"], Num: 7}], "obs E07")
	assert.Equal(map[ObsCode]int{"C1C": 120, "C2S": 0, "C2W": 120, "C5Q": 0, "D1C": 120, "D2S": 0, "D2W": 120, "D5Q": 0, "L1C": 120, "L2S": 0, "L2W": 120, "L5Q": 0, "S1C": 120, "S2S": 0, "S2W": 120, "S5Q": 0}, stat.ObsPerSat[gnss.PRN{Sys: sysPerAbbr["G"], Num: 11}], "obs G11")
	assert.Equal(map[ObsCode]int{"C5A": 119, "D5A": 119, "L5A": 72, "S5A": 119}, stat.ObsPerSat[gnss.PRN{Sys: sysPerAbbr["I"], Num: 6}], "obs I06")
	assert.Equal(map[ObsCode]int{"C1C": 94, "C2C": 94, "C2P": 94, "D1C": 94, "D2C": 94, "D2P": 94, "L1C": 92, "L2C": 92, "L2P": 92, "S1C": 94, "S2C": 94, "S2P": 94}, stat.ObsPerSat[gnss.PRN{Sys: sysPerAbbr["R"], Num: 19}], "obs R19")
	assert.Equal(map[ObsCode]int{"C2I": 117, "C7I": 0, "D2I": 117, "D7I": 0, "L2I": 116, "L7I": 0, "S2I": 117, "S7I": 0}, stat.ObsPerSat[gnss.PRN{Sys: sysPerAbbr["C"], Num: 22}], "obs C22")
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
	//assert.Equal(map[ObsCode]int{"C1C": 7, "C5Q": 7, "C7Q": 7, "C8Q": 7, "D1C": 7, "D5Q": 7, "D7Q": 7, "D8Q": 7, "L1C": 7, "L5Q": 7, "L7Q": 7, "L8Q": 7, "S1C": 7, "S5Q": 7, "S7Q": 7, "S8Q": 7}, stat.Obsstats[PRN{Sys: sysPerAbbr["E"], Num: 7}], "obs E07")
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

/* func getHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("could not get homeDir")
	}
	return homeDir
} */
