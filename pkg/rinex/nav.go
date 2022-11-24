package rinex

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
)

const (
	// TimeOfClockFormat is the time format within RINEX3 Nav records.
	TimeOfClockFormat string = "2006  1  2 15  4  5"
)

// Eph is the interface that wraps some methods for all types of ephemeris.
type Eph interface {
	// Validate checks the ephemeris.
	Validate() error

	// Returns the ephemermis' PRN.
	GetPRN() PRN

	// Returns the ephemermis' time of clock (toc).
	GetTime() time.Time

	//unmarshal(data []byte) error
}

// NewEph returns a new ephemeris having the concrete type.
func NewEph(sys gnss.System) Eph {
	var eph Eph
	switch sys {
	case gnss.SysGPS:
		eph = &EphGPS{}
	case gnss.SysGLO:
		eph = &EphGLO{}
	case gnss.SysGAL:
		eph = &EphGAL{}
	case gnss.SysQZSS:
		eph = &EphQZSS{}
	case gnss.SysBDS:
		eph = &EphBDS{}
	case gnss.SysNavIC:
		eph = &EphNavIC{}
	case gnss.SysSBAS:
		eph = &EphSBAS{}
	default:
		fmt.Printf("unknown satellite system: %v", sys)
		os.Exit(1)
	}

	return eph
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

func (eph *EphGPS) GetPRN() PRN        { return eph.PRN }
func (eph *EphGPS) GetTime() time.Time { return eph.TOC }
func (EphGPS) Validate() error         { return nil }

// EphGLO describes a GLONASS ephemeris.
type EphGLO struct {
	PRN PRN
	TOC time.Time
}

func (eph *EphGLO) GetPRN() PRN        { return eph.PRN }
func (eph *EphGLO) GetTime() time.Time { return eph.TOC }
func (EphGLO) Validate() error         { return nil }

// EphGAL describes a Galileo ephemeris.
type EphGAL struct {
	PRN PRN
	TOC time.Time
}

func (eph *EphGAL) GetPRN() PRN        { return eph.PRN }
func (eph *EphGAL) GetTime() time.Time { return eph.TOC }
func (EphGAL) Validate() error         { return nil }

// EphQZSS describes a QZSS ephemeris.
type EphQZSS struct {
	PRN PRN
	TOC time.Time
}

func (eph *EphQZSS) GetPRN() PRN        { return eph.PRN }
func (eph *EphQZSS) GetTime() time.Time { return eph.TOC }
func (EphQZSS) Validate() error         { return nil }

// EphBDS describes a chinese BDS ephemeris.
type EphBDS struct {
	PRN PRN
	TOC time.Time
}

func (eph *EphBDS) GetPRN() PRN        { return eph.PRN }
func (eph *EphBDS) GetTime() time.Time { return eph.TOC }
func (EphBDS) Validate() error         { return nil }

// EphNavIC describes an indian IRNSS/NavIC ephemeris.
type EphNavIC struct {
	PRN PRN
	TOC time.Time
}

func (eph *EphNavIC) GetPRN() PRN        { return eph.PRN }
func (eph *EphNavIC) GetTime() time.Time { return eph.TOC }
func (EphNavIC) Validate() error         { return nil }

// EphSBAS describes a SBAS payload.
type EphSBAS struct {
	PRN PRN
	TOC time.Time
}

func (eph *EphSBAS) GetPRN() PRN        { return eph.PRN }
func (eph *EphSBAS) GetTime() time.Time { return eph.TOC }
func (EphSBAS) Validate() error         { return nil }

// A NavHeader containes the RINEX Navigation Header information.
// All header parameters are optional and may comprise different types of ionospheric model parameters
// and time conversion parameters.
type NavHeader struct {
	RINEXVersion float32     // RINEX Format version
	RINEXType    string      // RINEX File type. O for Obs
	SatSystem    gnss.System // Satellite System. System is "Mixed" if more than one.

	Pgm   string // name of program creating this file
	RunBy string // name of agency creating this file
	Date  string // date and time of file creation TODO time.Time

	Comments []string // * comment lines

	Labels []string // all Header Labels found
}

// A NavDecoder reads and decodes header and data records from a RINEX Nav input stream.
type NavDecoder struct {
	// The Header is valid after NewNavDecoder or Reader.Reset. The header must not necessarily exist,
	// e.g. if you want to read from a stream. Then ErrNoHeader will be returned.
	Header   NavHeader
	sc       *bufio.Scanner
	eph      Eph
	lineNum  int
	fastMode bool // In fast mode, only the eph type and TOC are read.
	err      error
}

// NewNavDecoder creates a new decoder for RINEX Navigation data.
// The RINEX header will be read implicitly if it exists. The header must not exist, that is usful e.g.
// for reading from streams.
//
// It is the caller's responsibility to call Close on the underlying reader when done!
func NewNavDecoder(r io.Reader) (*NavDecoder, error) {
	var err error
	dec := &NavDecoder{
		Header:   NavHeader{},
		sc:       bufio.NewScanner(r),
		eph:      nil,
		lineNum:  0,
		fastMode: false,
		err:      err,
	}
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

// readHeader reads a RINEX Navigation header. If the Header does not exist,
// a ErrNoHeader error will be returned.
func (dec *NavDecoder) readHeader() (hdr NavHeader, err error) {
	maxLines := 300
readln:
	for dec.readLine() {
		line := dec.line()

		// The header always begins with "RINEX VERSION / TYPE".
		if dec.lineNum == 1 {
			if !strings.Contains(line, "RINEX VERSION / TYPE") {
				err = ErrNoHeader
				return
			}
		}
		if dec.lineNum > maxLines {
			return hdr, fmt.Errorf("reading header failed: line %d reached without finding end of header", maxLines)
		}
		if len(line) < 60 {
			continue
		}

		// RINEX files are ASCII
		val := line[:60]
		key := strings.TrimSpace(line[60:])

		hdr.Labels = append(hdr.Labels, key)

		switch key {
		case "RINEX VERSION / TYPE":
			if f64, err := strconv.ParseFloat(strings.TrimSpace(val[:20]), 32); err == nil {
				hdr.RINEXVersion = float32(f64)
			} else {
				return hdr, fmt.Errorf("could not parse RINEX VERSION: %v", err)
			}
			hdr.RINEXType = strings.TrimSpace(val[20:21])

			s := strings.TrimSpace(val[40:41])
			/* 			if hdr.RINEXVersion < 3 {
				hlp := map[string]gnss.System{
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
			break readln
		default:
			fmt.Printf("Header field %q not handled yet\n", key)
		}
	}

	err = dec.sc.Err()
	return hdr, err
}

// NextEphemeris reads the next Ephemeris into the buffer.
// It returns false when the scan stops, either by reaching the end of the input or an error.
//
// TODO: read all values
func (dec *NavDecoder) NextEphemeris() bool {
	if dec.Header.RINEXVersion < 3 {
		// TODO
		//return dec.nextEphemerisv2()
		panic("rinex-2 not implemented yet")
	}
	if dec.Header.RINEXVersion >= 4 {
		// TODO
		panic("rinex-4 not implemented yet")
	}
	// RINEX Version 3
	return dec.nextEphemerisv3()
}

// decode RINEX Version 3 ephemeris.
func (dec *NavDecoder) nextEphemerisv3() bool {
	for dec.readLine() {
		line := dec.line()
		if len(line) < 1 {
			continue
		}

		if !strings.ContainsAny(line[:1], "GREJCIS") {
			fmt.Printf("rinex: stream does not start with epoch line: %q\n", line) // must not be an error
			continue
		}

		sys, ok := sysPerAbbr[line[:1]]
		if !ok {
			dec.setErr(fmt.Errorf("rinex: invalid satellite system: %q: line %d", line[:1], dec.lineNum))
			return false
		}

		var err error
		switch sys {
		case gnss.SysGPS:
			err = dec.decodeGPS()
		case gnss.SysGLO:
			err = dec.decodeGLO()
		case gnss.SysGAL:
			err = dec.decodeGAL()
		case gnss.SysQZSS:
			err = dec.decodeQZSS()
		case gnss.SysBDS:
			err = dec.decodeBDS()
		case gnss.SysNavIC:
			err = dec.decodeNavIC()
		case gnss.SysSBAS:
			err = dec.decodeSBAS()
		default:
			fmt.Printf("unknown satellite system: %v", sys)
			os.Exit(1)
		}

		if err != nil {
			dec.setErr(err)
			return false
		}

		return true
	}

	if err := dec.sc.Err(); err != nil {
		dec.setErr(fmt.Errorf("rinex: read epochs: %v", err))
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

// readLine reads the next line into buffer. It returns false if an error
// occurs or EOF was reached.
func (dec *NavDecoder) readLine() bool {
	if ok := dec.sc.Scan(); !ok {
		return ok
	}
	dec.lineNum++
	return true
}

// skip i lines.
func (dec *NavDecoder) skipLines(i int) bool {
	for l := 0; l < i; l++ {
		if ok := dec.readLine(); !ok {
			return false
		}
	}
	return true
}

// line returns the current line.
func (dec *NavDecoder) line() string {
	return dec.sc.Text()
}

func (dec *NavDecoder) decodeGPS() (err error) {
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
	eph := &EphGPS{}
	dec.eph = eph

	// reread first line
	line := dec.line()
	eph.PRN, err = newPRN(line[0:3])
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("could not parse TOC: '%s': %v", line, err)
	}

	// In fast mode we only read only the TOC.
	if dec.fastMode {
		dec.skipLines(7)
		return nil
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

	// Line 2
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	line = dec.line()
	eph.IODE, eph.Crs, eph.DeltaN, eph.M0, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	// Line 3
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	line = dec.line()
	eph.Cuc, eph.Ecc, eph.Cus, eph.SqrtA, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	// Line 4
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	line = dec.line()
	eph.Toe, eph.Cic, eph.Omega0, eph.Cis, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	// Line 5
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	line = dec.line()
	eph.I0, eph.Crc, eph.Omega, eph.OmegaDot, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	// Line 6
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	line = dec.line()
	eph.IDOT, eph.L2Codes, eph.ToeWeek, eph.L2PFlag, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	// Line 7
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	line = dec.line()
	eph.URA, eph.Health, eph.TGD, eph.IODC, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	// Line 8
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	line = dec.line()
	eph.Tom, eph.FitInterval, _, _, err = parseFloatsNavLine(line)
	if err != nil {
		return
	}

	return nil
}

func (dec *NavDecoder) decodeGLO() (err error) {
	eph := &EphGLO{}
	dec.eph = eph

	// reread first line
	line := dec.line()
	eph.PRN, err = newPRN(line[0:3])
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("could not parse TOC: '%s': %v", line, err)
	}

	// In fast mode we read only the TOC.
	if dec.fastMode {
		dec.skipLines(3)
		return nil
	}

	// TODO parse remaining lines
	dec.skipLines(3)

	return nil
}

func (dec *NavDecoder) decodeGAL() (err error) {
	eph := &EphGAL{}
	dec.eph = eph

	// reread first line
	line := dec.line()
	eph.PRN, err = newPRN(line[0:3])
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("could not parse TOC: '%s': %v", line, err)
	}

	// In fast mode we read only the TOC.
	if dec.fastMode {
		dec.skipLines(7)
		return nil
	}

	// TODO parse remaining lines
	dec.skipLines(7)

	return nil
}

func (dec *NavDecoder) decodeQZSS() (err error) {
	eph := &EphQZSS{}
	dec.eph = eph

	// reread first line
	line := dec.line()
	eph.PRN, err = newPRN(line[0:3])
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("could not parse TOC: '%s': %v", line, err)
	}

	// In fast mode we read only the TOC.
	if dec.fastMode {
		dec.skipLines(7)
		return nil
	}

	// TODO parse remaining lines
	dec.skipLines(7)

	return nil
}

func (dec *NavDecoder) decodeBDS() (err error) {
	eph := &EphBDS{}
	dec.eph = eph

	// reread first line
	line := dec.line()
	eph.PRN, err = newPRN(line[0:3])
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("could not parse TOC: '%s': %v", line, err)
	}

	// In fast mode we read only the TOC.
	if dec.fastMode {
		dec.skipLines(7)
		return nil
	}

	// TODO parse remaining lines
	dec.skipLines(7)

	return nil
}

func (dec *NavDecoder) decodeNavIC() (err error) {
	eph := &EphNavIC{}
	dec.eph = eph

	// reread first line
	line := dec.line()
	eph.PRN, err = newPRN(line[0:3])
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("could not parse TOC: '%s': %v", line, err)
	}

	// In fast mode we read only the TOC.
	if dec.fastMode {
		dec.skipLines(7)
		return nil
	}

	// TODO parse remaining lines
	dec.skipLines(7)

	return nil
}

func (dec *NavDecoder) decodeSBAS() (err error) {
	eph := &EphSBAS{}
	dec.eph = eph

	// reread first line
	line := dec.line()

	eph.PRN, err = newPRN(line[0:3])
	if err != nil {
		return err
	}

	eph.TOC, err = time.Parse(TimeOfClockFormat, line[4:23])
	if err != nil {
		return fmt.Errorf("could not parse TOC: '%s': %v", line, err)
	}

	// In fast mode we read only the TOC.
	if dec.fastMode {
		dec.skipLines(3)
		return nil
	}

	// TODO parse remaining lines
	dec.skipLines(3)

	return nil
}

// NavStats holds some statistics about a RINEX nav file, derived from the data.
type NavStats struct {
	NumEphemeris    int          `json:"numEphemeris"`    // The number of epochs in the file.
	SatSystems      gnss.Systems `json:"systems"`         // The satellite systems contained.
	Satellites      []PRN        `json:"satellites"`      // The ephemeris' satellites.
	EarliestEphTime time.Time    `json:"earliestEphTime"` // Time of the earliest ephemeris.
	LatestEphTime   time.Time    `json:"latestEphTime"`   // Time of the latest ephemeris.
}

// A NavFile contains fields and methods for RINEX navigation files and includes common methods for
// handling RINEX Nav files.
// It is useful e.g. for operations on the RINEX filename.
// If you do not need these file-related features, use the NavDecoder instead.
type NavFile struct {
	*RnxFil
	Header *NavHeader
	Stats  *NavStats // Some statistics.
}

// NewNavFile returns a new Navigation File object. The file must exist and the name will be parsed.
func NewNavFile(filepath string) (*NavFile, error) {
	navFil := &NavFile{RnxFil: &RnxFil{Path: filepath}}
	err := navFil.parseFilename()
	return navFil, err
}

// Parse and return the Header lines.
func (f *NavFile) ReadHeader() (NavHeader, error) {
	r, err := os.Open(f.Path)
	if err != nil {
		return NavHeader{}, err
	}
	defer r.Close()
	dec, err := NewNavDecoder(r)
	if err != nil {
		return NavHeader{}, err
	}
	f.Header = &dec.Header
	return dec.Header, nil
}

// GetStats reads the file and retuns some statistics.
func (f *NavFile) GetStats() (stats NavStats, err error) {
	r, err := os.Open(f.Path)
	if err != nil {
		return
	}
	defer r.Close()
	dec, err := NewNavDecoder(r)
	if err != nil {
		return
	}
	f.Header = &dec.Header
	dec.fastMode = true

	earliestTOC, latestTOC := time.Time{}, time.Time{}
	seenSystems := make(map[gnss.System]int, 5)
	seenSatellites := make(map[PRN]int, 50)
	nEphs := 0
	for dec.NextEphemeris() {
		eph := dec.Ephemeris()
		nEphs++

		prn := eph.GetPRN()
		if _, exists := seenSystems[prn.Sys]; !exists {
			seenSystems[prn.Sys]++
		}

		if _, exists := seenSatellites[prn]; !exists {
			seenSatellites[prn]++
		}

		stats.Satellites = append(stats.Satellites, prn)

		toc := eph.GetTime()
		if earliestTOC.IsZero() || toc.Before(earliestTOC) {
			earliestTOC = toc
		}
		if latestTOC.IsZero() || toc.After(latestTOC) {
			latestTOC = toc
		}

	}
	if err := dec.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading ephemerides:", err)
	}

	stats.NumEphemeris = nEphs
	stats.EarliestEphTime = earliestTOC
	stats.LatestEphTime = latestTOC

	stats.SatSystems = make([]gnss.System, 0, len(seenSystems))
	for sys := range seenSystems {
		stats.SatSystems = append(stats.SatSystems, sys)
	}

	stats.Satellites = make([]PRN, 0, len(seenSatellites))
	for prn := range seenSatellites {
		stats.Satellites = append(stats.Satellites, prn)
	}
	sort.Sort(ByPRN(stats.Satellites))

	return stats, err
}

// Rnx3Filename returns the filename following the RINEX3 convention.
// In most cases we must read the read the header. The countrycode must come from an external source.
// DO NOT USE! Must parse header first!

// Rnx3Filename returns the filename following the RINEX3 convention.
// TODO !!!
func (f *NavFile) Rnx3Filename() (string, error) {
	// Station Identifier
	if len(f.FourCharID) != 4 {
		return "", fmt.Errorf("FourCharID: %s", f.FourCharID)
	}

	if len(f.CountryCode) != 3 {
		return "", fmt.Errorf("CountryCode: %s", f.CountryCode)
	}

	var fn strings.Builder
	fn.WriteString(f.FourCharID)
	fn.WriteString(strconv.Itoa(f.MonumentNumber))
	fn.WriteString(strconv.Itoa(f.ReceiverNumber))
	fn.WriteString(f.CountryCode)

	fn.WriteString("_")

	if f.DataSource == "" {
		fn.WriteString("U")
	} else {
		fn.WriteString(f.DataSource)
	}

	fn.WriteString("_")

	// StartTime
	// AREG00PER_R_20201690000_01D_MN.rnx

	fn.WriteString(strconv.Itoa(f.StartTime.Year()))
	fn.WriteString(fmt.Sprintf("%03d", f.StartTime.YearDay()))
	fn.WriteString(fmt.Sprintf("%02d", f.StartTime.Hour()))
	fn.WriteString(fmt.Sprintf("%02d", f.StartTime.Minute()))
	fn.WriteString("_")

	fn.WriteString(string(f.FilePeriod))
	fn.WriteString("_")

	fn.WriteString(f.DataType)
	fn.WriteString(".rnx")

	if len(fn.String()) != 34 {
		return "", fmt.Errorf("invalid filename: %s", fn.String())
	}

	return fn.String(), nil
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
