package rinex

// Note: fmt.Scanf is pretty slow in Go!? https://github.com/golang/go/issues/12275#issuecomment-133796990
//
// TODO: read headers that may occur at any position in the file.

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
)

// The RINEX observation code that specifies frequency, signal and tracking mode like "L1C".
type ObsCode string

// Options for global settings.
type Options struct {
	SatSys string // satellite systems GRE... Why not gnss.System?
}

// DiffOptions sets options for file comparison.
type DiffOptions struct {
	SatSys      string // satellite systems GRE...
	CheckHeader bool   // also compare the RINEX header
}

// Coord defines a XYZ coordinate.
type Coord struct {
	X, Y, Z float64
}

// CoordNEU defines a North-, East-, Up-coordinate or eccentrity
type CoordNEU struct {
	N, E, Up float64
}

// Obs specifies a RINEX observation.
type Obs struct {
	Val float64 // The observation itself.
	LLI int8    // LLI is the loss of lock indicator.
	SNR int8    // SNR is the signal-to-noise ratio.
}

// PRN specifies a GNSS satellite.
type PRN struct {
	Sys gnss.System // The satellite system.
	Num int8        // The satellite number.
	// flags
}

// newPRN returns a new PRN for the string prn that is e.g. G12.
func newPRN(prn string) (PRN, error) {
	sys, ok := sysPerAbbr[prn[:1]]
	if !ok {
		return PRN{}, fmt.Errorf("invalid satellite system: %q", prn)
	}
	snum, err := strconv.Atoi(strings.TrimSpace(prn[1:3]))
	if err != nil {
		return PRN{}, fmt.Errorf("parse sat num: %q: %v", prn, err)
	}
	if snum < 1 {
		return PRN{}, fmt.Errorf("check satellite number '%v%d'", sys, snum)
	}
	return PRN{Sys: sys, Num: int8(snum)}, nil
}

// String is a PRN Stringer.
func (prn PRN) String() string {
	return fmt.Sprintf("%s%02d", prn.Sys.Abbr(), prn.Num)
}

// ByPRN implements sort.Interface based on the PRN.
type ByPRN []PRN

func (p ByPRN) Len() int {
	return len(p)
}
func (p ByPRN) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p ByPRN) Less(i, j int) bool {
	return p[i].String() < p[j].String()
}

// SatObs contains all observations for a satellite per epoch.
type SatObs struct {
	Prn  PRN             // The satellite number or PRN.
	Obss map[ObsCode]Obs // A map of observations with the obs-code as key. L1C: Obs{Val:0, LLI:0, SNR:0}, L2C: Obs{Val:...},...
}

// SyncEpochs contains two epochs from different files with the same timestamp.
type SyncEpochs struct {
	Epo1 *Epoch
	Epo2 *Epoch
}

// Epoch contains a RINEX obs data epoch.
type Epoch struct {
	Time    time.Time // The epoch time.
	Flag    int8      // The epoch flag 0:OK, 1:power failure between previous and current epoch, >1 : Special event.
	NumSat  uint8     // The number of satellites per epoch.
	ObsList []SatObs  // The list of observations per epoch.
	//Error   error // e.g. parse error
}

// Print pretty prints the epoch.
func (epo *Epoch) Print() {
	//fmt.Printf("%+v\n", epo)
	fmt.Printf("%s Flag: %d #prn: %d\n", epo.Time.Format(time.RFC3339Nano), epo.Flag, epo.NumSat)
	for _, satObs := range epo.ObsList {
		fmt.Printf("%v -------------------------------------\n", satObs.Prn)
		for code, obs := range satObs.Obss {
			fmt.Printf("%s: %+v\n", code, obs)
		}
	}
}

// PrintTab prints the epoch in a tabular format.
func (epo *Epoch) PrintTab(opts Options) {
	for _, obsPerSat := range epo.ObsList {
		printSys := false
		for _, useSys := range opts.SatSys {
			if obsPerSat.Prn.Sys.Abbr() == string(useSys) {
				printSys = true
				break
			}
		}

		if !printSys {
			continue
		}

		fmt.Printf("%s %v ", epo.Time.Format(time.RFC3339Nano), obsPerSat.Prn)
		for _, obs := range obsPerSat.Obss {
			fmt.Printf("%14.03f ", obs.Val)
		}
		fmt.Printf("\n")
	}
}

// ObsStats holds some statistics about a RINEX obs file, derived from the data.
type ObsStats struct {
	NumEpochs      int                     `json:"numEpochs"`      // The number of epochs in the file.
	NumSatellites  int                     `json:"numSatellites"`  // The number of satellites derived from the header.
	Sampling       time.Duration           `json:"sampling"`       // The sampling interval derived from the data.
	TimeOfFirstObs time.Time               `json:"timeOfFirstObs"` // Time of the first observation.
	TimeOfLastObs  time.Time               `json:"timeOfLastObs"`  // Time of the last observation.
	ObsPerSat      map[PRN]map[ObsCode]int `json:"obsstats"`       // Number of observations per PRN and observation-type.
}

