// Package sinex for reading SINEX files.
// Format description is available at https://www.iers.org/IERS/EN/Organization/AnalysisCoordinator/SinexFormat/sinex.html.
package sinex

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/de-bkg/gognss/pkg/site"
)

const (
	// dateFormat is the date part of a SINEX time string (15:243:02013 YY-year:DOY:sec of day).
	dateFormat string = "06:002"
)

// SiteCode is the site identifier, usually the FourCharID.
type SiteCode string

// ObservationTechnique used to arrive at the solutions obtained in this SINEX file, e.g. SLR, GPS, VLBI.
// It should be consistent with the IERS convention.
type ObservationTechnique int

// Observation techniques.
const (
	ObsTechCombined ObservationTechnique = iota + 1
	ObsTechDORIS
	ObsTechSLR
	ObsTechLLR
	ObsTechGPS
	ObsTechVLBI
)

func (techn ObservationTechnique) String() string {
	return [...]string{"", "Combined", "DORIS", "SLR", "LLR", "GPS", "VLBI", ""}[techn]
}

// SINEX contains everything about a SINEX file.
type SINEX struct {
	Header *Header
	Ref    *Reference
	Sites  map[SiteCode]*site.Site
}

// Header containes the information from the SINEX Header line.
type Header struct {
	Version            string               // Format version.
	Agency             string               // Agency creating the file.
	AgencyDataProvider string               // Agency providing the data in the file.
	CreationTime       time.Time            // Creation time of the file.
	StartTime          time.Time            // Start time of the data.
	EndTime            time.Time            // End time of the data.
	ObsTech            ObservationTechnique // Technique(s) used to generate the SINEX solution.
	NumEstimates       int                  // parameters estimated
	ConstraintCode     int                  // Single digit indicating the constraints:  0-fixed/tight constraints, 1-significant constraints, 2-unconstrained.
	SolutionTypes      []string             // Solution types contained in this SINEX file. Each character in this field may be one of the following:
	/* 	S - all station parameters, i.e. station coordinates, station velocities, biases, geocenter
	    O - Orbits
		  E - Earth Orientation Parameter
		  T - Troposphere
		  C - Celestial Reference Frame
	BLANK */

	//warnings []string
}

// Reference provides information on the Organization, point of contact, the software and hardware involved in the creation of the file.
type Reference struct {
	Description string // Organization(s).
	Output      string // File contents.
	Contact     string // Contact information.
	Software    string // SW used to generate the file.
	Hardware    string // Hardware on which above software was run.
	Input       string // Input used to generate this solution.
}

// Site contains information for each site.
// TODO setup new station
type Site struct {
	PointCode string
	ID        struct {
		// *CODE PT __DOMES__ T _STATION DESCRIPTION__ _LONGITUDE_ _LATITUDE__ HEIGHT_
		// ABMF  A 97103M001 P Les Abymes - Raizet ai 298 28 20.9  16 15 44.3   -25.6

	}
	// This block provides general information for each site containing estimated parameters.

}

// Decode decodes a SINEX input stream.
//
// It is the caller's responsibility to call Close on the underlying reader when done!
func Decode(r io.Reader) (*SINEX, error) {
	snx := &SINEX{Ref: &Reference{}}
	snx.Sites = make(map[SiteCode]*site.Site, 50)
	var err error
	dec := &decoder{sc: bufio.NewScanner(r)}
	snx.Header, err = dec.readHeader()
	if err != nil {
		return nil, err
	}
	err = dec.decode(snx)
	return snx, err
}

// decoder reads and decodes the SINEX input stream.
type decoder struct {
	sc      *bufio.Scanner
	lineNum int
	err     error
}

// read the SINEX header which is only one line.
func (dec *decoder) readHeader() (*Header, error) {
	dec.readLine()
	line := dec.line()
	err := dec.sc.Err()
	if err != nil {
		return nil, err
	}

	if line[:1] != "%" {
		return nil, fmt.Errorf("read Headerline: does not start with %q", "%")
	}

	hdr := &Header{}
	hdr.Version = line[6:10]
	hdr.Agency = line[11:14]
	hdr.CreationTime, err = parseTime(line[15:27])
	if err != nil {
		return nil, err
	}
	hdr.AgencyDataProvider = line[28:31]
	hdr.StartTime, err = parseTime(line[32:44])
	if err != nil {
		return nil, err
	}

	hdr.EndTime, err = parseTime(line[45:57])
	if err != nil {
		return nil, err
	}

	switch line[58:59] {
	case "C":
		hdr.ObsTech = ObsTechCombined
	case "D":
		hdr.ObsTech = ObsTechDORIS
	case "L":
		hdr.ObsTech = ObsTechSLR
	case "M":
		hdr.ObsTech = ObsTechLLR
	case "P":
		hdr.ObsTech = ObsTechGPS
	case "R":
		hdr.ObsTech = ObsTechVLBI
	default:
		return hdr, fmt.Errorf("unknown Observation Code: %q", line[58:59])
	}

	hdr.NumEstimates, err = strconv.Atoi(strings.TrimSpace(line[60:65]))
	if err != nil {
		return nil, err
	}

	hdr.ConstraintCode, err = strconv.Atoi(line[66:67])
	if err != nil {
		return nil, err
	}

	hdr.SolutionTypes = strings.Fields(strings.TrimSpace(line[68:]))

	return hdr, nil
}

