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

// A ClockHeader stores the RINEX Clock Header information.
// That header is exposed as the fields of the Decoder and Encoder structs.
// TODO Parse all fields.
type ClockHeader struct {
	RINEXVersion float32     // RINEX Format version
	RINEXType    string      // RINEX File type
	SatSystem    gnss.System // Satellite System. System is "Mixed" if more than one.

	Pgm   string    // name of program creating this file
	RunBy string    // name of agency creating this file
	Date  time.Time // Date and time of file creation

	TimeSystemID   string     // Time system used for time tags, 3 char (GPS, GAL, UTC, TAI,...)
	AC             string     // Analysis Center as 3-character IGS AC designator
	NumSolnSats    int        // Number of different satellites in the clock data records.
	StaCoordinates []string   // List of stations/receivers with coordinates.
	Sats           []gnss.PRN // List of all satellites reported in this file (PRN LIST).

	Comments []string // comments
	Labels   []string // all Header Labels found
}

// ClockDecoder reads and decodes from a RINEX Clock input stream.
type ClockDecoder struct {
	// The Header is valid after NewClockDecoder or Reader.Reset. The header must exist,
	// otherwise ErrNoHeader will be returned.
	Header  *ClockHeader
	sc      *bufio.Scanner
	lineNum int
	err     error
}

// NewClockDecoder returns a new RINEX clock decoder that reads from r.
// The RINEX header will be read implicitly. The header must exist.
//
// It is the caller's responsibility to call Close on the underlying reader when done!
func NewClockDecoder(r io.Reader) (*ClockDecoder, error) {
	dec := &ClockDecoder{sc: bufio.NewScanner(r)}
	dec.Header, dec.err = dec.readHeader()
	return dec, dec.err
}

// read the RUINEX clock header.
func (dec *ClockDecoder) readHeader() (*ClockHeader, error) {
	hdr, err := dec.readHeaderVersion()
	if err != nil {
		return nil, err
	}

	dec.Header = hdr

	if hdr.RINEXVersion >= 3.04 {
		return dec.readHeader304()
	}
	return dec.readHeader300()
}

// readHeaderVersion reads the first line of the header to get the version, type and satellite system.
// We need the version for further header parsing.
func (dec *ClockDecoder) readHeaderVersion() (*ClockHeader, error) {
	// We need to know the version from the first header line.
	hdr := &ClockHeader{}

	dec.readLine()
	line := dec.line()
	if !strings.Contains(line, "RINEX VERSION") {
		return nil, fmt.Errorf("parse RINEX VERSION failed: %q", line)
	}

	// RINEX Version
	if f64, err := strconv.ParseFloat(strings.TrimSpace(line[:9]), 32); err == nil {
		hdr.RINEXVersion = float32(f64)
	} else {
		return nil, fmt.Errorf("parse RINEX VERSION: %v", err)
	}

	// RINEX Filetype
	if hdr.RINEXVersion >= 3.04 {
		hdr.RINEXType = line[21:22]
	} else {
		hdr.RINEXType = line[20:21]
	}

	if hdr.RINEXType != "C" {
		return nil, fmt.Errorf("invalid RINEX TYPE: %q", hdr.RINEXType)
	}

	// Satellite system
	sys := ""
	if hdr.RINEXVersion >= 3.04 {
		sys = strings.TrimSpace(line[42:43])
	} else {
		sys = strings.TrimSpace(line[40:41])
	}

	if sys != "" {
		if s, ok := sysPerAbbr[sys]; ok {
			hdr.SatSystem = s
		} else {
			return nil, fmt.Errorf("read header: invalid satellite system: %q", sys)
		}
	}

	return hdr, nil
}

// readHeader reads a RINEX clock header. If the Header does not exist,
// an ErrNoHeader will be returned.
// There was a bigger change introduced in version 3.04.
func (dec *ClockDecoder) readHeader300() (*ClockHeader, error) {
	hdr := dec.Header

readln:
	for dec.readLine() {
		line := dec.line()
		if len(line) < 60 {
			continue
		}

		val := line[:60] // RINEX files are ASCII
		key := strings.TrimSpace(line[60:])
		hdr.Labels = append(hdr.Labels, key)

		switch key {
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
				log.Printf("parse header date: %v", err)
			}
		case "TIME SYSTEM ID":
			hdr.TimeSystemID = strings.TrimSpace(val[3:6])
		case "COMMENT":
			hdr.Comments = append(hdr.Comments, strings.TrimSpace(val))
		case "ANALYSIS CENTER":
			hdr.AC = strings.TrimSpace(val[:3])
		case "# OF SOLN SATS":
			nSats, err := strconv.Atoi(strings.TrimSpace(val[:6]))
			if err != nil {
				return hdr, fmt.Errorf("parse %q: %v", key, err)
			}
			hdr.NumSolnSats = nSats
		case "SOLN STA NAME / NUM":
			hdr.StaCoordinates = append(hdr.StaCoordinates, val)
		case "PRN LIST":
			if err := dec.parseHeaderPRNList(val); err != nil {
				return hdr, err
			}
		case "END OF HEADER":
			break readln
		default:
			log.Printf("Header field %q not handled yet", key)
		}
	}

	if hdr.RINEXVersion == 0 {
		return hdr, fmt.Errorf("unknown RINEX Version")
	}

	err := dec.sc.Err()
	return hdr, err
}

