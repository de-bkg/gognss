package rinex

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// TimeOfClockFormat is the time format within RINEX3 Nav records.
	TimeOfClockFormat string = "2006  1  2 15  4  5"
)

// Eph is the interface that wraps some methods for all types of ephemeris.
type Eph interface {
	// Validate checks the ephemeris.
	Validate() error

	unmarshal(data []byte) error
}

// NewEph returns a new ephemeris having the concrete type.
func NewEph(sys SatelliteSystem) Eph {
	var eph Eph
	switch sys {
	case SatSysGPS:
		eph = &EphGPS{}
	case SatSysGLO:
		eph = &EphGLO{}
	case SatSysGAL:
		eph = &EphGAL{}
	case SatSysQZSS:
		eph = &EphQZSS{}
	case SatSysBDS:
		eph = &EphBDS{}
	case SatSysIRNSS:
		eph = &EphIRNSS{}
	case SatSysSBAS:
		eph = &EphSBAS{}
	default:
		log.Fatalf("unknown satellite system: %v", sys)
	}

	return eph
}

// type Unmarshaler interface {
// 	Unmarshal([]string) error
// }

// UnmarshalEph parses the RINEX ephemeris given in lines and stores the result in the value pointed to by eph.
func UnmarshalEph(data []byte, eph Eph) error {
	return eph.unmarshal(data)
}

// EphGPS describes a GPS ephemeris.
type EphGPS struct {
	PRN PRN

	// Clock
	TOC            time.Time // Time of Clock, clock reference epoch
	ClockBias      float64   // sc clock bias in seconds
	ClockDrift     float64   // sec/sec
	ClockDriftRate float64   // sec/sec2

	IODE   float64 // Issue of Data, Ephemeris
	Crs    float64 // meters
	DeltaN float64 // radians/sec
	M0     float64 // radians

	Cuc   float64 // radians
	Ecc   float64 // Eccentricity
	Cus   float64 // radians
	SqrtA float64 // sqrt(m)

	Toe    float64 // time of ephemeris (sec of GPS week)
	Cic    float64 // radians
	Omega0 float64 // radians
	Cis    float64 // radians

	I0       float64 // radians
	Crc      float64 // meters
	Omega    float64 // radians
	OmegaDot float64 // radians/sec

	IDOT    float64 // radians/sec
	L2Codes float64
	ToeWeek float64 // GPS week (to go with TOE) Continuous
	L2PFlag float64

	URA    float64 // SV accuracy in meters
	Health float64 // SV health (bits 17-22 w 3 sf 1)
	TGD    float64 // seconds
	IODC   float64 // Issue of Data, clock

	Tom         float64 // transmission time of message, seconds of GPS week
	FitInterval float64 // Fit interval in hours
}

// EphGLO describes a GLONASS ephemeris.
type EphGLO struct {
	PRN PRN
	TOC time.Time
}

// EphGAL describes a Galileo ephemeris.
type EphGAL struct {
	PRN PRN
	TOC time.Time
}

// EphQZSS describes a QZSS ephemeris.
type EphQZSS struct {
	PRN PRN
	TOC time.Time
}

// EphBDS describes a chinese BDS ephemeris.
type EphBDS struct {
	PRN PRN
	TOC time.Time
}

// EphIRNSS describes an indian IRNSS/NavIC ephemeris.
type EphIRNSS struct {
	PRN PRN
	TOC time.Time
}

// EphSBAS describes a SBAS payload.
type EphSBAS struct {
	PRN PRN
	TOC time.Time
}

