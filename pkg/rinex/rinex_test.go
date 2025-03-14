package rinex

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
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

func Test_parseHeaderDate(t *testing.T) {
	assert := assert.New(t)
	tests := map[string]time.Time{
		"20221109 140100":     time.Date(2022, 11, 9, 14, 1, 0, 0, time.UTC),
		"20190927 095942 UTC": time.Date(2019, 9, 27, 9, 59, 42, 0, time.UTC),
		"19-FEB-98 10:42":     time.Date(1998, 2, 19, 10, 42, 0, 0, time.UTC),
		"05-Apr-2023 11:02":   time.Date(2023, 4, 5, 11, 2, 0, 0, time.UTC),   // inofficial!
		"10-May-17 22:01:54":  time.Date(2017, 5, 10, 22, 1, 54, 0, time.UTC), // inofficial!
		"2022-11-09 14:01":    time.Date(2022, 11, 9, 14, 1, 0, 0, time.UTC),  // inofficial!
	}

	for k, v := range tests {
		epTime, err := parseHeaderDate(k)
		assert.NoError(err)
		assert.Equal(v, epTime)
		fmt.Printf("epoch: %s\n", epTime)
	}
}

func TestRnxFil_StationName(t *testing.T) {
	fil1, err := NewObsFile("BRUX00BEL_R_20183101900_01H_30S_MO.rnx")
	assert.NoError(t, err)
	assert.Equal(t, "BRUX00BEL", fil1.StationName())

	fil2, err := NewObsFile("brux310t.18o")
	assert.NoError(t, err)
	assert.Equal(t, "BRUX", fil2.StationName())
}

func Test_rinexNamedCaptures(t *testing.T) {
	rnx3NamedPattern := regexp.MustCompile(`(?i)((([A-Z0-9]{4})(\d)(\d)(?P<countrycode>[A-Z]{3})_(?P<datasource>[RSU])_((\d{4})(\d{3})(\d{2})(\d{2}))_(\d{2}[A-Z])_?(\d{2}[CZSMHDU])?_([GREJCSM][MNO]))\.(rnx|crx))\.?([a-zA-Z0-9]+)?`)
	expNames := rnx3NamedPattern.SubexpNames()
	fn := "BRUX00BEL_R_20183101900_01H_30S_MO.rnx"

	res := rnx3NamedPattern.FindStringSubmatch(fn)
	for k, v := range res {
		fmt.Printf("%d. %s\n", k, v)
	}

	md := map[string]string{}
	for i, n := range expNames {
		fmt.Printf("%d. match=%q\tvalue=%q\n", i, n, res[i])
		if n != "" {
			md[n] = res[i]
		}
	}
	fmt.Printf("%+v\n", md)
}

func TestGetCaseSensitiveName(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "t1", path: "BRUX2450.22o", want: "brux2450.22o"},
		{name: "t1-withPath", path: "/path/to/BRUX2450.22O", want: "/path/to/brux2450.22o"},
		{name: "t2", path: "brux00bel_r_20183101900_01h_30s_mo.rnx", want: "BRUX00BEL_R_20183101900_01H_30S_MO.rnx"},
		{name: "t2-withPath", path: "/path/to/brux00bel_r_20183101900_01h_30s_mo.rnx", want: "/path/to/BRUX00BEL_R_20183101900_01H_30S_MO.rnx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCaseSensitiveName(tt.path); got != tt.want {
				t.Errorf("GetCaseSensitiveName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilePeriod_Duration(t *testing.T) {
	tests := []struct {
		name string
		p    FilePeriod
		want time.Duration
	}{
		{name: "t-15M", p: FilePeriod15Min, want: 15 * time.Minute},
		{name: "t-01D", p: FilePeriodDaily, want: 24 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Duration(); got != tt.want {
				t.Errorf("FilePeriod.Duration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseSeconds(t *testing.T) {
	tests := []struct {
		name    string
		secs    string
		wantS   int64
		wantNS  int64
		wantErr bool
	}{
		{name: "t1", secs: "0.000000", wantS: 0, wantNS: 0, wantErr: false},
		{name: "t2", secs: ".000000", wantS: 0, wantNS: 0, wantErr: false},
		{name: "t3", secs: "5.000000", wantS: 5, wantNS: 0, wantErr: false},
		{name: "t4", secs: "30.500000", wantS: 30, wantNS: 500000000, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseSeconds(tt.secs)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSeconds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantS {
				t.Errorf("parseSeconds() got = %v, want %v", got, tt.wantS)
			}
			if got1 != tt.wantNS {
				t.Errorf("parseNanoSeconds() got1 = %v, want %v", got1, tt.wantNS)
			}
		})
	}
}

func Test_parseYmdHMS(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantTi  time.Time
		wantErr bool
	}{
		{name: "t1", in: "1997     5    25     0     0     .000000",
			wantTi: time.Date(1997, 5, 25, 0, 0, 0, 0, time.UTC), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTi, err := parseYmdHMS(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseYmdHMS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotTi, tt.wantTi) {
				t.Errorf("parseYmdHMS() = %v, want %v", gotTi, tt.wantTi)
			}
		})
	}
}

func Test_parseTimeFirstObs(t *testing.T) {
	tests := []struct {
		name    string
		date    string
		want    time.Time
		wantErr bool
	}{
		{name: "t1", date: "1997     5    25     0     0     .000000",
			want: time.Date(1997, 5, 25, 0, 0, 0, 0, time.UTC), wantErr: false},
		{name: "t2", date: "96     1    25     0     0     .000000",
			want: time.Date(1996, 1, 25, 0, 0, 0, 0, time.UTC), wantErr: false},
		{name: "t3", date: "10     1    25     0     0    1.000000",
			want: time.Date(2010, 1, 25, 0, 0, 1, 0, time.UTC), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimeFirstObs(tt.date)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimeFirstObs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTimeFirstObs() = %v, want %v", got, tt.want)
			}
		})
	}
}
