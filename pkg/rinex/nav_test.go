package rinex

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
	"github.com/stretchr/testify/assert"
)

func TestNavDecoder_readHeader(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/AREG00PER_R_20201690000_01D_MN.rnx"
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewNavDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)

	assert.Equal(float32(3.04), dec.Header.RINEXVersion, "RINEX Version")
	assert.Equal("N", dec.Header.RINEXType, "RINEX Type")
	assert.Equal(gnss.SysMIXED, dec.Header.SatSystem, "Sat System")

	t.Logf("RINEX Header: %+v\n", dec)
}

func BenchmarkNavDecoder_Ephemerides(b *testing.B) {
	b.ReportAllocs()
	filepath := "testdata/white/AREG00PER_R_20201690000_01D_MN.rnx"
	r, err := os.Open(filepath)
	if err != nil {
		b.Fatalf("%v", err)
	}
	defer r.Close()

	dec, err := NewNavDecoder(r)
	if err != nil {
		b.Fatalf("%v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for dec.NextEphemeris() {
			eph := dec.Ephemeris()
			fmt.Printf("%v\n", eph)
		}
		if err := dec.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}
	}
}

// Loop over the ephemerides of a input stream.
func ExampleNavDecoder_loop() {
	filepath := "testdata/white/AREG00PER_R_20201690000_01D_MN.rnx"
	r, err := os.Open(filepath)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer r.Close()

	dec, err := NewNavDecoder(r)
	if err != nil {
		log.Fatalf("%v", err)
	}

	nEph := 0
	for dec.NextEphemeris() {
		nEph++
		eph := dec.Ephemeris()

		// Do something with eph
		_ = eph.Validate()
	}
	if err := dec.Err(); err != nil {
		log.Printf("reading epehmeris: %v", err)
	}

	fmt.Println(nEph)
	// Output: 3612
}

func TestNavDecoder_EphemeridesFromFile(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/AREG00PER_R_20201690000_01D_MN.rnx"
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewNavDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)

	nGPS, nGLO, nGAL, nQZSS, nBDS, nIRNSS, nSBAS := 0, 0, 0, 0, 0, 0, 0
	for dec.NextEphemeris() {
		eph := dec.Ephemeris()

		switch typ := eph.(type) {
		case *EphGPS:
			nGPS++
			fmt.Printf("GPS Eph: %v\n", eph)
		case *EphGLO:
			nGLO++
			fmt.Printf("GLONASS Eph: %v\n", eph)
		case *EphGAL:
			nGAL++
			fmt.Printf("Galileo Eph: %v\n", eph)
		case *EphQZSS:
			nQZSS++
			fmt.Printf("QZSS Eph: %v\n", eph)
		case *EphBDS:
			nBDS++
			fmt.Printf("BDS Eph: %v\n", eph)
		case *EphNavIC:
			nIRNSS++
			fmt.Printf("NavIC Eph: %v\n", eph)
		case *EphSBAS:
			nSBAS++
			fmt.Printf("SBAS payload: %v\n", eph)
		default:
			t.Fatalf("unknown type %T\n", typ)
		}
	}
	if err := dec.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	assert.Equal(193, nGPS, "number of GPS epemerides")
	assert.Equal(450, nGLO, "number of GLO epemerides")
	assert.Equal(1728, nGAL, "number of GAL epemerides")
	assert.Equal(0, nQZSS, "number of QZSS epemerides")
	assert.Equal(226, nBDS, "number of BDS epemerides")
	assert.Equal(0, nIRNSS, "number of NavIC epemerides")
	assert.Equal(1015, nSBAS, "number of SBAS epemerides")
}

func TestNavDecoder_EphemeridesFromStream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	assert := assert.New(t)

	conn, err := net.Dial("tcp", "localhost:7901")
	if err != nil {
		t.Fatal("could not open conn to local port")
	}

	dec, err := NewNavDecoder(conn)
	assert.EqualError(err, ErrNoHeader.Error())
	assert.NotNil(dec)

	nEphs := 0
	for dec.NextEphemeris() {
		nEphs++
		if nEphs == 4 {
			fmt.Println("close conn")
			conn.Close()
			break
		}
	}
	if err := dec.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	assert.GreaterOrEqual(nEphs, 4, "number of epemerides")
}

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