func (EphGPS) Validate() error { return nil }
func (eph *EphGPS) unmarshal(data []byte) (err error) {

	/*
			G12 2020 06 17 02 00 00 1.051961444318E-04-4.433786671143E-12 0.000000000000E+00
			     6.100000000000E+01 5.971875000000E+01 4.119457306218E-09-2.150395402634E+00
		         3.147870302200E-06 8.033315883949E-03 3.485009074211E-06 5.153677604675E+03
			     2.664000000000E+05 1.061707735062E-07 6.666502414356E-01-5.774199962616E-08
			     9.781878686511E-01 3.217500000000E+02 1.162895587886E+00-7.943902323989E-09
			     1.325055193867E-10 1.000000000000E+00 2.110000000000E+03 0.000000000000E+00
			     2.000000000000E+00 0.000000000000E+00-1.210719347000E-08 6.100000000000E+01
				 2.592180000000E+05 4.000000000000E+00
	*/

	r := bufio.NewReader(bytes.NewReader(data))
	line, err := r.ReadString('\n')
	if err != nil {
		return
	}

	snum, err := strconv.Atoi(line[1:3])
	if err != nil {
		return fmt.Errorf("Could not parse sat num: %q: %v", line, err)
	}
	eph.PRN, err = newPRN(SatSysGPS, int8(snum))
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("Could not parse TOC: '%s': %v", line, err)
	}

	eph.ClockBias, err = parseFloat(line[23 : 23+19])
	if err != nil {
		return
	}

	eph.ClockDrift, err = parseFloat(line[42 : 42+19])
	if err != nil {
		return
	}

	eph.ClockDriftRate, err = parseFloat(line[61 : 61+19])
	if err != nil {
		return
	}

	line, err = r.ReadString('\n')
	if err != nil {
		return
	}
	eph.IODE, eph.Crs, eph.DeltaN, eph.M0, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	line, err = r.ReadString('\n')
	if err != nil {
		return
	}
	eph.Cuc, eph.Ecc, eph.Cus, eph.SqrtA, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	line, err = r.ReadString('\n')
	if err != nil {
		return
	}
	eph.Toe, eph.Cic, eph.Omega0, eph.Cis, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	line, err = r.ReadString('\n')
	if err != nil {
		return
	}
	eph.I0, eph.Crc, eph.Omega, eph.OmegaDot, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	line, err = r.ReadString('\n')
	if err != nil {
		return
	}
	eph.IDOT, eph.L2Codes, eph.ToeWeek, eph.L2PFlag, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	line, err = r.ReadString('\n')
	if err != nil {
		return
	}
	eph.URA, eph.Health, eph.TGD, eph.IODC, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	line, err = r.ReadString('\n')
	if err != nil {
		return
	}
	eph.Tom, eph.FitInterval, _, _, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	return nil
}

func (EphGLO) Validate() error { return nil }
func (eph *EphGLO) unmarshal(data []byte) error {

	r := bufio.NewReader(bytes.NewReader(data))
	line, err := r.ReadString('\n')

	snum, err := strconv.Atoi(line[1:3])
	if err != nil {
		return fmt.Errorf("Could not parse sat num: %q: %v", line, err)
	}
	eph.PRN, err = newPRN(SatSysGLO, int8(snum))
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("Could not parse TOC: '%s': %v", line, err)
	}

	return nil
}

func (EphGAL) Validate() error { return nil }
func (eph *EphGAL) unmarshal(data []byte) error {
	r := bufio.NewReader(bytes.NewReader(data))
	line, err := r.ReadString('\n')

	snum, err := strconv.Atoi(line[1:3])
	if err != nil {
		return fmt.Errorf("Could not parse sat num: %q: %v", line, err)
	}
	eph.PRN, err = newPRN(SatSysGAL, int8(snum))
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("Could not parse TOC: '%s': %v", line, err)
	}

	return nil
}

func (EphQZSS) Validate() error { return nil }
func (eph *EphQZSS) unmarshal(data []byte) error {
	r := bufio.NewReader(bytes.NewReader(data))
	line, err := r.ReadString('\n')

	snum, err := strconv.Atoi(line[1:3])
	if err != nil {
		return fmt.Errorf("Could not parse sat num: %q: %v", line, err)
	}
	eph.PRN, err = newPRN(SatSysQZSS, int8(snum))
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("Could not parse TOC: '%s': %v", line, err)
	}

	return nil
}

