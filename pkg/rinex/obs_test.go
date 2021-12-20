package rinex

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//var homeDir string = getHomeDir()

func TestObsDecoder_readHeader(t *testing.T) {
	const header = `
     3.03           OBSERVATION DATA    M                   RINEX VERSION / TYPE
sbf2rin-12.3.1                          20181106 200225 UTC PGM / RUN BY / DATE
BRUX                                                        MARKER NAME
13101M010                                                   MARKER NUMBER
GEODETIC                                                    MARKER TYPE
ROB                 ROB                                     OBSERVER / AGENCY
3001376             SEPT POLARX4TR      2.9.6               REC # / TYPE / VERS
00464               JAVRINGANT_DM   NONE                    ANT # / TYPE
  4027881.8478   306998.2610  4919498.6554                  APPROX POSITION XYZ
        0.4689        0.0000        0.0010                  ANTENNA: DELTA H/E/N
G   14 C1C L1C S1C C1W S1W C2W L2W S2W C2L L2L S2L C5Q L5Q  SYS / # / OBS TYPES
       S5Q                                                  SYS / # / OBS TYPES
E   12 C1C L1C S1C C5Q L5Q S5Q C7Q L7Q S7Q C8Q L8Q S8Q      SYS / # / OBS TYPES
R   12 C1C L1C S1C C2P L2P S2P C2C L2C S2C C3Q L3Q S3Q      SYS / # / OBS TYPES
C    6 C2I L2I S2I C7I L7I S7I                              SYS / # / OBS TYPES
SEPTENTRIO RECEIVERS OUTPUT ALIGNED CARRIER PHASES.         COMMENT
NO FURTHER PHASE SHIFT APPLIED IN THE RINEX ENCODER.        COMMENT
G L1C                                                       SYS / PHASE SHIFT
G L2W                                                       SYS / PHASE SHIFT
G L2L  0.00000                                              SYS / PHASE SHIFT
G L5Q  0.00000                                              SYS / PHASE SHIFT
E L1C  0.00000                                              SYS / PHASE SHIFT
E L5Q  0.00000                                              SYS / PHASE SHIFT
E L7Q  0.00000                                              SYS / PHASE SHIFT
E L8Q  0.00000                                              SYS / PHASE SHIFT
R L1C                                                       SYS / PHASE SHIFT
R L2P  0.00000                                              SYS / PHASE SHIFT
R L2C                                                       SYS / PHASE SHIFT
R L3Q  0.00000                                              SYS / PHASE SHIFT
C L2I                                                       SYS / PHASE SHIFT
C L7I                                                       SYS / PHASE SHIFT
    30.000                                                  INTERVAL
  2018    11     6    19     0    0.0000000     GPS         TIME OF FIRST OBS
  2018    11     6    19    59   30.0000000     GPS         TIME OF LAST OBS
    43                                                      # OF SATELLITES
 C1C    0.000 C2C    0.000 C2P    0.000                     GLONASS COD/PHS/BIS
DBHZ                                                        SIGNAL STRENGTH UNIT
 11 R03  5 R04  6 R05  1 R06 -4 R13 -2 R14 -7 R15  0 R16 -1 GLONASS SLOT / FRQ #
    R22 -3 R23  3 R24  2                                    GLONASS SLOT / FRQ #
															END OF HEADER
> 2018 11 06 19 00  0.0000000  0 31`

	assert := assert.New(t)
	dec, err := NewObsDecoder(strings.NewReader(header))
	assert.NoError(err)
	assert.NotNil(dec)

	assert.Equal("O", dec.Header.RINEXType, "RINEX Type")
	assert.Equal("BRUX", dec.Header.MarkerName, "Markername")
	assert.Equal("3001376", dec.Header.ReceiverNumber, "ReceiverNumber")
	assert.Equal("SEPT POLARX4TR", dec.Header.ReceiverType, "ReceiverType")
	assert.Equal("2.9.6", dec.Header.ReceiverVersion, "ReceiverVersion")
	assert.Equal("DBHZ", dec.Header.SignalStrengthUnit, "Signal Strength Unit")
	assert.Equal("DBHZ", dec.Header.SignalStrengthUnit, "Signal Strength Unit")
	assert.Equal(time.Date(2018, 11, 6, 19, 0, 0, 0, time.UTC), dec.Header.TimeOfFirstObs, "TimeOfFirstObs")
	assert.Equal(time.Date(2018, 11, 6, 19, 59, 30, 0, time.UTC), dec.Header.TimeOfLastObs, "TimeOfLastObs")
	assert.Equal(30.000, dec.Header.Interval, "sampling interval")
	assert.Equal(43, dec.Header.NSatellites, "number of satellites")
	assert.Len(dec.Header.ObsTypes, 4, "number of GNSS")
	t.Logf("RINEX Header: %+v\n", dec)
}

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
	assert.Equal("15M", rnx.FilePeriod, "FilePeriod")
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
	assert.Equal("01H", rnx.FilePeriod, "FilePeriod")
	assert.Equal("crx", rnx.Format, "Format")
	assert.Equal(true, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("Z", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)

	rnx, err = NewObsFile("aggo237j.19n.Z ")
	assert.NoError(err)
	assert.Equal("AGGO", rnx.FourCharID, "FourCharID")
	assert.Equal(time.Date(2019, 8, 25, 9, 0, 0, 0, time.UTC), rnx.StartTime, "StartTime")
	assert.Equal("01H", rnx.FilePeriod, "FilePeriod")
	assert.Equal("rnx", rnx.Format, "Format")
	assert.Equal(false, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("Z", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)

	// highrates
	rnx, err = NewObsFile("adis240e00.19d.Z ")
	assert.NoError(err)
	assert.Equal("ADIS", rnx.FourCharID, "FourCharID")
	assert.Equal(time.Date(2019, 8, 28, 4, 0, 0, 0, time.UTC), rnx.StartTime, "StartTime")
	assert.Equal("15M", rnx.FilePeriod, "FilePeriod")
	assert.Equal("crx", rnx.Format, "Format")
	assert.Equal(true, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("Z", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)

	rnx, err = NewObsFile("adis240e15.19d.Z ")
	assert.NoError(err)
	assert.Equal("ADIS", rnx.FourCharID, "FourCharID")
	assert.Equal(time.Date(2019, 8, 28, 4, 15, 0, 0, time.UTC), rnx.StartTime, "StartTime")
	assert.Equal("15M", rnx.FilePeriod, "FilePeriod")
	assert.Equal("crx", rnx.Format, "Format")
	assert.Equal(true, rnx.IsHatanakaCompressed(), "Hatanaka")
	assert.Equal("Z", rnx.Compression, "Compression")
	t.Logf("RINEX: %+v\n", rnx)
}
func TestBuildFilename(t *testing.T) {
	assert := assert.New(t)

	rnx := &ObsFile{RnxFil: &RnxFil{StartTime: time.Date(2018, 11, 6, 19, 0, 0, 0, time.UTC), DataSource: "R",
		FilePeriod: "01H", DataFreq: "30S", DataType: "MO", Format: "rnx"}}

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

func BenchmarkReadEpochs(b *testing.B) {
	b.ReportAllocs()
	filepath := "testdata/white/REYK00ISL_S_20192701000_01H_30S_MO.rnx"
	r, err := os.Open(filepath)
	if err != nil {
		b.Fatalf("%v", err)
	}
	defer r.Close()

	dec, err := NewObsDecoder(r)
	if err != nil {
		b.Fatalf("%v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for dec.NextEpoch() {
			epo := dec.Epoch()
			fmt.Printf("%v\n", epo)
		}
		if err := dec.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}
	}
}

// Loop over the epochs of a observation data input stream.
func ExampleObsDecoder_loop() {
	filepath := "testdata/white/REYK00ISL_R_20192701000_01H_30S_MO.rnx"
	r, err := os.Open(filepath)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer r.Close()

	dec, err := NewObsDecoder(r)
	if err != nil {
		log.Fatalf("%v", err)
	}

	nEpochs := 0
	for dec.NextEpoch() {
		nEpochs++
		_ = dec.Epoch()
		// Do something with epoch
	}
	if err := dec.Err(); err != nil {
		log.Printf("reading epochs: %v", err)
	}

	fmt.Println(nEpochs)
	// Output: 120
}

func TestReadEpochs(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/REYK00ISL_R_20192701000_01H_30S_MO.rnx"
	//filepath := "testdata/white/BRUX00BEL_R_20183101900_01H_30S_MO.rnx"
	//filepath := filepath.Join(homeDir, "IGS000USA_R_20192180344_02H_01S_MO.rnx")
	//filepath := filepath.Join(homeDir, "TEST07DEU_S_20192180000_01D_01S_MO.rnx")
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewObsDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)

	numOfEpochs := 0
	for dec.NextEpoch() {
		numOfEpochs++
		epo := dec.Epoch()
		fmt.Printf("%v\n", epo)
	}
	if err := dec.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	assert.Equal(120, numOfEpochs, "# epochs")
	t.Logf("got all epochs: %d", numOfEpochs)
}

func TestPrintEpochs(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/REYK00ISL_R_20192701000_01H_30S_MO.rnx"
	//filepath := "testdata/white/BRUX00BEL_R_20183101900_01H_30S_MO.rnx"
	//filepath := filepath.Join(homeDir, "IGS000USA_R_20192180344_02H_01S_MO.rnx")
	//filepath := filepath.Join(homeDir, "TEST07DEU_S_20192180000_01D_01S_MO.rnx")
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewObsDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)

	numOfEpochs := 0
	for dec.NextEpoch() {
		numOfEpochs++
		epo := dec.Epoch()
		epo.PrintTab(Options{SatSys: "GR"})
	}
	if err := dec.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}