// A ObsHeader provides the RINEX Observation Header information.
type ObsHeader struct {
	RINEXVersion float32 // RINEX Format version
	RINEXType    string  // RINEX File type. O for Obs
	// The header satellite system. Note that system is "Mixed" if more than one. Use SatSystems() to get a list of all used systems.
	SatSystem gnss.System

	Pgm   string    // name of program creating this file
	RunBy string    // name of agency creating this file
	Date  time.Time // Date and time of file creation.

	Comments []string // * comment lines

	MarkerName   string // The name of the antenna marker, usually the 9-character station ID.
	MarkerNumber string // The IERS DOMES number assigned to the station marker is expected.
	MarkerType   string // Type of the marker. See RINEX specification. // TODO: make list.

	Observer, Agency string

	ReceiverNumber, ReceiverType, ReceiverVersion string
	AntennaNumber, AntennaType                    string

	Position     Coord    // Geocentric approximate marker position [m]
	AntennaDelta CoordNEU // North,East,Up deltas in [m]

	DOI          string   // Digital Object Identifier (DOI) for data citation i.e. https://doi.org/<DOI-number>.
	License      string   // Line(s) with the data license of use. Name of the license plus link to the specific version of the license. Using standard data license as from https://creativecommons.org/licenses/
	StationInfos []string // Line(s) with the link(s) to persistent URL with the station metadata (site log, GeodesyML, etc).

	ObsTypes map[gnss.System][]ObsCode // List of all observation types per GNSS.

	SignalStrengthUnit string
	Interval           float64 // Observation interval in seconds
	TimeOfFirstObs     time.Time
	TimeOfLastObs      time.Time
	GloSlots           map[PRN]int // GLONASS slot and frequency numbers.
	LeapSeconds        int         // The current number of leap seconds
	NSatellites        int         // Number of satellites, for which observations are stored in the file

	Labels []string // all Header Labels found.
}

// SatSystems returns all used satellite systems. The header must have been read before.
// For RINEX-2 files use SatSystem().
func (hdr *ObsHeader) SatSystems() []gnss.System {
	if hdr.ObsTypes == nil {
		return []gnss.System{}
	}
	sysList := make([]gnss.System, 0, len(hdr.ObsTypes))
	for sys := range hdr.ObsTypes {
		sysList = append(sysList, sys)
	}
	return sysList
}

// ObsDecoder reads and decodes header and data records from a RINEX Obs input stream.
type ObsDecoder struct {
	// The Header is valid after NewObsDecoder or Reader.Reset. The header must exist,
	// otherwise ErrNoHeader will be returned.
	Header  ObsHeader
	sc      *bufio.Scanner
	epo     *Epoch // the current epoch
	syncEpo *Epoch // the snchronized epoch from a second decoder
	lineNum int
	err     error
}

// NewObsDecoder creates a new decoder for RINEX Observation data.
// The RINEX header will be read implicitly. The header must exist.
//
// It is the caller's responsibility to call Close on the underlying reader when done!
func NewObsDecoder(r io.Reader) (*ObsDecoder, error) {
	dec := &ObsDecoder{sc: bufio.NewScanner(r)}
	dec.Header, dec.err = dec.readHeader(0)
	return dec, dec.err
}

// Err returns the first non-EOF error that was encountered by the decoder.
func (dec *ObsDecoder) Err() error {
	if dec.err == io.EOF {
		return nil
	}
	return dec.err
}

