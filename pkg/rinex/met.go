package rinex

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/de-bkg/gognss/pkg/gnss"
)

// MeteoFile contains fields and methods for Meteo RINEX files.
type MeteoFile struct {
	*RnxFil
	Header MeteoHeader
}

// A MeteoHeader provides the RINEX Meteo Header information.
type MeteoHeader struct {
	RINEXVersion float32     // RINEX Format version
	RINEXType    string      // RINEX File type. O for Obs
	SatSystem    gnss.System // Satellite System. System is "Mixed" if more than one.

	Pgm   string // name of program creating this file
	RunBy string // name of agency creating this file
	Date  string // date and time of file creation TODO time.Time

	MarkerName, MarkerNumber, MarkerType string // antennas' marker name, *number and type

	Observer, Agency string

	Comments []string

	warnings []string
}

// NewMeteoFile returns a new RINEX Meteo file.
func NewMeteoFile(filepath string) (*MeteoFile, error) {
	met := &MeteoFile{RnxFil: &RnxFil{Path: filepath}}
	err := met.parseFilename()
	return met, err
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
// TODO !!!
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

	fn.WriteString(f.FilePeriod)
	fn.WriteString("_")

	fn.WriteString(f.DataType)
	fn.WriteString(".rnx")

	if len(fn.String()) != 38 {
		return "", fmt.Errorf("invalid filename: %s", fn.String())
	}

	return fn.String(), nil
}