func (EphBDS) Validate() error { return nil }
func (eph *EphBDS) unmarshal(data []byte) error {
	r := bufio.NewReader(bytes.NewReader(data))
	line, err := r.ReadString('\n')

	snum, err := strconv.Atoi(line[1:3])
	if err != nil {
		return fmt.Errorf("Could not parse sat num: %q: %v", line, err)
	}
	eph.PRN, err = newPRN(SatSysBDS, int8(snum))
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("Could not parse TOC: '%s': %v", line, err)
	}

	return nil
}

func (EphIRNSS) Validate() error { return nil }
func (eph *EphIRNSS) unmarshal(data []byte) error {
	r := bufio.NewReader(bytes.NewReader(data))
	line, err := r.ReadString('\n')

	snum, err := strconv.Atoi(line[1:3])
	if err != nil {
		return fmt.Errorf("Could not parse sat num: %q: %v", line, err)
	}
	eph.PRN, err = newPRN(SatSysIRNSS, int8(snum))
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("Could not parse TOC: '%s': %v", line, err)
	}

	return nil
}

func (EphSBAS) Validate() error { return nil }
func (eph *EphSBAS) unmarshal(data []byte) error {
	r := bufio.NewReader(bytes.NewReader(data))
	line, err := r.ReadString('\n')

	snum, err := strconv.Atoi(line[1:3])
	if err != nil {
		return fmt.Errorf("Could not parse sat num: %q: %v", line, err)
	}
	eph.PRN, err = newPRN(SatSysSBAS, int8(snum))
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("Could not parse TOC: '%s': %v", line, err)
	}

	return nil
}

// A NavHeader containes the RINEX Navigation Header information.
// All header parameters are optional and may comprise different types of ionospheric model parameters
// and time conversion parameters.
type NavHeader struct {
	RINEXVersion float32         // RINEX Format version
	RINEXType    string          // RINEX File type. O for Obs
	SatSystem    SatelliteSystem // Satellite System. System is "Mixed" if more than one.

	Pgm   string // name of program creating this file
	RunBy string // name of agency creating this file
	Date  string // date and time of file creation TODO time.Time

	Comments []string // * comment lines

	labels   []string // all Header Labels found
	warnings []string
}

// A headerLabel is a RINEX Header Label.
type headerLabel struct {
	label    string
	official bool
	optional bool
}

// A NavDecoder reads and decodes header and data records from a RINEX Nav input stream.
type NavDecoder struct {
	// The Header is valid after NewNavDecoder or Reader.Reset. The header must not necessarily exist,
	// e.g. if you want to read from a stream. Then ErrNoHeader will be returned.
	Header NavHeader

	//b       *bufio.Reader
	sc  *bufio.Scanner
	eph Eph
	//ephLines []string
	buf     bytes.Buffer
	lineNum int
	err     error
}

// NewNavDecoder creates a new decoder for RINEX Navigation data.
// The RINEX header will be read implicitly if it exists. The header must not exist, that is usful e.g.
// for reading from streams.
//
// It is the caller's responsibility to call Close on the underlying reader when done!
func NewNavDecoder(r io.Reader) (*NavDecoder, error) {
	var err error
	//br := bufio.NewReader(r)
	/* 	rc, ok := r.(io.ReadCloser)
	   	if !ok && r != nil {
	   		log.Printf("WARN: new nav decoder: could not convert underlying reader to io.ReadCloser")
	   		rc = ioutil.NopCloser(r)
	   	}
	   	dec := &NavDecoder{r: rc} */
	dec := &NavDecoder{sc: bufio.NewScanner(r)}
	// TODO: reset reader?
	// if err := dec.Reset(r); err != nil {
	// 	return nil, err
	// }

	dec.Header, err = dec.readHeader()
	return dec, err
}