// readHeader reads a RINEX Observation header. If the Header does not exist,
// a ErrNoHeader error will be returned. Only maxLines header lines are read if maxLines > 0 (see epoch flags).
func (dec *ObsDecoder) readHeader(maxLines int) (hdr ObsHeader, err error) {
	hdr.ObsTypes = map[gnss.System][]ObsCode{} // TODO check Header reread for new ObsTypes
	var rememberSys gnss.System
	if maxLines == 0 {
		maxLines = 900
	}
readln:
	for dec.readLine() {
		line := dec.line()

		if dec.lineNum == 1 {
			if !strings.Contains(line, "RINEX VERS") { // "CRINEX VERS   / TYPE" or "RINEX VERSION / TYPE"
				err = ErrNoHeader
				return
			}
		}

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
			hdr.RINEXType = strings.TrimSpace(val[20:21])
			if sys, ok := sysPerAbbr[strings.TrimSpace(val[40:41])]; ok {
				hdr.SatSystem = sys
			} else {
				err = fmt.Errorf("read header: invalid satellite system in line %d: %s", dec.lineNum, line)
				return
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
		case "MARKER TYPE":
			hdr.MarkerType = strings.TrimSpace(val[20:40])
		case "OBSERVER / AGENCY":
			hdr.Observer = strings.TrimSpace(val[:20])
			hdr.Agency = strings.TrimSpace(val[20:])
		case "REC # / TYPE / VERS":
			hdr.ReceiverNumber = strings.TrimSpace(val[:20])
			hdr.ReceiverType = strings.TrimSpace(val[20:40])
			hdr.ReceiverVersion = strings.TrimSpace(val[40:])
		case "ANT # / TYPE":
			hdr.AntennaNumber = strings.TrimSpace(val[:20])
			hdr.AntennaType = strings.TrimSpace(val[20:40])
		case "APPROX POSITION XYZ":
			pos := strings.Fields(val)
			if len(pos) != 3 {
				return hdr, fmt.Errorf("parse approx. position from line: %s", line)
			}
			if f64, err := strconv.ParseFloat(pos[0], 64); err == nil {
				hdr.Position.X = f64
			}
			if f64, err := strconv.ParseFloat(pos[1], 64); err == nil {
				hdr.Position.Y = f64
			}
			if f64, err := strconv.ParseFloat(pos[2], 64); err == nil {
				hdr.Position.Z = f64
			}
		case "ANTENNA: DELTA H/E/N":
			ecc := strings.Fields(val)
			if len(ecc) != 3 {
				return hdr, fmt.Errorf("parse antenna deltas from line: %s", line)
			}
			if f64, err := strconv.ParseFloat(ecc[0], 64); err == nil {
				hdr.AntennaDelta.Up = f64
			}
			if f64, err := strconv.ParseFloat(ecc[1], 64); err == nil {
				hdr.AntennaDelta.E = f64
			}
			if f64, err := strconv.ParseFloat(ecc[2], 64); err == nil {
				hdr.AntennaDelta.N = f64
			}
		case "WAVELENGTH FACT L1/2": // optional (RINEX-2 only)
		case "SYS / # / OBS TYPES":
			var sys gnss.System
			if val[:1] == " " { // line continued
				sys = rememberSys
			} else {
				ok := false
				if sys, ok = sysPerAbbr[val[:1]]; !ok {
					err = fmt.Errorf("read header: invalid satellite system: %q: line %d", val[:1], dec.lineNum)
					return
				}
				rememberSys = sys
				nTypes, err := strconv.Atoi(strings.TrimSpace(val[3:6]))
				if err != nil {
					return hdr, fmt.Errorf("parse %q: %v", key, err)
				}
				hdr.ObsTypes[sys] = make([]ObsCode, 0, nTypes)
			}
			obscodes := convStringsToObscodes(strings.Fields(val[7:]))
			hdr.ObsTypes[sys] = append(hdr.ObsTypes[sys], obscodes...)
		case "# / TYPES OF OBSERV": // RINEX-2
			sys := hdr.SatSystem
			if strings.TrimSpace(val[:6]) != "" { // number of obs types
				nTypes, err := strconv.Atoi(strings.TrimSpace(val[:6]))
				if err != nil {
					return hdr, fmt.Errorf("parse %q: %v", key, err)
				}
				hdr.ObsTypes[sys] = make([]ObsCode, 0, nTypes)
			}
			obscodes := convStringsToObscodes(strings.Fields(val[7:]))
			hdr.ObsTypes[sys] = append(hdr.ObsTypes[sys], obscodes...)
		case "SIGNAL STRENGTH UNIT":
			hdr.SignalStrengthUnit = strings.TrimSpace(val[:20])
		case "INTERVAL":
			if f64, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
				hdr.Interval = f64
			}
		case "TIME OF FIRST OBS":
			t, err := time.Parse(epochTimeFormat, strings.TrimSpace(val[:43]))
			if err != nil {
				return hdr, fmt.Errorf("parse %q: %v", key, err)
			}
			hdr.TimeOfFirstObs = t
		case "TIME OF LAST OBS":
			t, err := time.Parse(epochTimeFormat, strings.TrimSpace(val[:43]))
			if err != nil {
				return hdr, fmt.Errorf("parse %q: %v", key, err)
			}
			hdr.TimeOfLastObs = t
		case "RCV CLOCK OFFS APPL": // TODO implement (field is optional)
		case "SYS / PHASE SHIFT": // optional. This header line is strongly deprecated and should be ignored by decoders.
		case "SYS / PHASE SHIFTS": // Rnx 3.01
		case "GLONASS SLOT / FRQ #":
			if strings.TrimSpace(val[:3]) != "" { // number of satellites
				nSat, err := strconv.Atoi(strings.TrimSpace(val[:3]))
				if err != nil {
					return hdr, fmt.Errorf("parse %q: %v", key, err)
				}
				hdr.GloSlots = make(map[PRN]int, nSat)
			}
			fields := strings.Fields(val[4:])
			for i := 0; i < len(fields)-1; i++ {
				prn, err := newPRN(fields[i])
				if err != nil {
					return hdr, fmt.Errorf("parse %q: %v", key, err)
				}
				frq, err := strconv.Atoi(fields[i+1])
				if err != nil {
					return hdr, fmt.Errorf("parse %q: %v", key, err)
				}
				hdr.GloSlots[prn] = frq
				i++
			}
		case "GLONASS COD/PHS/BIS": // optional. This header line is strongly deprecated and should be ignored by decoders.
		case "LEAP SECONDS": // optional. not complete! TODO: extend
			i, err := strconv.Atoi(strings.TrimSpace(val[:6]))
			if err != nil {
				return hdr, fmt.Errorf("parse %q: %v", key, err)
			}
			hdr.LeapSeconds = i
		case "# OF SATELLITES": // optional
			i, err := strconv.Atoi(strings.TrimSpace(val[:6]))
			if err != nil {
				return hdr, fmt.Errorf("parse %q: %v", key, err)
			}
			hdr.NSatellites = i
		case "PRN / # OF OBS": // optional
			// TODO
		case "END OF HEADER":
			break readln
		default:
			log.Printf("Header field %q not handled yet", key)
		}

		if maxLines > 0 && dec.lineNum == maxLines {
			break readln
		}
	}

	if hdr.RINEXVersion == 0 {
		return hdr, fmt.Errorf("unknown RINEX Version")
	}

	if err = dec.sc.Err(); err != nil {
		return hdr, err
	}

	return hdr, err
}

