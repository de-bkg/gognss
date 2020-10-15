package site

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
	"github.com/stretchr/testify/assert"
)

func TestEncodeSitelog(t *testing.T) {
	s := &Site{}
	s.FormInfo = FormInformation{PreparedBy: "Kevin De Bruyne", DatePrepared: time.Date(2020, 02, 25, 0, 0, 0, 0, time.UTC), ReportType: "UPDATE"}

	s.Location = Location{City: "Brussels", State: "Brabant", Country: "Belgium", TectonicPlate: "EURASIAN",
		ApproximatePosition: ApproximatePosition{CartesianPosition: CartesianPosition{Type: "Point", Coordinates: [3]float64{4027881.628, 306998.537, 4919498.984}},
			GeodeticPosition: GeodeticPosition{Type: "Point", Coordinates: [3]float64{504753.03, 42130.83, 158.3}}}, Notes: ""}

	s.Ident = Identification{Name: "Brussels", FourCharacterID: "BRUX", NineCharacterID: "", MonumentInscription: "", DOMESNumber: "13101M010",
		CDPNumber: "", MonumentDescription: "STEEL MAST", HeightOfMonument: 8, MonumentFoundation: "CONCRETE BLOCK", FoundationDepth: 3,
		MarkerDescription: "CENTER OF HOLE IN STEEL PLATE", DateInstalled: time.Date(2006, 07, 07, 0, 0, 0, 0, time.UTC),
		GeologicCharacteristic: "SAND", BedrockType: "SEDIMENTARY", BedrockCondition: "FRESH", FractureSpacing: "0 cm",
		FaultZonesNearby: "NO", DistanceActivity: "", Notes: ""}

	s.Receivers = append(s.Receivers, &Receiver{Type: "SEPT POLARX2", SatSystems: gnss.Systems{gnss.SysGPS}, SerialNum: "1436", Firmware: "2.6.2",
		ElevationCutoff: 0, TemperatureStabiliz: "", DateInstalled: time.Date(2006, 07, 07, 0, 0, 0, 0, time.UTC),
		DateRemoved: time.Date(2008, 02, 14, 9, 0, 0, 0, time.UTC), Notes: "hardware replacement of receiver with SN 1128, same receiver, but different serial number (now 1436)"},
		&Receiver{Type: "SEPT POLARX5TR", SatSystems: gnss.Systems{gnss.SysGPS, gnss.SysGLO, gnss.SysGAL, gnss.SysBDS, gnss.SysQZSS, gnss.SysIRNSS, gnss.SysSBAS}, SerialNum: "3057609", Firmware: "5.3.2",
			ElevationCutoff: 0, TemperatureStabiliz: "20.1 +/- 0.2", DateInstalled: time.Date(2020, 02, 25, 13, 30, 0, 0, time.UTC),
			DateRemoved: time.Date(0001, 01, 01, 0, 0, 0, 0, time.UTC), Notes: ""})

	s.Antennas = append(s.Antennas, &Antenna{Type: "ASH701945E_M    NONE", Radome: "NONE", RadomeSerialNum: "", SerialNum: "CR620023301", ReferencePoint: "BPA", EccUp: 0.1266, EccNorth: 0.001, EccEast: 0, AlignmentFromTrueNorth: 0,
		CableType: "ANDREW heliax LDF2-50A", CableLength: 60, DateInstalled: time.Date(2006, 07, 07, 0, 0, 0, 0, time.UTC),
		DateRemoved: time.Date(2008, 03, 19, 8, 45, 0, 0, time.UTC), Notes: ""},
		&Antenna{Type: "JAVRINGANT_DM   NONE", Radome: "NONE", RadomeSerialNum: "", SerialNum: "00464", ReferencePoint: "BPA",
			EccUp: 0.4689, EccNorth: 0.001, EccEast: 0, AlignmentFromTrueNorth: 0, CableType: "ANDREW heliax LDF2-50A", CableLength: 60,
			DateInstalled: time.Date(2018, 02, 01, 8, 15, 0, 0, time.UTC), DateRemoved: time.Date(0001, 01, 01, 0, 0, 0, 0, time.UTC),
			Notes: "To shield the antenna from reflections on the dome below it, a 0.8x0.8 m^2 metal shield was installed with Eccosorb ANW-77 on top. The spacing between the ARP and the top of the Eccosorb ANW-77 is 17.8 cm. On 2015-12-09, the Eccosorb was replaced by a new more resistant version. Surge protection device, dly: L2 553, L1 525 ps."})

	// Contact
	party := Party{IndividualName: "Kevin De Bruyne", OrganisationName: "FC Chelsea", Abbreviation: "Chels"}
	party.ContactInfo.Address.EmailAddresses = append(party.ContactInfo.Address.EmailAddresses, "kevin@chelsea.uk")
	s.Contacts = append(s.Contacts, Contact{Party: party})

	//w := &bytes.Buffer{}
	w := os.Stdout
	err := EncodeSitelog(w, s)
	assert.NoError(t, err)
}

