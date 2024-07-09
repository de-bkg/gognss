package rinex

import (
	"strings"
	"testing"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
	"github.com/stretchr/testify/assert"
)

func TestClockDecoder_readHeader(t *testing.T) {
	const header = `     3.00           C                                       RINEX VERSION / TYPE
CCLOCK              IGSACC @ GA MIT                         PGM / RUN BY / DATE
GPS week: 2285   Day: 2   MJD: 60241                        COMMENT
THE COMBINED CLOCKS ARE A WEIGHTED AVERAGE OF:              COMMENT
  esa gfz grg cod                                           COMMENT
THE FOLLOWING REFERENCE CLOCKS WERE USED BY ACs:            COMMENT
  HERS NNOR STJ3                                            COMMENT
THE COMBINED CLOCKS ARE ALIGNED TO GPS TIME                 COMMENT
USING THE SATELLITE BROADCAST EPHEMERIDES                   COMMENT
All clocks have been re-aligned to the IGS time scale: IGST COMMENT
    18                                                      LEAP SECONDS
     2    AR    AS                                          # / TYPES OF DATA
IGS  IGSACC @ GA MIT                                        ANALYSIS CENTER
   199    IGS20 : IGS REALIZATION of THE ITRF2020           # OF SOLN STA / TRF
ABMF 97103M001            2919785819 -5383744924  1774604918SOLN STA NAME / NUM
ADIS 31502M001            4913652485  3945922887   995383577SOLN STA NAME / NUM
ZECK 12351M001            3451174301  3060335713  4391955818SOLN STA NAME / NUM
   32                                                      # OF SOLN SATS
G01 G02 G03 G04 G05 G06 G07 G08 G09 G10 G11 G12 G13 G14 G15 PRN LIST
G16 G17 G18 G19 G20 G21 G22 G23 G24 G25 G26 G27 G28 G29 G30 PRN LIST
G31 G32                                                     PRN LIST
G                   igs20_2283.atx                          SYS / PCVS APPLIED
                                                            END OF HEADER
AR GPST 2023 10 24 00 00  0.000000  2   -3.586042358862e-09  0.000000000000e+00
AR AIRA 2023 10 24 00 00  0.000000  2   -3.167986726646e-08  3.643193552360e-12
	`

	assert := assert.New(t)
	dec, err := NewClockDecoder(strings.NewReader(header))
	assert.NoError(err)
	assert.NotNil(dec)

	hdr := dec.Header
	assert.Equal(float32(3.00), hdr.RINEXVersion)
	assert.Equal("C", hdr.RINEXType)
	assert.Equal("CCLOCK", hdr.Pgm)
	assert.Equal("IGSACC @ GA MIT", hdr.RunBy)
	assert.Equal("IGS", hdr.AC)
}

