package sinex

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDecoder(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/igs20P21161.snx"
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)

	hdr := dec.Header
	assert.Equal("2.02", hdr.Version, "Format Version")
	assert.Equal("IGN", hdr.Agency, "Agency")
	//t.Logf("Header: %+v\n", hdr)
}

func TestHeader_UnmarshalSINEX(t *testing.T) {
	assert := assert.New(t)
	str := "%=SNX 2.02 IGN 20:225:43202 IGN 20:208:75600 20:210:43200 C  1577 2 S E"
	hdr := &Header{}
	err := hdr.UnmarshalSINEX(str)
	assert.NoError(err)

	assert.Equal("2.02", hdr.Version, "Format Version")
	assert.Equal("IGN", hdr.Agency, "Agency")
	assert.Equal(time.Date(2020, 8, 12, 12, 0, 2, 0, time.UTC), hdr.CreationTime, "File Creation Time")
	assert.Equal("IGN", hdr.AgencyDataProvider, "Agency Data Provider")
	assert.Equal(time.Date(2020, 7, 26, 21, 0, 0, 0, time.UTC), hdr.StartTime, "Start Time")
	assert.Equal(time.Date(2020, 7, 28, 12, 0, 0, 0, time.UTC), hdr.EndTime, "End Time")
	assert.Equal(ObsTechCombined, hdr.ObsTech, "Obs Techn")
	assert.Equal(1577, hdr.NumEstimates, "Number of Estimates")
	assert.Equal(2, hdr.ConstraintCode, "Constraint Code")
	assert.Equal([]string{"S", "E"}, hdr.SolutionTypes, "Solution Types")
}

func TestSite_UnmarshalSINEX(t *testing.T) {
	assert := assert.New(t)
	str := " ABMF  A 97103M001 P Les Abymes - Raizet ai 298 28 20.9  16 15 44.3   -25.6"
	s := &Site{}
	err := s.UnmarshalSINEX(str)
	assert.NoError(err)
	t.Logf("%+v", s)
}

func TestAntenna_UnmarshalSINEX(t *testing.T) {
	assert := assert.New(t)
	str := " ABMF  A ---- P 12:024:43200 00:000:00000 TRM57971.00     NONE 14411"
	ant := &Antenna{}
	err := ant.UnmarshalSINEX(str)
	assert.NoError(err)
	assert.Equal("ABMF", string(ant.SiteCode), "sitecode")
	assert.Equal("A", ant.PointCode, "pointcode")
	assert.Equal("", ant.SolID, "solution nbr")
	assert.Equal(ObsTechGPS, ant.ObsTech, "techn")
	assert.Equal(time.Date(2012, 1, 24, 12, 0, 0, 0, time.UTC), ant.DateInstalled, "date installed")
	assert.True(ant.DateRemoved.IsZero(), "date removed")
	assert.Equal("TRM57971.00     NONE", ant.Type, "ant type")
	assert.Equal("NONE", ant.Radome, "radome")
	assert.Equal("14411", ant.SerialNum)
}

func TestReceiver_UnmarshalSINEX(t *testing.T) {
	assert := assert.New(t)
	str := " ABMF  A ---- P 20:038:36000 00:000:00000 SEPT POLARX5         45014 5.3.2"
	recv := &Receiver{}
	err := recv.UnmarshalSINEX(str)
	assert.NoError(err)
	assert.Equal("ABMF", string(recv.SiteCode), "sitecode")
	assert.Equal("A", recv.PointCode, "pointcode")
	assert.Equal("", recv.SolID, "solution nbr")
	assert.Equal(ObsTechGPS, recv.ObsTech, "techn")
	assert.Equal(time.Date(2020, 2, 7, 10, 0, 0, 0, time.UTC), recv.DateInstalled, "date installed")
	assert.True(recv.DateRemoved.IsZero(), "date removed")
	assert.Equal("SEPT POLARX5", recv.Type, "recv type")
	assert.Equal("45014", recv.SerialNum)
	assert.Equal("5.3.2", recv.Firmware)
}

