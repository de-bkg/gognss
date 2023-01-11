package rinex

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Meteorological observation type abbreviation PR, TD, etc.
type MeteoObsType string

/* type MeteoObsType int
// Meteorological observation types.
const (
	MeteoObsTypePressure      MeteoObsType = iota + 1 // Pressure in mbar.
	MeteoObsTypeDryTemp                               // Dry temperature in deg Celsius.
	MeteoObsTypeRelHumidity                           // Relative humidity in percent.
	MeteoObsTypeWZPD                                  // Wet zenith path delay (mm) (for WVR data).
	MeteoObsTypeDZPD                                  // Dry component of zen.path delay (mm).
	MeteoObsTypeTZPD                                  // Total zenith path delay (mm).
	MeteoObsTypeWindAzimuth                           // Wind azimuth (deg) (from where the wind blows).
	MeteoObsTypeWindSpeed                             // Wind speed in m/s.
	MeteoObsTypeRainIncr                              // Rain increment (1/10 mm) (Rain accumulation since last measure).
	MeteoObsTypeHailIndicator                         // Hail indicator non-zero (Hail detected since last measurement).
)
func (typ MeteoObsType) String() string {
	return [...]string{"", "Pressure", "Temp Dry", "Humidity Rel", "ZPD Wet", "ZPD Dry", "ZPD Total", "Wind Azi", "Wind Speed", "Rain Incr", "Hail Indi"}[typ]
}
// Abbr returns the systems' abbreviation used in RINEX.
func (typ MeteoObsType) Abbr() string {
	return [...]string{"", "PR", "TD", "HR", "ZW", "ZD", "ZT", "WD", "WS", "RI", "HI"}[typ]
} */

// MeteoSensor describes a meteorological seonsor.
type MeteoSensor struct {
	Model           string       // Model (manufacturer).
	Type            string       // The type.
	Accuracy        float64      // Accuracy with same units as obs values.
	ObservationType MeteoObsType // The observation type.
	Position        Coord        // Approx. position of the sensor - Geocentric coordinates X, Y, Z (ITRF or WGS84).
	Height          float64      // Ellipsoidal height.
}

// A MeteoHeader provides the RINEX Meteo Header information.
type MeteoHeader struct {
	RINEXVersion float32 // RINEX Format version
	RINEXType    string  // RINEX File type. O for Obs

	Pgm   string    // name of program creating this file
	RunBy string    // name of agency creating this file
	Date  time.Time // Date and time of file creation.

	MarkerName, MarkerNumber string // antennas' marker name, *number and type

	DOI          string   // Digital Object Identifier (DOI) for data citation i.e. https://doi.org/<DOI-number>.
	License      string   // Line(s) with the data license of use. Name of the license plus link to the specific version of the license. Using standard data license as from https://creativecommons.org/licenses/
	StationInfos []string // Line(s) with the link(s) to persistent URL with the station metadata (site log, GeodesyML, etc).

	ObsTypes []MeteoObsType // The different observation types stored in the file.
	Sensors  []*MeteoSensor // Description of the meteo sensors.
	Comments []string
	Labels   []string // all Header Labels found.
}

// MeteoEpoch contains a RINEX meteo epoch.
type MeteoEpoch struct {
	Time time.Time // The epoch time.
	Obs  []float64 // The observations in the same sequence as given in the header.
}

const (
	// meteoEpochTimeFormat is the time format for a epoch in RINEX version 3 meteo files.
	meteoEpochTimeFormat string = "2006  1  2 15  4  5"

	// meteoEpochTimeFormat is the time format for a epoch in RINEX version 3 meteo files.
	meteoEpochTimeFormatv2 string = "06  1  2 15  4  5"
)

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
				log.Printf("header date: %q, %v", val[40:], err)
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
			hdr.License = strings.TrimSpace(val)
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
			fmt.Printf("Header field %q not handled yet\n", key)
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

	err = dec.sc.Err()
	return hdr, err
}

// Err returns the first non-EOF error that was encountered by the decoder.
func (dec *MetDecoder) Err() error {
	if dec.err == io.EOF {
		return nil
	}
	return dec.err
}

// setErr records the first error encountered.
func (dec *MetDecoder) setErr(err error) {
	if dec.err == nil || dec.err == io.EOF {
		dec.err = err
	}
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
		return time.Parse(meteoEpochTimeFormatv2, line[1:18])
	}

	return time.Parse(meteoEpochTimeFormat, line[1:20])
}

// MeteoFile contains fields and methods for RINEX Meteo files.
type MeteoFile struct {
	*RnxFil
	Header *MeteoHeader
	Stats  *MeteoStats // Some Obersavation statistics.
}