func TestNavDecoder_decodeEphv2(t *testing.T) {
	assert := assert.New(t)

	navdata := `     2.11           N: GPS NAV DATA                         RINEX VERSION / TYPE
teqc  2019Feb25     BKG Frankfurt       20221124 21:10:15UTCPGM / RUN BY / DATE
Linux 2.6.32-573.12.1.x86_64|x86_64|gcc -static|Linux 64|=+ COMMENT
teqc  2019Feb25                         20221124 21:03:02UTCCOMMENT
    1.9558D-08 -1.4901D-08 -1.1921D-07  1.7881D-07          ION ALPHA
    1.2902D+05 -1.4746D+05  0.0000D+00 -6.5536D+04          ION BETA
   -2.793967723846D-09-4.440892098501D-15   589824     2237 DELTA-UTC: A0,A1,T,W
Concatenated RINEX files (6)                                COMMENT
                                                            END OF HEADER
26 22 11 24 20  0  0.0 2.367375418544D-04 1.023181539495D-12 0.000000000000D+00
    8.200000000000D+01 1.376562500000D+02 4.792342477443D-09 4.269679655010D-01
    7.273629307747D-06 7.515437668189D-03 1.032091677189D-05 5.153605495453D+03
    4.176000000000D+05 5.215406417847D-08-2.808004081807D+00 7.823109626770D-08
    9.360958109822D-01 1.641250000000D+02 3.988379102056D-01-8.139981920063D-09
   -2.592965150264D-10 1.000000000000D+00 2.237000000000D+03 0.000000000000D+00
    2.000000000000D+00 0.000000000000D+00 6.984919309616D-09 8.200000000000D+01
    4.146780000000D+05 4.000000000000D+00
 2 22 11 24 20  0  0.0-6.363084539771D-04 1.818989403546D-12 0.000000000000D+00
    8.700000000000D+01-1.420312500000D+02 4.175531070486D-09 1.741044635417D+00
   -7.731840014458D-06 2.000890567433D-02 2.821907401085D-06 5.153701881409D+03
    4.176000000000D+05 3.669410943985D-07-7.308842678977D-01-2.477318048477D-07
    9.668741546923D-01 3.261875000000D+02-1.360933979883D+00-7.969260523117D-09
   -3.232277494475D-10 1.000000000000D+00 2.237000000000D+03 0.000000000000D+00
    2.000000000000D+00 0.000000000000D+00-1.769512891769D-08 8.700000000000D+01
    4.129260000000D+05 4.000000000000D+00
11 22 11 24 20  0  0.0-4.512257874012D-05-6.480149750132D-12 0.000000000000D+00
    3.700000000000D+01-1.539062500000D+02 4.230533361553D-09-1.937381617115D+00
   -7.865950465202D-06 8.068794850260D-04 4.619359970093D-06 5.153675182343D+03
    4.176000000000D+05-4.097819328308D-08-5.986412719033D-01-3.539025783539D-08
    9.644632715494D-01 2.925625000000D+02-2.863460476908D+00-8.068907530958D-09
   -2.307238962907D-10 1.000000000000D+00 2.237000000000D+03 0.000000000000D+00
    2.000000000000D+00 0.000000000000D+00-8.847564458847D-09 8.050000000000D+02
    4.104180000000D+05 4.000000000000D+00
		`

	wantGPS := &EphGPS{PRN: PRN{gnss.SysGPS, 26}, TOC: time.Date(2022, 11, 24, 20, 0, 0, 0, time.UTC), ClockBias: 2.367375418544e-04, ClockDrift: 1.023181539495e-12, ClockDriftRate: 0,
		IODE: 82, Crs: 1.376562500000e+02, DeltaN: 4.792342477443e-09, M0: 4.269679655010e-01,
		Cuc: 7.273629307747e-06, Ecc: 7.515437668189e-03, Cus: 1.032091677189e-05, SqrtA: 5.153605495453e+03,
		Toe: 4.176000000000e+05, Cic: 5.215406417847e-08, Omega0: -2.808004081807e+00, Cis: 7.823109626770e-08,
		I0: 9.360958109822e-01, Crc: 1.641250000000e+02, Omega: 3.988379102056e-01, OmegaDot: -8.139981920063e-09,
		IDOT: -2.592965150264e-10, L2Codes: 1.0, ToeWeek: 2237, L2PFlag: 0,
		URA: 2.0, Health: 0, TGD: 6.984919309616e-09, IODC: 82,
		Tom: 4.146780000000e+05, FitInterval: 4}

	dec, err := NewNavDecoder(strings.NewReader(navdata))
	assert.NoError(err)

	//dec.fastMode = true

	nEphs := 0
	for dec.NextEphemeris() {
		if nEphs == 0 {
			eph := dec.Ephemeris()
			assert.Equal(wantGPS, eph, "GPS eph content")
		}
		nEphs++
	}
	if err := dec.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	assert.GreaterOrEqual(nEphs, 3, "number of epemerides")
}

