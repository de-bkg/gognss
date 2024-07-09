package rinex

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
	"github.com/stretchr/testify/assert"
)

func TestObsDecoder_readHeader(t *testing.T) {
	const header = `     3.03           OBSERVATION DATA    M                   RINEX VERSION / TYPE
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

	obsCodesGPSWanted := []ObsCode{"C1C", "L1C", "S1C", "C1W", "S1W", "C2W", "L2W", "S2W", "C2L", "L2L", "S2L", "C5Q", "L5Q", "S5Q"}
	gloSlotsWanted := map[gnss.PRN]int{{Sys: gnss.SysGLO, Num: 3}: 5, {Sys: gnss.SysGLO, Num: 4}: 6, {Sys: gnss.SysGLO, Num: 5}: 1,
		{Sys: gnss.SysGLO, Num: 6}: -4, {Sys: gnss.SysGLO, Num: 13}: -2, {Sys: gnss.SysGLO, Num: 14}: -7, {Sys: gnss.SysGLO, Num: 15}: 0,
		{Sys: gnss.SysGLO, Num: 16}: -1, {Sys: gnss.SysGLO, Num: 22}: -3, {Sys: gnss.SysGLO, Num: 23}: 3, {Sys: gnss.SysGLO, Num: 24}: 2}

	assert.Equal("O", dec.Header.RINEXType, "RINEX Type")
	assert.Equal(time.Date(2018, 11, 6, 20, 2, 25, 0, time.UTC), dec.Header.Date)
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
	assert.Equal(gloSlotsWanted, dec.Header.GloSlots)
	assert.ElementsMatch([]gnss.System{gnss.SysGPS, gnss.SysGAL, gnss.SysGLO, gnss.SysBDS}, dec.Header.SatSystems(), "used satellite systems")
	assert.Equal(obsCodesGPSWanted, dec.Header.ObsTypes[gnss.SysGPS], "observation types")
	t.Logf("RINEX Header: %+v\n", dec)
}

func TestObsDecoder_readRINEXHeaderV2(t *testing.T) {
	const header = `     2.11           OBSERVATION DATA    M (MIXED)           RINEX VERSION / TYPE
teqc  2019Feb25     IGN-RGP             20200603 08:03:25UTCPGM / RUN BY / DATE
Linux 2.6.32-573.12.1.x86_64|x86_64|gcc|Linux 64|=+         COMMENT
teqc  2019Feb25     Administrateur RGP  20200603 08:03:25UTCCOMMENT
teqc  2019Feb25     IGN-RGP             20200603 08:03:20UTCCOMMENT
teqc  2019Feb25     IGN-RGP             20200603 08:03:17UTCCOMMENT
  2.0430      (antenna height)                              COMMENT
 +48.38049068 (latitude)                                    COMMENT
  -4.49659762 (longitude)                                   COMMENT
0065.806      (elevation)                                   COMMENT
BIT 2 OF LLI FLAGS DATA COLLECTED UNDER A/S CONDITION       COMMENT
10004M004 (COGO code)                                       COMMENT
BRST                                                        MARKER NAME
10004M004                                                   MARKER NUMBER
Automatic           IGN                                     OBSERVER / AGENCY
5818R40023          TRIMBLE ALLOY       5.45                REC # / TYPE / VERS
1441017048          TRM57971.00     NONE                    ANT # / TYPE
  4231162.7880  -332746.9200  4745130.6890                  APPROX POSITION XYZ
        2.0431        0.0000        0.0000                  ANTENNA: DELTA H/E/N
     1     1                                                WAVELENGTH FACT L1/2
    22    L1    L2    C1    C2    P1    P2    D1    D2    S1# / TYPES OF OBSERV
          S2    L5    C5    D5    S5    L7    C7    D7    S7# / TYPES OF OBSERV
          L8    C8    D8    S8                              # / TYPES OF OBSERV
    30.0000                                                 INTERVAL
    18                                                      LEAP SECONDS
 SNR is mapped to RINEX snr flag value [0-9]                COMMENT
  L1 & L2: min(max(int(snr_dBHz/6), 0), 9)                  COMMENT
Forced Modulo Decimation to 30 seconds                      COMMENT
  2020     6     3     7     0    0.0000000     GPS         TIME OF FIRST OBS
                                                            END OF HEADER
`

	assert := assert.New(t)
	dec, err := NewObsDecoder(strings.NewReader(header))
	assert.NoError(err)
	assert.NotNil(dec)

	obsCodesWanted := []ObsCode{"L1", "L2", "C1", "C2", "P1", "P2", "D1", "D2", "S1", "S2", "L5", "C5", "D5", "S5", "L7", "C7", "D7", "S7", "L8", "C8", "D8", "S8"}

	assert.Equal("O", dec.Header.RINEXType, "RINEX Type")
	assert.Equal("BRST", dec.Header.MarkerName, "Markername")
	assert.Equal("10004M004", dec.Header.MarkerNumber, "Markernumber")
	assert.Equal("5818R40023", dec.Header.ReceiverNumber, "ReceiverNumber")
	assert.Equal("TRIMBLE ALLOY", dec.Header.ReceiverType, "ReceiverType")
	assert.Equal("5.45", dec.Header.ReceiverVersion, "ReceiverVersion")
	assert.Equal(time.Date(2020, 6, 3, 7, 0, 0, 0, time.UTC), dec.Header.TimeOfFirstObs, "TimeOfFirstObs")
	assert.Equal(30.000, dec.Header.Interval, "sampling interval")
	assert.Equal(obsCodesWanted, dec.Header.ObsTypes[dec.Header.SatSystem], "observation types")
	assert.Equal([]gnss.System{gnss.SysMIXED}, dec.Header.SatSystems(), "used satellite systems")
	assert.Equal(dec.Header.SatSystem, dec.Header.SatSystems()[0], "used satellite systems")
	t.Logf("RINEX Header: %+v\n", dec.Header)
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

	firstEpo := &Epoch{}
	numOfEpochs := 0
	for dec.NextEpoch() {
		numOfEpochs++
		epo := dec.Epoch()
		//fmt.Printf("%v\n", epo)
		if numOfEpochs == 1 {
			firstEpo = epo
		}
	}
	if err := dec.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	for _, obsPerSat := range firstEpo.ObsList {
		prn := obsPerSat.Prn
		if prn.Sys == gnss.SysGPS && prn.Num == 11 {
			wanted := SatObs{Prn: prn, Obss: map[ObsCode]Obs{
				"C1C": {Val: 20182171.481, LLI: 0, SNR: 0},
				"C2S": {Val: 0, LLI: 0, SNR: 0},
				"C2W": {Val: 2.0182168741e+07, LLI: 0, SNR: 0},
				"C5Q": {Val: 0, LLI: 0, SNR: 0},
				"D1C": {Val: 708.563, LLI: 0, SNR: 0},
				"D2S": {Val: 0, LLI: 0, SNR: 0},
				"D2W": {Val: 552.127, LLI: 0, SNR: 0},
				"D5Q": {Val: 0, LLI: 0, SNR: 0},
				"L1C": {Val: 1.06058033736e+08, LLI: 0, SNR: 8},
				"L2S": {Val: 0, LLI: 0, SNR: 0},
				"L2W": {Val: 8.2642616517e+07, LLI: 0, SNR: 8},
				"L5Q": {Val: 0, LLI: 0, SNR: 0},
				"S1C": {Val: 51.45, LLI: 0, SNR: 0},
				"S2S": {Val: 0, LLI: 0, SNR: 0},
				"S2W": {Val: 48.95, LLI: 0, SNR: 0},
				"S5Q": {Val: 0, LLI: 0, SNR: 0},
			}}
			assert.Equal(wanted, obsPerSat, "1st epoch G11")
		}
	}
	assert.Equal(120, numOfEpochs, "#epochs")
	t.Logf("got all epochs: %d", numOfEpochs)

	assert.Equal(5275, dec.lineNum, "# lines")
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
			_ = dec.Epoch()
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

	fmt.Printf("%d epochs found.", nEpochs)
	// Output: 120 epochs found.
}

func TestReadEpochsRINEX2(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/brst155h.20o"
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewObsDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)
	//t.Logf("%+v", dec.Header)

	firstEpo := &Epoch{}
	numOfEpochs := 0
	for dec.NextEpoch() {
		numOfEpochs++
		epo := dec.Epoch()
		if numOfEpochs == 1 {
			firstEpo = epo
		}
	}
	if err := dec.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	fmt.Printf("%+v\n", firstEpo)
	assert.Equal(120, numOfEpochs, "#epochs")
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

func Test_decodeObs(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		wantObs Obs
		wantErr bool
	}{
		{name: "t1", s: " 204471670.07007", wantObs: Obs{Val: float64(204471670.07), LLI: int8(0), SNR: int8(7)}, wantErr: false},
		{name: "t2", s: " 204471670.07017", wantObs: Obs{Val: float64(204471670.07), LLI: int8(1), SNR: int8(7)}, wantErr: false},
		{name: "t3", s: "        43.600", wantObs: Obs{Val: float64(43.6), LLI: int8(0), SNR: int8(0)}, wantErr: false},
		{name: "t4", s: "      -105.814  ", wantObs: Obs{Val: float64(-105.814), LLI: int8(0), SNR: int8(0)}, wantErr: false},
		{name: "t5", s: "      -105.814a_", wantObs: Obs{Val: float64(-105.814)}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotObs, err := decodeObs(tt.s, EpochFlagOK)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeObs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotObs, tt.wantObs) {
				t.Errorf("decodeObs() = %v, want %v", gotObs, tt.wantObs)
			}
		})
	}
}

func Test_decodeEpoLineHlp(t *testing.T) {
	// Helper test. Go slices are inclusive-exclusive.
	// Rnx2
	line := " 20  6  3  7  0 30.0000000  0 32S25G14R07G29G24R06R24G15G02G12G19G10"
	assert.Equal(t, "20  6  3  7  0 30.0000000", line[1:26], "epoch time")
	assert.Equal(t, "0", line[28:29], "epoch flag")
	assert.Equal(t, " 32", line[29:32], "num Satellites")

	// Rnx3
	line = "> 2018 11 25 22 59 30.0000000  0 26"
	assert.Equal(t, "2018 11 25 22 59 30.0000000", line[2:29], "epoch time")
	assert.Equal(t, "0", line[31:32], "epoch flag")
	assert.Equal(t, " 26", line[32:35], "num Satellites")
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

func TestParseEpochTime(t *testing.T) {
	assert := assert.New(t)
	tests := map[string]time.Time{
		"2018 11 06 19 00  0.0000000":              time.Date(2018, 11, 6, 19, 0, 0, 0, time.UTC),
		"2018 11 06 19 00 30.0000000":              time.Date(2018, 11, 6, 19, 0, 30, 0, time.UTC),
		"2019  8  6  3 44 29.0000000":              time.Date(2019, 8, 6, 3, 44, 29, 0, time.UTC),
		"2019  8  6  4 44  0.0000000":              time.Date(2019, 8, 6, 4, 44, 0, 0, time.UTC),
		"2019  8  6  5  5  8.0000000":              time.Date(2019, 8, 6, 5, 5, 8, 0, time.UTC),
		"2019  8  6  5  5  8.9538000":              time.Date(2019, 8, 6, 5, 5, 8, 953800000, time.UTC),
		"2023     4    23     0     0    1.000000": time.Date(2023, 4, 23, 0, 0, 1, 0, time.UTC),
	}

	for k, v := range tests {
		epTime, err := time.Parse(epochTimeFormat, k)
		assert.NoError(err)
		assert.Equal(v, epTime)
		fmt.Printf("epoch v3: %s\n", epTime)
	}

	// RINEX version 2
	tests = map[string]time.Time{
		"20  6  3  7  0 30.0000000": time.Date(2020, 6, 3, 7, 0, 30, 0, time.UTC),
		"20 06 03 07 59 30.0000000": time.Date(2020, 6, 3, 7, 59, 30, 0, time.UTC),
		"99 12  3  0  1  0.0000000": time.Date(1999, 12, 3, 0, 1, 0, 0, time.UTC),
	}

	for k, v := range tests {
		epTime, err := time.Parse(epochTimeFormatv2, k)
		assert.NoError(err)
		assert.Equal(v, epTime, "RINEX-2 epoch")
		fmt.Printf("epoch v2: %s\n", epTime)
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

func TestReadEpochsWithFlag4(t *testing.T) {
	assert := assert.New(t)
	// this file contains header at the end of the file
	// TODO find better example
	filepath := "testdata/white/kais329w.18o"
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewObsDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)

	numOfEpochs := 0
	for dec.NextEpoch() {
		numOfEpochs++
		_ = dec.Epoch()
	}
	err = dec.Err()
	assert.NoError(err)
}

func Test_parseEpochFlag(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    EpochFlag
		wantErr bool
	}{
		{name: "t1-2", in: "2", want: EpochFlagMovingAntenna, wantErr: false},
		{name: "t1-6", in: "6", want: EpochFlagCycleSlip, wantErr: false},
		{name: "t-neg", in: "-1", want: EpochFlagMovingAntenna, wantErr: true},
		{name: "t-toobig", in: "7", want: EpochFlagMovingAntenna, wantErr: true},
		{name: "t-toobig", in: "2000", want: EpochFlagMovingAntenna, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEpochFlag(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseEpochFlag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if got != tt.want {
					t.Errorf("parseEpochFlag() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
