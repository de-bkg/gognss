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
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
)

// EpochFlag for indicating special occurrences during the tracking in RINEX observation files.
type EpochFlag uint8

const (
	EpochFlagOK            EpochFlag = iota // 0: OK
	EpochFlagPowerFailure                   // 1: power failure between previous and current epoch
	EpochFlagMovingAntenna                  // 2: start moving antenna
	EpochFlagNewSite                        // 3: new site occupation (end of kinematic data) (at least MARKER NAME record follows)
	EpochFlagHeaderInfo                     // 4: header information follows
	EpochFlagExternalEvent                  // 5: external event (epoch is significant, same time frame as observation time tags)
	EpochFlagCycleSlip                      // 6: cycle slip records follow to optionally report detected and repaired cycle slips
)

func (flag EpochFlag) String() string {
	return [...]string{"OK", "Power Failure", "Moving Antenna", "New Site", "Header Info", "External Event", "Cycle Slip Records"}[flag]
}

// ObsCode is the RINEX observation code that specifies frequency, signal and tracking mode like "L1C".
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

// SatObs contains all observations for a satellite per epoch.
type SatObs struct {
	Prn  gnss.PRN        // The satellite number or PRN.
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
	Flag    EpochFlag // The epoch flag 0:OK, 1:power failure between previous and current epoch, >1 : Special event.
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
	NumEpochs      int                          `json:"numEpochs"`      // The number of epochs in the file.
	NumSatellites  int                          `json:"numSatellites"`  // The number of satellites derived from the header.
	Sampling       time.Duration                `json:"sampling"`       // The sampling interval derived from the data.
	TimeOfFirstObs time.Time                    `json:"timeOfFirstObs"` // Time of the first observation.
	TimeOfLastObs  time.Time                    `json:"timeOfLastObs"`  // Time of the last observation.
	ObsPerSat      map[gnss.PRN]map[ObsCode]int `json:"obsstats"`       // Number of observations per PRN and observation-type.
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
	Licenses     []string // Line(s) with the data license of use. Name of the license plus link to the specific version of the license. Using standard data license as from https://creativecommons.org/licenses/
	StationInfos []string // Line(s) with the link(s) to persistent URL with the station metadata (site log, GeodesyML, etc).

	ObsTypes map[gnss.System][]ObsCode // List of all observation types per GNSS.

	SignalStrengthUnit string
	Interval           float64 // Observation interval in seconds
	TimeOfFirstObs     time.Time
	TimeOfLastObs      time.Time
	GloSlots           map[gnss.PRN]int // GLONASS slot and frequency numbers.
	LeapSeconds        int              // The current number of leap seconds
	NSatellites        int              // Number of satellites, for which observations are stored in the file

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

// Write the header to w.
func (hdr *ObsHeader) Write(w io.Writer) error {
	bw := bufio.NewWriter(w)

	fmt.Fprintf(bw, "%9.2f%-11s%-20s%-20s%-s\n", hdr.RINEXVersion, " ", "OBSERVATION DATA", hdr.SatSystem.Abbr(), "RINEX VERSION / TYPE")
	if hdr.Pgm != "" {
		fmt.Fprintf(bw, "%-20s%-20s%-20s%-s\n", hdr.Pgm, hdr.RunBy, hdr.Date.Format("20060102 150405 UTC"), "PGM / RUN BY / DATE")
	}

	if len(hdr.Comments) > 0 {
		for _, c := range hdr.Comments {
			fmt.Fprintf(bw, "%-60.60s%-s\n", c, "COMMENT")
		}
	}
	fmt.Fprintf(bw, "%-60s%-s\n", hdr.MarkerName, "MARKER NAME")
	if hdr.MarkerNumber != "" {
		fmt.Fprintf(bw, "%-20s%-40s%-s\n", hdr.MarkerNumber, " ", "MARKER NUMBER")
	}
	fmt.Fprintf(bw, "%-20s%-40s%-s\n", hdr.MarkerType, " ", "MARKER TYPE")
	fmt.Fprintf(bw, "%-20s%-20.20s%-20.20s%-s\n", hdr.Observer, hdr.Agency, " ", "OBSERVER / AGENCY")
	fmt.Fprintf(bw, "%-20.20s%-20.20s%-20.20s%-s\n", hdr.ReceiverNumber, hdr.ReceiverType, hdr.ReceiverVersion, "REC # / TYPE / VERS")
	fmt.Fprintf(bw, "%-20s%-20.20s%-20.20s%-s\n", hdr.AntennaNumber, hdr.AntennaType, "", "ANT # / TYPE")
	fmt.Fprintf(bw, "%14.4f%14.4f%14.4f%-18s%-s\n", hdr.Position.X, hdr.Position.Y, hdr.Position.Z, " ", "APPROX POSITION XYZ")
	fmt.Fprintf(bw, "%14.4f%14.4f%14.4f%-18s%-s\n", hdr.AntennaDelta.Up, hdr.AntennaDelta.E, hdr.AntennaDelta.N, " ", "ANTENNA: DELTA H/E/N")
	if hdr.DOI != "" {
		fmt.Fprintf(bw, "%-60s%-s\n", hdr.DOI, "DOI")
	}
	hdr.writeObsCodes(bw)
	if hdr.Interval != 0 {
		fmt.Fprintf(bw, "%10.3f%-50s%-s\n", hdr.Interval, " ", "INTERVAL")
	}

	// TODO must not be GPS!
	if !hdr.TimeOfFirstObs.IsZero() {
		fmt.Fprintf(bw, "%s%-5s%-12s%-s\n", hdr.formatFirstObsTime(hdr.TimeOfFirstObs), " ", "GPS", "TIME OF FIRST OBS")
	}

	if !hdr.TimeOfLastObs.IsZero() {
		fmt.Fprintf(bw, "%s%-5s%-12s%-s\n", hdr.formatFirstObsTime(hdr.TimeOfLastObs), " ", "GPS", "TIME OF LAST OBS")
	}

	hdr.writeGloSlotsAndFreqs(bw)

	fmt.Fprintf(bw, "%-60s%-s\n", " ", "END OF HEADER")

	return bw.Flush()
}

// writes the Observation Types to w, in the format of a RINEX header.
func (hdr *ObsHeader) writeObsCodes(w io.Writer) {
	for sys, codes := range hdr.ObsTypes {
		numCodes := len(codes)
		if numCodes < 1 {
			continue
		}

		fmt.Fprintf(w, "%-3s%3d", sys.Abbr(), numCodes)
		iChunk := 0
		for chunk := range slices.Chunk(codes, 13) {
			if iChunk > 0 {
				fmt.Fprint(w, "      ")
			}
			numCodesInLine := len(chunk)
			for _, code := range chunk {
				fmt.Fprintf(w, " %-3s", code)
			}
			pad := strings.Repeat("    ", 13-numCodesInLine)
			fmt.Fprintf(w, "%s  %-s\n", pad, "SYS / # / OBS TYPES")
			iChunk++
		}
	}
}

// writes the GLONASS slots & frequency header to w.
func (hdr *ObsHeader) writeGloSlotsAndFreqs(w io.Writer) {
	// Sort by prn
	keys := make([]gnss.PRN, 0, len(hdr.GloSlots))
	for k := range hdr.GloSlots {
		keys = append(keys, k)
	}
	sort.Sort(gnss.ByPRN(keys))

	iChunk := 0
	for chunk := range slices.Chunk(keys, 8) {
		if iChunk == 0 {
			fmt.Fprintf(w, "%3d ", len(keys))
		} else {
			fmt.Fprintf(w, "%4s", " ")
		}
		numCodesInLine := len(chunk)
		for _, prn := range chunk {
			fmt.Fprintf(w, "%-3s %2d ", prn, hdr.GloSlots[prn])
		}

		pad := strings.Repeat("       ", 8-numCodesInLine)
		fmt.Fprintf(w, "%s%-s\n", pad, "GLONASS SLOT / FRQ #")
		iChunk++
	}
}

// Formats the given time ti in the layout of TIME OF FIRST OBS and TIME OF LAST OBS.
func (hdr *ObsHeader) formatFirstObsTime(ti time.Time) string {
	return fmt.Sprintf("%6d%6d%6d%6d%6d%13.7f", ti.Year(), ti.Month(), ti.Day(), ti.Hour(), ti.Minute(),
		float64(ti.Second())+float64(ti.Nanosecond())/1e9)
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

	obsstats := make(map[gnss.PRN]map[ObsCode]int, numSat)
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
			log.Printf("WARN! rnx2crx: %s", stderr.Bytes())
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
			log.Printf("WARN! crx2rnx: %s", stderr.Bytes())
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

func getObsByPRN(obslist []SatObs, prn gnss.PRN) (SatObs, error) {
	for _, obs := range obslist {
		if obs.Prn == prn {
			return obs, nil
		}
	}

	return SatObs{}, fmt.Errorf("no oberservations found for prn %v", prn)
}

func diffObs(obs1, obs2 SatObs, epoTime time.Time, prn gnss.PRN) string {
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
