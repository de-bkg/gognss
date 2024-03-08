package sinex

import (
	"log"
	"os"
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

func TestDecoder_NextEstimate(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/igs20P21161.snx"
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)

	for dec.NextBlock() {
		name := dec.CurrentBlock()
		//t.Logf("block: %s", name)
		numRecords := 0
		if name == BlockSolEstimate {
			for dec.NextBlockLine() {
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

	for dec.NextBlock() {
		name := dec.CurrentBlock()
		if name == BlockSolEstimate {
			for dec.NextBlockLine() {
				var est Estimate
				err := dec.Decode(&est)
				if err != nil {
					log.Fatal(err)
				}

				// Do something with est
				//fmt.Printf("%s: %.5f\n", est.SiteCode, est.Value)
			}
		}
	}
}

func TestDecoder_NextBlock(t *testing.T) {
	assert := assert.New(t)
	filepath := "testdata/igs20P21161.snx"
	r, err := os.Open(filepath)
	assert.NoError(err)
	defer r.Close()

	dec, err := NewDecoder(r)
	assert.NoError(err)
	assert.NotNil(dec)

	for dec.NextBlock() {
		name := dec.CurrentBlock()
		//t.Logf("block: %s", name)

		// Read receiver records.
		if name == BlockSiteReceiver {
			for dec.NextBlockLine() {
				var recv Receiver
				err := dec.Decode(&recv)
				if err != nil {
					t.Fatal(err)
				}

				//fmt.Printf("%s: %s\n", recv.SiteCode, recv.Type)
			}
		}
	}
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