// NextEpoch reads the observations for the next epoch.
// It returns false when the scan stops, either by reaching the end of the input or an error.
// TODO: add phase shifts
func (dec *ObsDecoder) NextEpoch() bool {
	if dec.Header.RINEXVersion < 3 {
		return dec.nextEpochv2()
	}
	return dec.nextEpoch()
}

// Read RINEX version 2 obs file.
func (dec *ObsDecoder) nextEpochv2() bool {
readln:
	for dec.readLine() {
		line := dec.line()
		if len(line) < 1 {
			continue
		}

		epoFlag, err := strconv.Atoi(line[28:29])
		if err != nil {
			dec.setErr(fmt.Errorf("rinex2: parse epoch flag in line %d: %q", dec.lineNum, line))
			return false
		}

		// flag == 2: start moving antenna - no action
		if epoFlag >= 3 {
			numSpecialRecords, err := strconv.Atoi(strings.TrimSpace(line[29:32]))
			if err != nil {
				dec.setErr(fmt.Errorf("rinex: line %d: %v", dec.lineNum, err))
				return false
			}
			if epoFlag == 3 || epoFlag == 4 {
				// TODO reread header
				// dec.readHeader(numSpecialRecords)
				for ii := 1; ii <= numSpecialRecords; ii++ {
					if ok := dec.readLine(); !ok {
						break readln
					}
				}
			} else {
				for ii := 1; ii <= numSpecialRecords; ii++ {
					if ok := dec.readLine(); !ok {
						break readln
					}
				}
			}
			continue
		}

		epoTime, err := time.Parse(epochTimeFormatv2, line[1:26])
		if err != nil {
			dec.setErr(fmt.Errorf("rinex2: line %d: %v", dec.lineNum, err))
			return false
		}

		// Number of satellites
		numSat, err := strconv.Atoi(strings.TrimSpace(line[29:32]))
		if err != nil {
			dec.setErr(fmt.Errorf("rinex2: line %d: %v", dec.lineNum, err))
			return false
		}

		// Read list of PRNs
		pos := 32
		sats := make([]PRN, 0, numSat)
		for iSat := 0; iSat < numSat; iSat++ {
			if iSat > 0 && iSat%12 == 0 {
				if ok := dec.readLine(); !ok {
					break readln
				}
				line = dec.line()
				pos = 32
			}

			// sysShort := string(line[pos])
			// if sysShort == " " {
			// 	sysShort = "G"
			// }

			prn, err := newPRN(line[pos : pos+3])
			if err != nil {
				dec.setErr(fmt.Errorf("rinex2: new PRN in line %d: %q: %v", dec.lineNum, line, err))
				return false
			}
			sats = append(sats, prn)
			pos += 3
		}

		dec.epo = &Epoch{Time: epoTime, Flag: int8(epoFlag), NumSat: uint8(numSat),
			ObsList: make([]SatObs, 0, numSat)}

		// Read observations
		obsTypes := dec.Header.ObsTypes[dec.Header.SatSystem]
		for _, prn := range sats {
			if ok := dec.readLine(); !ok {
				break readln
			}
			line = dec.line()
			linelen := len(line)

			obsPerTyp := make(map[ObsCode]Obs, len(obsTypes))
			pos := 0
			for ityp, typ := range obsTypes {
				if ityp > 0 && ityp%5 == 0 {
					if ok := dec.readLine(); !ok {
						break readln
					}
					line = dec.line()
					linelen = len(line)
					pos = 0
				}
				if pos >= linelen {
					obsPerTyp[typ] = Obs{}
					continue
				}
				end := pos + 16
				if end > linelen {
					end = linelen
				}
				obs, err := decodeObs(line[pos:end], epoFlag)
				if err != nil {
					dec.setErr(fmt.Errorf("rinex2: parse %s observation in line %d: %q: %v", typ, dec.lineNum, line, err))
					return false
				}
				obsPerTyp[typ] = obs
				pos += 16
			}
			dec.epo.ObsList = append(dec.epo.ObsList, SatObs{Prn: prn, Obss: obsPerTyp})
		}
		return true
	}

	if err := dec.sc.Err(); err != nil {
		dec.setErr(fmt.Errorf("rinex2: read epochs: %v", err))
	}

	return false // EOF
}