func TestNavDecoder_decodeEphv4(t *testing.T) {
	assert := assert.New(t)

	navdata := `     4.00           NAVIGATION DATA     MIXED               RINEX VERSION / TYPE
JPS2RIN v.2.1.216   JAVAD GNSS          20221129 001843 UTC PGM / RUN BY / DATE 
gfzrnx-3499         FILE MERGE          20221130 075926 UTC PGM / RUN BY / DATE 
    18                                                      LEAP SECONDS
                                                            END OF HEADER
> EPH C01 D1  
C01 2022 11 29 12 00 00 9.344602003694e-04-3.997691067070e-12 0.000000000000e+00
     1.000000000000e+00-3.418906250000e+02-3.509431896211e-09-3.073247862435e+00
    -1.103850081563e-05 6.337405648082e-04-9.746756404638e-06 6.493360338211e+03
     2.160000000000e+05 2.626329660416e-07 8.175155760052e-02 1.629814505577e-08
     1.088206710577e-01 2.943281250000e+02 2.415359474789e+00 4.576262048254e-09
     4.846630453041e-10 0.000000000000e+00 8.820000000000e+02 0.000000000000e+00
     2.000000000000e+00 0.000000000000e+00-4.700000000000e-09-1.000000000000e-08
     2.182470000000e+05 0.000000000000e+00 0.000000000000e+00 0.000000000000e+00
> EPH G22 LNAV
G22 2022 11 29 04 00 00 3.741933032870e-04 7.730704965070e-12 0.000000000000e+00
     6.000000000000e+01-6.025000000000e+01 4.208032424298e-09 2.742321292461e+00
    -3.069639205933e-06 1.356723881327e-02 9.909272193909e-06 5.153760629654e+03
     1.872000000000e+05 1.378357410431e-07 1.402224437442e+00-8.940696716309e-08
     9.616292729991e-01 1.858437500000e+02-1.843594565182e+00-7.519598935764e-09
     2.732256666600e-10 1.000000000000e+00 2.238000000000e+03 0.000000000000e+00
     2.000000000000e+00 0.000000000000e+00-8.381903000000e-09 6.000000000000e+01
     1.858800000000e+05 4.000000000000e+00 0.000000000000e+00 0.000000000000e+00
> EPH E01 FNAV
E01 2022 11 29 05 10 00-5.723080830649e-04-7.347011887759e-12 0.000000000000e+00
     6.300000000000e+01 2.993750000000e+01 3.187275619966e-09-8.481673963194e-01
     1.402571797371e-06 2.489964244887e-04 7.089227437973e-06 5.440602836609e+03
     1.914000000000e+05 1.117587089539e-08-2.940363735278e+00 3.166496753693e-08
     9.719678613571e-01 1.898750000000e+02-2.943531074341e-01-5.568446233851e-09
    -3.614436270064e-10 2.580000000000e+02 2.238000000000e+03 0.000000000000e+00
     3.119999885559e+00 8.000000000000e+00 4.656612873077e-10 0.000000000000e+00
     1.961500000000e+05 0.000000000000e+00 0.000000000000e+00 0.000000000000e+00
> EPH E01 INAV
E01 2022 11 29 05 10 00-5.723083741032e-04-7.332801033044e-12 0.000000000000e+00
     6.300000000000e+01 2.993750000000e+01 3.187275619966e-09-8.481673963194e-01
     1.402571797371e-06 2.489964244887e-04 7.089227437973e-06 5.440602836609e+03
     1.914000000000e+05 1.117587089539e-08-2.940363735278e+00 3.166496753693e-08
     9.719678613571e-01 1.898750000000e+02-2.943531074341e-01-5.568446233851e-09
    -3.614436270064e-10 5.160000000000e+02 2.238000000000e+03 0.000000000000e+00
     3.119999885559e+00 6.500000000000e+01 4.656612873077e-10 4.656612873077e-10
     1.961440000000e+05 0.000000000000e+00 0.000000000000e+00 0.000000000000e+00
> EPH R22 FDMA
R22 2022 11 29 10 45 00 1.968629658222e-05 0.000000000000e+00 2.106300000000e+05
    -1.174041748047e+04-7.016086578369e-01 0.000000000000e+00 1.000000000000e+00
     2.063836816406e+04 1.077777862549e+00 3.725290298462e-09-3.000000000000e+00
    -9.277129882813e+03 3.275458335876e+00-9.313225746155e-10 0.000000000000e+00
                         .999999999999e+09 1.500000000000e+01 
`
	dec, err := NewNavDecoder(strings.NewReader(navdata))
	assert.NoError(err)

	//dec.fastMode = true

	nEphs := 0
	for dec.NextEphemeris() {
		if nEphs == 0 {
			eph := dec.Ephemeris()
			t.Logf("%+v", eph)
		}
		nEphs++
	}
	if err := dec.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	assert.GreaterOrEqual(nEphs, 5, "number of epemerides")
}
func TestNavDecoder_decodeEph(t *testing.T) {
	assert := assert.New(t)

	navdata := `     3.04           N: GNSS NAV DATA    M: MIXED            RINEX VERSION / TYPE
sbf2rin-13.4.3                          20200618 001127 UTC PGM / RUN BY / DATE 
GPSA   5.5879E-09  1.4901E-08 -5.9605E-08 -1.1921E-07       IONOSPHERIC CORR    
GPSB   8.3968E+04  9.8304E+04 -6.5536E+04 -5.2429E+05       IONOSPHERIC CORR    
GAL    2.3750E+01  1.5625E-02  1.2329E-02  0.0000E+00       IONOSPHERIC CORR    
GPUT  3.4924596548E-10-1.154631946E-14 503808 2110          TIME SYSTEM CORR    
GAUT -9.3132257462E-10 0.000000000E+00 259200 2110          TIME SYSTEM CORR    
GAGP  9.0221874416E-10 4.884981308E-15 345600 2110          TIME SYSTEM CORR    
    18                                                      LEAP SECONDS        
                                                            END OF HEADER 
G20 2020 06 18 00 00 00 5.274894647300E-04-1.136868377216E-13 0.000000000000E+00
     8.300000000000E+01 2.078125000000E+01 5.373438110980E-09-2.252452975616E+00
     1.156702637672E-06 5.203154985793E-03 7.405877113342E-06 5.153647661209E+03
     3.456000000000E+05-1.247972249985E-07-2.679776962713E+00 2.048909664154E-08
     9.344138223835E-01 2.252500000000E+02 2.669542608731E+00-8.333918569731E-09
     4.632335812523E-10 1.000000000000E+00 2.110000000000E+03 0.000000000000E+00
     2.000000000000E+00 0.000000000000E+00-8.847564458847E-09 8.300000000000E+01
     3.393480000000E+05 4.000000000000E+00
E26 2020 06 17 04 20 00 3.064073505811E-03-4.352784799266E-11 0.000000000000E+00
     7.400000000000E+01-1.238437500000E+02 2.376527563341E-09 3.130998000440E+00
    -5.731359124184E-06 2.621184103191E-05 1.052953302860E-05 5.440627540588E+03
     2.748000000000E+05 1.303851604462E-08 2.421956189340E+00-2.607703208923E-08
     9.848811109258E-01 1.224062500000E+02 1.660149314991E+00-5.262004897911E-09
     8.571785620706E-11 5.170000000000E+02 2.110000000000E+03
     3.120000000000E+00 0.000000000000E+00 3.958120942116E-09 4.423782229424E-09
     2.754650000000E+05
	`
	wantGPS := &EphGPS{PRN: PRN{gnss.SysGPS, 20}, TOC: time.Date(2020, 6, 18, 0, 0, 0, 0, time.UTC), ClockBias: 5.274894647300e-04, ClockDrift: -1.136868377216e-13, ClockDriftRate: 0,
		IODE: 83, Crs: 2.078125000000e+01, DeltaN: 5.373438110980e-09, M0: -2.252452975616,
		Cuc: 1.156702637672e-06, Ecc: 5.203154985793e-03, Cus: 7.405877113342e-06, SqrtA: 5.153647661209e+03,
		Toe: 3.456000000000e+05, Cic: -1.247972249985e-07, Omega0: -2.679776962713, Cis: 2.048909664154e-08,
		I0: 9.344138223835e-01, Crc: 2.252500000000e+02, Omega: 2.669542608731, OmegaDot: -8.333918569731e-09,
		IDOT: 4.632335812523e-10, L2Codes: 1.0, ToeWeek: 2110, L2PFlag: 0,
		URA: 2.0, Health: 0, TGD: -8.847564458847e-09, IODC: 83,
		Tom: 3.393480000000e+05, FitInterval: 4}

	wantGAL := &EphGAL{PRN: PRN{gnss.SysGAL, 26}, TOC: time.Date(2020, 6, 17, 4, 20, 0, 0, time.UTC)}

	dec, err := NewNavDecoder(strings.NewReader(navdata))
	assert.NoError(err)

	// GPS
	ok := dec.NextEphemeris()
	assert.True(ok)
	eph := dec.Ephemeris()
	gpsEhp, ok := eph.(*EphGPS)
	assert.True(ok)
	assert.Equal(wantGPS, gpsEhp, "GPS eph content")

	// Galileo
	ok = dec.NextEphemeris()
	assert.True(ok)
	eph = dec.Ephemeris()
	galEhp, ok := eph.(*EphGAL)
	assert.True(ok)
	assert.Equal(wantGAL, galEhp, "Galileo eph content")

	assert.NoError(dec.Err())
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

func Test_decodeEphLineHlp(t *testing.T) {
	// Helper test. Go slices are inclusive-exclusive.
	// Rnx2 GPS
	line := "26 22 11 24 20  0  0.0 2.367375418544D-04 1.023181539495D-12 0.000000000000D+00"
	assert.Equal(t, "26", line[:2], "prn")
	assert.Equal(t, "22 11 24 20  0  0.0", line[3:22], "epoch time")
	assert.Equal(t, " 2.367375418544D-04", line[22:22+19], "clock bias")
	assert.Equal(t, " 1.023181539495D-12", line[41:41+19], "clock drift")
	assert.Equal(t, " 0.000000000000D+00", line[60:60+19], "clock drift rate")

	// Rnx4
	line = "> EPH G22 LNAV"
	assert.Equal(t, "EPH", line[2:5], "rec type")
	assert.Equal(t, "G", line[6:7], "sys")
	assert.Equal(t, "G22", line[6:9], "prn")
	assert.Equal(t, "LNAV", strings.TrimSpace(line[10:]), "mess type")
}

func TestNavDecoder_parseToC(t *testing.T) {
	assert := assert.New(t)
	tests := map[string]time.Time{
		"2022 11 29 04 00 00": time.Date(2022, 11, 29, 4, 0, 0, 0, time.UTC),
	}

	for k, v := range tests {
		epTime, err := time.Parse(TimeOfClockFormat, k)
		assert.NoError(err)
		assert.Equal(v, epTime)
		fmt.Printf("RINEX-3: %s\n", epTime)
	}

	// RINEX version 2
	tests = map[string]time.Time{
		"22 11 24 20  0  0.0": time.Date(2022, 11, 24, 20, 0, 0, 0, time.UTC),
		"06 11 25  1 59 44.0": time.Date(2006, 11, 25, 1, 59, 44, 0, time.UTC),
	}

	for k, v := range tests {
		epTime, err := time.Parse(TimeOfClockFormatv2, k)
		assert.NoError(err)
		assert.Equal(v, epTime, "RINEX-2 epoch")
		fmt.Printf("RINEX-2: %s\n", epTime)
	}
}