// Err returns the first non-EOF error that was encountered by the decoder.
func (dec *NavDecoder) Err() error {
	if dec.err == io.EOF {
		return nil
	}

	return dec.err
}

func (dec *NavDecoder) unmarshal(sys SatelliteSystem) error {
	eph := NewEph(sys)
	err := eph.unmarshal(dec.buf.Bytes())
	if err != nil {
		dec.setErr(err)
		return err
	}
	dec.eph = eph
	return nil
}

// Close closes the wrapped io.Reader originally passed to NewReader.
/* func (dec *NavDecoder) Close() error {
	dec.err = dec.r.Close()
	return dec.err
} */

// Reset discards the Reader z's state and makes it equivalent to the
// result of its original state from NewReader, but reading from r instead.
// This permits reusing a Reader rather than allocating a new one.
/* func (dec *NavDecoder) Reset(r io.Reader) error {
	*dec = NavDecoder{
		decompressor: z.decompressor,
	}

	if rr, ok := r.(flate.Reader); ok {
		dec.r = rr
	} else {
		dec.r = bufio.NewReader(r)
	}

	dec.NavHeader, dec.err = dec.readHeader()
	return dec.err
} */

// readHeader reads a RINEX Navigation header. If the Header does not exist,
// a ErrNoHeader error will be returned.
func (dec *NavDecoder) readHeader() (hdr NavHeader, err error) {
	// The header always begins with "RINEX VERSION / TYPE".
	/* 	line1 := []byte("")
	   	line1, err = dec.b.Peek(80)
	   	if err != nil {
	   		return
	   	}
	   	if !bytes.Contains(line1, []byte("RINEX VERSION / TYPE")) {
	   		err = ErrNoHeader
	   		return
	   	} */

	// Now we can read the header
	maxLines := 300
read:
	for dec.sc.Scan() {
		dec.lineNum++
		line := dec.sc.Text()
		//fmt.Print(line)

		// The header always begins with "RINEX VERSION / TYPE".
		if dec.lineNum == 1 {
			if !strings.Contains(line, "RINEX VERSION / TYPE") {
				err = ErrNoHeader
				return
			}
		}
		if dec.lineNum > maxLines {
			return hdr, fmt.Errorf("Reading header failed: line %d reached without finding end of header", maxLines)
		}
		if len(line) < 60 {
			continue
		}

		// RINEX files are ASCII
		val := line[:60]
		key := strings.TrimSpace(line[60:])

		hdr.labels = append(hdr.labels, key)

		switch key {
		case "RINEX VERSION / TYPE":
			if f64, err := strconv.ParseFloat(strings.TrimSpace(val[:20]), 32); err == nil {
				hdr.RINEXVersion = float32(f64)
			} else {
				log.Printf("Could not parse RINEX VERSION: %v", err)
			}
			hdr.RINEXType = strings.TrimSpace(val[20:21])

			s := strings.TrimSpace(val[40:41])
			/* 			if hdr.RINEXVersion < 3 {
				hlp := map[string]SatelliteSystem{
					"N": SatSysGPS,
					"G": SatSysGLO,
					"E": SatSysGAL,
					"L": SatSysGAL, // gfzrnx
					"S": SatSysSBAS,
				}
			}  */
			// Rnx2.11: GPS G, GLO R, Gal E, GEO Nav H, GEO SBAS S
			// L: GALILEO NAV DATA (aubg207w.16l)
			// E: GALILEO NAV DATA (gfzrnx)
			/* 				if ( $val =~ /[NGELCHS]/ ) {
			   					my %translate = ( N => "G", G => "R", E => "E", L => "E", C => "C", H => "S", S => "S" );
			   					$self->satSystem( $translate{$val} );
			   				}
			   				else { $ok = 0 } */

			if sys, ok := sysPerAbbr[s]; ok {
				hdr.SatSystem = sys
			} else {
				err = fmt.Errorf("read header: invalid satellite system in line %d: %s", dec.lineNum, line)
				return
			}
		case "PGM / RUN BY / DATE":
			hdr.Pgm = strings.TrimSpace(val[:20])
			hdr.RunBy = strings.TrimSpace(val[20:40])
			hdr.Date = strings.TrimSpace(val[40:])
		case "COMMENT":
			hdr.Comments = append(hdr.Comments, strings.TrimSpace(val))
		case "IONOSPHERIC CORR":
			// TODO
		case "TIME SYSTEM CORR":
			// TODO
		case "LEAP SECONDS":
			// TODO
			// my @lsecs = split ( " ", trim($val) );
			// $self->leapSecs( $lsecs[0] );    # ab Vers. 3 hier mehrere Werte moeglich!
		case "END OF HEADER":
			break read
		default:
			log.Printf("Header field %q not handled yet", key)
		}
	}

	err = dec.sc.Err()
	return
}