// readHeader reads a RINEX clock header. If the Header does not exist,
// a ErrNoHeader error will be returned.
// There was a bigger change introduced in version 3.04.
func (dec *ClockDecoder) readHeader304() (*ClockHeader, error) {
	hdr := dec.Header

readln:
	for dec.readLine() {
		line := dec.line()
		if len(line) < 60 {
			continue
		}

		val := line[:65] // RINEX files are ASCII
		key := strings.TrimSpace(line[65:])
		hdr.Labels = append(hdr.Labels, key)

		switch key {
		case "PGM / RUN BY / DATE":
			// Additional lines of this type can appear together after the second line, if needed to preserve the history of previous actions on the file.
			if hdr.Pgm != "" {
				continue
				// TODO additional lines
			}
			hdr.Pgm = strings.TrimSpace(val[:19])
			hdr.RunBy = strings.TrimSpace(val[21:40])
			if date, err := parseHeaderDate(strings.TrimSpace(val[42:])); err == nil {
				hdr.Date = date
			} else {
				log.Printf("parse header date: %q, %v", val[42:], err)
			}
		case "TIME SYSTEM ID":
			hdr.TimeSystemID = strings.TrimSpace(val[3:6])
		case "COMMENT":
			hdr.Comments = append(hdr.Comments, strings.TrimSpace(val))
		case "ANALYSIS CENTER":
			hdr.AC = strings.TrimSpace(val[:3])
		case "# OF SOLN SATS":
			nSats, err := strconv.Atoi(strings.TrimSpace(val[:6]))
			if err != nil {
				return hdr, fmt.Errorf("parse %q: %v", key, err)
			}
			hdr.NumSolnSats = nSats
		case "SOLN STA NAME / NUM":
			hdr.StaCoordinates = append(hdr.StaCoordinates, val)
		case "PRN LIST":
			if err := dec.parseHeaderPRNList(val); err != nil {
				return hdr, err
			}
		case "END OF HEADER":
			break readln
		default:
			log.Printf("Header field %q not handled yet", key)
		}
	}

	if hdr.RINEXVersion == 0 {
		return hdr, fmt.Errorf("unknown RINEX Version")
	}

	err := dec.sc.Err()
	return hdr, err
}

// parse header field "PRN LIST".
func (dec *ClockDecoder) parseHeaderPRNList(s string) error {
	sats := strings.Fields(s)
	for _, sat := range sats {
		prn, err := gnss.NewPRN(sat)
		if err != nil {
			log.Printf("WARN: read header: %s: %v", "PRN LIST", err)
			continue
		}
		dec.Header.Sats = append(dec.Header.Sats, prn)
	}
	return nil
}

// Err returns the first non-EOF error that was encountered by the decoder.
func (dec *ClockDecoder) Err() error {
	if dec.err == io.EOF {
		return nil
	}
	return dec.err
}

// setErr adds an error.
func (dec *ClockDecoder) setErr(err error) {
	dec.err = errors.Join(dec.err, err)
}

// readLine reads the next line into buffer. It returns false if an error
// occurs or EOF was reached.
func (dec *ClockDecoder) readLine() bool {
	if ok := dec.sc.Scan(); !ok {
		return ok
	}
	dec.lineNum++
	return true
}

// line returns the current line.
func (dec *ClockDecoder) line() string {
	return dec.sc.Text()
}

// splitAt returns the strings between start/end position pairs in string s, in argument order.
// Leading and trailing white spaces are trimmed.
/* func splitAt(s string, pos ...int) []string {
	fields := []string{}

	for i := 0; i < len(pos); i++ {
		// Set last position to the end of the string, if out of range.
		if i == len(pos)-2 && pos[i+1] > len(s) {
			pos[i+1] = len(s)
		}

		fields = append(fields, strings.TrimSpace(s[pos[i]:pos[i+1]]))
		i++
	}
	return fields
} */