func (dec *ObsDecoder) nextEpoch() bool {
readln:
	for dec.readLine() {
		line := dec.line()
		if len(line) < 1 {
			continue
		}

		if !strings.HasPrefix(line, "> ") {
			log.Printf("rinex: stream does not start with epoch line: %q", line) // must not be an error
			continue
		}

		epoFlag, err := strconv.Atoi(line[31:32])
		if err != nil {
			dec.setErr(fmt.Errorf("rinex: parse epoch flag in line %d: %q: %v", dec.lineNum, line, err))
			return false
		}

		// flag == 2: start moving antenna - no action
		if epoFlag >= 3 {
			numSpecialRecords, err := strconv.Atoi(strings.TrimSpace(line[32:35]))
			if err != nil {
				dec.setErr(fmt.Errorf("rinex: line %d: %v", dec.lineNum, err))
				return false
			}
			if epoFlag == 3 || epoFlag == 4 {
				// TODO reread header
				// dec.readHeader(numSpecialRecords)
				for ii := 1; ii <= numSpecialRecords; ii++ {
					if ok := dec.readLine(); !ok {
						break readln
					}
				}
			} else {
				for ii := 1; ii <= numSpecialRecords; ii++ {
					if ok := dec.readLine(); !ok {
						break readln
					}
				}
			}
			continue
		}

		epoTime, err := time.Parse(epochTimeFormat, line[2:29])
		if err != nil {
			dec.setErr(fmt.Errorf("rinex: line %d: %v", dec.lineNum, err))
			return false
		}

		numSat, err := strconv.Atoi(strings.TrimSpace(line[32:35]))
		if err != nil {
			dec.setErr(fmt.Errorf("rinex: line %d: %v", dec.lineNum, err))
			return false
		}

		dec.epo = &Epoch{Time: epoTime, Flag: int8(epoFlag), NumSat: uint8(numSat),
			ObsList: make([]SatObs, 0, numSat)}

		// Read observations
		for ii := 1; ii <= numSat; ii++ {
			if ok := dec.readLine(); !ok {
				break readln
			}
			line = dec.line()
			linelen := len(line)

			prn, err := newPRN(line[0:3])
			if err != nil {
				dec.setErr(fmt.Errorf("rinex: parse sat num in line %d: %q: %v", dec.lineNum, line, err))
				return false
			}

			if strings.TrimSpace(line[3:]) == "" { // ??
				continue
			}

			sys := sysPerAbbr[line[:1]]
			ntypes := len(dec.Header.ObsTypes[sys])
			obsPerTyp := make(map[ObsCode]Obs, ntypes)
			for ityp, typ := range dec.Header.ObsTypes[sys] {
				pos := 3 + 16*ityp
				if pos >= linelen {
					obsPerTyp[typ] = Obs{}
					continue
				}
				end := pos + 16
				if end > linelen {
					end = linelen
				}
				obs, err := decodeObs(line[pos:end], epoFlag)
				if err != nil {
					dec.setErr(fmt.Errorf("rinex: parse %s observation in line %d: %q: %v", typ, dec.lineNum, line, err))
					return false
				}
				obsPerTyp[typ] = obs
			}
			dec.epo.ObsList = append(dec.epo.ObsList, SatObs{Prn: prn, Obss: obsPerTyp})
		}
		return true
	}

	if err := dec.sc.Err(); err != nil {
		dec.setErr(fmt.Errorf("rinex: read epochs: %v", err))
	}

	return false // EOF
}

// Epoch returns the most recent epoch generated by a call to NextEpoch.
func (dec *ObsDecoder) Epoch() *Epoch {
	return dec.epo
}

// SyncEpoch returns the current pair of time-synchronized epochs from two RINEX Obs input streams.
func (dec *ObsDecoder) SyncEpoch() SyncEpochs {
	return SyncEpochs{dec.epo, dec.syncEpo}
}

// setErr adds an error.
func (dec *ObsDecoder) setErr(err error) {
	dec.err = errors.Join(dec.err, err)
}

// sync returns a stream of time-synchronized epochs from two RINEX Obs input streams.
func (dec *ObsDecoder) sync(dec2 *ObsDecoder) bool {
	var epoF1, epoF2 *Epoch
	for dec.NextEpoch() {
		epoF1 = dec.Epoch()
		//fmt.Printf("%s: got f1\n", epoF1.Time)

		if epoF2 != nil {
			if epoF1.Time.Equal(epoF2.Time) {
				dec.syncEpo = epoF2
				return true
			} else if epoF2.Time.After(epoF1.Time) {
				continue // next epo1 needed
			}
		}

		// now we need the next epo2
		for dec2.NextEpoch() {
			epoF2 = dec2.Epoch()
			//fmt.Printf("%s: got f2\n", epoF2.Time)
			if epoF2.Time.Equal(epoF1.Time) {
				dec.syncEpo = epoF2
				return true
			} else if epoF2.Time.After(epoF1.Time) {
				break // next epo1 needed
			}
		}
	}

	if err := dec2.Err(); err != nil {
		dec.setErr(fmt.Errorf("stream2 decoder error: %v", err))
	}
	return false
}

// readLine reads the next line into buffer. It returns false if an error
// occurs or EOF was reached.
func (dec *ObsDecoder) readLine() bool {
	if ok := dec.sc.Scan(); !ok {
		return ok
	}
	dec.lineNum++
	return true
}

// line returns the current line.
func (dec *ObsDecoder) line() string {
	return dec.sc.Text()
}

// decode an observation of a GNSS obs file.
func decodeObs(s string, flag int) (obs Obs, err error) {
	val := 0.0
	lli := 0
	snr := 0

	if strings.TrimSpace(s) == "" {
		return obs, err
	}

	// Value
	valStr := strings.TrimSpace(s[:14])
	if valStr != "" {
		val, err = strconv.ParseFloat(valStr, 64)
		if err != nil {
			return obs, fmt.Errorf("parse obs: %q: %v", s, err)
		}
	}
	obs.Val = val

	// LLI
	if len(s) > 14 {
		if s[14:15] != " " {
			lli, err = strconv.Atoi(s[14:15])
			if err != nil {
				return obs, fmt.Errorf("parse LLI: %q: %v", s, err)
			}
		}
	}
	// flag power failure
	if flag == 1 {
		lli |= 1
	}
	obs.LLI = int8(lli)

	// SNR
	if len(s) > 15 {
		if s[15:16] != " " {
			snr, err = strconv.Atoi(s[15:16])
			if err != nil {
				return obs, fmt.Errorf("parse LLI: %q: %v", s, err)
			}
		}
	}
	obs.SNR = int8(snr)
	return obs, err
}