func TestStat(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/REYK00ISL_R_20192701000_01H_30S_MO.rnx"
	obsFil, err := NewObsFile(filepath)
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.NotNil(obsFil)
	stat, err := obsFil.Stat()
	assert.NoError(err)
	t.Logf("%+v", stat)
}

func TestParseEpochTime(t *testing.T) {
	assert := assert.New(t)
	tests := map[string]time.Time{
		"2018 11 06 19 00  0.0000000": time.Date(2018, 11, 6, 19, 0, 0, 0, time.UTC),
		"2018 11 06 19 00 30.0000000": time.Date(2018, 11, 6, 19, 0, 30, 0, time.UTC),
		"2019  8  6  3 44 29.0000000": time.Date(2019, 8, 6, 3, 44, 29, 0, time.UTC),
		"2019  8  6  4 44  0.0000000": time.Date(2019, 8, 6, 4, 44, 0, 0, time.UTC),
		"2019  8  6  5  5  8.0000000": time.Date(2019, 8, 6, 5, 5, 8, 0, time.UTC),
	}

	for k, v := range tests {
		epTime, err := time.Parse(epochTimeFormat, k)
		assert.NoError(err)
		assert.Equal(v, epTime)
		fmt.Printf("epoch: %s\n", epTime)
	}
}

