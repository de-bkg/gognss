package sinex

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
)

const (
	// dateFormat is the date part of a SINEX time string (15:243:02013 YY-year:DOY:sec of day).
	dateFormat string = "06:002"
)

var (
	// oberservation techn lookup map.
	obsTechnMap = map[string]ObservationTechnique{
		"C": ObsTechCombined,
		"D": ObsTechDORIS,
		"L": ObsTechSLR,
		"M": ObsTechLLR,
		"P": ObsTechGPS,
		"R": ObsTechVLBI,
	}
)

// Decoder reads and decodes the SINEX input stream.
type Decoder struct {
	Header *Header

	fileRef   *FileReference
	scan      *bufio.Scanner
	currBlock string // The name of the current block.
	lineNum   int
	//err       error
}

// NewDecoder returns a new decoder that reads from r.
// The header line and FILE/REFERENCE block will be read implicitely.
//
// It is the caller's responsibility to call Close on the underlying reader when done!
func NewDecoder(r io.Reader) (*Decoder, error) {
	dec := &Decoder{scan: bufio.NewScanner(r)}
	dec.fileRef = &FileReference{}

	if err := dec.decodeHeader(); err != nil {
		return nil, err
	}

	return dec, nil
}

// Return the FILE/REFERENCE data.
func (dec *Decoder) GetFileReference() FileReference {
	return *dec.fileRef
}

// NextBlock reports whether there is another block available and moves the reader to the begin of the this block.
// Use CurrentBlock() to get the name of the current block.
func (dec *Decoder) NextBlock() bool {
	for dec.readLine() {
		if dec.scan.Err() != nil {
			return false
		}

		if dec.isBlockBegin() {
			return true
		}
	}

	return false
}

// NextBlockLine reports whether there is another data line in the current block and reads that line into the buffer.
// It returns false when reaching the end of the block.
func (dec *Decoder) NextBlockLine() bool {
	for dec.readLine() {
		if dec.scan.Err() != nil {
			return false
		}

		if dec.isCommentLine() {
			continue
		}

		if dec.isDataLine() {
			return true
		}
		return false
	}

	return false
}

// Returns the name of the current block.
func (dec *Decoder) CurrentBlock() string {
	return dec.currBlock
}

// Reports wheter the current line is a comment.
func (dec *Decoder) isCommentLine() bool {
	return strings.HasPrefix(dec.Line(), "*")
}

// Reports wheter the current line is the begin of a block.
func (dec *Decoder) isBlockBegin() bool {
	return strings.HasPrefix(dec.Line(), "+")
}

// Reports wheter the current line is a data line, that means no comment, block begin etc.
func (dec *Decoder) isDataLine() bool {
	line := dec.Line()
	if len(line) < 1 {
		return false
	}
	return !strings.ContainsAny(line[:1], "-+*%")
}

// decodeHeader decodes the first the header line as well as some mandatory first blocks like FILE/REFERENCE.
func (dec *Decoder) decodeHeader() error {
	err := dec.readHeaderLine()
	if err != nil {
		return err
	}

	// Parse FILE/REFERENCE that should be always the first block.
	if ok := dec.NextBlock(); !ok {
		return fmt.Errorf("no blocks found")
	}

	name := dec.CurrentBlock()
	if name != BlockFileReference {
		return fmt.Errorf("is not the first block: %q", BlockFileReference)
	}

	for dec.NextBlockLine() {
		line := dec.Line()
		key := strings.TrimSpace(line[1:19])
		val := strings.TrimSpace(line[20:])

		switch key {
		case "DESCRIPTION":
			dec.fileRef.Description = val
		case "OUTPUT":
			dec.fileRef.Output = val
		case "CONTACT":
			dec.fileRef.Contact = val
		case "SOFTWARE":
			dec.fileRef.Software = val
		case "HARDWARE":
			dec.fileRef.Hardware = val
		case "INPUT":
			dec.fileRef.Input = val
		default:
			//dec.setErr()
			return fmt.Errorf("invalid %s field: %q", BlockFileReference, key)
		}
	}

	return nil
}

