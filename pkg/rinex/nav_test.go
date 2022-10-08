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
	//defer dec.Close() // obsolet? s.o.

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

func TestNavDecoder_EphemeridesFromBuf(t *testing.T) {
	assert := assert.New(t)

	const ehems = `
G20 2020 06 18 00 00 00 5.274894647300E-04-1.136868377216E-13 0.000000000000E+00
     8.300000000000E+01 2.078125000000E+01 5.373438110980E-09-2.252452975616E+00
     1.156702637672E-06 5.203154985793E-03 7.405877113342E-06 5.153647661209E+03
     3.456000000000E+05-1.247972249985E-07-2.679776962713E+00 2.048909664154E-08
     9.344138223835E-01 2.252500000000E+02 2.669542608731E+00-8.333918569731E-09
     4.632335812523E-10 1.000000000000E+00 2.110000000000E+03 0.000000000000E+00
     2.000000000000E+00 0.000000000000E+00-8.847564458847E-09 8.300000000000E+01
     3.393480000000E+05 4.000000000000E+00
R21 2020 06 17 09 45 00-1.319693401456E-04-2.728484105319E-12 2.937000000000E+05
    -1.042075537109E+04 2.813003540039E+00-2.793967723846E-09 0.000000000000E+00
    -6.330877929688E+03-1.233654975891E+00 0.000000000000E+00 4.000000000000E+00
    -2.240664208984E+04-9.621353149414E-01 9.313225746155E-10 0.000000000000E+00
E26 2020 06 17 04 20 00 3.064073505811E-03-4.352784799266E-11 0.000000000000E+00
     7.400000000000E+01-1.238437500000E+02 2.376527563341E-09 3.130998000440E+00
    -5.731359124184E-06 2.621184103191E-05 1.052953302860E-05 5.440627540588E+03
     2.748000000000E+05 1.303851604462E-08 2.421956189340E+00-2.607703208923E-08
     9.848811109258E-01 1.224062500000E+02 1.660149314991E+00-5.262004897911E-09
     8.571785620706E-11 5.170000000000E+02 2.110000000000E+03
     3.120000000000E+00 0.000000000000E+00 3.958120942116E-09 4.423782229424E-09
     2.754650000000E+05
	`

	dec, err := NewNavDecoder(strings.NewReader(ehems))
	assert.EqualError(err, ErrNoHeader.Error())
	assert.NotNil(dec)

	nEphs := 0
	for dec.NextEphemeris() {
		eph := dec.Ephemeris()

		nEphs++
		switch typ := eph.(type) {
		case *EphGPS:
			fmt.Printf("GPS Eph: %v\n", eph)
		case *EphGLO:
			fmt.Printf("GLO Eph: %v\n", eph)
		case *EphGAL:
			fmt.Printf("Gal Eph: %v\n", eph)
		case *EphQZSS:
			fmt.Printf("QZSS Eph: %v\n", eph)
		case *EphBDS:
			fmt.Printf("BDS Eph: %v\n", eph)
		case *EphNavIC:
			fmt.Printf("NavIC Eph: %v\n", eph)
		case *EphSBAS:
			fmt.Printf("SBAS payload: %v\n", eph)
		default:
			t.Fatalf("unknown type %T\n", typ)
		}
	}
	if err := dec.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading ephemerides:", err)
	}

	assert.Equal(3, nEphs, "number of epemerides")

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

