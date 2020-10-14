package site

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSite_cleanAntennas(t *testing.T) {
	antennas := []*Antenna{{Type: "ASH701945E_M NONE", Radome: "NONE", RadomeSerialNum: "",
		SerialNum: "CR620023301", ReferencePoint: "BPA", EccUp: 0.1266, EccNorth: 0.001, EccEast: 0, AlignmentFromTrueNorth: 0,
		CableType: "ANDREW heliax LDF2-50A", CableLength: 60, DateInstalled: time.Date(2006, 07, 07, 0, 0, 0, 0, time.UTC),
		DateRemoved: time.Date(2008, 03, 19, 8, 45, 0, 0, time.UTC), Notes: ""},
		{Type: "LEIAR25.R3", Radome: "LEIT", RadomeSerialNum: "",
			SerialNum: "CR620023301", ReferencePoint: "BPA", EccUp: 0.1266, EccNorth: 0.001, EccEast: 0, AlignmentFromTrueNorth: 0,
			CableType: "ANDREW heliax LDF2-50A", CableLength: 60, DateInstalled: time.Date(2008, 3, 19, 9, 0, 0, 0, time.UTC),
			DateRemoved: time.Date(2008, 05, 20, 10, 0, 0, 0, time.UTC), Notes: ""},
	}
	s := Site{Antennas: antennas}
	err := s.cleanAntennas()
	assert.NoError(t, err)
	assert.Equal(t, "ASH701945E_M    NONE", s.Antennas[0].Type, "ANT TYPE string length")
	assert.Equal(t, "LEIAR25.R3      LEIT", s.Antennas[1].Type, "ANT TYPE string length")
}