// NextEphemeris reads the next Ephemeris into the buffer.
// It returns false when the scan stops, either by reaching the end of the input or an error.
//
// If there is no header we suppose the format is RINEX3.
// TODO: read all values
func (dec *NavDecoder) NextEphemeris() bool {
	for dec.sc.Scan() {
		dec.lineNum++
		//line := dec.sc.Text()
		line := dec.sc.Bytes()

		// RINEX 3
		if dec.Header.RINEXVersion == 0 || dec.Header.RINEXVersion >= 3 {
			//if !strings.ContainsAny(line[:1], "GREJCIS") {
			if !bytes.ContainsAny(line[:1], "GREJCIS") {
				log.Printf("stream does not start with epoch line: %q", line) // must not be an error
				continue
			}

			sys, ok := sysPerAbbr[string(line[:1])]
			if !ok {
				dec.setErr(fmt.Errorf("invalid satellite system: %q: line %d", line[:1], dec.lineNum))
				return false
			}

			// Write the ephemeris data into the buffer.
			nLines := 8
			switch sys {
			case SatSysGLO, SatSysSBAS:
				nLines = 4
			}

			dec.buf.Reset()
			dec.buf.Write(line)
			dec.buf.WriteByte('\n')
			//dec.ephLines = dec.ephLines[:0] // reuse
			//dec.ephLines = append(dec.ephLines, string(line))
			for ii := 1; ii < nLines; ii++ {
				dec.sc.Scan()
				dec.lineNum++
				if err := dec.sc.Err(); err != nil {
					dec.setErr(fmt.Errorf("read eph lines scanner error: %v", err))
					return false
				}
				//dec.ephLines = append(dec.ephLines, dec.sc.Text())
				dec.buf.Write(dec.sc.Bytes())
				dec.buf.WriteByte('\n')
			}

			err := dec.unmarshal(sys)
			if err != nil {
				dec.setErr(err)
				return false
			}

			return true
		}

		// RINEX 2
		dec.setErr(fmt.Errorf("RINEX 2 not supported so far"))
		return false
	}

	if err := dec.sc.Err(); err != nil {
		dec.setErr(fmt.Errorf("read eph scanner error: %v", err))
	}

	return false // EOF
}

// Ephemeris returns the most recent ephemeris generated by a call to NextEphemeris.
func (dec *NavDecoder) Ephemeris() Eph {
	return dec.eph
}

// setErr records the first error encountered.
func (dec *NavDecoder) setErr(err error) {
	if dec.err == nil || dec.err == io.EOF {
		dec.err = err
	}
}

// A NavFile contains fields and methods for RINEX navigation files and includes common methods for
// handling RINEX Nav files.
// It is useful e.g. for operations on the RINEX filename.
// If you do not need these file-related features, use the NavDecoder instead.
type NavFile struct {
	*RnxFil
	Header NavHeader
}

