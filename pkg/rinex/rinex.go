// Package rinex provides functions for reading RINEX files.
package rinex

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
)

const (
	// epochTimeFormat is the time format for the epoch-time in RINEX3 files.
	epochTimeFormat string = "2006  1  2 15  4  5.0000000"

	// rnx3StartTimeFormat is the time format for the start time in RINEX3 file names.
	rnx3StartTimeFormat string = "20060021504"
)

// errors
var (
	// ErrNoHeader is returned when reading RINEX data that does not begin with a RINEX Header.
	ErrNoHeader = errors.New("RINEX: no header")
)

var (
	// Rnx2FileNamePattern is the regex for RINEX2 filenames.
	Rnx2FileNamePattern = regexp.MustCompile(`(([a-z0-9]{4})(\d{3})([a-x0])(\d{2})?\.(\d{2})([domnglqfph]))\.?([a-zA-Z0-9]+)?`)

	// Rnx3FileNamePattern is the regex for RINEX3 filenames.
	Rnx3FileNamePattern = regexp.MustCompile(`((([A-Z0-9]{4})(\d)(\d)([A-Z]{3})_([RSU])_((\d{4})(\d{3})(\d{2})(\d{2}))_(\d{2}[A-Z])_?(\d{2}[CZSMHDU])?_([GREJCSM][MNO]))\.(rnx|crx))\.?([a-zA-Z0-9]+)?`)

	sysPerAbbr = map[string]gnss.System{
		"G": gnss.SysGPS,
		"R": gnss.SysGLO,
		"E": gnss.SysGAL,
		"J": gnss.SysQZSS,
		"C": gnss.SysBDS,
		"I": gnss.SysIRNSS,
		"S": gnss.SysSBAS,
		"M": gnss.SysMIXED,
	}

	// rnxTypMap maps RINEX3 data-types to RINEX2 types.
	rnxTypMap = map[string]string{"GO": "o", "RO": "o", "EO": "o", "JO": "o", "CO": "o", "IO": "o", "SO": "o", "MO": "o",
		"GN": "n", "RN": "g", "EN": "l", "JN": "q", "CN": "f", "SN": "h", "MN": "p", "MM": "m"}
)

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

	FourCharID     string
	MonumentNumber int
	ReceiverNumber int
	CountryCode    string // ISO 3char
	StartTime      time.Time
	DataSource     string // [RSU]
	FilePeriod     string // 15M, 01D
	DataFreq       string // 30S, not for nav files // TODO make type frequency!!
	DataType       string // The data type abbreviations GO, RO, MN, MM, ...
	Format         string // rnx, crx, etc. Attention: Format and Hatanaka are dependent!
	Compression    string // gz, ...
	//IsHatanaka     bool   // true if file is Hatanaka compressed
}