// NewMeteoFile returns a new RINEX Meteo file. The file must exist and the name will be parsed.
func NewMeteoFile(filepath string) (*MeteoFile, error) {
	met := &MeteoFile{RnxFil: &RnxFil{Path: filepath}}
	err := met.ParseFilename()
	return met, err
}

// Parse and return the Header lines.
func (f *MeteoFile) ReadHeader() (MeteoHeader, error) {
	r, err := os.Open(f.Path)
	if err != nil {
		return MeteoHeader{}, err
	}
	defer r.Close()
	dec, err := NewMetDecoder(r)
	if err != nil {
		return MeteoHeader{}, err
	}
	f.Header = &dec.Header
	return dec.Header, nil
}

// Compress a meteo file using the gzip format.
// The source file will be removed if the compression finishes without errors.
/* func (f *MeteoFile) Compress() error {
	if IsCompressed(f.Path) {
		return nil
	}

	err := archiver.CompressFile(f.Path, f.Path+".gz")
	if err != nil {
		return err
	}
	os.Remove(f.Path)
	f.Path = f.Path + ".gz"
	f.Compression = "gz"
	return nil
} */

// Rnx3Filename returns the filename following the RINEX3 convention.
func (f *MeteoFile) Rnx3Filename() (string, error) {
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

	fn.WriteString(strconv.Itoa(f.StartTime.Year()))
	fn.WriteString(fmt.Sprintf("%03d", f.StartTime.YearDay()))
	fn.WriteString(fmt.Sprintf("%02d", f.StartTime.Hour()))
	fn.WriteString(fmt.Sprintf("%02d", f.StartTime.Minute()))
	fn.WriteString("_")

	fn.WriteString(string(f.FilePeriod))
	fn.WriteString("_")

	fn.WriteString(f.DataFreq)
	fn.WriteString("_")

	fn.WriteString("MM") // f.DataType
	fn.WriteString(".rnx")

	if len(fn.String()) != 38 {
		return "", fmt.Errorf("invalid filename: %s", fn.String())
	}

	return fn.String(), nil
}

// ComputeObsStats reads the file and computes some statistics on the observations.
func (f *MeteoFile) ComputeObsStats() (stats MeteoStats, err error) {
	r, err := os.Open(f.Path)
	if err != nil {
		return
	}
	defer r.Close()
	dec, err := NewMetDecoder(r)
	if err != nil {
		return
	}
	f.Header = &dec.Header

	numOfEpochs := 0
	intervals := make([]time.Duration, 0, 10)
	var epo, epoPrev *MeteoEpoch

	for dec.NextEpoch() {
		numOfEpochs++
		epo = dec.Epoch()
		if numOfEpochs == 1 {
			stats.TimeOfFirstObs = epo.Time
		}

		if epoPrev != nil && len(intervals) <= 10 {
			intervals = append(intervals, epo.Time.Sub(epoPrev.Time))
		}
		epoPrev = epo
	}
	if err = dec.Err(); err != nil {
		return stats, err
	}

	// Sampling rate
	sort.Slice(intervals, func(i, j int) bool { return intervals[i] < intervals[j] })
	stats.Sampling = intervals[int(len(intervals)/2)]
	stats.TimeOfLastObs = epoPrev.Time
	stats.NumEpochs = numOfEpochs
	f.Stats = &stats

	return stats, err
}

// MeteoStats holds some statistics about a RINEX meteo file, derived from the data.
type MeteoStats struct {
	NumEpochs      int           `json:"numEpochs"`      // The number of epochs in the file.
	Sampling       time.Duration `json:"sampling"`       // The saampling interval derived from the data.
	TimeOfFirstObs time.Time     `json:"timeOfFirstObs"` // Time of the first observation.
	TimeOfLastObs  time.Time     `json:"timeOfLastObs"`  // Time of the last observation.
}

// Parse a header sensor position line.
func parseSensorPosition(line string) (coord Coord, height float64, err error) {
	coord.X, err = parseFloat(strings.TrimSpace(line[0:14]))
	if err != nil {
		return coord, height, fmt.Errorf("rinex met header: parse sensor position: %v", err)
	}

	coord.Y, err = parseFloat(strings.TrimSpace(line[14:28]))
	if err != nil {
		return coord, height, fmt.Errorf("rinex met header: parse sensor position: %v", err)
	}

	coord.Z, err = parseFloat(strings.TrimSpace(line[28:42]))
	if err != nil {
		return coord, height, fmt.Errorf("rinex met header: parse sensor position: %v", err)
	}

	height, err = parseFloat(strings.TrimSpace(line[44:56]))
	if err != nil {
		return coord, height, fmt.Errorf("rinex met header: parse sensor position: %v", err)
	}

	return coord, height, nil
}
