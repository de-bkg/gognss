package rinex

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
)

// The NavRecordType specifies the Navigation Data Record Type introduced in RINEX Vers 4.
type NavRecordType string

// The Navigation Data Record Types.
const (
	NavRecordTypeEPH NavRecordType = "EPH" // Ephemerides data including orbit, clock, biases, accuracy and status parameters.
	NavRecordTypeSTO NavRecordType = "STO" // System Time and UTC proxy offset parameters.
	NavRecordTypeEOP NavRecordType = "EOP" // Earth Orientation Parameters.
	NavRecordTypeION NavRecordType = "ION" // Global/Regional ionospheric model parameters.
)

// Eph is the interface that wraps some methods for all types of ephemeris.
type Eph interface {
	// Validate checks the ephemeris.
	Validate() error

	// Returns the ephemermis' PRN.
	GetPRN() gnss.PRN

	// Returns the ephemermis' time of clock (toc).
	GetTime() time.Time

	//unmarshal(data []byte) error
}

// NewEph returns a new ephemeris having the concrete type.
func NewEph(sys gnss.System) (Eph, error) {
	var eph Eph
	switch sys {
	case gnss.SysGPS:
		eph = &EphGPS{}
	case gnss.SysGLO:
		eph = &EphGLO{}
	case gnss.SysGAL:
		eph = &EphGAL{}
	case gnss.SysQZSS:
		eph = &EphQZSS{}
	case gnss.SysBDS:
		eph = &EphBDS{}
	case gnss.SysNavIC:
		eph = &EphNavIC{}
	case gnss.SysSBAS:
		eph = &EphSBAS{}
	default:
		return eph, fmt.Errorf("unknown satellite system: %v", sys)
	}

	return eph, nil
}

// EphGPS describes a GPS ephemeris.
type EphGPS struct {
	PRN         gnss.PRN
	MessageType string // Navigation Message Type, LNAV etc., see RINEX 4 spec.

	// Clock
	TOC            time.Time // Time of Clock, clock reference epoch
	ClockBias      float64   // sc clock bias in seconds
	ClockDrift     float64   // sec/sec
	ClockDriftRate float64   // sec/sec2

	IODE   float64 // Issue of Data, Ephemeris
	Crs    float64 // meters
	DeltaN float64 // radians/sec
	M0     float64 // radians

	Cuc   float64 // radians
	Ecc   float64 // Eccentricity
	Cus   float64 // radians
	SqrtA float64 // sqrt(m)

	Toe    float64 // time of ephemeris (sec of GPS week)
	Cic    float64 // radians
	Omega0 float64 // radians
	Cis    float64 // radians

	I0       float64 // radians
	Crc      float64 // meters
	Omega    float64 // radians
	OmegaDot float64 // radians/sec

	IDOT    float64 // radians/sec
	L2Codes float64
	ToeWeek float64 // GPS week (to go with TOE) Continuous
	L2PFlag float64

	URA    float64 // SV accuracy in meters
	Health float64 // SV health (bits 17-22 w 3 sf 1)
	TGD    float64 // seconds
	IODC   float64 // Issue of Data, clock

	Tom         float64 // transmission time of message, seconds of GPS week
	FitInterval float64 // Fit interval in hours
}

func (eph *EphGPS) GetPRN() gnss.PRN   { return eph.PRN }
func (eph *EphGPS) GetTime() time.Time { return eph.TOC }
func (EphGPS) Validate() error         { return nil }

// EphGLO describes a GLONASS ephemeris.
type EphGLO struct {
	PRN         gnss.PRN
	MessageType string // Navigation Message Type.
	TOC         time.Time
}

func (eph *EphGLO) GetPRN() gnss.PRN   { return eph.PRN }
func (eph *EphGLO) GetTime() time.Time { return eph.TOC }
func (EphGLO) Validate() error         { return nil }

// EphGAL describes a Galileo ephemeris.
type EphGAL struct {
	PRN         gnss.PRN
	MessageType string // Navigation Message Type.
	TOC         time.Time
}

