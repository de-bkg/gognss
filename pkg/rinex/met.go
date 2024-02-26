package rinex

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// MeteoObsType is a meteorological observation type abbreviation PR, TD, etc.
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

	if numOfEpochs == 0 {
		stats.NumEpochs = numOfEpochs
		f.Stats = &stats
		return stats, nil
	}

	// Sampling rate
	if len(intervals) > 1 {
		sort.Slice(intervals, func(i, j int) bool { return intervals[i] < intervals[j] })
		stats.Sampling = intervals[int(len(intervals)/2)]
	}

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