// read the SINEX header which is only one line.
func (dec *Decoder) readHeaderLine() error {
	dec.readLine()
	if dec.scan.Err() != nil {
		return dec.scan.Err()
	}

	line := dec.Line()
	if line[:1] != "%" {
		return fmt.Errorf("read Headerline: does not start with %q", "%")
	}

	hdr := &Header{}
	if err := hdr.UnmarshalSINEX(line); err != nil {
		return err
	}
	dec.Header = hdr
	return nil
}

// readLine reads the next line into buffer. It returns false if an error occurs or EOF was reached.
// It also sets the currBlock in the decoder if a begin or end-block was reached.
func (dec *Decoder) readLine() bool {
	if ok := dec.scan.Scan(); !ok {
		return false
	}

	line := dec.Line()
	if strings.HasPrefix(line, "+") {
		dec.currBlock = strings.TrimSpace(line[1:])
	} else if strings.HasPrefix(line, "-") {
		dec.currBlock = ""
	}

	dec.lineNum++
	return true
}

// Line returns the current Line in buffer.
func (dec *Decoder) Line() string {
	return dec.scan.Text()
}

type Unmarshaler interface {
	UnmarshalSINEX(string) error
}

// Decode the current line into out.
func (dec *Decoder) Decode(out Unmarshaler) error {
	return out.UnmarshalSINEX(dec.Line())
}

// Unmarshal in to out.
func Unmarshal(in string, out Unmarshaler) error {
	return out.UnmarshalSINEX(in)
}

// Unmarshall the header line.
func (hdr *Header) UnmarshalSINEX(in string) error {
	var err error
	hdr.Version = in[6:10]
	hdr.Agency = in[11:14]
	hdr.CreationTime, err = parseTime(in[15:27])
	if err != nil {
		return err
	}
	hdr.AgencyDataProvider = in[28:31]
	hdr.StartTime, err = parseTime(in[32:44])
	if err != nil {
		return err
	}

	hdr.EndTime, err = parseTime(in[45:57])
	if err != nil {
		return err
	}

	if techn, ok := obsTechnMap[in[58:59]]; ok {
		hdr.ObsTech = techn
	} else {
		return fmt.Errorf("unknown observation code: %q", in[58:59])
	}

	hdr.NumEstimates, err = strconv.Atoi(strings.TrimSpace(in[60:65]))
	if err != nil {
		return err
	}

	hdr.ConstraintCode, err = strconv.Atoi(in[66:67])
	if err != nil {
		return err
	}

	hdr.SolutionTypes = strings.Fields(strings.TrimSpace(in[68:]))
	return nil
}

// Unmarshall a SITE/ID record.
func (s *Site) UnmarshalSINEX(in string) error {
	// *CODE PT __DOMES__ T _STATION DESCRIPTION__ _LONGITUDE_ _LATITUDE__ HEIGHT_
	//  ABMF  A 97103M001 P Les Abymes - Raizet ai 298 28 20.9  16 15 44.3   -25.6
	s.Code = SiteCode(cleanField(in[1:5]))
	s.PointCode = cleanField(in[6:8])
	s.DOMESNumber = cleanField(in[9:18])

	if techn, ok := obsTechnMap[in[19:20]]; ok {
		s.ObsTech = techn
	} else {
		return fmt.Errorf("unknown observation code: %q", in[14:15])
	}

	s.Description = in[21:43]
	s.Lon = strings.TrimSpace(in[44:55]) // TODO convert to float
	s.Lat = strings.TrimSpace(in[56:67])

	if hgt, err := strconv.ParseFloat(strings.TrimSpace(in[68:]), 64); err == nil {
		s.Height = hgt
	} else {
		return fmt.Errorf("parse HEIGHT: %v", err)
	}

	return nil
}

