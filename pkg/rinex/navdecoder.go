package rinex

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
)

const (
	// TimeOfClockFormat is the time format within RINEX-3 Nav records.
	TimeOfClockFormat string = "2006  1  2 15  4  5"

	// TimeOfClockFormatv2 is the time format within RINEX-2 Nav records.
	TimeOfClockFormatv2 string = "06  1  2 15  4  5.0"
)

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

			if hdr.RINEXVersion < 3 {
				// Only N and G are RINEX conform.
				switch hdr.RINEXType {
				case "N":
					hdr.SatSystem = gnss.SysGPS
				case "G":
					hdr.SatSystem = gnss.SysGLO
				case "E", "L":
					hdr.SatSystem = gnss.SysGAL
				case "C":
					hdr.SatSystem = gnss.SysBDS
				case "J":
					hdr.SatSystem = gnss.SysQZSS
				case "S":
					hdr.SatSystem = gnss.SysSBAS
				default:
					return hdr, fmt.Errorf("read RINEX-2 header: invalid satellite system: %s", hdr.RINEXType)
				}
				continue
			}

			// version >= 3:
			s := strings.TrimSpace(val[40:41])
			if sys, ok := sysPerAbbr[s]; ok {
				hdr.SatSystem = sys
			} else {
				return hdr, fmt.Errorf("read RINEX-3 header: invalid satellite system: %s", s)
			}
		case "PGM / RUN BY / DATE":
			// Additional lines of this type can appear together after the second line, if needed to preserve the history of previous actions on the file.
			if hdr.Pgm != "" {
				continue
				// TODO additional lines
			}
			hdr.Pgm = strings.TrimSpace(val[:20])
			hdr.RunBy = strings.TrimSpace(val[20:40])
			if date, err := parseHeaderDate(strings.TrimSpace(val[40:])); err == nil {
				hdr.Date = date
			} else {
				log.Printf("parse header date: %q, %v", val[40:], err)
			}
		case "COMMENT":
			hdr.Comments = append(hdr.Comments, strings.TrimSpace(val))
		case "MERGED FILE":
			nStr := strings.TrimSpace(val[:9])
			if nStr != "" {
				n, err := strconv.Atoi(nStr)
				if err != nil {
					return hdr, fmt.Errorf("parse %q: %v", key, err)
				}
				hdr.MergedFiles = n
			}
		case "DOI":
			hdr.DOI = strings.TrimSpace(val)
		case "LICENSE OF USE":
			hdr.Licenses = append(hdr.Licenses, strings.TrimSpace(val))
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
			log.Printf("Header field %q not handled yet", key)
		}
	}

	if err := dec.sc.Err(); err != nil {
		return hdr, err
	}

	return hdr, err
}

// NextEphemeris reads the next Ephemeris into the buffer.
// It returns false when the scan stops, either by reaching the end of the input or an error.
//
// TODO: read all values
func (dec *NavDecoder) NextEphemeris() bool {
	if dec.Header.RINEXVersion < 3 {
		return dec.nextEphemerisv2()
	}
	if dec.Header.RINEXVersion < 4 {
		return dec.nextEphemerisv3()
	}
	return dec.nextEphemerisv4()
}

