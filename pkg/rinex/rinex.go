// Package rinex provides functions for reading RINEX files.
package rinex

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
	"github.com/mholt/archiver/v3"
)

// The ContentType defines the content of a RINEX file.
type ContentType int

const (
	ContentTypeObs ContentType = iota + 1 // Observation content.
	ContentTypeNav                        // Navigation content.
	ContentTypeMet                        // Meteo content.
)

func (c ContentType) String() string {
	return [...]string{"", "obs", "nav", "met"}[c]
}

// The FilePeriod specifies the intended (nominal) collection period of a file.
type FilePeriod string

const (
	FilePeriodUnspecified FilePeriod = "00U"
	FilePeriod15Min       FilePeriod = "15M" // 15 minutes, usually for high-rate 1Hz files.
	FilePeriodHourly      FilePeriod = "01H"
	FilePeriodDaily       FilePeriod = "01D"
	FilePeriodYearly      FilePeriod = "01Y"
)

// Duration returns the duration of the file period.
func (p FilePeriod) Duration() time.Duration {
	switch p {
	case FilePeriodUnspecified:
		return 0
	case FilePeriod15Min:
		return 15 * time.Minute
	case FilePeriodHourly:
		return 1 * time.Hour
	case FilePeriodDaily:
		return 24 * time.Hour
	case FilePeriodYearly:
		return 365 * 24 * time.Hour
	default:
		return 0
	}
}

const (
	// epochTimeFormat is the time format for a epoch in RINEX version 3 files.
	epochTimeFormat string = "2006  1  2 15  4  5.9999999" // .99... see https://stackoverflow.com/questions/41617401/how-do-i-parse-string-timestamps-of-different-length-to-time

	// epochTimeFormatv2 is the time format for a epoch in RINEX version 2 files.
	epochTimeFormatv2 string = "06  1  2 15  4  5.9999999"

	// rnx3StartTimeFormat is the time format for the start time in RINEX3 file names.
	rnx3StartTimeFormat string = "20060021504"

	// The Date/Time format in the PGM / RUN BY / DATE header record.
	headerDateFormat string = "20060102 150405"

	// The Date/Time format with time zone in the PGM / RUN BY / DATE header record.
	//
	// Format: "yyyymmdd hhmmss zone" with 3–4 character code for the time zone.
	headerDateWithZoneFormat string = "20060102 150405 MST"

	// The RINEX-2 Date/Time format in the PGM / RUN BY / DATE header record.
	headerDateFormatv2 string = "02-Jan-06 15:04"
)

// General RINEX errors.
var (
	// ErrNoHeader is returned when reading RINEX data that does not begin with a RINEX Header.
	ErrNoHeader = errors.New("rinex: no header")

	// ErrParser is returned on any RINEX parsing error.
	ErrParser = errors.New("rinex: parse error")
)

// A RinexError holds a warning or error that may occur during the processing a RINEX file.
// RinexError implements the error interface.
type RinexError struct {
	Err error

	// The line where the error occured.
	// Line int

	// Additional information. This can have any type, mainly used struct, map or string.
	Meta any
}

// Unwrap returns the wrapped error, to allow interoperability with errors.Is(), errors.As() and errors.Unwrap()
func (e *RinexError) Unwrap() error { return e.Err }

// RinexError implements the error interface.
func (e *RinexError) Error() string {
	if e.Meta != nil {
		return fmt.Sprintf("%s: %+v", e.Err.Error(), e.Meta)
	}
	return e.Err.Error()
}