// Unmarshall a SITE/ANTENNA record.
func (ant *Antenna) UnmarshalSINEX(in string) error {
	if ant.Antenna == nil {
		ant.Antenna = &gnss.Antenna{}
	}

	ant.SiteCode = SiteCode(cleanField(in[1:5]))
	ant.PointCode = cleanField(in[6:8])
	ant.SolID = cleanField(in[9:13])

	if techn, ok := obsTechnMap[in[14:15]]; ok {
		ant.ObsTech = techn
	} else {
		return fmt.Errorf("unknown observation code: %q", in[14:15])
	}

	if installedAt, err := parseTime(in[16:28]); err == nil {
		ant.DateInstalled = installedAt
	} else {
		return fmt.Errorf("parse DATA START %q: %v", in, err)
	}

	if remAt, err := parseTime(in[29:41]); err == nil {
		ant.DateRemoved = remAt
	} else {
		return fmt.Errorf("parse DATA_END %q: %v", in, err)
	}

	ant.Type = cleanField(in[42:62])
	radom := cleanField(in[58:62])
	if len(radom) == 4 {
		ant.Radome = radom
	}
	ant.SerialNum = cleanField(in[63:])
	return nil
}

// Unmarshall a SITE/RECEIVER record.
func (recv *Receiver) UnmarshalSINEX(in string) error {
	if recv.Receiver == nil {
		recv.Receiver = &gnss.Receiver{}
	}

	recv.SiteCode = SiteCode(cleanField(in[1:5]))
	recv.PointCode = cleanField(in[6:8])
	recv.SolID = cleanField(in[9:13])

	if techn, ok := obsTechnMap[in[14:15]]; ok {
		recv.ObsTech = techn
	} else {
		return fmt.Errorf("unknown observation code: %q", in[14:15])
	}

	if installedAt, err := parseTime(in[16:28]); err == nil {
		recv.DateInstalled = installedAt
	} else {
		return fmt.Errorf("parse DATA START %q: %v", in, err)
	}

	if remAt, err := parseTime(in[29:41]); err == nil {
		recv.DateRemoved = remAt
	} else {
		return fmt.Errorf("parse DATA_END %q: %v", in, err)
	}

	recv.Type = cleanField(in[42:62])
	recv.SerialNum = cleanField(in[63:68])
	recv.Firmware = cleanField(in[69:])
	return nil
}

// Unmarshall a SOLUTION/ESTIMATE record.
func (est *Estimate) UnmarshalSINEX(in string) error {
	var err error
	if est.Idx, err = strconv.Atoi(strings.TrimSpace(in[1:6])); err != nil {
		return fmt.Errorf("parse INDEX: %v", err)
	}

	est.ParType = ParameterType(strings.TrimSpace(in[7:13]))

	est.SiteCode = SiteCode(cleanField(in[14:18]))

	est.PointCode = cleanField(in[19:21])

	est.SolID = cleanField(in[22:26])

	if ti, err := parseTime(in[27:39]); err == nil {
		est.Epoch = ti
	} else {
		return fmt.Errorf("parse TIME %q: %v", in, err)
	}

	est.Unit = strings.TrimSpace(in[40:44])
	est.ConstraintCode = in[45:46]

	if est.Value, err = strconv.ParseFloat(strings.TrimSpace(in[47:68]), 64); err != nil {
		return fmt.Errorf("parse ESTIMATED_VALUE: %v", err)
	}

	if len(in) < 70 {
		return nil
	}
	if est.Stddev, err = strconv.ParseFloat(strings.TrimSpace(in[69:80]), 64); err != nil {
		return fmt.Errorf("parse STD_DEV: %v", err)
	}

	return nil
}

// parseTime parses a SINEX time string.
//
//	Time | YY:DDD:SSSSS. "UTC"         | I2.2,    |
//	YY = last 2 digits of the year,    | 1H:,I3.3,|
//	if YY <= 50 implies 21-st century, | 1H:,I5.5 |
//	if YY > 50 implies 20-th century,
//	DDD = 3-digit day in year
//	SSSSS = 5-digit seconds in day
func parseTime(str string) (time.Time, error) {
	if str == "00:000:00000" { //  __DATA_END__ means open end
		return time.Time{}, nil // zero time
	}

	t, err := time.Parse(dateFormat, str[:6])
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date: %q: %v", str, err)
	}

	secs, err := strconv.Atoi(str[7:12])
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time: %q: %v", str, err)
	}

	return t.Add(time.Duration(secs) * time.Second), nil
}

// Clean field values. Return an empty string instead of "----" for unknown values.
func cleanField(in string) string {
	s := strings.TrimSpace(in)
	return strings.Trim(s, "-")
}