func TestEstimate_UnmarshalSINEX(t *testing.T) {
	str := "     1 STAX   ABMF  A    3 20:209:43200 m    2  2.91978579389317e+06 8.34951e-04"
	esti := &Estimate{}
	err := esti.UnmarshalSINEX(str)
	assert.NoError(t, err)
	t.Logf("%+v", esti)
}

func TestDiscontinuity_UnmarshalSINEX(t *testing.T) {
	assert := assert.New(t)
	str := " AB02  A    2 P 11:175:11380 15:208:17386 P - EQ M6.9 - Fox Islands, Aleutian Islands, Alaska"
	dis := &Discontinuity{}
	err := dis.UnmarshalSINEX(str)
	assert.NoError(err)
	t.Logf("%+v", dis)
	assert.Equal("AB02", string(dis.SiteCode), "sitecode")
	assert.Equal("A", string(dis.ParType), "parameter type")
	assert.Equal(2, dis.Idx, "Index")
	assert.Equal("P", string(dis.Type), "disc type")
	assert.Equal(time.Date(2011, 6, 24, 3, 9, 40, 0, time.UTC), dis.StartTime, "start time")
	assert.Equal(time.Date(2015, 7, 27, 4, 49, 46, 0, time.UTC), dis.EndTime, "end time")
	assert.Equal("EQ M6.9 - Fox Islands, Aleutian Islands, Alaska", dis.Event, "event")
}

