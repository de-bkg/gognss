package rinex

// Note: fmt.Scanf is pretty slow in Go!? https://github.com/golang/go/issues/12275#issuecomment-133796990

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
)

// Options for global settings.
type Options struct {
	SatSys string // satellite systems GRE...
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

	snum, err := strconv.Atoi(prn[1:3])
	if err != nil {
		return PRN{}, fmt.Errorf("parse sat num: %q: %v", prn, err)
	}
	if snum < 1 || snum > 60 {
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
	Prn  PRN
	Obss map[string]Obs // L1C: Obs{Val:0, LLI:0, SNR:0}, L2C: Obs{Val:...},...
}

// SyncEpochs contains two epochs from different files with the same timestamp.
type SyncEpochs struct {
	Epo1 *Epoch
	Epo2 *Epoch
}

// Epoch contains a RINEX data epoch.
type Epoch struct {
	Time    time.Time // epoch time
	Flag    int8      // Epoch flag 0:OK, 1:power failure between previous and current epoch, >1 : Special event.
	NumSat  uint8     // The number of satellites in this epoch.
	ObsList []SatObs  // A list of observations per PRN.
	//Error   error // e.g. parsing error
}

// Print pretty prints the epoch.
func (epo *Epoch) Print() {
	//fmt.Printf("%+v\n", epo)
	fmt.Printf("%s Flag: %d #prn: %d\n", epo.Time.Format(time.RFC3339Nano), epo.Flag, epo.NumSat)
	for _, satObs := range epo.ObsList {
		fmt.Printf("%v -------------------------------------\n", satObs.Prn)
		for typ, obs := range satObs.Obss {
			fmt.Printf("%s: %+v\n", typ, obs)
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

// ObsMeta stores some metadata about a RINEX obs file.
type ObsMeta struct {
	NumEpochs      int                    `json:"numEpochs"`
	NumSatellites  int                    `json:"numSatellites"` // The number of satellites derived from the header.
	Sampling       time.Duration          `json:"sampling"`      // The saampling interval derived from the data.
	TimeOfFirstObs time.Time              `json:"timeOfFirstObs"`
	TimeOfLastObs  time.Time              `json:"timeOfLastObs"`
	Obsstats       map[PRN]map[string]int `json:"obsstats"` // Number of observations per PRN and observation-type.
}

// A ObsHeader provides the RINEX Observation Header information.
type ObsHeader struct {
	RINEXVersion float32     // RINEX Format version
	RINEXType    string      // RINEX File type. O for Obs
	SatSystem    gnss.System // Satellite System. System is "Mixed" if more than one.

	Pgm   string // name of program creating this file
	RunBy string // name of agency creating this file
	Date  string // date and time of file creation TODO time.Time

	Comments []string // * comment lines

	MarkerName, MarkerNumber, MarkerType string // antennas' marker name, *number and type

	Observer, Agency string

	ReceiverNumber, ReceiverType, ReceiverVersion string
	AntennaNumber, AntennaType                    string

	Position     Coord    // Geocentric approximate marker position [m]
	AntennaDelta CoordNEU // North,East,Up deltas in [m]

	ObsTypes map[gnss.System][]string // List of all observation types per GNSS.

	SignalStrengthUnit string
	Interval           float64 // Observation interval in seconds
	TimeOfFirstObs     time.Time
	TimeOfLastObs      time.Time
	LeapSeconds        int // The current number of leap seconds
	NSatellites        int // Number of satellites, for which observations are stored in the file

	labels []string // all Header Labels found
}

// ObsDecoder reads and decodes header and data records from a RINEX Obs input stream.
type ObsDecoder struct {
	// The Header is valid after NewObsDecoder or Reader.Reset. The header must exist,
	// otherwise ErrNoHeader will be returned.
	Header ObsHeader
	//b       *bufio.Reader // remove!!!
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
	dec.Header, dec.err = dec.readHeader()
	return dec, dec.err
}

// Err returns the first non-EOF error that was encountered by the decoder.
func (dec *ObsDecoder) Err() error {
	if dec.err == io.EOF {
		return nil
	}

	return dec.err
}

// readHeader reads a RINEX Navigation header. If the Header does not exist,
// a ErrNoHeader error will be returned.
func (dec *ObsDecoder) readHeader() (hdr ObsHeader, err error) {
	hdr.ObsTypes = map[gnss.System][]string{}
	maxLines := 800
	rememberMe := ""
read:
	for dec.sc.Scan() {
		dec.lineNum++
		line := dec.sc.Text()
		//fmt.Print(line)

		if dec.lineNum > maxLines {
			return hdr, fmt.Errorf("reading header failed: line %d reached without finding end of header", maxLines)
		}
		if len(line) < 60 {
			continue
		}

		// RINEX files are ASCII, so we can write:
		val := line[:60]
		key := strings.TrimSpace(line[60:])

		hdr.labels = append(hdr.labels, key)

		switch key {
		//if strings.EqualFold(key, "RINEX VERSION / TYPE") {
		case "RINEX VERSION / TYPE":
			if f64, err := strconv.ParseFloat(strings.TrimSpace(val[:20]), 32); err == nil {
				hdr.RINEXVersion = float32(f64)
			} else {
				return hdr, fmt.Errorf("parsing RINEX VERSION: %v", err)
			}
			hdr.RINEXType = strings.TrimSpace(val[20:21])
			if sys, ok := sysPerAbbr[strings.TrimSpace(val[40:41])]; ok {
				hdr.SatSystem = sys
			} else {
				err = fmt.Errorf("read header: invalid satellite system in line %d: %s", dec.lineNum, line)
				return
			}
		case "PGM / RUN BY / DATE":
			hdr.Pgm = strings.TrimSpace(val[:20])
			hdr.RunBy = strings.TrimSpace(val[20:40])
			hdr.Date = strings.TrimSpace(val[40:])
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
				return hdr, fmt.Errorf("parsing approx. position from line: %s", line)
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
				return hdr, fmt.Errorf("parsing antenna deltas from line: %s", line)
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
		case "SYS / # / OBS TYPES":
			sysStr := val[:1]
			if sysStr == " " { // line continued
				sysStr = rememberMe
			} else {
				rememberMe = sysStr
			}

			sys, ok := sysPerAbbr[sysStr]
			if !ok {
				err = fmt.Errorf("invalid satellite system: %q: line %d", val[:1], dec.lineNum)
				return
			}

			if strings.TrimSpace(val[3:6]) != "" { // number of obstypes
				hdr.ObsTypes[sys] = strings.Fields(val[7:])
			} else {
				hdr.ObsTypes[sys] = append(hdr.ObsTypes[sys], strings.Fields(val[7:])...)
			}
		case "# / TYPES OF OBSERV": // RINEX-2
			sys := hdr.SatSystem
			if strings.TrimSpace(val[:6]) != "" { // number of obstypes
				hdr.ObsTypes[sys] = strings.Fields(val[7:])
			} else {
				hdr.ObsTypes[sys] = append(hdr.ObsTypes[sys], strings.Fields(val[7:])...)
			}
		case "SIGNAL STRENGTH UNIT":
			hdr.SignalStrengthUnit = strings.TrimSpace(val[:20])
		case "INTERVAL":
			if f64, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
				hdr.Interval = f64
			}
		case "TIME OF FIRST OBS":
			t, err := time.Parse(epochTimeFormat, strings.TrimSpace(val[:43]))
			if err != nil {
				return hdr, fmt.Errorf("parsing %q: %v", key, err)
			}
			hdr.TimeOfFirstObs = t
		case "TIME OF LAST OBS":
			t, err := time.Parse(epochTimeFormat, strings.TrimSpace(val[:43]))
			if err != nil {
				return hdr, fmt.Errorf("parsing %q: %v", key, err)
			}
			hdr.TimeOfLastObs = t
		case "SYS / PHASE SHIFT": // optional. This header line is strongly deprecated and should be ignored by decoders.
		case "LEAP SECONDS": // optional. not complete! TODO: extend
			i, err := strconv.Atoi(strings.TrimSpace(val[:6]))
			if err != nil {
				return hdr, fmt.Errorf("parsing %q: %v", key, err)
			}
			hdr.LeapSeconds = i
		case "# OF SATELLITES": // optional
			i, err := strconv.Atoi(strings.TrimSpace(val[:6]))
			if err != nil {
				return hdr, fmt.Errorf("parsing %q: %v", key, err)
			}
			hdr.NSatellites = i
		case "PRN / # OF OBS": // optional
			// TODO
		case "END OF HEADER":
			break read
		default:
			fmt.Printf("Header field %q not handled yet\n", key)
		}
	}

	err = dec.sc.Err()
	return
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
	for dec.sc.Scan() {
		dec.lineNum++
		line := dec.sc.Text()

		if len(line) < 1 {
			continue
		}

		epTime, err := time.Parse(epochTimeFormatv2, line[1:26])
		if err != nil {
			dec.setErr(fmt.Errorf("error in line %d: %v", dec.lineNum, err))
			return false
		}

		epochFlag, err := strconv.Atoi(line[28:29])
		if err != nil {
			dec.setErr(fmt.Errorf("parsing epoch flag in line %d: %q", dec.lineNum, line))
			return false
		}

		// Number of satellites
		numSat, err := strconv.Atoi(strings.TrimSpace(line[30:32]))
		if err != nil {
			dec.setErr(fmt.Errorf("error in line %d: %v", dec.lineNum, err))
			return false
		}

		// Read list of PRNs
		pos := 32
		sats := make([]PRN, 0, numSat)
		for iSat := 0; iSat < numSat; iSat++ {
			if iSat > 0 && iSat%12 == 0 {
				dec.sc.Scan()
				dec.lineNum++
				line = dec.sc.Text()
				pos = 32
			}

			// sysShort := string(line[pos])
			// if sysShort == " " {
			// 	sysShort = "G"
			// }

			prn, err := newPRN(line[pos : pos+3])
			if err != nil {
				dec.setErr(fmt.Errorf("new PRN in line %d: %q: %v", dec.lineNum, line, err))
				return false
			}
			sats = append(sats, prn)
			//_currEpo.rnxSat[iSat].prn.set(sys, satNum);
			pos += 3
		}

		dec.epo = &Epoch{Time: epTime, Flag: int8(epochFlag), NumSat: uint8(numSat),
			ObsList: make([]SatObs, 0, numSat)}

		// Read observation records
		for iSat := 0; iSat < numSat; iSat++ {
			dec.sc.Scan()
			dec.lineNum++
			if err := dec.sc.Err(); err != nil {
				dec.setErr(fmt.Errorf("error in line %d: %v", dec.lineNum, err))
				return false
			}
			line = dec.sc.Text()

			prn, err := newPRN(line[0:3])
			if err != nil {
				dec.setErr(fmt.Errorf("new PRN in line %d: %q: %v", dec.lineNum, line, err))
				return false
			}

			if strings.TrimSpace(line[3:]) == "" { // ??
				continue
			}

			sys := sysPerAbbr[line[0:1]]
			obsPerTyp := make(map[string]Obs, 30) // cap
			pos := 3                              // line position
			for _, typ := range dec.Header.ObsTypes[sys] {
				var val float64
				if pos+14 > len(line) {
					// error ??
					dec.setErr(fmt.Errorf("obstype %s out of range in line %d: %q", typ, dec.lineNum, line))
					return false
				}

				//fmt.Printf("%q\n", line[pos:pos+14])
				obsStr := strings.TrimSpace(line[pos : pos+14])
				if obsStr != "" {
					val, err = strconv.ParseFloat(obsStr, 64)
					if err != nil {
						dec.setErr(fmt.Errorf("parsing the %s observation in line %d: %q", typ, dec.lineNum, line))
						return false
					}
				}
				pos += 14

				// LLI
				if pos+1 > len(line) {
					obsPerTyp[typ] = Obs{Val: val}
					break
				}
				pos++
				lli, err := parseFlag(line[pos-1 : pos])
				if err != nil {
					dec.setErr(fmt.Errorf("parsing the %s LLI in line %d: %q: %v", typ, dec.lineNum, line, err))
					return false
				}

				// SNR
				if pos+1 > len(line) {
					obsPerTyp[typ] = Obs{Val: val, LLI: int8(lli)}
					break
				}
				pos++
				snr, err := parseFlag(line[pos-1 : pos])
				if err != nil {
					dec.setErr(fmt.Errorf("parsing the %s SNR in line %d: %q: %v", typ, dec.lineNum, line, err))
					return false
				}

				obsPerTyp[typ] = Obs{Val: val, LLI: int8(lli), SNR: int8(snr)}
			}
			dec.epo.ObsList = append(dec.epo.ObsList, SatObs{Prn: prn, Obss: obsPerTyp})
		}
		return true
	}

	if err := dec.sc.Err(); err != nil {
		dec.setErr(fmt.Errorf("read epoch scanner error: %v", err))
	}

	return false // EOF
}

func (dec *ObsDecoder) nextEpoch() bool {
	for dec.sc.Scan() {
		dec.lineNum++
		line := dec.sc.Text()

		if len(line) < 1 {
			continue
		}

		if !strings.HasPrefix(line, "> ") {
			fmt.Printf("stream does not start with epoch line: %q\n", line) // must not be an error
			continue
		}

		//> 2018 11 06 19 00  0.0000000  0 31
		epTime, err := time.Parse(epochTimeFormat, line[2:29])
		if err != nil {
			dec.setErr(fmt.Errorf("error in line %d: %v", dec.lineNum, err))
			return false
		}

		epochFlag, err := strconv.Atoi(line[31:32])
		if err != nil {
			dec.setErr(fmt.Errorf("parsing epoch flag in line %d: %q", dec.lineNum, line))
			return false
		}

		numSat, err := strconv.Atoi(strings.TrimSpace(line[32:35]))
		if err != nil {
			dec.setErr(fmt.Errorf("error in line %d: %v", dec.lineNum, err))
			return false
		}

		dec.epo = &Epoch{Time: epTime, Flag: int8(epochFlag), NumSat: uint8(numSat),
			ObsList: make([]SatObs, 0, numSat)}

		// Read observations
		for ii := 1; ii <= numSat; ii++ {
			dec.sc.Scan()
			dec.lineNum++
			if err := dec.sc.Err(); err != nil {
				dec.setErr(fmt.Errorf("error in line %d: %v", dec.lineNum, err))
				return false
			}
			line = dec.sc.Text()
			linelen := len(line)

			prn, err := newPRN(line[0:3])
			if err != nil {
				dec.setErr(fmt.Errorf("parsing sat num in line %d: %q: %v", dec.lineNum, line, err))
				return false
			}

			if strings.TrimSpace(line[3:]) == "" { // ??
				continue
			}

			sys := sysPerAbbr[line[:1]]
			obsPerTyp := make(map[string]Obs, 30)
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
				obs, err := decodeObs(line[pos:end])
				if err != nil {
					dec.setErr(fmt.Errorf("parsing the %s observation in line %d: %q: %v", typ, dec.lineNum, line, err))
					return false
				}
				obsPerTyp[typ] = obs
			}
			dec.epo.ObsList = append(dec.epo.ObsList, SatObs{Prn: prn, Obss: obsPerTyp})
		}
		return true
	}

	if err := dec.sc.Err(); err != nil {
		dec.setErr(fmt.Errorf("read epoch scanner error: %v", err))
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

// setErr records the first error encountered.
func (dec *ObsDecoder) setErr(err error) {
	if dec.err == nil || dec.err == io.EOF {
		dec.err = err
	}
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

// decode an observation of a GNSS obs file.
func decodeObs(s string) (obs Obs, err error) {
	val := 0.0
	lli := 0
	snr := 0

	if strings.TrimSpace(s) == "" {
		return
	}

	// Value
	valStr := strings.TrimSpace(s[:14])
	if valStr != "" {
		val, err = strconv.ParseFloat(valStr, 64)
		if err != nil {
			err = fmt.Errorf("parse obs: %q: %v", s, err)
			return
		}
	}
	obs.Val = val

	// LLI
	if len(s) > 14 {
		if s[14:15] != " " {
			lli, err = strconv.Atoi(s[14:15])
			if err != nil {
				err = fmt.Errorf("parse LLI: %q: %v", s, err)
				return
			}
		}
	}
	// TODO flag powerfail
	// if (_flgPowerFail) {
	// 	lli |= 1;
	//   }
	obs.LLI = int8(lli)

	// SNR
	if len(s) > 15 {
		if s[15:16] != " " {
			snr, err = strconv.Atoi(s[15:16])
			if err != nil {
				err = fmt.Errorf("parse LLI: %q: %v", s, err)
				return
			}
		}
	}
	obs.SNR = int8(snr)
	return
}

// ObsFile contains fields and methods for RINEX observation files.
// Use NewObsFil() to instantiate a new ObsFile.
type ObsFile struct {
	*RnxFil
	Header *ObsHeader
	Opts   *Options
}

// NewObsFile returns a new ObsFile.
func NewObsFile(filepath string) (*ObsFile, error) {
	// must file exist?
	obsFil := &ObsFile{RnxFil: &RnxFil{Path: filepath}, Header: &ObsHeader{}, Opts: &Options{}}
	err := obsFil.parseFilename()
	return obsFil, err
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
			fmt.Printf("diff: %s\n", diff)
		}
	}
	if err := dec.Err(); err != nil {
		return fmt.Errorf("read epochs error: %v", err)
	}

	return nil
}

// Meta reads the file and returns some metadata.
func (f *ObsFile) Meta() (stat ObsMeta, err error) {
	r, err := os.Open(f.Path)
	if err != nil {
		return
	}
	defer r.Close()
	dec, err := NewObsDecoder(r)
	if err != nil {
		return
	}
	f.Header = &dec.Header

	numSat := 60
	if f.Header.NSatellites > 0 {
		numSat = f.Header.NSatellites
	}

	satmap := make(map[string]int, numSat)

	obsstats := make(map[PRN]map[string]int, numSat)
	numOfEpochs := 0
	intervals := make([]time.Duration, 0, 10)
	var epo, epoPrev *Epoch

	for dec.NextEpoch() {
		numOfEpochs++
		epo = dec.Epoch()
		if numOfEpochs == 1 {
			stat.TimeOfFirstObs = epo.Time
		}

		for _, obsPerSat := range epo.ObsList {
			prn := obsPerSat.Prn

			// list of all satellites
			if _, exists := satmap[prn.String()]; !exists {
				satmap[prn.String()] = 1
			}

			// observations per sat and obs-type
			for obstype, obs := range obsPerSat.Obss {
				if prn.Sys == gnss.SysGPS && prn.Num == 11 {
					fmt.Printf("%s: %s: %+v\n", prn, obstype, obs)
				}
				if _, exists := obsstats[prn]; !exists {
					obsstats[prn] = map[string]int{}
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
		return
	}

	stat.TimeOfLastObs = epoPrev.Time
	stat.NumEpochs = numOfEpochs
	stat.NumSatellites = len(satmap)
	stat.Obsstats = obsstats

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
	sort.Slice(intervals, func(i, j int) bool { return intervals[i] < intervals[j] })
	stat.Sampling = intervals[int(len(intervals)/2)]

	// LLIs

	return
}

// Rnx3Filename returns the filename following the RINEX3 convention.
// In most cases we must read the read the header. The countrycode must come from an external source.
// DO NOT USE! Must parse header first!
func (f *ObsFile) Rnx3Filename() (string, error) {
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
	return f.Format == "crx"
}

// Rnx2crx Hatanaka compresses a RINEX obs file (compact RINEX) and returns the compressed filename.
// The rnxFilename must be a valid RINEX filename.
// see http://terras.gsi.go.jp/ja/crx2rnx.html
func Rnx2crx(rnxFilename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(rnxFilename))

	// Check if file is already Hata decompressed
	if ext == "crx" || ext == "d" {
		return rnxFilename, nil
	}

	tool, err := exec.LookPath("RNX2CRX")
	if err != nil {
		return "", err
	}

	dir, rnxFil := filepath.Split(rnxFilename)

	// Build name of target file
	crxFil := ""
	if Rnx2FileNamePattern.MatchString(rnxFil) {
		crxFil = Rnx2FileNamePattern.ReplaceAllString(rnxFil, "${2}${3}${4}${5}.${6}d")
	} else if Rnx3FileNamePattern.MatchString(rnxFil) {
		crxFil = Rnx3FileNamePattern.ReplaceAllString(rnxFil, "${2}.crx")
	} else {
		return "", fmt.Errorf("file %s with no standard RINEX extension", rnxFil)
	}

	//fmt.Printf("rnxFil: %s - crxFil: %s\n", rnxFil, crxFil)

	if crxFil == "" || rnxFil == crxFil {
		return "", fmt.Errorf("could not build compressed filename for %s", rnxFil)
	}

	// Run compression tool
	cmd := exec.Command(tool, rnxFilename, "-d", "-f")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("cmd %s failed: %v: %s", tool, err, stderr.Bytes())
	}

	// Return filepath
	crxFilePath := filepath.Join(dir, crxFil)
	if _, err := os.Stat(crxFilePath); os.IsNotExist(err) {
		return "", fmt.Errorf("compressed file does not exist: %s", crxFilePath)
	}
	return crxFilePath, nil
}

// Crx2rnx decompresses a Hatanaka-compressed RINEX obs file and returns the decompressed filename.
// The crxFilename must be a valid RINEX filename.
// see http://terras.gsi.go.jp/ja/crx2rnx.html
func Crx2rnx(crxFilename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(crxFilename))

	// Check if file is already Hata decompressed
	if ext == "rnx" || ext == "o" {
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
		rnxFil = Rnx2FileNamePattern.ReplaceAllString(crxFil, "${2}${3}${4}${5}.${6}o")
	} else if Rnx3FileNamePattern.MatchString(crxFil) {
		rnxFil = Rnx3FileNamePattern.ReplaceAllString(crxFil, "${2}.rnx")
	} else {
		return "", fmt.Errorf("file %s with no standard RINEX extension", crxFil)
	}

	if rnxFil == "" || rnxFil == crxFil {
		return "", fmt.Errorf("could not build uncompressed filename for %s", crxFil)
	}

	// Run compression tool
	cmd := exec.Command(tool, crxFilename, "-d", "-f")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("cmd %s failed: %v: %s", tool, err, stderr.Bytes())
	}

	// Return filepath
	rnxFilePath := filepath.Join(dir, rnxFil)
	if _, err := os.Stat(rnxFilePath); os.IsNotExist(err) {
		return "", fmt.Errorf("compressed file does not exist: %s", rnxFilePath)
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
			fmt.Printf("%v\n", err)
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
			if strings.HasPrefix(k, "L") { // phase observations
				val1 = getDecimal(val1)
				val2 = getDecimal(val2)
			}
			if (o1.LLI != o2.LLI) || (math.Abs(val1-val2) > deltaPhase) {
				fmt.Printf("%s %v %02d %s %s %14.03f %d %d | %14.03f %d %d\n", epoTime.Format(time.RFC3339Nano), prn.Sys, prn.Num, k[:1], k, val1, o1.LLI, o1.SNR, val2, o2.LLI, o2.SNR)
			} else if checkSNR && o1.SNR != o2.SNR {
				fmt.Printf("%s %v %02d %s %s %14.03f %d %d | %14.03f %d %d\n", epoTime.Format(time.RFC3339Nano), prn.Sys, prn.Num, k[:1], k, val1, o1.LLI, o1.SNR, val2, o2.LLI, o2.SNR)
			}

			// if o1.SNR != o2.SNR {
			// 	fmt.Printf("%s: SNR: %s: %d %d\n", epoTime.Format(time.RFC3339Nano), k, o1.SNR, o2.SNR)
			// }
			// if val1 != val2 {
			// 	fmt.Printf("%s: val: %s: %14.03f %14.03f\n", epoTime.Format(time.RFC3339Nano), k, val1, val2)
			// }
		} else {
			fmt.Printf("Key %q does not exist\n", k)
		}

	}

	return ""
}