func TestNavFile_Validate(t *testing.T) {
	assert := assert.New(t)
	navFilepath := "testdata/white/AREG00PER_R_20201690000_01D_MN.rnx"
	nf, err := NewNavFile(navFilepath)
	assert.NoError(err)

	err = nf.Validate()
	assert.NoError(err)
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

func TestUnmarshalEph(t *testing.T) {
	assert := assert.New(t)

	gps := []byte(`G20 2020 06 18 00 00 00 5.274894647300E-04-1.136868377216E-13 0.000000000000E+00
     8.300000000000E+01 2.078125000000E+01 5.373438110980E-09-2.252452975616E+00
     1.156702637672E-06 5.203154985793E-03 7.405877113342E-06 5.153647661209E+03
     3.456000000000E+05-1.247972249985E-07-2.679776962713E+00 2.048909664154E-08
     9.344138223835E-01 2.252500000000E+02 2.669542608731E+00-8.333918569731E-09
     4.632335812523E-10 1.000000000000E+00 2.110000000000E+03 0.000000000000E+00
     2.000000000000E+00 0.000000000000E+00-8.847564458847E-09 8.300000000000E+01
     3.393480000000E+05 4.000000000000E+00
	`)

	gal := []byte(`E26 2020 06 17 04 20 00 3.064073505811E-03-4.352784799266E-11 0.000000000000E+00
	7.400000000000E+01-1.238437500000E+02 2.376527563341E-09 3.130998000440E+00
   -5.731359124184E-06 2.621184103191E-05 1.052953302860E-05 5.440627540588E+03
	2.748000000000E+05 1.303851604462E-08 2.421956189340E+00-2.607703208923E-08
	9.848811109258E-01 1.224062500000E+02 1.660149314991E+00-5.262004897911E-09
	8.571785620706E-11 5.170000000000E+02 2.110000000000E+03
	3.120000000000E+00 0.000000000000E+00 3.958120942116E-09 4.423782229424E-09
	2.754650000000E+05
	`)

	type args struct {
		eph  Eph
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    Eph
		wantErr bool
	}{
		{
			name: "GPS eph",
			args: args{NewEph(gnss.SysGPS), gps},
			want: &EphGPS{PRN: PRN{gnss.SysGPS, 20}, TOC: time.Date(2020, 6, 18, 0, 0, 0, 0, time.UTC), ClockBias: 5.274894647300e-04, ClockDrift: -1.136868377216e-13, ClockDriftRate: 0,
				IODE: 83, Crs: 2.078125000000e+01, DeltaN: 5.373438110980e-09, M0: -2.252452975616,
				Cuc: 1.156702637672e-06, Ecc: 5.203154985793e-03, Cus: 7.405877113342e-06, SqrtA: 5.153647661209e+03,
				Toe: 3.456000000000e+05, Cic: -1.247972249985e-07, Omega0: -2.679776962713, Cis: 2.048909664154e-08,
				I0: 9.344138223835e-01, Crc: 2.252500000000e+02, Omega: 2.669542608731, OmegaDot: -8.333918569731e-09,
				IDOT: 4.632335812523e-10, L2Codes: 1.0, ToeWeek: 2110, L2PFlag: 0,
				URA: 2.0, Health: 0, TGD: -8.847564458847e-09, IODC: 83,
				Tom: 3.393480000000e+05, FitInterval: 4},
			wantErr: false,
		},
		{
			name:    "GAL eph",
			args:    args{NewEph(gnss.SysGAL), gal},
			want:    &EphGAL{PRN: PRN{gnss.SysGAL, 26}, TOC: time.Date(2020, 6, 17, 4, 20, 0, 0, time.UTC)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UnmarshalEph(tt.args.data, tt.args.eph); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalEph() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(tt.want, tt.args.eph, "content check")

			/* 			switch typ := tt.want.(type) {
			   			case *EphGPS:
			   				assert.Equal(tt.want, tt.args.eph, "GPS check")
			   			case *EphGLO:
			   				//fmt.Printf("GLO Eph: %v\n", eph)
			   			case *EphGAL:
			   				//fmt.Printf("Gal Eph: %v\n", eph)
			   			case *EphQZSS:
			   				//fmt.Printf("QZSS Eph: %v\n", eph)
			   			case *EphBDS:
			   				//fmt.Printf("BDS Eph: %v\n", eph)
			   			case *EphIRNSS:
			   				//fmt.Printf("NavIC Eph: %v\n", eph)
			   			case *EphSBAS:
			   				//fmt.Printf("SBAS payload: %v\n", eph)
			   			default:
			   				t.Fatalf("unknown type %T\n", typ)
			   			} */

		})
	}
}