func (dec *decoder) decode(snx *SINEX) (err error) {
	block := ""
	for dec.readLine() {
		line := dec.line()
		firstChar := line[:1]
		if firstChar == "*" { // comment
			continue
		} else if firstChar == "+" { // BEGIN BLOCK
			block = strings.TrimSpace(line[1:])
			//fmt.Printf("block %q\n", block)
			continue
		} else if firstChar == "-" { // END BLOCK
			block = ""
			continue
		}

		if firstChar != " " || block == "" {
			continue
		}

		if block == "FILE/REFERENCE" {
			key := strings.TrimSpace(line[1:19])
			val := strings.TrimSpace(line[20:])

			switch key {
			case "DESCRIPTION":
				snx.Ref.Description = val
			case "OUTPUT":
				snx.Ref.Output = val
			case "CONTACT":
				snx.Ref.Contact = val
			case "SOFTWARE":
				snx.Ref.Software = val
			case "HARDWARE":
				snx.Ref.Hardware = val
			case "INPUT":
				snx.Ref.Input = val
			default:
				fmt.Fprintf(os.Stderr, "invalid FILE/REFERENCE field: %q\n", key)
			}
		} else if block == "SITE/ID" {
			s := &site.Site{}
			s.Ident.FourCharacterID = line[1:5]
			s.Ident.DOMESNumber = line[9:18]
			s.Location.City = line[21:43]
			snx.Sites[SiteCode(line[1:5])] = s

		} else if block == "SITE/RECEIVER" {
			//*CODE PT SOLN T _DATA START_ __DATA_END__ ___RECEIVER_TYPE____ _S/N_ _FIRMWARE__
			// ABMF  A ---- P 20:038:36000 00:000:00000 SEPT POLARX5         45014 5.3.2
			sitecode := SiteCode(line[1:5])
			recv := &site.Receiver{}
			recv.DateInstalled, err = parseTime(line[16:28])
			if err != nil {
				return fmt.Errorf("parse line %q: %v", line, err)
			}
			recv.DateRemoved, err = parseTime(line[29:41])
			if err != nil {
				return fmt.Errorf("parse line %q: %v", line, err)
			}
			recv.Type = strings.TrimSpace(line[42:62])
			recv.SerialNum = strings.TrimSpace(line[63:68])
			recv.Firmware = strings.TrimSpace(line[69:])
			snx.Sites[sitecode].Receivers = append(snx.Sites[sitecode].Receivers, recv)
		}

		/* 		if ( $block eq 'SITE/RECEIVER' ) {
		# ' ALME  A    1 P 15:228:00000 15:234:86370 TRIMBLE NETRS        ----- -----------'
		$TEMPLATE = "x A4 x A2 x A4 x A1 x A12 x A12 x A20 x A5 x A11";
		eval { @fields = unpack ( $TEMPLATE, $_ ) };
		if ($@) {
		    $self->_set_error( "block $block : could not unpack line: '" . $_ . "'\n$@" );
		    next;
		}
		$fields[7] =~ s/-//g;    # remove placeholder
		$fields[8] =~ s/-//g;
		_cleanFields( \@fields );
		my $tInst   = $self->_parseDate( $fields[4] );
		my $tRem    = $self->_parseDate( $fields[5] );
		my $siteCod = $fields[0];
		if ( $site_rec{$siteCod} ) { WARN "$siteCod with more lines in $block" }
		push (
		    @{ $site_rec{$siteCod} },
		    {                           # can be more than one for a site!
		       'Pointcode'     => $fields[1],
		       'SolID'         => $fields[2],
		       'ObsCod'        => $fields[3],
		       'TimeInstalled' => $tInst,
		       'TimeRemoved'   => $tRem,
		       'RecType'       => $fields[6],
		       'RecSerialNum'  => $fields[7],
		       'RecFirmware'   => $fields[8]
		    }
		); */

	}
	if err := dec.sc.Err(); err != nil {
		return fmt.Errorf("decode sinex: %v", err)
	}

	/* 	FILE/REFERENCE Block (Mandatory)
	Description:
	This block provides information on the Organization, point of contact, the
	software and hardware involved in the creation of the file. */
	return nil
}

// readLine reads the next line into buffer. It returns false if an error
// occurs or EOF was reached.
func (dec *decoder) readLine() bool {
	if ok := dec.sc.Scan(); !ok {
		return ok
	}
	dec.lineNum++
	return true
}

// line returns the current line.
func (dec *decoder) line() string {
	return dec.sc.Text()
}

// setErr records the first error encountered.
func (dec *decoder) setErr(err error) {
	if dec.err == nil || dec.err == io.EOF {
		dec.err = err
	}
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