// ObsFile contains fields and methods for RINEX observation files.
// Use NewObsFil() to instantiate a new ObsFile.
type ObsFile struct {
	*RnxFil
	Header *ObsHeader
	Opts   *Options
	Stats  *ObsStats // Some Obersavation statistics.
}

// NewObsFile returns a new ObsFile. The file must exist and the name will be parsed.
func NewObsFile(filepath string) (*ObsFile, error) {
	// must file exist?
	obsFil := &ObsFile{RnxFil: &RnxFil{Path: filepath}, Header: &ObsHeader{}, Opts: &Options{}}
	err := obsFil.ParseFilename()
	return obsFil, err
}

// Parse and return the Header lines.
func (f *ObsFile) ReadHeader() (ObsHeader, error) {
	r, err := os.Open(f.Path)
	if err != nil {
		return ObsHeader{}, err
	}
	defer r.Close()
	dec, err := NewObsDecoder(r)
	if err != nil {
		return ObsHeader{}, err
	}
	f.Header = &dec.Header
	return dec.Header, nil
}

// Diff compares two RINEX obs files.
func (f *ObsFile) Diff(obsFil2 *ObsFile) error {
	// file 1
	r, err := os.Open(f.Path)
	if err != nil {
		return fmt.Errorf("open obs file: %v", err)
	}
	defer r.Close()
	dec, err := NewObsDecoder(r)
	if err != nil {
		return err
	}

	// file 2
	r2, err := os.Open(obsFil2.Path)
	if err != nil {
		return fmt.Errorf("open obs file: %v", err)
	}
	defer r2.Close()
	dec2, err := NewObsDecoder(r2)
	if err != nil {
		return err
	}

	nSyncEpochs := 0
	for dec.sync(dec2) {
		nSyncEpochs++
		syncEpo := dec.SyncEpoch()

		diff := diffEpo(syncEpo, *f.Opts)
		if diff != "" {
			log.Printf("diff: %s", diff)
		}
	}
	if err := dec.Err(); err != nil {
		return fmt.Errorf("read epochs error: %v", err)
	}

	return nil
}