func TestUnmarshal(t *testing.T) {
	type args struct {
		in  string
		out Unmarshaler
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "t1-esti", args: args{in: "     1 STAX   ABMF  A    3 20:209:43200 m    2  2.91978579389317e+06 8.34951e-04",
			out: &Estimate{}}, wantErr: false},
		{name: "t1-recv", args: args{in: " ALX2  A ---- P 08:063:00000 00:000:00000 LEICA GRX1200GGPRO   ----  ----",
			out: &Receiver{}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Unmarshal(tt.args.in, tt.args.out); (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecoder_Blocks(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/igs20P21161.snx"
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)

	numBlocks := 0
	for name, err := range dec.Blocks() {
		if err != nil {
			t.Fatal(err)
		}
		numBlocks++
		t.Logf("block: %s", name)
	}

	assert.Equal(13, numBlocks, "number of blocks") // with FILE/REFERENCE it would be 14
}

func TestDecoder_NextEstimate(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/igs20P21161.snx"
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)

	for name, err := range dec.Blocks() {
		if err != nil {
			t.Fatal(err)
		}

		//t.Logf("block: %s", name)
		numRecords := 0
		if name == BlockSolEstimate {
			for _, err := range dec.BlockLines() {
				if err != nil {
					t.Fatal(err)
				}

				numRecords++
				var est Estimate
				err := dec.Decode(&est)
				if err != nil {
					t.Fatal(err)
				}

				//fmt.Printf("%s: %s: %f\n", est.SiteCode, est.ParType, est.Value)

				// Test first record
				if numRecords == 1 {
					assert.Equal(1, est.Idx, "INDEX")
					assert.Equal("STAX", string(est.ParType), "type")
					assert.Equal("ABMF", string(est.SiteCode), "sitecode")
					assert.Equal("A", est.PointCode, "pointcode")
					assert.Equal("3", est.SolID, "soln")
					assert.Equal("m", est.Unit, "unit")
					assert.Equal("2", est.ConstraintCode, "constraint")
					assert.Equal(time.Date(2020, 07, 27, 12, 0, 0, 0, time.UTC), est.Epoch, "epoch")
					assert.Equal(2.91978579389317e+06, est.Value, "value est")
					assert.Equal(8.34951e-04, est.Stddev, "stddev")
				}
			}
			assert.Equal(1577, numRecords, "number of estimates")
		}
	}
}

// Loop over the epochs of a observation data input stream.
func ExampleDecoder_estimates() {
	r, err := os.Open("testdata/igs20P21161.snx")
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	dec, err := NewDecoder(r)
	if err != nil {
		log.Fatal(err)
	}

	var estimates []Estimate
	for name, err := range dec.Blocks() {
		if err != nil {
			log.Fatal(err)
		}

		if name == BlockSolEstimate {
			for _, err := range dec.BlockLines() { // Reads the next line into buffer.
				if err != nil {
					log.Fatal(err)
				}

				var est Estimate
				err := dec.Decode(&est)
				if err != nil {
					log.Fatal(err)
				}
				estimates = append(estimates, est)

				// Do something with est
				// fmt.Printf("%s: %.5f\n", est.SiteCode, est.Value)
			}
		}
	}

	for rec := range AllStationCoordinates(estimates) {
		fmt.Printf("%v\n", rec)
	}
}

func TestDecoder_Discontinuities(t *testing.T) {
	assert := assert.New(t)

	const data = `%=SNX 2.02 IGN 21:329:57332 IGN 94:002:00000 21:001:00000 P     0 1 X
*-------------------------------------------------------------------------------
+SOLUTION/DISCONTINUITY
 00NA  A    1 P 00:000:00000 16:302:00000 P - antenna change
 00NA  A    2 P 16:302:00000 00:000:00000 P - 
 00NA  A    1 P 00:000:00000 00:000:00000 V - 
*
 7601  A    1 P 00:000:00000 22:261:00000 P - unknown
 7601  A    2 P 22:261:00000 00:000:00000 P - 
 7601  A    1 P 00:000:00000 00:000:00000 V - 
*
 AB01  A    1 P 00:000:00000 11:245:39354 P - EQ M6.9 - 170 km E of Atka, Alaska
 AB01  A    5 P 16:072:65205 21:210:22549 P - EQ M8.2 - 99 km SE of Perryville, Alaska
 AB01  A    6 P 21:210:22549 00:000:00000 P - 
 AB01  A    1 P 00:000:00000 13:242:59103 V - EQ M7.0 - 101 km SW of Atka, Alaska
 AB01  A    2 P 13:242:59103 00:000:00000 V - 
*
 AB02  A    1 P 00:000:00000 11:175:11380 P - EQ M7.3 - Fox Islands, Aleutian Islands, Alaska
 AB02  A    4 P 20:336:58960 22:011:41743 P - EQ M6.8 - 100 km SE of Nikolski, Alaska
 AB02  A    5 P 22:011:41743 00:000:00000 P - 
`

	dec, err := NewDecoder(strings.NewReader(data))
	if !errors.Is(err, ErrMandatoryBlockNotFound) {
		t.Fatal(err)
	}
	assert.NotNil(dec)

	numRecords := 0
	err = dec.GoToBlock(BlockSolDiscontinuity) // Special as the input has no file/reference.
	if err != nil {
		t.Fatal(err)
	}

	for _, err := range dec.BlockLines() {
		if err != nil {
			log.Fatal(err)
		}
		numRecords++
		var dis Discontinuity
		err := dec.Decode(&dis)
		if err != nil {
			t.Fatal(err)
		}
		//t.Logf("%+v", dis)
	}

	assert.Equal(14, numRecords, "number of discontinuities")
}

func Test_parseTime(t *testing.T) {
	tests := map[string]time.Time{
		"95:120:86399": time.Date(1995, 4, 30, 23, 59, 59, 0, time.UTC),
		"20:038:00000": time.Date(2020, 2, 7, 0, 0, 0, 0, time.UTC),
		"20:038:36000": time.Date(2020, 2, 7, 10, 0, 0, 0, time.UTC),
		"20:211:43184": time.Date(2020, 7, 29, 11, 59, 44, 0, time.UTC),
		"00:000:00000": time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	for k, v := range tests {
		ti, err := parseTime(k) // or "2006__2"
		assert.NoError(t, err)
		assert.Equal(t, ti, v)
	}
}