var (
	// Rnx2FileNamePattern is the regex for RINEX2 filenames.
	Rnx2FileNamePattern = regexp.MustCompile(`(?i)(([a-z0-9]{4})(\d{3})([a-x0])(\d{2})?\.(\d{2})([domnglqfphs]))\.?([a-zA-Z0-9]+)?`)

	// Rnx3FileNamePattern is the regex for RINEX3 filenames.
	Rnx3FileNamePattern = regexp.MustCompile(`(?i)((([A-Z0-9]{4})(\d)(\d)([A-Z]{3})_([RSU])_((\d{4})(\d{3})(\d{2})(\d{2}))_(\d{2}[A-Z])_?(\d{2}[CZSMHDU])?_([GREJCISM][MNO]))\.(rnx|crx))\.?([a-zA-Z0-9]+)?`)

	sysPerAbbr = map[string]gnss.System{
		"G": gnss.SysGPS,
		"R": gnss.SysGLO,
		"E": gnss.SysGAL,
		"J": gnss.SysQZSS,
		"C": gnss.SysBDS,
		"I": gnss.SysNavIC,
		"S": gnss.SysSBAS,
		"M": gnss.SysMIXED,
	}

	// periodMap helps to get the FilePeriod.
	periodMap = map[string]FilePeriod{
		"15M": FilePeriod15Min,
		"01H": FilePeriodHourly,
		"01D": FilePeriodDaily,
		"24H": FilePeriodDaily,
		"01Y": FilePeriodYearly,
	}

	// rnxTypMap maps RINEX3 data-types to RINEX2 types.
	rnxTypMap = map[string]string{"GO": "o", "RO": "o", "EO": "o", "JO": "o", "CO": "o", "IO": "o", "SO": "o", "MO": "o",
		"GN": "n", "RN": "g", "EN": "l", "JN": "q", "CN": "f", "SN": "h", "MN": "p", "MM": "m"}
)

// FilerHandler is the interface for RINEX files..
type FilerHandler interface {
	// Compress compresses the RINEX file dependend of its file type.
	//Compress() error

	// Rnx3Filename returns the filename following the RINEX3 convention.
	Rnx3Filename() (string, error)
}

/* // DataFrequency is a measurement of cycle per second, stored as an int64 micro Hertz.
// Observation interval in seconds
type DataFrequency int64

// String returns the frequency formatted as a string in Hertz.
func (f DataFrequency) String() string {
	return microAsString(int64(f)) + "Hz"
} */

// RnxFil contains fields and methods that can be used by all RINEX file types.
// Usually you won't instantiate a RnxFil directly and use NewObsFil() and NewNavFileReader() instead.
// Both ObsFil and NavFile embed RnxFil.
type RnxFil struct {
	Path string

	FourCharID     string     // The 4char ID of the file or site.
	MonumentNumber int        // The site monument number.
	ReceiverNumber int        // The site receiver number.
	CountryCode    string     // ISO 3char
	StartTime      time.Time  // StartTime is the nominal start time derived from the filename.
	DataSource     string     // [RSU]
	FilePeriod     FilePeriod // The intended collection period of the file.
	DataFreq       string     // 30S, not for nav files // TODO make type frequency!!
	DataType       string     // The data type abbreviations GO, RO, MN, MM, ...
	Format         string     // rnx, crx, etc. Attention: Format and Hatanaka are dependent!
	Compression    string     // gz, ...
	//IsHatanaka     bool   // true if file is Hatanaka compressed

	Warnings []string // List of warnings that might occur when reading the file.
}

// NewFile returns a new RINEX file object.
func NewFile(filepath string) (FilerHandler, error) {
	rnx := &RnxFil{Path: filepath}
	err := rnx.ParseFilename()
	if err != nil {
		return nil, err
	}

	var f FilerHandler
	if rnx.IsObsType() {
		f = &ObsFile{RnxFil: rnx}
	} else if rnx.IsNavType() {
		f = &NavFile{RnxFil: rnx}
	} else if rnx.IsMeteoType() {
		f = &MeteoFile{RnxFil: rnx}
	} else {
		return nil, fmt.Errorf("no valid RINEX file: %s", filepath)
	}

	return f, nil
}

// SetStationName sets the station or project name.
// IGS users should follow XXXXMRCCC (9 char) site and station naming convention described above.
// GNSS industry users could use the 9 characters to indicate the project name and/or number.
func (f *RnxFil) SetStationName(name string) error {
	if len(name) == 4 {
		f.FourCharID = strings.ToUpper(name)
	} else if len(name) == 9 {
		f.FourCharID = strings.ToUpper(name[:4])
		f.MonumentNumber, _ = strconv.Atoi(name[4:5])
		f.ReceiverNumber, _ = strconv.Atoi(name[5:6])
		f.CountryCode = strings.ToUpper(name[6:])
	} else {
		return fmt.Errorf("weird station identifier %s", name)
	}

	return nil
}

