package rinex

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetDecoder_NextEpoch(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/white/BAUT00DEU_R_20223131300_01H_10S_MM.rnx"
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewMetDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)

	firstEpo := &MeteoEpoch{}
	numOfEpochs := 0
	for dec.NextEpoch() {
		numOfEpochs++
		epo := dec.Epoch()
		//fmt.Printf("%v\n", epo)
		if numOfEpochs == 1 {
			firstEpo = epo
		}
	}
	assert.NoError(dec.Err())
	t.Logf("1st epoch: %+v", firstEpo)

	assert.Equal(360, numOfEpochs, "#epochs")
	assert.Equal(380, dec.lineNum, "#lines")
}

func Test_decodeMeteoLineHlp(t *testing.T) {
	// Helper test. Go slices are inclusive-exclusive.
	// Rnx3
	line := " 2022 11  9 13  0  1  993.4   12.1   63.5  214.0    1.1    0.0"
	assert.Equal(t, "2022 11  9 13  0  1", line[1:20], "datetime")
	assert.Equal(t, "  993.4", line[20:27], "1st obs")
	assert.Equal(t, "   12.1", line[27:34], "2st obs")
}