func (eph *EphGAL) GetPRN() gnss.PRN   { return eph.PRN }
func (eph *EphGAL) GetTime() time.Time { return eph.TOC }
func (EphGAL) Validate() error         { return nil }

// EphQZSS describes a QZSS ephemeris.
type EphQZSS struct {
	PRN         gnss.PRN
	MessageType string // Navigation Message Type.
	TOC         time.Time
}

func (eph *EphQZSS) GetPRN() gnss.PRN   { return eph.PRN }
func (eph *EphQZSS) GetTime() time.Time { return eph.TOC }
func (EphQZSS) Validate() error         { return nil }

// EphBDS describes a chinese BDS ephemeris.
type EphBDS struct {
	PRN         gnss.PRN
	MessageType string // Navigation Message Type.
	TOC         time.Time
}

func (eph *EphBDS) GetPRN() gnss.PRN   { return eph.PRN }
func (eph *EphBDS) GetTime() time.Time { return eph.TOC }
func (EphBDS) Validate() error         { return nil }

// EphNavIC describes an indian IRNSS/NavIC ephemeris.
type EphNavIC struct {
	PRN         gnss.PRN
	MessageType string // EPH Navigation Message Type.
	TOC         time.Time
}

func (eph *EphNavIC) GetPRN() gnss.PRN   { return eph.PRN }
func (eph *EphNavIC) GetTime() time.Time { return eph.TOC }
func (EphNavIC) Validate() error         { return nil }

// EphSBAS describes a SBAS payload.
type EphSBAS struct {
	PRN         gnss.PRN
	MessageType string // EPH Navigation Message Type.
	TOC         time.Time
}

func (eph *EphSBAS) GetPRN() gnss.PRN   { return eph.PRN }
func (eph *EphSBAS) GetTime() time.Time { return eph.TOC }
func (EphSBAS) Validate() error         { return nil }

// A NavHeader containes the RINEX Navigation Header information.
// All header parameters are optional and may comprise different types of ionospheric model parameters
// and time conversion parameters.
type NavHeader struct {
	RINEXVersion float32     // RINEX Format version
	RINEXType    string      // RINEX File type. N for Nav, O for Obs
	SatSystem    gnss.System // Satellite System. System is "Mixed" if more than one.

	Pgm   string    // name of program creating this file
	RunBy string    // name of agency creating this file
	Date  time.Time // Date and time of file creation.

	DOI          string   // Digital Object Identifier (DOI) for data citation i.e. https://doi.org/<DOI-number>.
	Licenses     []string // Line(s) with the data license of use. Name of the license plus link to the specific version of the license. Using standard data license as from https://creativecommons.org/licenses/
	StationInfos []string // Line(s) with the link(s) to persistent URL with the station metadata (site log, GeodesyML, etc).

	Comments    []string // Comment lines
	MergedFiles int      // The number of files merged, if any.

	Labels []string // all Header Labels found
}

// NavStats holds some statistics about a RINEX nav file, derived from the data.
type NavStats struct {
	NumEphemeris    int          `json:"numEphemeris"`    // The number of epochs in the file.
	SatSystems      gnss.Systems `json:"systems"`         // The satellite systems contained.
	Satellites      []gnss.PRN   `json:"satellites"`      // The ephemeris' satellites.
	EarliestEphTime time.Time    `json:"earliestEphTime"` // Time of the earliest ephemeris.
	LatestEphTime   time.Time    `json:"latestEphTime"`   // Time of the latest ephemeris.
	Errors          error        `json:"err"`             // Any errors that occur e.g. at decoding.
}

// A NavFile contains fields and methods for RINEX navigation files and includes common methods for
// handling RINEX Nav files.
// It is useful e.g. for operations on the RINEX filename.
// If you do not need these file-related features, use the NavDecoder instead.
type NavFile struct {
	*RnxFil
	Header *NavHeader
	Stats  *NavStats // Some statistics.
}