// NewNavFile returns a new Navigation File object.
func NewNavFile(filepath string) (*NavFile, error) {
	navFil := &NavFile{RnxFil: &RnxFil{Path: filepath}}
	err := navFil.parseFilename()
	return navFil, err
}

// Validate validates the RINEX Nav file. It is valid if no error is returned.
func (f *NavFile) Validate() error {
	log.Printf("validate nav file %s", f.Path)
	r, err := os.Open(f.Path)
	if err != nil {
		return fmt.Errorf("open nav file: %v", err)
	}
	defer r.Close()

	// Read the header
	dec, err := NewNavDecoder(r)
	if err != nil {
		return err
	}
	f.Header = dec.Header

	// TODO add checks
	err = dec.Header.Validate()
	if err != nil {
		return err
	}

	return nil
}

var rnx3HeaderLables = []headerLabel{
	// mandatory
	{label: "RINEX VERSION / TYPE", official: true, optional: false},
	{label: "PGM / RUN BY / DATE", official: true, optional: false},
	{label: "END OF HEADER", official: true, optional: false},
	// optional
	{label: "COMMENT", official: true, optional: true},
	{label: "IONOSPHERIC CORR", official: true, optional: true},
	{label: "TIME SYSTEM CORR", official: true, optional: true},
	{label: "LEAP SECONDS", official: true, optional: true},
}

var navHeaderLables = map[float32][]headerLabel{
	2: {
		// mandatory
		headerLabel{label: "RINEX VERSION / TYPE", official: true, optional: false},
		headerLabel{label: "PGM / RUN BY / DATE", official: true, optional: false},
		headerLabel{label: "END OF HEADER", official: true, optional: false},
		// optional
		headerLabel{label: "COMMENT", official: true, optional: true},
		headerLabel{label: "ION ALPHA", official: true, optional: true},
		headerLabel{label: "ION BETA", official: true, optional: true},
		headerLabel{label: "DELTA-UTC: A0,A1,T,W", official: true, optional: true},
		headerLabel{label: "LEAP SECONDS", official: true, optional: true},
	},
	2.01: {
		// mandatory
		headerLabel{label: "RINEX VERSION / TYPE", official: true, optional: false},
		headerLabel{label: "PGM / RUN BY / DATE", official: true, optional: false},
		headerLabel{label: "END OF HEADER", official: true, optional: false},
		// optional
		headerLabel{label: "COMMENT", official: true, optional: true},
		headerLabel{label: "ION ALPHA", official: true, optional: true},
		headerLabel{label: "ION BETA", official: true, optional: true},
		headerLabel{label: "DELTA-UTC: A0,A1,T,W", official: true, optional: true},
		headerLabel{label: "LEAP SECONDS", official: true, optional: true},
		headerLabel{label: "CORR TO SYSTEM TIME", official: true, optional: true},
	},
	2.10: {
		// mandatory
		headerLabel{label: "RINEX VERSION / TYPE", official: true, optional: false},
		headerLabel{label: "PGM / RUN BY / DATE", official: true, optional: false},
		headerLabel{label: "END OF HEADER", official: true, optional: false},
		// optional
		headerLabel{label: "COMMENT", official: true, optional: true},
		headerLabel{label: "ION ALPHA", official: true, optional: true},
		headerLabel{label: "ION BETA", official: true, optional: true},
		headerLabel{label: "DELTA-UTC: A0,A1,T,W", official: true, optional: true},
		headerLabel{label: "LEAP SECONDS", official: true, optional: true},
		headerLabel{label: "CORR TO SYSTEM TIME", official: true, optional: true},
	},
	2.11: {
		// The "CORR TO SYSTEM TIME" header record (in 2.10 for GLONASS Nav) has been replaced by the more general record "D-UTC A0,A1,T,W,S,U" in Version 2.11.
		// mandatory
		headerLabel{label: "RINEX VERSION / TYPE", official: true, optional: false},
		headerLabel{label: "PGM / RUN BY / DATE", official: true, optional: false},
		headerLabel{label: "END OF HEADER", official: true, optional: false},
		// optional
		headerLabel{label: "COMMENT", official: true, optional: true},
		headerLabel{label: "ION ALPHA", official: true, optional: true},
		headerLabel{label: "ION BETA", official: true, optional: true},
		headerLabel{label: "DELTA-UTC: A0,A1,T,W", official: true, optional: true},
		headerLabel{label: "LEAP SECONDS", official: true, optional: true},
		headerLabel{label: "CORR TO SYSTEM TIME", official: true, optional: true}, // ??
	},
	3.00: rnx3HeaderLables,
	3.01: rnx3HeaderLables,
	3.02: rnx3HeaderLables,
	3.03: rnx3HeaderLables,
	3.04: rnx3HeaderLables,
	4: {
		// unofficial CNAV files
		// mandatory
		headerLabel{label: "RINEX VERSION / TYPE", optional: false},
		headerLabel{label: "PGM / RUN BY / DATE", optional: false},
		headerLabel{label: "END OF HEADER", optional: false},
		// optional
		headerLabel{label: "COMMENT", optional: true},
		headerLabel{label: "IONOSPHERIC CORR", optional: true},
		headerLabel{label: "TIME SYSTEM CORR", optional: true},
		headerLabel{label: "LEAP SECONDS", optional: true},
	},
}