func TestClockDecoder_readHeader304(t *testing.T) {
	const header = `3.04                 C                    M                      RINEX VERSION / TYPE
CCRNXC V5.3          AIUB                 21-AUG-20 05:54        PGM / RUN BY / DATE
Center for Orbit Determination in Europe (CODE)                  COMMENT
Final GNSS clock information for year-day 2020-229               COMMENT
Clock information consistent with phase and C1W/C2W code data    COMMENT
Satellite/receiver clock values at intervals of 5/300 sec        COMMENT
High-rate (5 sec) clock interpolation based on phase data        COMMENT
Product reference: DOI 10.7892/boris.75876.4                     COMMENT
   GPS                                                           TIME SYSTEM ID
    18                                                           LEAP SECONDS GNSS
G  CLKEST V5.3        IGS14                                      SYS / PCVS APPLIED
R  CLKEST V5.3        IGS14                                      SYS / PCVS APPLIED
G  CLKEST V5.3        CODE.BIA @ ftp.aiub.unibe.ch/CODE/         SYS / DCBS APPLIED
R  CLKEST V5.3        CODE.BIA @ ftp.aiub.unibe.ch/CODE/         SYS / DCBS APPLIED
     2    AR    AS                                               # / TYPES OF DATA
COD  Center for Orbit Determination in Europe                    ANALYSIS CENTER
     1                                                           # OF CLK REF
TIDB00AUS 50103M108                           0.000000000000E+00 ANALYSIS CLK REF
   306    IGb14                                                  # OF SOLN STA / TRF
TIDB00AUS 50103M108           -4460996987  2682557086 -3674442611SOLN STA NAME / NUM
ZIMM00CHE 14001M004            4331296853   567556159  4633134121SOLN STA NAME / NUM
    53                                                           # OF SOLN SATS
G01 G02 G03 G04 G05 G06 G07 G08 G09 G10 G11 G12 G13 G15 G16 G17  PRN LIST
G18 G19 G20 G21 G22 G24 G25 G26 G27 G28 G29 G30 G31 G32 R01 R02  PRN LIST
R03 R04 R05 R07 R08 R09 R11 R12 R13 R14 R15 R16 R17 R18 R19 R20  PRN LIST
R21 R22 R23 R24 R26                                              PRN LIST
                                                                 END OF HEADER
	`

	prnListWanted := []gnss.PRN{{Sys: gnss.SysGPS, Num: 1}, {Sys: gnss.SysGPS, Num: 2}, {Sys: gnss.SysGPS, Num: 3},
		{Sys: gnss.SysGPS, Num: 4}, {Sys: gnss.SysGPS, Num: 5}, {Sys: gnss.SysGPS, Num: 6}, {Sys: gnss.SysGPS, Num: 7},
		{Sys: gnss.SysGPS, Num: 8}, {Sys: gnss.SysGPS, Num: 9}, {Sys: gnss.SysGPS, Num: 10}, {Sys: gnss.SysGPS, Num: 11},
		{Sys: gnss.SysGPS, Num: 12}, {Sys: gnss.SysGPS, Num: 13}, {Sys: gnss.SysGPS, Num: 15}, {Sys: gnss.SysGPS, Num: 16},
		{Sys: gnss.SysGPS, Num: 17}, {Sys: gnss.SysGPS, Num: 18}, {Sys: gnss.SysGPS, Num: 19}, {Sys: gnss.SysGPS, Num: 20},
		{Sys: gnss.SysGPS, Num: 21}, {Sys: gnss.SysGPS, Num: 22}, {Sys: gnss.SysGPS, Num: 24}, {Sys: gnss.SysGPS, Num: 25},
		{Sys: gnss.SysGPS, Num: 26}, {Sys: gnss.SysGPS, Num: 27}, {Sys: gnss.SysGPS, Num: 28}, {Sys: gnss.SysGPS, Num: 29},
		{Sys: gnss.SysGPS, Num: 30}, {Sys: gnss.SysGPS, Num: 31}, {Sys: gnss.SysGPS, Num: 32}, {Sys: gnss.SysGLO, Num: 1},
		{Sys: gnss.SysGLO, Num: 2}, {Sys: gnss.SysGLO, Num: 3}, {Sys: gnss.SysGLO, Num: 4}, {Sys: gnss.SysGLO, Num: 5},
		{Sys: gnss.SysGLO, Num: 7}, {Sys: gnss.SysGLO, Num: 8}, {Sys: gnss.SysGLO, Num: 9}, {Sys: gnss.SysGLO, Num: 11},
		{Sys: gnss.SysGLO, Num: 12}, {Sys: gnss.SysGLO, Num: 13}, {Sys: gnss.SysGLO, Num: 14}, {Sys: gnss.SysGLO, Num: 15},
		{Sys: gnss.SysGLO, Num: 16}, {Sys: gnss.SysGLO, Num: 17}, {Sys: gnss.SysGLO, Num: 18}, {Sys: gnss.SysGLO, Num: 19},
		{Sys: gnss.SysGLO, Num: 20}, {Sys: gnss.SysGLO, Num: 21}, {Sys: gnss.SysGLO, Num: 22}, {Sys: gnss.SysGLO, Num: 23},
		{Sys: gnss.SysGLO, Num: 24}, {Sys: gnss.SysGLO, Num: 26}}

	assert := assert.New(t)
	dec, err := NewClockDecoder(strings.NewReader(header))
	assert.NoError(err)
	assert.NotNil(dec)

	hdr := dec.Header
	assert.Equal(float32(3.04), hdr.RINEXVersion)
	assert.Equal("C", hdr.RINEXType)
	assert.Equal(hdr.SatSystem, gnss.SysMIXED)
	assert.Equal("CCRNXC V5.3", hdr.Pgm)
	assert.Equal("AIUB", hdr.RunBy)
	assert.Equal("COD", hdr.AC)
	assert.Equal("GPS", hdr.TimeSystemID)
	assert.Equal(53, hdr.NumSolnSats)
	assert.Equal(time.Date(2020, 8, 21, 5, 54, 0, 0, time.UTC), hdr.Date)
	assert.Equal(prnListWanted, hdr.Sats, "PRN List")

	//t.Logf("RINEX Header: %+v\n", hdr)
}

/* func Test_splitAt(t *testing.T) {
	type args struct {
		s   string
		pos []int
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{name: "t1-304", args: args{s: "3.04                 C                    M                      ", pos: []int{0, 4, 21, 22, 42, 43}},
			want: []string{"3.04", "C", "M"}},
		{name: "t1-304-2", args: args{s: "3.04                 C                    M                      ", pos: []int{0, 4, 21, 22, 42, 99}},
			want: []string{"3.04", "C", "M"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitAt(tt.args.s, tt.args.pos...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitAt() = %v, want %v", got, tt.want)
			}
		})
	}
} */
