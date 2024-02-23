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
)

// A ClockHeader stores the RINEX Clock Header information.
// That header is exposed as the fields of the Decoder and Encoder structs.
// TODO Parse all fields.
type ClockHeader struct {
	RINEXVersion float32 // RINEX Format version
	RINEXType    string  // RINEX File type

	Pgm   string    // name of program creating this file
	RunBy string    // name of agency creating this file
	Date  time.Time // Date and time of file creation

	Comments []string // comments
	Labels   []string // all Header Labels found
}

// ClockDecoder reads and decodes from a RINEX Clock input stream.
type ClockDecoder struct {
	// The Header is valid after NewClockDecoder or Reader.Reset. The header must exist,
	// otherwise ErrNoHeader will be returned.
	Header  ClockHeader
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

// readHeader reads a RINEX clock header. If the Header does not exist,
// a ErrNoHeader error will be returned.
func (dec *ClockDecoder) readHeader() (hdr ClockHeader, err error) {
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
		case "RINEX VERSION / TYPE":
			if f64, err := strconv.ParseFloat(strings.TrimSpace(val[:20]), 32); err == nil {
				hdr.RINEXVersion = float32(f64)
			} else {
				return hdr, fmt.Errorf("parse RINEX VERSION: %v", err)
			}
			hdr.RINEXType = val[20:21]
			if hdr.RINEXType != "C" {
				return hdr, fmt.Errorf("invalid RINEX TYPE: %q", hdr.RINEXType)
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
		case "END OF HEADER":
			break readln
		default:
			log.Printf("Header field %q not handled yet", key)
		}
	}

	if hdr.RINEXVersion == 0 {
		return hdr, fmt.Errorf("unknown RINEX Version")
	}

	err = dec.sc.Err()
	return hdr, err
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