// NewFile returns a new RINEX file object.
func NewFile(filepath string) (*RnxFil, error) {
	fil := &RnxFil{Path: filepath}
	err := fil.parseFilename()
	return fil, err
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

// Rnx2Filename returns the filename following the RINEX2 convention.
func (f *RnxFil) Rnx2Filename() (string, error) {
	// Station Identifier
	if len(f.FourCharID) != 4 {
		return "", fmt.Errorf("FourCharID: %s", f.FourCharID)
	}

	var fn strings.Builder
	fn.WriteString(strings.ToLower(f.FourCharID))
	fn.WriteString(fmt.Sprintf("%03d", f.StartTime.YearDay()))
	if f.FilePeriod == "01D" {
		fn.WriteString("0")
	} else {
		fn.WriteString(getHourAsChar(f.StartTime.Hour()))
	}

	if f.FilePeriod == "15M" { // 15min highrates
		d := time.Duration(f.StartTime.Minute()) * time.Minute
		fn.WriteString(fmt.Sprintf("%02d", int(d.Truncate(15*time.Minute).Minutes())))
	}

	yyyy := strconv.Itoa(f.StartTime.Year())
	fn.WriteString("." + yyyy[2:])

	rnx2Typ, ok := rnxTypMap[f.DataType]
	if !ok {
		return "", fmt.Errorf("Could not map type %s to RINEX2", f.DataType)
	}
	if f.IsObsType() && f.Format == "crx" {
		fn.WriteString("d")
	} else {
		fn.WriteString(rnx2Typ)
	}

	// Checks
	shouldLength := 12
	if f.FilePeriod == "15M" { // 15min highrates
		shouldLength = 14
	}

	length := len(fn.String())
	if length != shouldLength {
		return "", fmt.Errorf("wrong filename length: %s: %d (should: %d)", fn.String(), length, shouldLength)
	}

	return fn.String(), nil
}

// Rnx3Filename returns the filename following the RINEX3 convention.
// In most cases we must read the read the header. The countrycode must come from an external source.
// DO NOT USE! Must parse header first!
func (f *RnxFil) Rnx3Filename() (string, error) {
	if f.IsObsType() {
		if f.DataFreq == "" || f.FilePeriod == "" {
			r, err := os.Open(f.Path)
			if err != nil {
				return "", err
			}
			defer r.Close()
			dec, err := NewObsDecoder(r)
			if err != nil {
				return "", err
			}

			if dec.Header.Interval != 0 {
				f.DataFreq = fmt.Sprintf("%02d%s", int(dec.Header.Interval), "S")
			}

			f.DataType = fmt.Sprintf("%s%s", dec.Header.SatSystem.Abbr(), "O")
		}
	} else {
		return "", fmt.Errorf("nav and meteo not implemented yet")
	}

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
	//BRUX00BEL_R_20183101900_01H_30S_MO.rnx
	fn.WriteString(strconv.Itoa(f.StartTime.Year()))
	fn.WriteString(fmt.Sprintf("%03d", f.StartTime.YearDay()))
	fn.WriteString(fmt.Sprintf("%02d", f.StartTime.Hour()))
	fn.WriteString(fmt.Sprintf("%02d", f.StartTime.Minute()))
	fn.WriteString("_")

	fn.WriteString(f.FilePeriod)
	fn.WriteString("_")

	fn.WriteString(f.DataFreq)
	fn.WriteString("_")

	fn.WriteString(f.DataType)

	if f.IsObsType() && f.Format == "crx" {
		fn.WriteString(".crx")
	} else {
		fn.WriteString(".rnx")
	}

	// Checks
	length := len(fn.String())
	if f.IsObsType() {
		if length != 38 {
			return "", fmt.Errorf("wrong filename length: %s: %d", fn.String(), length)
		}
	}
	// Rnx3 Filename: total: 41-42 obs, 37-38 eph.

	return fn.String(), nil
}

// IsObsType returns true if the file is a RINEX observation file type.
func (f *RnxFil) IsObsType() bool {
	if strings.HasSuffix(f.DataType, "O") {
		return true
	}
	return false
}

// IsNavType returns true if the file is a RINEX navigation file type.
func (f *RnxFil) IsNavType() bool {
	if strings.HasSuffix(f.DataType, "N") {
		return true
	}
	return false
}

// IsMeteoType returns true if the file is a RINEX meteo file type.
func (f *RnxFil) IsMeteoType() bool {
	if strings.HasSuffix(f.DataType, "M") {
		return true
	}
	return false
}

// parseFilename parses the specified filename, which must be a valid RINEX filename,
// and fills its fields.
func (f *RnxFil) parseFilename() error {
	if f.Path == "" {
		return fmt.Errorf("could not parse filename: Path is empty")
	}

	fn := filepath.Base(f.Path)
	if len(fn) > 20 { // Rnx3
		res := Rnx3FileNamePattern.FindStringSubmatch(fn)
		for k, v := range res {
			//fmt.Printf("%d. %s\n", k, v)
			switch k {
			case 3:
				f.FourCharID = strings.ToUpper(v)
			case 4:
				i, err := strconv.Atoi(v)
				f.MonumentNumber = i
				if err != nil {
					return fmt.Errorf("could not parse MonumentNumber: %s", v)
				}
			case 5:
				i, err := strconv.Atoi(v)
				f.ReceiverNumber = i
				if err != nil {
					return fmt.Errorf("could not parse ReceiverNumber: %s", v)
				}
			case 6:
				f.CountryCode = strings.ToUpper(v)
			case 7:
				f.DataSource = strings.ToUpper(v)
			case 8:
				t, err := time.Parse(rnx3StartTimeFormat, v)
				if err != nil {
					return fmt.Errorf("could not parse start time: %s: %v", v, err)
				}
				f.StartTime = t
			case 13:
				f.FilePeriod = strings.ToUpper(v)
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
		for k, v := range res {
			//fmt.Printf("%d. %s\n", k, v)
			switch k {
			case 2:
				f.FourCharID = strings.ToUpper(v)
			case 5: // highrate minutes
				if res[4] == "0" {
					f.FilePeriod = "01D"
					f.DataFreq = "30S"
				} else {
					if v != "" {
						f.FilePeriod = "15M"
						f.DataFreq = "01S"
					} else {
						f.FilePeriod = "01H"
						f.DataFreq = "30S"
					}
				}
			case 6: // yr
				doy, err := time.Parse("06002", v+res[3])
				if err != nil {
					return fmt.Errorf("could not parse DoY: %v", err)
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
					f.Format = "rnx"
					f.DataType = "MO"
				case "d":
					f.DataType = "MO"
					f.Format = "crx"
				case "n":
					f.DataType = "GN"
					f.Format = "rnx"
				case "g":
					f.DataType = "RN"
					f.Format = "rnx"
				default:
					return fmt.Errorf("could not determine the DATA TYPE")
				}
			case 8:
				f.Compression = v
			}
		}
	}

	return nil
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
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
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