// Validate validates the RINEX Nav file. It is valid if no error is returned.
func (hdr NavHeader) Validate() error {
	if hdr.RINEXVersion >= 3 {
		if hdr.RINEXType != "N" {
			return fmt.Errorf("invalid RINEX TYPE: %q", hdr.RINEXType)
		}
	}

	// unofficial RINEX 2.12
	if hdr.RINEXVersion == 2.12 {
		return fmt.Errorf("invalid RINEX VERSION: %.2f", 2.12)
	}

	// Check header lines
	if hLablesMust, ok := navHeaderLables[hdr.RINEXVersion]; ok {
		// https://stackoverflow.com/questions/10485743/contains-method-for-a-slice

		// Check existence of mandatory header lines
		hlpmap := make(map[string]struct{}, len(hdr.labels))
		for _, l := range hdr.labels {
			hlpmap[l] = struct{}{}
		}

		ok := false
		for _, f := range hLablesMust {
			if !f.optional {
				if _, ok = hlpmap[f.label]; !ok {
					hdr.warnings = append(hdr.warnings, fmt.Sprintf("mandatory header label does not exist: %s", f.label))
				}
			}
		}

		// Vice versa, check found header lines
		hlpmap = make(map[string]struct{}, len(hLablesMust))
		for _, h := range hLablesMust {
			hlpmap[h.label] = struct{}{}
		}
		for _, l := range hdr.labels {
			if _, ok = hlpmap[l]; !ok {
				hdr.warnings = append(hdr.warnings, fmt.Sprintf("invalid RINEX %.2f header label: %s", hdr.RINEXVersion, l))
			}
		}

	} else {
		return fmt.Errorf("invalid RINEX VERSION: %.2f", hdr.RINEXVersion)
	}

	return nil
}

// parseFloatsNavLine parses a common data line of a nav file, having four floats 4X,4D19.12.
func parseFloatsNavLine(s string) (f1, f2, f3, f4 float64, err error) {
	f1, err = parseFloat(s[4 : 4+19])
	if err != nil {
		return
	}

	f2, err = parseFloat(s[23 : 23+19])
	if err != nil {
		return
	}

	if len(s) < 45 {
		return
	}
	f3, err = parseFloat(s[42 : 42+19])
	if err != nil {
		return
	}

	if len(s) < 64 {
		return
	}
	f4, err = parseFloat(s[61 : 61+19])
	return
}