// StationName returns the long 9-char station name if possible, otherwiese - mainly for RINEX-2 files - it returns the fourCharID.
// The returned name is uppercase.
func (f *RnxFil) StationName() string {
	if f.CountryCode != "" {
		return f.FourCharID + strconv.Itoa(f.MonumentNumber) + strconv.Itoa(f.ReceiverNumber) + f.CountryCode
	}
	return f.FourCharID
}

// IsObsType returns true if the file is a RINEX observation file type.
func (f *RnxFil) IsObsType() bool {
	return strings.HasSuffix(f.DataType, "O")
}

// IsNavType returns true if the file is a RINEX navigation file type.
func (f *RnxFil) IsNavType() bool {
	return strings.HasSuffix(f.DataType, "N")
}

// IsMeteoType returns true if the file is a RINEX meteo file type.
func (f *RnxFil) IsMeteoType() bool {
	return strings.HasSuffix(f.DataType, "M")
}

// Returns true if it is a hourly file.
func (f *RnxFil) IsHourly() bool { return f.FilePeriod == FilePeriodHourly }

// Returns true if it is a daily file.
func (f *RnxFil) IsDaily() bool { return f.FilePeriod == FilePeriodDaily }

// ParseFilename parses the specified filename, which must be a valid RINEX filename,
// and fills its fields.
func (f *RnxFil) ParseFilename() error {
	if f.Path == "" {
		return fmt.Errorf("parse filename: empty path")
	}

	fn := filepath.Base(f.Path)
	if len(fn) > 20 { // Rnx3
		res := Rnx3FileNamePattern.FindStringSubmatch(fn)
		if len(res) == 0 {
			return fmt.Errorf("parse filename: name did not match: %s", fn)
		}
		for k, v := range res {
			//fmt.Printf("%d. %s\n", k, v)
			switch k {
			case 3:
				f.FourCharID = strings.ToUpper(v)
			case 4:
				i, err := strconv.Atoi(v)
				f.MonumentNumber = i
				if err != nil {
					return fmt.Errorf("parse filename: could not parse MonumentNumber: %s", v)
				}
			case 5:
				i, err := strconv.Atoi(v)
				f.ReceiverNumber = i
				if err != nil {
					return fmt.Errorf("parse filename: could not parse ReceiverNumber: %s", v)
				}
			case 6:
				f.CountryCode = strings.ToUpper(v)
			case 7:
				f.DataSource = strings.ToUpper(v)
			case 8:
				t, err := time.Parse(rnx3StartTimeFormat, v)
				if err != nil {
					return fmt.Errorf("parse filename: could not parse start time: %s: %v", v, err)
				}
				f.StartTime = t
			case 13:
				f.FilePeriod = periodMap[strings.ToUpper(v)]
			case 14:
				f.DataFreq = strings.ToUpper(v)
			case 15:
				f.DataType = strings.ToUpper(v)
			case 16:
				f.Format = strings.ToLower(v)
			case 17:
				f.Compression = v
			}
		}
	} else { // Rnx2
		res := Rnx2FileNamePattern.FindStringSubmatch(fn)
		if len(res) == 0 {
			return fmt.Errorf("parse filename: name did not match: %s", fn)
		}
		for k, v := range res {
			//fmt.Printf("%d. %s\n", k, v)
			switch k {
			case 2:
				f.FourCharID = strings.ToUpper(v)
			case 5: // highrate minutes
				if res[4] == "0" {
					f.FilePeriod = FilePeriodDaily
					f.DataFreq = "30S"
				} else {
					if v != "" {
						f.FilePeriod = FilePeriod15Min
						f.DataFreq = "01S"
					} else {
						f.FilePeriod = FilePeriodHourly
						f.DataFreq = "30S"
					}
				}
			case 6: // yr
				doy, err := time.Parse("06002", v+res[3])
				if err != nil {
					return fmt.Errorf("parse filename: could not parse DoY: %v", err)
				}
				hr, _ := getHourAsDigit(rune((res[4])[0]))
				min := 0
				if res[5] != "" && res[5] != "00" { // highrate minutes
					min, _ = strconv.Atoi(res[5])
				}
				f.StartTime = doy.Add(time.Duration(hr)*time.Hour + time.Duration(min)*time.Minute)
			case 7:
				switch strings.ToLower(v) {
				case "o":
					f.DataType = "MO"
					f.Format = "rnx"
				case "d":
					f.DataType = "MO"
					f.Format = "crx"
				case "n":
					f.DataType = "GN"
					f.Format = "rnx"
				case "g":
					f.DataType = "RN"
					f.Format = "rnx"
				case "m":
					f.DataType = "MM"
					f.Format = "rnx"
				case "f":
					f.DataType = "CN"
					f.Format = "rnx"
				case "l":
					f.DataType = "EN"
					f.Format = "rnx"
				case "q":
					f.DataType = "JN"
					f.Format = "rnx"
				case "s": // RINEX summary file
					f.DataType = "SM" // This datatype is not official!
					f.Format = "rnx"
				default:
					return fmt.Errorf("parse filename: could not determine DATA TYPE: %q", v)
				}
			case 8:
				f.Compression = v
			}
		}
	}

	return nil
}

