package rinex

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileNamePattern(t *testing.T) {
	// Rnx2
	res := Rnx2FileNamePattern.FindStringSubmatch("adar335t.18d.Z") // obs hourly
	assert.Greater(t, len(res), 7)
	for k, v := range res {
		fmt.Printf("%d. %s\n", k, v)
	}
	fmt.Println("----------------------------")

	res = Rnx2FileNamePattern.FindStringSubmatch("bcln332d15.18o") // obs highrate
	assert.Greater(t, len(res), 7)
	for k, v := range res {
		fmt.Printf("%d. %s\n", k, v)
	}
	fmt.Println("----------------------------")

	// Rnx3
	res = Rnx3FileNamePattern.FindStringSubmatch("ALGO00CAN_R_20121601000_15M_01S_GO.rnx") // obs highrate
	assert.Greater(t, len(res), 7)
	for k, v := range res {
		fmt.Printf("%d. %s\n", k, v)
	}
	fmt.Println("----------------------------")

	res = Rnx3FileNamePattern.FindStringSubmatch("ALGO00CAN_R_20121600000_01D_MN.rnx.gz") // nav
	assert.Greater(t, len(res), 7)
	for k, v := range res {
		fmt.Printf("%d. %s\n", k, v)
	}
}

// Convert a RINEX 2 filename to a RINEX 3 one.
func ExampleRnx3Filename() {
	rnx2name, err := Rnx3Filename("testdata/white/brst155h.20o", "FRA")
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(rnx2name)
	// Output: BRST00FRA_R_20201550700_01H_30S_MO.rnx
}

func TestRnx3Filename(t *testing.T) {
	type args struct {
		rnx2filepath string
		countryCode  string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Rnx2 hourly obs file", args: args{"testdata/white/brst155h.20o", "FRA"}, want: "BRST00FRA_R_20201550700_01H_30S_MO.rnx", wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Rnx3Filename(tt.args.rnx2filepath, tt.args.countryCode)
			if (err != nil) != tt.wantErr {
				t.Errorf("Rnx3Filename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Rnx3Filename() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TODO Thats not good. Create static function Rnx3Filename() instead!
/* func Test_Rnx3Filenameeee(t *testing.T) {
	rnx, err := NewFile("testdata/white/brst155h.20o")
	if err != nil {
		log.Fatalln(err)
	}

	// if v, ok := iv.(string); ok {
	// 	fmt.Printf("iv is of type string and has value '%s'\n", v)
	// 	return
	// }

	switch typ := rnx.(type) {
	case *ObsFile:
		obs, _ := rnx.(*ObsFile)
		obs.CountryCode = "FRA"
		obs.DataSource = "R"
		rnx3name, err := obs.Rnx3Filename()
		assert.NoError(t, err)
		assert.Equal(t, "BRST00FRA_R_20201550700_01H_30S_MO.rnx", rnx3name)
	case *NavFile:
		nav, _ := rnx.(*NavFile)
		nav.CountryCode = "FRA"
		nav.DataSource = "R"
	default:
		t.Fatalf("unknown type %T\n", typ)
	}

	// file.CountryCode = "FRA"
	// file.DataSource = "R"

	// rnx3name, err := file.Rnx3Filename()
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// fmt.Println(rnx3name)
} */

func TestRnx2Filename(t *testing.T) {
	tests := []struct {
		name         string
		rnx3filepath string
		want         string
		wantErr      bool
	}{
		{
			name: "Rnx3 hourly obs file", rnx3filepath: "BRUX00BEL_R_20183101900_01H_30S_MO.rnx", want: "brux310t.18o", wantErr: false,
		},
		{
			name: "Rnx3 obs Hatanaka file", rnx3filepath: "BRUX00BEL_R_20183101900_01H_30S_MO.crx", want: "brux310t.18d", wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Rnx2Filename(tt.rnx3filepath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Rnx3Filename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Rnx3Filename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDoy(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(time.Date(2001, 12, 31, 0, 0, 0, 0, time.UTC), ParseDoy(2001, 365))
	assert.Equal(time.Date(2018, 12, 5, 0, 0, 0, 0, time.UTC), ParseDoy(2018, 339))
	assert.Equal(time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC), ParseDoy(2017, 1))
	assert.Equal(time.Date(2016, 12, 31, 0, 0, 0, 0, time.UTC), ParseDoy(2016, 366))
	assert.Equal(time.Date(2016, 12, 31, 0, 0, 0, 0, time.UTC), ParseDoy(16, 366))
	assert.Equal(time.Date(1998, 1, 2, 0, 0, 0, 0, time.UTC), ParseDoy(98, 2))

	// parse Rnx3 starttime
	tests := map[string]time.Time{
		"20121601000": time.Date(2012, 6, 8, 10, 0, 0, 0, time.UTC),
		"20192681900": time.Date(2019, 9, 25, 19, 0, 0, 0, time.UTC),
		"20192660415": time.Date(2019, 9, 23, 4, 15, 0, 0, time.UTC),
	}

	for k, v := range tests {
		t, err := time.Parse(rnx3StartTimeFormat, k) // or "2006__2"
		assert.NoError(err)
		assert.Equal(t, v)
		fmt.Printf("epoch: %s\n", t)
	}
}