// ComputeObsStats reads the file and computes some statistics on the observations.
func (f *ObsFile) ComputeObsStats() (stats ObsStats, err error) {
	r, err := os.Open(f.Path)
	if err != nil {
		return
	}
	defer r.Close()
	dec, err := NewObsDecoder(r)
	if err != nil {
		if dec.Header.RINEXType != "" {
			f.Header = &dec.Header
		}
		return
	}
	f.Header = &dec.Header

	numSat := 60
	if f.Header.NSatellites > 0 {
		numSat = f.Header.NSatellites
	}

	satmap := make(map[string]int, numSat)

	obsstats := make(map[PRN]map[ObsCode]int, numSat)
	numOfEpochs := 0
	intervals := make([]time.Duration, 0, 10)
	var epo, epoPrev *Epoch

	for dec.NextEpoch() {
		numOfEpochs++
		epo = dec.Epoch()
		if numOfEpochs == 1 {
			stats.TimeOfFirstObs = epo.Time
		}

		for _, obsPerSat := range epo.ObsList {
			prn := obsPerSat.Prn

			// list of all satellites
			if _, exists := satmap[prn.String()]; !exists {
				satmap[prn.String()] = 1
			}

			// number of observations per sat and obs-type
			for obstype, obs := range obsPerSat.Obss {
				if _, exists := obsstats[prn]; !exists {
					obsstats[prn] = map[ObsCode]int{}
				}
				if _, exists := obsstats[prn][obstype]; !exists {
					obsstats[prn][obstype] = 0
				}
				if obs.Val != 0 {
					obsstats[prn][obstype]++
				}
			}
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

	stats.TimeOfLastObs = epoPrev.Time
	stats.NumEpochs = numOfEpochs
	stats.NumSatellites = len(satmap)
	stats.ObsPerSat = obsstats

	// Some checks (TODO make a separate function for checks)
	// Check observation types, see #637
	if types, exists := f.Header.ObsTypes[gnss.SysGPS]; exists {
		for _, typ := range types {
			if typ == "L2P" || typ == "C2P" {
				f.Warnings = append(f.Warnings, "observation types 'L2P' and 'C2P' are not reasonable for GPS")
				break
			}
		}
	}

	// Sampling rate
	if len(intervals) > 1 {
		sort.Slice(intervals, func(i, j int) bool { return intervals[i] < intervals[j] })
		stats.Sampling = intervals[int(len(intervals)/2)]
	}

	// LLIs

	f.Stats = &stats

	return stats, err
}

// Rnx3Filename returns the filename following the RINEX3 convention.
// In most cases we must read the read the header. The countrycode must come from an external source.
func (f *ObsFile) Rnx3Filename() (string, error) {
	if f.DataFreq == "" || f.FilePeriod == "" {
		// Parse header first.
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

	fn.WriteString(string(f.FilePeriod))
	fn.WriteString("_")

	fn.WriteString(f.DataFreq)
	fn.WriteString("_")

	fn.WriteString(f.DataType)

	if f.Format == "crx" {
		fn.WriteString(".crx")
	} else {
		fn.WriteString(".rnx")
	}

	if len(fn.String()) != 38 {
		return "", fmt.Errorf("invalid filename: %s", fn.String())
	}

	// Rnx3 Filename: total: 41-42 obs, 37-38 eph.

	return fn.String(), nil
}

// Compress an observation file using Hatanaka first and then gzip.
// The source file will be removed if the compression finishes without errors.
/* func (f *ObsFile) Compress() error {
	if f.Format == "crx" && f.Compression == "gz" {
		return nil
	}
	if f.Format == "rnx" && f.Compression != "" {
		return fmt.Errorf("compressed file is not Hatanaka compressed: %s", f.Path)
	}

	err := f.Rnx2crx()
	if err != nil {
		return err
	}

	err = archiver.CompressFile(f.Path, f.Path+".gz")
	if err != nil {
		return err
	}
	os.Remove(f.Path)
	f.Path = f.Path + ".gz"
	f.Compression = "gz"

	return nil
} */

// IsHatanakaCompressed returns true if the obs file is Hatanaka compressed, otherwise false.
func (f *ObsFile) IsHatanakaCompressed() bool {
	if f.Format != "" {
		return f.Format == "crx"
	}
	return IsHatanakaCompressed(f.Path)
}

// Returns true if the file giveb by filename is Hatanaka compressed.
// This is checked by the filenames' extension.
func IsHatanakaCompressed(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".crx" || strings.HasSuffix(ext, "d") { // .21d
		return true
	}
	return false
}

// Rnx2crx Hatanaka compresses a RINEX obs file (compact RINEX) and returns the compressed filename.
// The rnxFilename must be a valid RINEX filename.
// see http://terras.gsi.go.jp/ja/crx2rnx.html
func Rnx2crx(rnxFilename string) (string, error) {
	// Check if file is already Hata decompressed.
	if IsHatanakaCompressed(rnxFilename) {
		return rnxFilename, nil
	}

	tool, err := exec.LookPath("RNX2CRX")
	if err != nil {
		return "", err
	}

	dir, rnxFil := filepath.Split(rnxFilename)

	// Build name of target file.
	crxFil := ""
	if Rnx2FileNamePattern.MatchString(rnxFil) {
		typ := "d"
		if strings.HasSuffix(rnxFil, "O") {
			typ = "D"
		}
		crxFil = Rnx2FileNamePattern.ReplaceAllString(rnxFil, "${2}${3}${4}${5}.${6}"+typ)
	} else if Rnx3FileNamePattern.MatchString(rnxFil) {
		crxFil = Rnx3FileNamePattern.ReplaceAllString(rnxFil, "${2}.crx")
	} else {
		return "", fmt.Errorf("rnx2crx: file has no standard RINEX extension")
	}

	//fmt.Printf("rnxFil: %s - crxFil: %s\n", rnxFil, crxFil)

	if crxFil == "" || rnxFil == crxFil {
		return "", fmt.Errorf("rnx2crx: could not build compressed filename")
	}
	crxFilePath := filepath.Join(dir, crxFil)

	// Run compression tool
	cmd := exec.Command(tool, rnxFilename, "-d", "-f")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Launch as new process group so that signals (ex: SIGINT) are not sent also the the child process,
	// see https://stackoverflow.com/questions/66232825/child-process-receives-sigint-which-should-be-handled-only-by-parent-process-re
	cmd.SysProcAttr = &syscall.SysProcAttr{
		//CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP, // windows
		Setpgid: true, // linux
	}

	err = cmd.Run()
	if err != nil {
		rc := cmd.ProcessState.ExitCode()
		if rc == 2 { // Warning
			log.Printf("W! rnx2crx: %s", stderr.Bytes())
		} else { // Error
			if _, err := os.Stat(crxFilePath); !errors.Is(err, os.ErrNotExist) {
				os.Remove(crxFilePath)
			}
			return "", fmt.Errorf("rnx2crx: rc:%d: %v: %s", rc, err, stderr.Bytes())
		}
	}

	// Return filepath
	if _, err := os.Stat(crxFilePath); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("rnx2crx: no such file: %s", crxFilePath)
	}
	return crxFilePath, nil
}

// Crx2rnx decompresses a Hatanaka-compressed RINEX obs file and returns the decompressed filename.
// The crxFilename must be a valid RINEX filename, RINEX v2 lowercase and RINEX v3 uppercase.
// see http://terras.gsi.go.jp/ja/crx2rnx.html
func Crx2rnx(crxFilename string) (string, error) {
	// Check if file is already Hata decompressed.
	if !IsHatanakaCompressed(crxFilename) {
		return crxFilename, nil
	}

	tool, err := exec.LookPath("CRX2RNX")
	if err != nil {
		return "", err
	}

	dir, crxFil := filepath.Split(crxFilename)

	// Build name of target file
	rnxFil := ""
	if Rnx2FileNamePattern.MatchString(crxFil) {
		typ := "o"
		if strings.HasSuffix(crxFil, "D") {
			typ = "O"
		}
		rnxFil = Rnx2FileNamePattern.ReplaceAllString(crxFil, "${2}${3}${4}${5}.${6}"+typ)
	} else if Rnx3FileNamePattern.MatchString(crxFil) {
		rnxFil = Rnx3FileNamePattern.ReplaceAllString(crxFil, "${2}.rnx")
	} else {
		return "", fmt.Errorf("crx2rnx: file has no standard RINEX extension")
	}

	if rnxFil == "" || rnxFil == crxFil {
		return "", fmt.Errorf("crx2rnx: could not build uncompressed filename")
	}
	rnxFilePath := filepath.Join(dir, rnxFil)

	// Run compression tool
	cmd := exec.Command(tool, crxFilename, "-d", "-f")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		//CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP, // windows
		Setpgid: true, // linux
	}

	err = cmd.Run()
	if err != nil {
		rc := cmd.ProcessState.ExitCode()
		if rc == 2 { // Warning
			log.Printf("W! crx2rnx: %s", stderr.Bytes())
		} else { // Error
			if _, err := os.Stat(rnxFilePath); !errors.Is(err, os.ErrNotExist) {
				os.Remove(rnxFilePath)
			}
			return "", fmt.Errorf("crx2rnx: rc:%d: %v: %s", rc, err, stderr.Bytes())
		}
	}

	// Return filepath
	if _, err := os.Stat(rnxFilePath); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("crx2rnx: no such file: %s", rnxFilePath)
	}
	return rnxFilePath, nil
}