// NewNavFile returns a new Navigation File object. The file must exist and the name will be parsed.
func NewNavFile(filepath string) (*NavFile, error) {
	navFil := &NavFile{RnxFil: &RnxFil{Path: filepath}}
	err := navFil.ParseFilename()
	return navFil, err
}

// Parse and return the Header lines.
func (f *NavFile) ReadHeader() (NavHeader, error) {
	r, err := os.Open(f.Path)
	if err != nil {
		return NavHeader{}, err
	}
	defer r.Close()
	dec, err := NewNavDecoder(r)
	if err != nil {
		if dec.Header.RINEXType != "" {
			f.Header = &dec.Header
		}
		return NavHeader{}, err
	}
	f.Header = &dec.Header
	return dec.Header, nil
}

// GetStats reads the file and retuns some statistics.
func (f *NavFile) GetStats() (stats NavStats, err error) {
	r, err := os.Open(f.Path)
	if err != nil {
		return
	}
	defer r.Close()
	dec, err := NewNavDecoder(r)
	if err != nil {
		return
	}
	f.Header = &dec.Header
	dec.fastMode = true

	earliestTOC, latestTOC := time.Time{}, time.Time{}
	seenSystems := make(map[gnss.System]int, 5)
	seenSatellites := make(map[gnss.PRN]int, 50)
	nEphs := 0
	for dec.NextEphemeris() {
		eph := dec.Ephemeris()
		nEphs++

		prn := eph.GetPRN()
		if _, exists := seenSystems[prn.Sys]; !exists {
			seenSystems[prn.Sys]++
		}

		if _, exists := seenSatellites[prn]; !exists {
			seenSatellites[prn]++
		}

		stats.Satellites = append(stats.Satellites, prn)

		toc := eph.GetTime()
		if earliestTOC.IsZero() || toc.Before(earliestTOC) {
			earliestTOC = toc
		}
		if latestTOC.IsZero() || toc.After(latestTOC) {
			latestTOC = toc
		}

	}
	if err := dec.Err(); err != nil {
		stats.Errors = errors.Join(stats.Errors, err)
	}

	if nEphs == 0 {
		stats.NumEphemeris = nEphs
		f.Stats = &stats
		return stats, nil
	}

	stats.NumEphemeris = nEphs
	stats.EarliestEphTime = earliestTOC
	stats.LatestEphTime = latestTOC

	stats.SatSystems = make([]gnss.System, 0, len(seenSystems))
	for sys := range seenSystems {
		stats.SatSystems = append(stats.SatSystems, sys)
	}

	stats.Satellites = make([]gnss.PRN, 0, len(seenSatellites))
	for prn := range seenSatellites {
		stats.Satellites = append(stats.Satellites, prn)
	}
	sort.Sort(gnss.ByPRN(stats.Satellites))

	f.Stats = &stats

	return stats, err
}

// Rnx3Filename returns the filename following the RINEX3 convention.
// In most cases we must read the read the header. The countrycode must come from an external source.
// DO NOT USE! Must parse header first!

// Rnx3Filename returns the filename following the RINEX3 convention.
// TODO !!!
func (f *NavFile) Rnx3Filename() (string, error) {
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
	// AREG00PER_R_20201690000_01D_MN.rnx

	fn.WriteString(strconv.Itoa(f.StartTime.Year()))
	fn.WriteString(fmt.Sprintf("%03d", f.StartTime.YearDay()))
	fn.WriteString(fmt.Sprintf("%02d", f.StartTime.Hour()))
	fn.WriteString(fmt.Sprintf("%02d", f.StartTime.Minute()))
	fn.WriteString("_")

	fn.WriteString(string(f.FilePeriod))
	fn.WriteString("_")

	fn.WriteString(f.DataType)
	fn.WriteString(".rnx")

	if len(fn.String()) != 34 {
		return "", fmt.Errorf("invalid filename: %s", fn.String())
	}

	return fn.String(), nil
}
