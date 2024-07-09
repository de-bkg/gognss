// Package gnss contains common constants and type definitions.
package gnss

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// System is a satellite system.
type System int

// Available satellite systems.
const (
	SysGPS System = iota + 1
	SysGLO
	SysGAL
	SysQZSS
	SysBDS
	SysNavIC
	SysSBAS
	SysMIXED
)

func (sys System) String() string {
	return [...]string{"", "GPS", "GLO", "GAL", "QZSS", "BDS", "NavIC", "SBAS", "MIXED"}[sys]
}

// Abbr returns the systems' abbreviation used in RINEX.
func (sys System) Abbr() string {
	return [...]string{"", "G", "R", "E", "J", "C", "I", "S", "M"}[sys]
}

// For JSON encoding implement the json.Marshaler interface.
func (sys System) MarshalJSON() ([]byte, error) {
	return json.Marshal(sys.Abbr())
}

// Systems specifies a list of satellite systems.
type Systems []System

// String returns the contained systems in sitelog manner GPS+GLO+...
func (syss Systems) String() string {
	str := make([]string, 0, len(syss))
	for _, sys := range syss {
		str = append(str, sys.String())
	}
	return strings.Join(str, "+")
}

// Lookup table to get the GNSS system by its 1-char abbreviation.
var sysPerAbbr = map[string]System{
	"G": SysGPS,
	"R": SysGLO,
	"E": SysGAL,
	"J": SysQZSS,
	"C": SysBDS,
	"I": SysNavIC,
	"S": SysSBAS,
	"M": SysMIXED,
}

// PRN specifies a GNSS satellite.
type PRN struct {
	Sys System // The satellite system.
	Num int8   // The satellite number.
	// flags
}

// NewPRN returns a new PRN for the string prn that is e.g. G12.
func NewPRN(prn string) (PRN, error) {
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

// Receiver is a GNSS receiver.
type Receiver struct {
	Type                 string    `json:"type" validate:"required"`
	SatSystemsDeprecated string    `json:"satelliteSystem"`                      // Sattelite System for compatibility with GA JSON, deprecated!
	SatSystems           Systems   `json:"satelliteSystems" validate:"required"` // Sattelite System
	SerialNum            string    `json:"serialNumber" validate:"required"`
	Firmware             string    `json:"firmwareVersion"`
	ElevationCutoff      float64   `json:"elevationCutoffSetting"`   // degree
	TemperatureStabiliz  string    `json:"temperatureStabilization"` // none or tolerance in degrees C
	DateInstalled        time.Time `json:"dateInstalled" validate:"required"`
	DateRemoved          time.Time `json:"dateRemoved"`
	Notes                string    `json:"notes"` // Additional Information
}

// Equal reports whether two receivers have the same values for the significant parameters.
// Note for STATION INFOTMATION files: Some generators do not consider the receiver firmware
// for this comparision, e.g. EUREF.STA.
func (recv Receiver) Equal(recv2 *Receiver) bool {
	return recv.Type == recv2.Type && recv.SerialNum == recv2.SerialNum && recv.Firmware == recv2.Firmware
}

// Antenna is a GNSS antenna.
type Antenna struct {
	Type                   string    `json:"type" validate:"required"`
	Radome                 string    `json:"antennaRadomeType"`
	RadomeSerialNum        string    `json:"radomeSerialNumber"`
	SerialNum              string    `json:"serialNumber" validate:"required"`
	ReferencePoint         string    `json:"antennaReferencePoint"`
	EccUp                  float64   `json:"markerArpUpEcc"`
	EccNorth               float64   `json:"markerArpNorthEcc"`
	EccEast                float64   `json:"markerArpEastEcc"`
	AlignmentFromTrueNorth float64   `json:"alignmentFromTrueNorth"` // in deg; + is clockwise/east
	CableType              string    `json:"antennaCableType"`       // vendor & type number
	CableLength            float32   `json:"antennaCableLength"`     // in meter
	DateInstalled          time.Time `json:"dateInstalled"`
	DateRemoved            time.Time `json:"dateRemoved"`
	Notes                  string    `json:"notes"` // Additional Information
}

// Equal reports whether two antennas have the same values for the significant parameters.
func (ant Antenna) Equal(ant2 *Antenna) bool {
	return ant.Type == ant2.Type && ant.Radome == ant2.Radome && ant.SerialNum == ant2.SerialNum && ant.EccNorth == ant2.EccNorth && ant.EccEast == ant2.EccEast && ant.EccUp == ant2.EccUp
}