// Rnx2Filename returns the filename following the RINEX2 convention.
func (f *RnxFil) Rnx2Filename() (string, error) {
	// Station Identifier
	if len(f.FourCharID) != 4 {
		return "", fmt.Errorf("FourCharID: %s", f.FourCharID)
	}

	var fn strings.Builder
	fn.WriteString(strings.ToLower(f.FourCharID))
	fn.WriteString(fmt.Sprintf("%03d", f.StartTime.YearDay()))
	if f.IsDaily() {
		fn.WriteString("0")
	} else {
		fn.WriteString(getHourAsChar(f.StartTime.Hour()))
	}

	if f.FilePeriod == FilePeriod15Min { // 15min highrates
		d := time.Duration(f.StartTime.Minute()) * time.Minute
		fn.WriteString(fmt.Sprintf("%02d", int(d.Truncate(15*time.Minute).Minutes())))
	}

	yyyy := strconv.Itoa(f.StartTime.Year())
	fn.WriteString("." + yyyy[2:])

	rnx2Typ, ok := rnxTypMap[f.DataType]
	if !ok {
		return "", fmt.Errorf("could not map type %s to RINEX2", f.DataType)
	}
	if f.IsObsType() && f.Format == "crx" {
		fn.WriteString("d")
	} else {
		fn.WriteString(rnx2Typ)
	}

	// Checks
	shouldLength := 12
	if f.FilePeriod == FilePeriod15Min { // 15min highrates
		shouldLength = 14
	}

	length := len(fn.String())
	if length != shouldLength {
		return "", fmt.Errorf("wrong filename length: %s: %d (should: %d)", fn.String(), length, shouldLength)
	}

	return fn.String(), nil
}

// Rnx3Filename returns the RTCM RINEX-3 compliant filename for the given RINEX-2 file.
// The countryCode must be the 3 char ISO ?? code.
// Datasource as option!?
func Rnx3Filename(rnx2filepath string, countryCode string) (string, error) {
	if len(countryCode) != 3 {
		return "", fmt.Errorf("invalid countryCode %q", countryCode)
	}
	rnx := &RnxFil{Path: rnx2filepath, CountryCode: countryCode, DataSource: "R"}
	err := rnx.ParseFilename()
	if err != nil {
		return "", err
	}

	if rnx.IsObsType() {
		f := &ObsFile{RnxFil: rnx}
		return f.Rnx3Filename()
	} else if rnx.IsNavType() {
		f := &NavFile{RnxFil: rnx}
		return f.Rnx3Filename()
	} else if rnx.IsMeteoType() {
		return "", fmt.Errorf("meteo files not implemented yet")
	}

	return "", fmt.Errorf("no valid RINEX filename: %s", rnx2filepath)
}

// Rnx2Filename returns the filename following the RINEX2 convention.
func Rnx2Filename(rnx3filepath string) (string, error) {
	rnx := &RnxFil{Path: rnx3filepath}
	err := rnx.ParseFilename()
	if err != nil {
		return "", err
	}

	return rnx.Rnx2Filename()
}

