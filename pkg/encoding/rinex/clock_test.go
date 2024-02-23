package rinex

import (
	"strings"
	"testing"

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
	//assert.Equal(time.Date(2019, 11, 3, 0, 7, 0, 0, time.UTC), hdr.Date)
	//t.Logf("RINEX Header: %+v\n", hdr)
}