// decode RINEX Version 2 ephemeris.
func (dec *NavDecoder) nextEphemerisv2() bool {
	for dec.readLine() {
		line := dec.line()
		if len(line) < 3 {
			continue
		}

		if strings.TrimSpace(line[:2]) == "" {
			continue
		}

		err := dec.decodeEPH(dec.Header.SatSystem)
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

// decode RINEX Version 3 ephemeris.
func (dec *NavDecoder) nextEphemerisv3() bool {
	for dec.readLine() {
		line := dec.line()
		if len(line) < 1 {
			continue
		}

		if !strings.ContainsAny(line[:1], "GREJCIS") {
			log.Printf("rinex: line %d: stream does not start with epoch line: %q", dec.lineNum, line) // must not be an error
			continue
		}

		sys, ok := sysPerAbbr[line[:1]]
		if !ok {
			dec.setErr(fmt.Errorf("rinex: line %d: invalid satellite system: %q", dec.lineNum, line[:1]))
			return false
		}

		if len(line) < 23 { // ToC. simple test to prevent panicing.
			dec.setErr(fmt.Errorf("rinex: invalid line %d: %q", dec.lineNum, line))
			continue
		}

		err := dec.decodeEPH(sys)
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

// decode RINEX Version 4 ephemeris.
func (dec *NavDecoder) nextEphemerisv4() bool {
	for dec.readLine() {
		line := dec.line()
		if len(line) < 1 {
			continue
		}

		if !strings.HasPrefix(line, "> ") {
			continue
		}

		rectyp := line[2:5]
		if rectyp == string(NavRecordTypeEPH) {
			sys, ok := sysPerAbbr[line[6:7]]
			if !ok {
				dec.setErr(fmt.Errorf("rinex: invalid satellite system in: %q (line %d)", line, dec.lineNum))
				return false
			}

			err := dec.decodeEPH(sys)
			if err != nil {
				dec.setErr(err)
				return false
			}
			return true
		}

		// TODO read other record types

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

// setErr adds an error.
func (dec *NavDecoder) setErr(err error) {
	dec.err = errors.Join(dec.err, err)
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

// parse the prn from the data record.
func (dec *NavDecoder) parsePRN() (gnss.PRN, error) {
	line := dec.line()
	if dec.Header.RINEXVersion < 3 {
		return gnss.NewPRN(fmt.Sprintf("%s%s", dec.Header.SatSystem.Abbr(), line[0:2]))
	}
	return gnss.NewPRN(line[0:3])
}

// parse the time of eph from the data record.
func (dec *NavDecoder) parseToC() (time.Time, error) {
	line := dec.line()
	if dec.Header.RINEXVersion < 3 {
		toc := line[3:22]
		if toc[0] == ' ' { // year can be only 1 char, so pad with 0.
			toc = strings.Replace(toc, " ", "0", 1)
		}
		return time.Parse(TimeOfClockFormatv2, toc)
	}

	return time.Parse(TimeOfClockFormat, line[4:23])
}

// parseFloatsFromLine parses a common data line of a nav file, having 4 floats 4X,4D19.12.
// For RINEX-2 it is 3X,4D19.12, so we have a shift of -1.
func (dec *NavDecoder) parseFloatsFromLine(shift int) (f1, f2, f3, f4 float64, err error) {
	line := dec.line()
	f1, err = parseFloat(line[4+shift : 4+shift+19])
	if err != nil {
		return
	}

	f2, err = parseFloat(line[23+shift : 23+shift+19])
	if err != nil {
		return
	}

	if len(line) < 45+shift {
		return
	}
	f3, err = parseFloat(line[42+shift : 42+shift+19])
	if err != nil {
		return
	}

	if len(line) < 64+shift {
		return
	}
	f4, err = parseFloat(line[61+shift : 61+shift+19])
	return
}

func (dec *NavDecoder) decodeEPH(sys gnss.System) (err error) {
	switch sys {
	case gnss.SysGPS:
		return dec.decodeGPS()
	case gnss.SysGLO:
		return dec.decodeGLO()
	case gnss.SysGAL:
		return dec.decodeGAL()
	case gnss.SysQZSS:
		return dec.decodeQZSS()
	case gnss.SysBDS:
		return dec.decodeBDS()
	case gnss.SysNavIC:
		return dec.decodeNavIC()
	case gnss.SysSBAS:
		return dec.decodeSBAS()
	}

	return fmt.Errorf("rinex: not supported satellite system: %v", sys)
}

func (dec *NavDecoder) decodeGPS() (err error) {
	eph := &EphGPS{}
	dec.eph = eph

	// reread first line
	line := dec.line()

	if dec.Header.RINEXVersion >= 4 {
		eph.MessageType = strings.TrimSpace(line[10:])
		if ok := dec.readLine(); !ok {
			return fmt.Errorf("could not read line")
		}
		line = dec.line()
	}

	eph.PRN, err = dec.parsePRN()
	if err != nil {
		return fmt.Errorf("parse prn: '%s': %v", line, err)
	}

	eph.TOC, err = dec.parseToC()
	if err != nil {
		return fmt.Errorf("parse ToC: '%s': %v", line, err)
	}

	// In fast mode we only read only the TOC.
	if dec.fastMode {
		dec.skipLines(7)
		return nil
	}

	shift := 0
	if dec.Header.RINEXVersion < 3 {
		shift = -1
	}

	eph.ClockBias, err = parseFloat(line[23+shift : 23+shift+19])
	if err != nil {
		return
	}

	eph.ClockDrift, err = parseFloat(line[42+shift : 42+shift+19])
	if err != nil {
		return
	}

	eph.ClockDriftRate, err = parseFloat(line[61+shift : 61+shift+19])
	if err != nil {
		return
	}

	// Line 2
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	eph.IODE, eph.Crs, eph.DeltaN, eph.M0, err = dec.parseFloatsFromLine(shift)
	if err != nil {
		return
	}

	// Line 3
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	eph.Cuc, eph.Ecc, eph.Cus, eph.SqrtA, err = dec.parseFloatsFromLine(shift)
	if err != nil {
		return
	}

	// Line 4
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	eph.Toe, eph.Cic, eph.Omega0, eph.Cis, err = dec.parseFloatsFromLine(shift)
	if err != nil {
		return
	}

	// Line 5
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	eph.I0, eph.Crc, eph.Omega, eph.OmegaDot, err = dec.parseFloatsFromLine(shift)
	if err != nil {
		return
	}

	// Line 6
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	eph.IDOT, eph.L2Codes, eph.ToeWeek, eph.L2PFlag, err = dec.parseFloatsFromLine(shift)
	if err != nil {
		return
	}

	// Line 7
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	eph.URA, eph.Health, eph.TGD, eph.IODC, err = dec.parseFloatsFromLine(shift)
	if err != nil {
		return
	}

	// Line 8
	if ok := dec.readLine(); !ok {
		return fmt.Errorf("could not read line")
	}
	eph.Tom, eph.FitInterval, _, _, err = dec.parseFloatsFromLine(shift)
	if err != nil {
		return
	}

	return nil
}

func (dec *NavDecoder) decodeGLO() (err error) {
	eph := &EphGLO{}
	dec.eph = eph

	nLines := 4
	if dec.Header.RINEXVersion >= 3.05 {
		nLines = 5
	}

	// reread first line
	line := dec.line()

	if dec.Header.RINEXVersion >= 4 {
		eph.MessageType = strings.TrimSpace(line[10:])
		if ok := dec.readLine(); !ok {
			return fmt.Errorf("could not read line")
		}
		line = dec.line()
	}

	eph.PRN, err = dec.parsePRN()
	if err != nil {
		return fmt.Errorf("parse prn: '%s': %v", line, err)
	}

	eph.TOC, err = dec.parseToC()
	if err != nil {
		return fmt.Errorf("parse ToC: '%s': %v", line, err)
	}

	// In fast mode we read only the TOC.
	if dec.fastMode {
		dec.skipLines(nLines - 1)
		return nil
	}

	// TODO parse remaining lines
	dec.skipLines(nLines - 1)

	return nil
}

func (dec *NavDecoder) decodeGAL() (err error) {
	eph := &EphGAL{}
	dec.eph = eph

	// reread first line
	line := dec.line()

	if dec.Header.RINEXVersion >= 4 {
		eph.MessageType = strings.TrimSpace(line[10:])
		if ok := dec.readLine(); !ok {
			return fmt.Errorf("could not read line")
		}
		line = dec.line()
	}

	eph.PRN, err = dec.parsePRN()
	if err != nil {
		return fmt.Errorf("parse prn: '%s': %v", line, err)
	}

	eph.TOC, err = dec.parseToC()
	if err != nil {
		return fmt.Errorf("parse ToC: '%s': %v", line, err)
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

	if dec.Header.RINEXVersion >= 4 {
		eph.MessageType = strings.TrimSpace(line[10:])
		if ok := dec.readLine(); !ok {
			return fmt.Errorf("could not read line")
		}
		line = dec.line()
	}

	eph.PRN, err = dec.parsePRN()
	if err != nil {
		return fmt.Errorf("parse prn: '%s': %v", line, err)
	}

	eph.TOC, err = dec.parseToC()
	if err != nil {
		return fmt.Errorf("parse ToC: '%s': %v", line, err)
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

	if dec.Header.RINEXVersion >= 4 {
		eph.MessageType = strings.TrimSpace(line[10:])
		if ok := dec.readLine(); !ok {
			return fmt.Errorf("could not read line")
		}
		line = dec.line()
	}

	eph.PRN, err = dec.parsePRN()
	if err != nil {
		return fmt.Errorf("parse prn: '%s': %v", line, err)
	}

	eph.TOC, err = dec.parseToC()
	if err != nil {
		return fmt.Errorf("parse ToC: '%s': %v", line, err)
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

	if dec.Header.RINEXVersion >= 4 {
		eph.MessageType = strings.TrimSpace(line[10:])
		if ok := dec.readLine(); !ok {
			return fmt.Errorf("could not read line")
		}
		line = dec.line()
	}

	eph.PRN, err = dec.parsePRN()
	if err != nil {
		return fmt.Errorf("parse prn: '%s': %v", line, err)
	}

	eph.TOC, err = dec.parseToC()
	if err != nil {
		return fmt.Errorf("parse ToC: '%s': %v", line, err)
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

	if dec.Header.RINEXVersion >= 4 {
		eph.MessageType = strings.TrimSpace(line[10:])
		if ok := dec.readLine(); !ok {
			return fmt.Errorf("could not read line")
		}
		line = dec.line()
	}

	eph.PRN, err = dec.parsePRN()
	if err != nil {
		return fmt.Errorf("parse prn: '%s': %v", line, err)
	}

	eph.TOC, err = dec.parseToC()
	if err != nil {
		return fmt.Errorf("parse ToC: '%s': %v", line, err)
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
