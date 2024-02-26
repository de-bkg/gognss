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

	"github.com/de-bkg/gognss/pkg/gnss"
)

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

			// G or blank: GPS
			myprn := line[pos : pos+3]
			if myprn[0] == ' ' {
				myprn = "G" + myprn[1:3]
			}

			prn, err := newPRN(myprn)
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
	oEnd := 14
	if len(s) < oEnd {
		oEnd = len(s)
	}
	valStr := strings.TrimSpace(s[:oEnd])
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
