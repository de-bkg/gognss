// Package sinex for reading SINEX files.
// Format description is available at https://www.iers.org/IERS/EN/Organization/AnalysisCoordinator/SinexFormat/sinex.html.

package sinex

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllStationCoordinates(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/igs20P21161.snx"
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)

	var estimates []Estimate
	for name, err := range dec.Blocks() {
		if err != nil {
			t.Fatal(err)
		}

		if name == BlockSolEstimate {
			for _, err := range dec.BlockLines() {
				if err != nil {
					t.Fatal(err)
				}

				var est Estimate
				err := dec.Decode(&est)
				if err != nil {
					t.Fatal(err)
				}
				estimates = append(estimates, est)
			}
		}
	}

	assert.Equal(1577, len(estimates), "number of estimates")

	numCrds := 0
	for crd := range AllStationCoordinates(estimates) {
		numCrds++
		//t.Logf("%v", crd)
		if crd.SiteCode == "" {
			t.Fatalf("record w/o sitecode: %+v", crd)
		}
	}

	assert.Equal(523, numCrds, "number of stations with estimated coordinates")
}