func TestReadSitelog(t *testing.T) {
	assert := assert.New(t)

	f, err := os.Open("testdata/brux_20200225.log")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer f.Close()

	site, err := DecodeSitelog(f)
	assert.NoError(err)

	// Form information
	fInfo := FormInformation{PreparedBy: "Kevin De Bruyne", DatePrepared: time.Date(2020, 02, 25, 0, 0, 0, 0, time.UTC), ReportType: "UPDATE"}
	assert.Equal(fInfo, site.FormInfo)

	// Site Identification
	ident := Identification{Name: "Brussels", FourCharacterID: "BRUX", NineCharacterID: "", MonumentInscription: "", DOMESNumber: "13101M010",
		CDPNumber: "", MonumentDescription: "STEEL MAST", HeightOfMonument: 8, MonumentFoundation: "CONCRETE BLOCK", FoundationDepth: 3,
		MarkerDescription: "CENTER OF HOLE IN STEEL PLATE", DateInstalled: time.Date(2006, 07, 07, 0, 0, 0, 0, time.UTC),
		GeologicCharacteristic: "SAND", BedrockType: "SEDIMENTARY", BedrockCondition: "FRESH", FractureSpacing: "0 cm",
		FaultZonesNearby: "NO", DistanceActivity: "", Notes: ""}
	assert.Equal(ident, site.Ident)

	// Site location
	location := Location{City: "Brussels", State: "Brabant", Country: "Belgium", TectonicPlate: "EURASIAN",
		ApproximatePosition: ApproximatePosition{CartesianPosition: CartesianPosition{Type: "Point", Coordinates: [3]float64{4027881.628, 306998.537, 4919498.984}},
			GeodeticPosition: GeodeticPosition{Type: "Point", Coordinates: [3]float64{504753.03, 42130.83, 158.3}}}, Notes: ""}
	assert.Equal(location, site.Location)

	// Receiver
	assert.Len(site.Receivers, 14, "number of receivers")
	recvFirst := Receiver{Type: "SEPT POLARX2", SatSystems: gnss.Systems{gnss.SysGPS}, SerialNum: "1436",
		Firmware: "2.6.2", ElevationCutoff: 0, TemperatureStabiliz: "", DateInstalled: time.Date(2006, 07, 07, 0, 0, 0, 0, time.UTC),
		DateRemoved: time.Date(2008, 02, 14, 9, 0, 0, 0, time.UTC), Notes: "hardware replacement of receiver with SN 1128, same receiver, but different serial number (now 1436)"}
	recvLast := Receiver{Type: "SEPT POLARX5TR", SatSystems: gnss.Systems{gnss.SysGPS, gnss.SysGLO, gnss.SysGAL, gnss.SysBDS, gnss.SysQZSS, gnss.SysIRNSS, gnss.SysSBAS},
		SerialNum: "3057609", Firmware: "5.3.2", ElevationCutoff: 0, TemperatureStabiliz: "20.1 +/- 0.2",
		DateInstalled: time.Date(2020, 02, 25, 13, 30, 0, 0, time.UTC), DateRemoved: time.Date(0001, 01, 01, 0, 0, 0, 0, time.UTC), Notes: ""}
	assert.Equal(recvFirst, *site.Receivers[0], "first receiver")
	assert.Equal(recvLast, *site.Receivers[len(site.Receivers)-1], "last receiver")
	assert.True(site.Receivers[10].DateRemoved.IsZero(), "Date Removed from receiver 3.11 not set")

	// Antennas
	assert.Len(site.Antennas, 9, "number of antennas")
	antFirst := Antenna{Type: "ASH701945E_M    NONE", Radome: "NONE", RadomeSerialNum: "",
		SerialNum: "CR620023301", ReferencePoint: "BPA", EccUp: 0.1266, EccNorth: 0.001, EccEast: 0, AlignmentFromTrueNorth: 0,
		CableType: "ANDREW heliax LDF2-50A", CableLength: 60, DateInstalled: time.Date(2006, 07, 07, 0, 0, 0, 0, time.UTC),
		DateRemoved: time.Date(2008, 03, 19, 8, 45, 0, 0, time.UTC), Notes: ""}
	antLast := Antenna{Type: "JAVRINGANT_DM   NONE", Radome: "NONE",
		RadomeSerialNum: "", SerialNum: "00464", ReferencePoint: "BPA", EccUp: 0.4689,
		EccNorth: 0.001, EccEast: 0, AlignmentFromTrueNorth: 0, CableType: "ANDREW heliax LDF2-50A", CableLength: 60,
		DateInstalled: time.Date(2018, 02, 01, 8, 15, 0, 0, time.UTC), DateRemoved: time.Date(0001, 01, 01, 0, 0, 0, 0, time.UTC),
		Notes: "To shield the antenna from reflections on the dome below it, a 0.8x0.8 m^2 metal shield was installed with Eccosorb ANW-77 on top. The spacing between the ARP and the top of the Eccosorb ANW-77 is 17.8 cm. On 2015-12-09, the Eccosorb was replaced by a new more resistant version. Surge protection device, dly: L2 553, L1 525 ps."}
	assert.Equal(antFirst, *site.Antennas[0], "first antenna")
	assert.Equal(antLast, *site.Antennas[len(site.Antennas)-1], "last antenna")
	assert.True(site.Antennas[2].DateInstalled.IsZero(), "Date Installed from antenna 4.3 not set")

	// Local ties
	assert.Len(site.LocalTies, 3, "number of local ties")
	locTies := LocalTies{MarkerName: "BRUS", MarkerUsage: "IGS and EPN station", MarkerCDPNumber: "",
		MarkerDomesNumber: "13101M004", DifferentialFromMarker: DeltaXYZ{Dx: 12.164, Dy: 47.324, Dz: -23.739}, Accuracy: 3,
		SurveyMethod: "TRIANGULATION", DateMeasured: time.Date(2012, 03, 23, 0, 0, 0, 0, time.UTC), Notes: "accuracy = 1 sigma"}
	assert.Equal(locTies, site.LocalTies[0], "first local ties")

	// Frequency Standard
	assert.Len(site.FrequencyStandards, 11, "number of Frequency Standards")
	frq := FrequencyStandard{Type: "EXTERNAL H-MASER", InputFrequency: "10 MHz",
		EffectiveDates: EffectiveDates{From: time.Date(2006, 07, 07, 0, 0, 0, 0, time.UTC), To: time.Date(2010, 12, 21, 0, 0, 0, 0, time.UTC)}, Notes: ""}
	assert.Equal(frq, site.FrequencyStandards[0], "first Frequency Standard 6.1")

	// Collocation
	assert.Len(site.Collocations, 2, "number of Collocations")
	coll1 := Collocation{InstrumentType: "CRYOGENIC GRAVIMETER", Status: "PERMANENT",
		EffectiveDates: EffectiveDates{From: time.Date(1993, 10, 20, 0, 0, 0, 0, time.UTC), To: time.Date(2000, 8, 21, 0, 0, 0, 0, time.UTC)}, Notes: ""}
	assert.Equal(coll1, site.Collocations[0], "first collocation 7.1")

	// Meteo Sensors
	assert.Len(site.HumiditySensors, 1, "number of Humidity Sensors")
	assert.Len(site.PressureSensors, 1, "number of Pressure Sensors")
	assert.Len(site.TemperatureSensors, 1, "number of Temperature Sensors")
	assert.Len(site.WaterVaporSensors, 1, "number of Water Vapor Sensors")
	pressSens := PressureSensor{Type: "WXTPTU", Manufacturer: "Vaisala", SerialNumber: "J2420010", DataSamplingInterval: 10,
		Accuracy: 1, HeightDiffToAntenna: -0.5, CalibrationDate: time.Date(2013, 6, 14, 0, 0, 0, 0, time.UTC),
		EffectiveDates: EffectiveDates{From: time.Date(2018, 8, 23, 0, 0, 0, 0, time.UTC), To: time.Date(0001, 1, 1, 0, 0, 0, 0, time.UTC)},
		Notes:          "Vaisala WXT520 SN: J2440011 Pressure Sensor is 0.5 m under the antenna capacitive silicon BAROCAP sensor"}
	wvr := WaterVaporSensor{Type: "CTGR129502", Manufacturer: "Captec, Bern / ETH, Zuerich", SerialNumber: "",
		DistanceToAntenna: 27, HeightDiffToAntenna: 1, CalibrationDate: time.Date(0001, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectiveDates: EffectiveDates{From: time.Date(1997, 4, 15, 0, 0, 0, 0, time.UTC), To: time.Date(0001, 1, 1, 0, 0, 0, 0, time.UTC)}, Notes: ""}
	assert.Equal(pressSens, site.PressureSensors[0], "Pressure Sensor")
	assert.Equal(wvr, site.WaterVaporSensors[0], "WaterVapor Sensor")

	assert.Len(site.LocalEpisodicEffectLogItems, 1, "number of Local Episodic Effects")

	// 13. More Info
	moreInf := MoreInformation{PrimaryDataCenter: "ROB", SecondaryDataCenter: "BKG", URLForMoreInformation: "", SiteMap: "", SiteDiagram: "", HorizonMask: "", MonumentDescription: "",
		SitePictures: "", Notes: "", AntennaGraphicsWithDimensions: "", InsertTextGraphicFromAntenna: "", Doi: ""}
	assert.Equal(moreInf, site.MoreInformation, "More Information")

	log.Printf("%+v", site)

	// Clean data
	err = site.ValidateAndClean(false)
	assert.NoError(err)

	assert.Equal(time.Date(2017, 3, 19, 23, 59, 59, 0, time.UTC), site.Receivers[10].DateRemoved, "Set Date Removed from receiver 3.11")
	assert.Equal(time.Date(2008, 6, 17, 9, 0, 1, 0, time.UTC), site.Antennas[2].DateInstalled, "Set Date Installed from antenna 4.3")
}

func Test_parseDate(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]time.Time{
		"2018-08-23":           time.Date(2018, 8, 23, 0, 0, 0, 0, time.UTC),
		"2018-02-01T08:15Z":    time.Date(2018, 2, 1, 8, 15, 0, 0, time.UTC),
		"2003-02-01 12:00":     time.Date(2003, 2, 1, 12, 0, 0, 0, time.UTC),
		"2014-11-13T09:50:00Z": time.Date(2014, 11, 13, 9, 50, 0, 0, time.UTC),
		"1991-07-22Thh:mmZ":    time.Date(1991, 7, 22, 0, 0, 0, 0, time.UTC),
		"2009-9-29":            time.Date(2009, 9, 29, 0, 0, 0, 0, time.UTC),
		"1999-04-DD":           time.Date(1999, 4, 1, 0, 0, 0, 0, time.UTC),
		"2002":                 time.Date(2002, 1, 1, 0, 0, 0, 0, time.UTC),
		"CCYY-MM-DD":           time.Date(0001, 1, 1, 0, 0, 0, 0, time.UTC),
		"YYYY-MM-DDThh:mmZ":    time.Date(0001, 1, 1, 0, 0, 0, 0, time.UTC),
		"NONE":                 time.Date(0001, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	for k, v := range tests {
		ti, err := parseDate(k) // or "2006__2"
		assert.NoError(err)
		assert.Equal(v, ti)
		//t.Logf("epoch: %s\n", ti)
	}
}

func Test_parseEffectiveDates(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]EffectiveDates{
		"2018-02-01/CCYY-MM-DD": {time.Date(2018, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(0001, 1, 1, 0, 0, 0, 0, time.UTC)},
		"2018-02-01/":           {time.Date(2018, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(0001, 1, 1, 0, 0, 0, 0, time.UTC)},
		"2007-01-31":            {time.Date(2007, 1, 31, 0, 0, 0, 0, time.UTC), time.Date(0001, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	for k, v := range tests {
		effDates, err := parseEffectiveDates(k) // or "2006__2"
		assert.NoError(err)
		assert.Equal(v, effDates)
		//t.Logf("epoch: %s\n", ti)
	}
}