func parseFlag(str string) (int, error) {
	if str == " " {
		return 0, nil
	}
	return strconv.Atoi(str)
}

// get decimal part of a float.
func getDecimal(f float64) float64 {
	// or big.NewFloat(f).Text("f", 6)
	fBig := big.NewFloat(f)
	fint, _ := fBig.Int(nil)
	intf := new(big.Float).SetInt(fint)
	//fmt.Printf("accuracy: %d\n", acc)
	resBig := new(big.Float).Sub(fBig, intf)
	ff, _ := resBig.Float64()
	return ff
}

// compare two epochs
func diffEpo(epochs SyncEpochs, opts Options) string {
	epo1, epo2 := epochs.Epo1, epochs.Epo2
	epoTime := epo1.Time
	// if epo1.NumSat != epo2.NumSat {
	// 	return fmt.Sprintf("epo %s: different number of satellites: fil1: %d fil2:%d", epoTime, epo1.NumSat, epo2.NumSat)
	// }

	for _, obs := range epo1.ObsList {
		printSys := false
		for _, useSys := range opts.SatSys {
			if obs.Prn.Sys.Abbr() == string(useSys) {
				printSys = true
				break
			}
		}

		if !printSys {
			continue
		}

		obs2, err := getObsByPRN(epo2.ObsList, obs.Prn)
		if err != nil {
			log.Printf("%v", err)
			continue
		}

		diffObs(obs, obs2, epoTime, obs.Prn)
	}

	return ""
}

func getObsByPRN(obslist []SatObs, prn PRN) (SatObs, error) {
	for _, obs := range obslist {
		if obs.Prn == prn {
			return obs, nil
		}
	}

	return SatObs{}, fmt.Errorf("no oberservations found for prn %v", prn)
}

func diffObs(obs1, obs2 SatObs, epoTime time.Time, prn PRN) string {
	deltaPhase := 0.005
	checkSNR := false
	for k, o1 := range obs1.Obss {
		if o2, ok := obs2.Obss[k]; ok {
			val1, val2 := o1.Val, o2.Val
			if strings.HasPrefix(string(k), "L") { // phase observations
				val1 = getDecimal(val1)
				val2 = getDecimal(val2)
			}
			if (o1.LLI != o2.LLI) || (math.Abs(val1-val2) > deltaPhase) {
				log.Printf("%s %v %02d %s %s %14.03f %d %d | %14.03f %d %d", epoTime.Format(time.RFC3339Nano), prn.Sys, prn.Num, k[:1], k, val1, o1.LLI, o1.SNR, val2, o2.LLI, o2.SNR)
			} else if checkSNR && o1.SNR != o2.SNR {
				log.Printf("%s %v %02d %s %s %14.03f %d %d | %14.03f %d %d", epoTime.Format(time.RFC3339Nano), prn.Sys, prn.Num, k[:1], k, val1, o1.LLI, o1.SNR, val2, o2.LLI, o2.SNR)
			}

			// if o1.SNR != o2.SNR {
			// 	fmt.Printf("%s: SNR: %s: %d %d\n", epoTime.Format(time.RFC3339Nano), k, o1.SNR, o2.SNR)
			// }
			// if val1 != val2 {
			// 	fmt.Printf("%s: val: %s: %14.03f %14.03f\n", epoTime.Format(time.RFC3339Nano), k, val1, val2)
			// }
		} else {
			log.Printf("Key %q does not exist", k)
		}

	}

	return ""
}

// Convert strings to Obscodes.
func convStringsToObscodes(strs []string) []ObsCode {
	obscodes := make([]ObsCode, 0, len(strs))
	for _, str := range strs {
		obscodes = append(obscodes, ObsCode(str))
	}
	return obscodes
}
