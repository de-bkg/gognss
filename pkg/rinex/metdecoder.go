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

const (
	// meteoEpochTimeFormat is the time format for a epoch in RINEX version 3 meteo files.
	meteoEpochTimeFormat string = "2006  1  2 15  4  5"

	// meteoEpochTimeFormat is the time format for a epoch in RINEX version 3 meteo files.
	meteoEpochTimeFormatv2 string = "06  1  2 15  4  5"
)

// MeteoEpoch contains a RINEX meteo epoch.
type MeteoEpoch struct {
	Time time.Time // The epoch time.
	Obs  []float64 // The observations in the same sequence as given in the header.
}

// MetDecoder reads and decodes header and data records from a RINEX Meteo input stream.
type MetDecoder struct {
	// The Header is valid after NewMetDecoder or Reader.Reset. The header must exist,
	// otherwise ErrNoHeader will be returned.
	Header  MeteoHeader
	sc      *bufio.Scanner
	epo     *MeteoEpoch // the current epoch
	lineNum int
	err     error
}

// NewMetDecoder creates a new decoder for RINEX Meteo data.
// The RINEX header will be read implicitly. The header must exist.
//
// It is the caller's responsibility to call Close on the underlying reader when done!
func NewMetDecoder(r io.Reader) (*MetDecoder, error) {
	dec := &MetDecoder{sc: bufio.NewScanner(r)}
	dec.Header, dec.err = dec.readHeader()
	return dec, dec.err
}

// readHeader reads a RINEX Observation header. If the Header does not exist,
// a ErrNoHeader error will be returned. Only maxLines header lines are read if maxLines > 0 (see epoch flags).
func (dec *MetDecoder) readHeader() (hdr MeteoHeader, err error) {
	sensPositions := []string{}
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
		//if strings.EqualFold(key, "RINEX VERSION / TYPE") {
		case "RINEX VERSION / TYPE":
			if f64, err := strconv.ParseFloat(strings.TrimSpace(val[:20]), 32); err == nil {
				hdr.RINEXVersion = float32(f64)
			} else {
				return hdr, fmt.Errorf("parse RINEX VERSION: %v", err)
			}
			hdr.RINEXType = val[20:21]
			if hdr.RINEXType != "M" {
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
		case "MARKER NAME":
			hdr.MarkerName = strings.TrimSpace(val)
		case "MARKER NUMBER":
			hdr.MarkerNumber = strings.TrimSpace(val[:20])
		case "DOI":
			hdr.DOI = strings.TrimSpace(val)
		case "LICENSE OF USE":
			hdr.Licenses = append(hdr.Licenses, strings.TrimSpace(val))
		case "STATION INFORMATION":
			hdr.StationInfos = append(hdr.StationInfos, strings.TrimSpace(val))
		case "# / TYPES OF OBSERV":
			for _, v := range strings.Fields(val[6:]) {
				hdr.ObsTypes = append(hdr.ObsTypes, MeteoObsType(v))
			}
		case "SENSOR MOD/TYPE/ACC":
			sens := &MeteoSensor{}
			sens.Model = strings.TrimSpace(val[:20])
			sens.Type = strings.TrimSpace(val[20:40])
			sens.Accuracy, err = parseFloat(strings.TrimSpace(val[40:53]))
			if err != nil {
				log.Printf("rinex met header: parse accuracy: %v", err)
			}
			sens.ObservationType = MeteoObsType(val[57:59])
			hdr.Sensors = append(hdr.Sensors, sens)
		case "SENSOR POS XYZ/H":
			// Process them at the end as they can appear before the sensor model line.
			sensPositions = append(sensPositions, val)
		case "END OF HEADER":
			break readln
		default:
			log.Printf("Header field %q not handled yet", key)
		}
	}

	// At the end store the sensor positions.
	for _, posline := range sensPositions {
		obstype := MeteoObsType(posline[57:59])
		xyz, height, err := parseSensorPosition(posline)
		if err != nil {
			return hdr, err
		}

		found := false
		for _, sensor := range hdr.Sensors {
			if sensor.ObservationType == obstype {
				sensor.Position = xyz
				sensor.Height = height
				found = true
				break
			}
		}

		if !found {
			return hdr, fmt.Errorf("position, but no model defined for %q", string(obstype))
		}
	}

	if err := dec.sc.Err(); err != nil {
		return hdr, err
	}

	return hdr, err
}

// Err returns the first non-EOF error that was encountered by the decoder.
func (dec *MetDecoder) Err() error {
	if dec.err == io.EOF {
		return nil
	}
	return dec.err
}

// setErr adds an error.
func (dec *MetDecoder) setErr(err error) {
	dec.err = errors.Join(dec.err, err)
}

// readLine reads the next line into buffer. It returns false if an error
// occurs or EOF was reached.
func (dec *MetDecoder) readLine() bool {
	if ok := dec.sc.Scan(); !ok {
		return ok
	}
	dec.lineNum++
	return true
}

// line returns the current line.
func (dec *MetDecoder) line() string {
	return dec.sc.Text()
}

// NextEpoch reads the observations for the next epoch.
// It returns false when the scan stops, either by reaching the end of the input or an error.
func (dec *MetDecoder) NextEpoch() bool {
	numObs := len(dec.Header.ObsTypes)
readln:
	for dec.readLine() {
		line := dec.line()
		if len(line) < 1 {
			continue
		}

		epoTime, err := dec.parseEpochTime(line)
		if err != nil {
			dec.setErr(fmt.Errorf("rinex meteo: line %d: %v", dec.lineNum, err))
			return false
		}

		obsList := make([]float64, 0, numObs)
		pos := 20
		if dec.Header.RINEXVersion < 3 {
			pos = 18
		}
		for iObs := 0; iObs < numObs; iObs++ {
			if iObs > 0 && iObs%8 == 0 { // read continuation line
				if ok := dec.readLine(); !ok {
					break readln
				}
				line = dec.line()
				pos = 4
			}

			if pos+7 > len(line) {
				break
			}

			obs, err := parseFloat(line[pos : pos+7])
			if err != nil {
				dec.setErr(fmt.Errorf("rinex met: line %d: %v", dec.lineNum, err))
				return false
			}
			obsList = append(obsList, obs)
			pos += 7
		}

		dec.epo = &MeteoEpoch{Time: epoTime, Obs: obsList}
		return true
	}

	if err := dec.sc.Err(); err != nil {
		dec.setErr(fmt.Errorf("rinex: read epoch: %v", err))
	}

	return false // EOF
}

// Epoch returns the most recent epoch generated by a call to NextEpoch.
func (dec *MetDecoder) Epoch() *MeteoEpoch {
	return dec.epo
}

func (dec *MetDecoder) parseEpochTime(line string) (time.Time, error) {
	if dec.Header.RINEXVersion < 3 {
		if len(line) < 18 {
			return time.Time{}, fmt.Errorf("incomplete line: %q", line)
		}
		return time.Parse(meteoEpochTimeFormatv2, line[1:18])
	}

	if len(line) < 20 {
		return time.Time{}, fmt.Errorf("incomplete line: %q", line)
	}
	return time.Parse(meteoEpochTimeFormat, line[1:20])
}