func TestParseDoY(t *testing.T) {
	assert := assert.New(t)

	// Go 1.13+ !!!

	// yyyy
	tests := map[string]time.Time{
		"2019001": time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
		"2018365": time.Date(2018, 12, 31, 0, 0, 0, 0, time.UTC),
		"1999360": time.Date(1999, 12, 26, 0, 0, 0, 0, time.UTC),
	}

	for k, v := range tests {
		ti, err := time.Parse("2006002", k) // or "2006__2"
		assert.NoError(err)
		assert.Equal(v, ti)
		fmt.Printf("epoch: %s\n", ti)
	}

	// yy
	tests = map[string]time.Time{
		"19001": time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
		"18365": time.Date(2018, 12, 31, 0, 0, 0, 0, time.UTC),
	}

	for k, v := range tests {
		ti, err := time.Parse("06002", k) // or "06__2"
		assert.NoError(err)
		assert.Equal(v, ti)
		fmt.Printf("epoch: %s\n", ti)
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

func TestSyncEpochs(t *testing.T) {
	assert := assert.New(t)
	//filePath1 := filepath.Join(homeDir, "IGS000USA_R_20192180344_02H_01S_MO.rnx")
	//filePath2 := filepath.Join(homeDir, "TEST07DEU_S_20192180000_01D_01S_MO.rnx")
	filePath1 := "testdata/white/REYK00ISL_R_20192701000_01H_30S_MO.rnx"
	filePath2 := "testdata/white/REYK00ISL_S_20192701000_01H_30S_MO.rnx"
	// file 1
	r1, err := os.Open(filePath1)
	assert.NoError(err)
	defer r1.Close()
	dec, err := NewObsDecoder(r1)
	assert.NoError(err)

	// file 2
	r2, err := os.Open(filePath2)
	assert.NoError(err)
	defer r2.Close()
	dec2, err := NewObsDecoder(r2)
	assert.NoError(err)

	numOfSyncEpochs := 0
	for dec.sync(dec2) {
		numOfSyncEpochs++
		syncEpo := dec.SyncEpoch()

		if numOfSyncEpochs == 1 {
			fmt.Printf("1st synced epoch: %s\n", syncEpo.Epo1.Time)
		}
	}
	if err := dec.Err(); err != nil {
		t.Fatalf("read error: %v", err)
	}

	assert.Equal(115, numOfSyncEpochs, "#synced epochs") // 325
}

func TestRnx2crx(t *testing.T) {
	assert := assert.New(t)
	tempDir := t.TempDir()
	/* 	t.Cleanup(func() {
		t.Logf("clean up dir %s", tempDir)
		os.RemoveAll(tempDir)
	}) */

	// Rnx3
	rnxFilePath, err := copyToTempDir("testdata/white/BRUX00BEL_R_20183101900_01H_30S_MO.rnx", tempDir)
	if err != nil {
		t.Fatalf("Could not copy to temp dir: %v", err)
	}
	crxFilePath, err := Rnx2crx(rnxFilePath)
	if err != nil {
		t.Fatalf("Could not Hatanaka compress file %s: %v", crxFilePath, err)
	}
	assert.Equal(filepath.Join(tempDir, "BRUX00BEL_R_20183101900_01H_30S_MO.crx"), crxFilePath, "rnx3 crx file")

	// Rnx2
	rnxFilePath, err = copyToTempDir("testdata/white/brst155h.20o", tempDir)
	if err != nil {
		t.Fatalf("Could not copy to temp dir: %v", err)
	}
	crxFilePath, err = Rnx2crx(rnxFilePath)
	if err != nil {
		t.Fatalf("Could not Hatanaka compress file: %v", err)
	}
	assert.Equal(filepath.Join(tempDir, "brst155h.20d"), crxFilePath, "rnx3 crx file")
}

func TestCrx2rnx(t *testing.T) {
	assert := assert.New(t)
	tempDir := t.TempDir()

	// Rnx3
	crxFilePath, err := copyToTempDir("testdata/white/BRUX00BEL_R_20202302000_01H_30S_MO.crx", tempDir)
	if err != nil {
		t.Fatalf("Could not copy to temp dir: %v", err)
	}

	rnxFilePath, err := Crx2rnx(crxFilePath)
	if err != nil {
		t.Fatalf("Could not Hatanaka decompress file: %v", err)
	}
	assert.Equal(filepath.Join(tempDir, "BRUX00BEL_R_20202302000_01H_30S_MO.rnx"), rnxFilePath, "rnx2 file")

	// Rnx2
	crxFilePath, err = copyToTempDir("testdata/white/brst155h.20d", tempDir)
	if err != nil {
		t.Fatalf("Could not copy to temp dir: %v", err)
	}

	rnxFilePath, err = Crx2rnx(crxFilePath)
	if err != nil {
		t.Fatalf("Could not Hatanaka decompress file: %v", err)
	}
	assert.Equal(filepath.Join(tempDir, "brst155h.20o"), rnxFilePath, "rnx2 file")
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
