// Package gnss contains common constants and type definitions.
package gnss

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystems_MarshalJSON(t *testing.T) {
	systems := Systems{SysGAL, SysBDS}
	sysJSON, err := json.Marshal(systems)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "[\"E\",\"C\"]", string(sysJSON), "marshall gnss")
}

func TestParseSatSystems(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    Systems
		wantErr bool
	}{

		{name: "t1", s: "GPS+GLO+GAL+BDS+SBAS+IRNSS",
			want: Systems{SysGPS, SysGLO, SysGAL, SysBDS, SysSBAS, SysNavIC}, wantErr: false},
		{name: "t1-blanks", s: "GPS+GLO+GAL+BDS+SBAS+IRNSS",
			want: Systems{SysGPS, SysGLO, SysGAL, SysBDS, SysSBAS, SysNavIC}, wantErr: false},
		{name: "t2", s: "GPS+GLO-GAL+BDS+SBAS+IRNSS", want: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSatSystems(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSatSystems() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseSatSystems() = %v, want %v", got, tt.want)
			}
		})
	}
}