// Returns the RINEX filename with the correct case sensitivity. RINEX-3 long filenames must be uppercase
// except format, whereas RINEX-2 short names have to be lowercase.
func GetCaseSensitiveName(path string) string {
	dir := filepath.Dir(path)
	fname := filepath.Base(path)
	if len(fname) < 17 { // RINEX-2 short names
		return filepath.Join(dir, strings.ToLower(fname))
	}

	// RINEX-3
	ext := filepath.Ext(fname)
	fnameWoExt := strings.TrimSuffix(fname, ext)
	return filepath.Join(dir, strings.ToUpper(fnameWoExt)+strings.ToLower(ext))
}

// IsCompressed returns true if the src is compressed, otherwise false.
// This checks NOT for Hatanaka compression.
func IsCompressed(src string) bool {
	ext := filepath.Ext(src)
	if ext == "" {
		return false
	}

	if ext == ".z" || ext == ".Z" {
		return true
	}

	_, err := archiver.ByExtension(src)
	return err == nil
}

// ParseDoy returns the UTC-Time corresponding to the given year and day of year.
// Added in Go 1.13 !!!
func ParseDoy(year, doy int) time.Time {
	y := year
	if year > 80 && year <= 99 {
		y += 1900
	} else if year <= 80 {
		y += 2000
	}
	t := time.Date(y, 1, 0, 0, 0, 0, 0, time.UTC)
	return t.Add(time.Duration(doy) * time.Hour * 24)
}

func parseFloat(s string) (float64, error) {
	//s. bncutils::readDbl
	if strings.TrimSpace(s) == "" {
		return 0, nil
	}
	r := strings.NewReplacer("d", "e", "D", "e")
	scleaned := r.Replace(strings.TrimSpace(s))
	return strconv.ParseFloat(scleaned, 64)
}

// Parse the Date/Time in the PGM / RUN BY / DATE header record.
// It is recommended to use UTC as the time zone. Set zone to LCL if an unknown local time was used.
func parseHeaderDate(date string) (time.Time, error) {
	format := headerDateFormat
	if len(date) == 19 || len(date) == 20 {
		format = headerDateWithZoneFormat
	} else if len(date) == 15 && strings.Contains(date, "-") {
		format = headerDateFormatv2
	} else if len(date) == 18 && strings.Contains(date, "-") {
		format = "02-Jan-06 15:04:05" // unofficial!
	} else if len(date) == 17 && strings.Contains(date, "-") {
		format = "02-Jan-2006 15:04" // unofficial!
	} else if len(date) == 16 && strings.Contains(date, "-") {
		format = "2006-01-02 15:04" // unofficial!
	}

	ti, err := time.Parse(format, date)
	if err != nil {
		return time.Time{}, err
	}
	return ti, nil
}

func getHourAsChar(hr int) string {
	return string(rune(hr + 97))
}

func getHourAsDigit(char rune) (int, error) {
	hr := int(char) - int('a')
	if hr < 0 || hr > 23 {
		return 0, fmt.Errorf("could not get hour for %c", char)
	}
	return hr, nil
}

/* func parseEpochTime(str string) (time.Time, error) {
	//> 2018 11 06 19 00  0.0000000  0 31
	epTime, err := time.Parse("2006 01 02 15 04  5.0000000", str)
	if err == nil {
		return epTime, nil
	}

	// if blanks instead of zeros
	// '2019  8  6  3 44 29.0000000'
	f := strings.Fields(str)
	m, _ := strconv.Atoi(f[1])
	d, _ := strconv.Atoi(f[2])
	hr, _ := strconv.Atoi(f[3])
	min, _ := strconv.Atoi(f[4])
	newStr := fmt.Sprintf("%s %02d %02d %02d %02d %s", f[0], m, d, hr, min, f[5])
	epTime, err = time.Parse("2006 01 02 15 04 05.0000000", newStr)
	if err == nil {
		return epTime, nil
	}

	return time.Time{}, fmt.Errorf("Could not parse date from string: '%s': %v", str, err)
} */
