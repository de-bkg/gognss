// Package gnss contains common constants and type definitions.
package gnss

import "strings"

// System is a satellite system.
type System int

// Available satellite systems.
const (
	SysGPS System = iota + 1
	SysGLO
	SysGAL
	SysQZSS
	SysBDS
	SysIRNSS
	SysSBAS
	SysMIXED
)

func (sys System) String() string {
	// TODO change to NavIC or NAVIC
	return [...]string{"", "GPS", "GLO", "GAL", "QZSS", "BDS", "IRNSS", "SBAS", "MIXED"}[sys]
}

// Abbr returns the systems' abbreviation used in RINEX.
func (sys System) Abbr() string {
	return [...]string{"", "G", "R", "E", "J", "C", "I", "S", "M"}[sys]
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
