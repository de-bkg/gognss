// Package site handles a GNSS site with its antenna, receiver etc. including the history.
package site

import (
	"fmt"
	"strings"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
	"github.com/go-playground/validator/v10"
)

// Site specifies a GNSS site.
type Site struct {
	ID               int       `json:"id"`
	EntryDate        time.Time `json:"entryDate"`
	LastModifiedDate time.Time `json:"lastModifiedDate"`
	//SiteLogText                 time.Time                `json:"siteLogText"`
	FormInfo FormInformation `json:"formInformation"`
	Ident    Identification  `json:"siteIdentification"`
	Location Location        `json:"siteLocation"`

	Receivers []*Receiver `json:"gnssReceivers" validate:"required,min=1,dive,required"`
	Antennas  []*Antenna  `json:"gnssAntennas" validate:"required,min=1,dive,required"`

	LocalTies                   []LocalTies             `json:"surveyedLocalTies"`
	FrequencyStandards          []FrequencyStandard     `json:"frequencyStandards"`
	Collocations                []Collocation           `json:"collocationInformation"`
	HumiditySensors             []HumiditySensor        `json:"humiditySensors"`
	PressureSensors             []PressureSensor        `json:"pressureSensors"`
	TemperatureSensors          []TemperatureSensor     `json:"temperatureSensors"`
	WaterVaporSensors           []WaterVaporSensor      `json:"waterVaporSensors"`
	OtherInstrumentationLogItem []interface{}           `json:"otherInstrumentationLogItem"` // 8.5 Other Instrumentation
	RadioInterferences          []interface{}           `json:"radioInterferences"`          // 9.1
	MultipathSourceLogItems     []interface{}           `json:"multipathSourceLogItems"`     // 9.2
	SignalObstructionLogItems   []interface{}           `json:"signalObstructionLogItems"`   // 9.3
	LocalEpisodicEffectLogItems []LocalEpisodicEffect   `json:"localEpisodicEffectLogItems"` // 10
	Contacts                    []Contact               `json:"siteContacts"`                // 11. On-Site, Point of Contact Agency Information
	ResponsibleAgencies         []ResponsibleAgency     `json:"responsibleParties"`          // 12. Responsible Agency
	MoreInformation             MoreInformation         `json:"moreInformation"`             // 13
	SiteMetadataCustodians      []SiteMetadataCustodian `json:"siteMetadataCustodians"`
	//EquipmentLogItems         []EquipmentLogItems      `json:"equipmentLogItems"` // ??
	//Links                     Links                    `json:"_links"`

	Warnings []error
}

// FormInformation stores sitelog metdadata.
type FormInformation struct {
	PreparedBy   string    `json:"preparedBy"`
	DatePrepared time.Time `json:"datePrepared" validate:"required"`
	ReportType   string    `json:"reportType"` // NEW/UPDATE
	/* 	If Update:
	   	Previous Site Log       : brux00bel_20181112.log
	   	Modified/Added Sections : 3.13, 3.14 */
}

// Identification holds common fields about this site.
type Identification struct {
	Name                   string    `json:"siteName" validate:"required"` // City or nearest town
	FourCharacterID        string    `json:"fourCharacterId"`
	NineCharacterID        string    `json:"nineCharacterId"`        // or store singel fields? ID
	MonumentInscription    string    `json:"monumentInscription"`    //
	DOMESNumber            string    `json:"iersDOMESNumber"`        // IERS Domes number, A9
	CDPNumber              string    `json:"cdpNumber"`              // whats that? A4
	MonumentDescription    string    `json:"monumentDescription"`    // PILLAR/BRASS PLATE/STEEL MAST/etc
	HeightOfMonument       float64   `json:"heightOfMonument"`       // in meter?
	MonumentFoundation     string    `json:"monumentFoundation"`     // STEEL RODS, CONCRETE BLOCK, ROOF, etc
	FoundationDepth        float64   `json:"foundationDepth"`        // in meter
	MarkerDescription      string    `json:"markerDescription"`      // CHISELLED CROSS/DIVOT/BRASS NAIL/etc
	DateInstalled          time.Time `json:"dateInstalled"`          //
	GeologicCharacteristic string    `json:"geologicCharacteristic"` // BEDROCK/CLAY/CONGLOMERATE/GRAVEL/SAND/etc
	BedrockType            string    `json:"bedrockType"`            // IGNEOUS/METAMORPHIC/SEDIMENTARY  -> new type BedrockType
	BedrockCondition       string    `json:"bedrockCondition"`       // FRESH/JOINTED/WEATHERED
	FractureSpacing        string    `json:"fractureSpacing"`        // 1-10 cm/11-50 cm/51-200 cm/over 200 cm
	FaultZonesNearby       string    `json:"faultZonesNearby"`       // YES/NO/Name of the zone
	DistanceActivity       string    `json:"distanceActivity"`
	Notes                  string    `json:"notes"`
}

// Location holds information about the location.
type Location struct {
	City                string              `json:"city"`
	State               string              `json:"state"`
	Country             string              `json:"country"`
	TectonicPlate       string              `json:"tectonicPlate"`
	ApproximatePosition ApproximatePosition `json:"approximatePosition" validate:"required"` // ITRF
	Notes               string              `json:"notes"`
}

// Receiver is a GNSS receiver.
type Receiver struct {
	Type                string       `json:"type" validate:"required"`
	SatSystems          gnss.Systems `json:"satelliteSystem" validate:"required"` // Sattelite System
	SerialNum           string       `json:"serialNumber" validate:"required"`
	Firmware            string       `json:"firmwareVersion"`
	ElevationCutoff     float64      `json:"elevationCutoffSetting"`   // degree
	TemperatureStabiliz string       `json:"temperatureStabilization"` // none or tolerance in degrees C
	DateInstalled       time.Time    `json:"dateInstalled" validate:"required"`
	DateRemoved         time.Time    `json:"dateRemoved"`
	Notes               string       `json:"notes"` // Additional Information

	/* 	"dateInserted": "1999-07-31T01:00:00Z",
	   	"dateDeleted": null,
	   	"deletedReason": null,
	   	"effectiveDates": {
	   	  "from": "1999-07-31T01:00:00Z",
	   	  "to": "2000-01-14T01:50:00Z"
	   	} */
}

// Antenna is a GNSS antenna.
type Antenna struct {
	Type                   string    `json:"type" validate:"required"`
	Radome                 string    `json:"antennaRadomeType"`
	RadomeSerialNum        string    `json:"radomeSerialNumber"`
	SerialNum              string    `json:"serialNumber" validate:"required"`
	ReferencePoint         string    `json:"antennaReferencePoint"`
	EccUp                  float64   `json:"markerArpUpEcc"`
	EccNorth               float64   `json:"markerArpNorthEcc"`
	EccEast                float64   `json:"markerArpEastEcc"`
	AlignmentFromTrueNorth float64   `json:"alignmentFromTrueNorth"` // in deg; + is clockwise/east
	CableType              string    `json:"antennaCableType"`       // vendor & type number
	CableLength            float32   `json:"antennaCableLength"`     // in meter
	DateInstalled          time.Time `json:"dateInstalled"`
	DateRemoved            time.Time `json:"dateRemoved"`
	Notes                  string    `json:"notes"` // Additional Information

	/* 	"dateInserted": "2003-06-15T03:30:00Z",
	   	"dateDeleted": null,
	   	"deletedReason": null,
	   	"effectiveDates": {
	   	  "from": "2003-06-15T03:30:00Z",
	   	  "to": "2011-07-20T00:00:00Z"
	   	} */
}

// CartesianPosition is a point specified by its XYZ-coordinates.
type CartesianPosition struct {
	Type        string     `json:"type"` // "Point"
	Coordinates [3]float64 `json:"coordinates"`
}

// NewCartesianPosition inits a Cartesian Point Position.
func NewCartesianPosition() CartesianPosition {
	return CartesianPosition{Type: "Point"}
}

// GeodeticPosition is a point specified by lat,lon and ellipsoid height.
type GeodeticPosition struct {
	Type        string     `json:"type"` // "Point"
	Coordinates [3]float64 `json:"coordinates"`
}

// NewGeodeticPosition inits a Geodetic Point Position.
func NewGeodeticPosition() GeodeticPosition {
	return GeodeticPosition{Type: "Point"}
}

// ApproximatePosition stores the approximate position of the site.
type ApproximatePosition struct {
	CartesianPosition CartesianPosition `json:"cartesianPosition"`
	GeodeticPosition  GeodeticPosition  `json:"geodeticPosition"`
}

// DeltaXYZ stores deltas to a cartesian coordinate.
type DeltaXYZ struct {
	Dx float64 `json:"dx"`
	Dy float64 `json:"dy"`
	Dz float64 `json:"dz"`
}

// EffectiveDates holds a start- and enddate.
type EffectiveDates struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// LocalTies stores the surveyed local ties from one measurement.
type LocalTies struct {
	MarkerName             string    `json:"tiedMarkerName"`
	MarkerUsage            string    `json:"tiedMarkerUsage"`        // SLR/VLBI/LOCAL CONTROL/FOOTPRINT/etc
	MarkerCDPNumber        string    `json:"tiedMarkerCdpNumber"`    // A4
	MarkerDomesNumber      string    `json:"tiedMarkerDomesNumber"`  // A9
	DifferentialFromMarker DeltaXYZ  `json:"differentialFromMarker"` // in meter
	Accuracy               float64   `json:"localSiteTieAccuracy"`   // in mm
	SurveyMethod           string    `json:"surveyMethod"`           // GPS CAMPAIGN/TRILATERATION/TRIANGULATION/etc
	DateMeasured           time.Time `json:"dateMeasured"`
	Notes                  string    `json:"notes"`

	//
	/* 	DateInserted           time.Time      `json:"dateInserted"`
	   	DateDeleted            time.Time      `json:"dateDeleted"`
	   	DeletedReason          time.Time      `json:"deletedReason"` */
}

// FrequencyStandard describes the internal or external frequency input.
type FrequencyStandard struct {
	Type           string         `json:"type"`           // INTERNAL or EXTERNAL H-MASER/CESIUM/etc.
	InputFrequency string         `json:"inputFrequency"` // if external
	EffectiveDates EffectiveDates `json:"effectiveDates"`
	Notes          string         `json:"notes"`

	/* 	DateInserted   time.Time      `json:"dateInserted"`
	   	DateDeleted    time.Time   `json:"dateDeleted"`
	   	DeletedReason  time.Time    `json:"deletedReason"`
	   	SerialNumber   interface{}    `json:"serialNumber"` */
}

// Collocation describes collocation instruments.
type Collocation struct {
	InstrumentType string         `json:"instrumentType"` // GPS/GLONASS/DORIS/PRARE/SLR/VLBI/TIME/etc
	Status         string         `json:"status"`         // PERMANENT/MOBILE
	EffectiveDates EffectiveDates `json:"effectiveDates"`
	Notes          string         `json:"notes"`

	/* 	DateInserted   time.Time      `json:"dateInserted"`
	   	DateDeleted    time.Time   `json:"dateDeleted"`
	   	DeletedReason  time.Time   `json:"deletedReason"` */
}

// HumiditySensor specifies a humidity sensor.
type HumiditySensor struct {
	Type                 string         `json:"type"` // Humidity Sensor Model
	Manufacturer         string         `json:"manufacturer"`
	SerialNumber         string         `json:"serialNumber"`
	DataSamplingInterval float64        `json:"dataSamplingInterval"`            // in secs
	Accuracy             float64        `json:"accuracyPercentRelativeHumidity"` // in % relative humidity
	Aspiration           string         `json:"aspiration"`                      // UNASPIRATED, NATURAL, FAN etc.
	HeightDiffToAntenna  float64        `json:"heightDiffToAntenna"`             // in meter
	CalibrationDate      time.Time      `json:"calibrationDate"`
	EffectiveDates       EffectiveDates `json:"effectiveDates"`
	Notes                string         `json:"notes"`

	/* 	DateInserted  time.Time `json:"dateInserted"`
	   	DateDeleted   time.Time `json:"dateDeleted"`
		DeletedReason string    `json:"deletedReason"` */
}

// PressureSensor specifies a pressure sensor.
type PressureSensor struct {
	Type                 string         `json:"type"` // Pressure Sensor Model
	Manufacturer         string         `json:"manufacturer"`
	SerialNumber         string         `json:"serialNumber"`
	DataSamplingInterval float64        `json:"dataSamplingInterval"` // in secs
	Accuracy             float64        `json:"accuracyHPa"`          // in hPa
	HeightDiffToAntenna  float64        `json:"heightDiffToAntenna"`  // in meter
	CalibrationDate      time.Time      `json:"calibrationDate"`
	EffectiveDates       EffectiveDates `json:"effectiveDates"`
	Notes                string         `json:"notes"`
}

// TemperatureSensor specifies a temperature sensor.
type TemperatureSensor struct {
	Type                 string         `json:"type"` // Pressure Sensor Model
	Manufacturer         string         `json:"manufacturer"`
	SerialNumber         string         `json:"serialNumber"`
	DataSamplingInterval float64        `json:"dataSamplingInterval"`   // in secs
	Accuracy             float64        `json:"accuracyDegreesCelcius"` // in degrees
	Aspiration           string         `json:"aspiration"`             // UNASPIRATED, NATURAL, FAN etc.
	HeightDiffToAntenna  float64        `json:"heightDiffToAntenna"`    // in meter
	CalibrationDate      time.Time      `json:"calibrationDate"`
	EffectiveDates       EffectiveDates `json:"effectiveDates"`
	Notes                string         `json:"notes"`
}

// WaterVaporSensor specifies a water-vapor sensor.
type WaterVaporSensor struct {
	Type                string         `json:"type"`
	Manufacturer        string         `json:"manufacturer"`
	SerialNumber        string         `json:"serialNumber"`
	DistanceToAntenna   float64        `json:"distanceToAntenna"`
	HeightDiffToAntenna float64        `json:"heightDiffToAntenna"`
	CalibrationDate     time.Time      `json:"calibrationDate"`
	EffectiveDates      EffectiveDates `json:"effectiveDates"`
	Notes               string         `json:"notes"`
}

// LocalEpisodicEffect is a local episodic effect that possibly affects data quality, defined in 10.
type LocalEpisodicEffect struct {
	EffectiveDates EffectiveDates `json:"effectiveDates"`
	Event          string         `json:"event"` // TREE CLEARING/CONSTRUCTION/etc
}

// MoreInformation about data centers, pictures etc., sitelog block 13
type MoreInformation struct {
	PrimaryDataCenter             string `json:"primaryDataCenter"`
	SecondaryDataCenter           string `json:"secondaryDataCenter"`
	URLForMoreInformation         string `json:"urlForMoreInformation"`
	SiteMap                       string `json:"siteMap"`
	SiteDiagram                   string `json:"siteDiagram"`
	HorizonMask                   string `json:"horizonMask"`
	MonumentDescription           string `json:"monumentDescription"`
	SitePictures                  string `json:"sitePictures"`
	Notes                         string `json:"notes"`
	AntennaGraphicsWithDimensions string `json:"antennaGraphicsWithDimensions"`
	InsertTextGraphicFromAntenna  string `json:"insertTextGraphicFromAntenna"`
	Doi                           string `json:"doi"`
}

// Contacts

// Standard is a no idea what.
type Standard struct {
}

// Address stores an address. It's not possible to parse that information from a sitelog.
type Address struct {
	PostalCode         string   `json:"postalCode"`
	City               string   `json:"city"`
	Country            string   `json:"country"`
	AdministrativeArea string   `json:"administrativeArea"` // Bundesland
	DeliveryPoints     []string `json:"deliveryPoints"`     // Postfach?
	EmailAddresses     []string `json:"electronicMailAddresses" validate:"dive,email"`
	Standard           Standard `json:"standard"`   // ?
	Modifiable         bool     `json:"modifiable"` // ?
	Interface          string   `json:"interface"`  // "org.opengis.metadata.citation.ResponsibleParty"
}

// Phone stores phone and facsimile numbers of a contact.
type Phone struct {
	Voices     []string `json:"voices"`
	Facsimiles []string `json:"facsimiles"`
}

// ContactInfo stores the address, phones etc. of an party/organisation.
type ContactInfo struct {
	ContactInstructions interface{} `json:"contactInstructions"`
	HoursOfService      interface{} `json:"hoursOfService"`
	OnLineResource      interface{} `json:"onLineResource"`
	Address             Address     `json:"address"`
	Phone               Phone       `json:"phone"`
	Standard            Standard    `json:"standard"`
	Modifiable          bool        `json:"modifiable"`
	Interface           string      `json:"interface"` // "org.opengis.metadata.citation.Contact"
}
type Role struct {
}

// Party describes an organisation with contacts, addresses included.
type Party struct {
	IndividualName   string      `json:"individualName"`
	OrganisationName string      `json:"organisationName"` // abbreviation
	Abbreviation     string      `json:"abbreviation"`     // additional field added by wiese
	PositionName     string      `json:"positionName"`     // ?
	ContactInfo      ContactInfo `json:"contactInfo"`
	Role             Role        `json:"role"`
	Standard         Standard    `json:"standard"`
	Modifiable       bool        `json:"modifiable"`
	Interface        string      `json:"interface"`
}

// ResponsibleAgency is the responsible agency.
type ResponsibleAgency struct {
	ContactTypeID int   `json:"contactTypeId"`
	Party         Party `json:"party"`
}

// Contact is the on-site point of contact.
type Contact struct {
	ContactTypeID int   `json:"contactTypeId"`
	Party         Party `json:"party"`
}
type SiteMetadataCustodian struct {
	ContactTypeID int   `json:"contactTypeId"`
	Party         Party `json:"party"`
}

// GA - what's this for?
/* type EquipmentLogItems struct {
	DateInserted                    time.Time      `json:"dateInserted"`
	DateDeleted                     interface{}    `json:"dateDeleted"`
	DeletedReason                   interface{}    `json:"deletedReason"`
	Type                            string         `json:"type"`
	SatelliteSystem                 string         `json:"satelliteSystem,omitempty"`
	SerialNumber                    string         `json:"serialNumber"`
	FirmwareVersion                 string         `json:"firmwareVersion,omitempty"`
	ElevationCutoffSetting          string         `json:"elevationCutoffSetting,omitempty"`
	DateInstalled                   time.Time      `json:"dateInstalled,omitempty"`
	DateRemoved                     time.Time      `json:"dateRemoved,omitempty"`
	TemperatureStabilization        interface{}    `json:"temperatureStabilization,omitempty"`
	Notes                           interface{}    `json:"notes"`
	EffectiveDates                  EffectiveDates `json:"effectiveDates"`
	AntennaReferencePoint           string         `json:"antennaReferencePoint,omitempty"`
	MarkerArpUpEcc                  float64        `json:"markerArpUpEcc,omitempty"`
	MarkerArpNorthEcc               float64        `json:"markerArpNorthEcc,omitempty"`
	MarkerArpEastEcc                float64        `json:"markerArpEastEcc,omitempty"`
	AlignmentFromTrueNorth          string         `json:"alignmentFromTrueNorth,omitempty"`
	AntennaRadomeType               string         `json:"antennaRadomeType,omitempty"`
	RadomeSerialNumber              string         `json:"radomeSerialNumber,omitempty"`
	AntennaCableType                string         `json:"antennaCableType,omitempty"`
	AntennaCableLength              string         `json:"antennaCableLength,omitempty"`
	Manufacturer                    string         `json:"manufacturer,omitempty"`
	HeightDiffToAntenna             interface{}    `json:"heightDiffToAntenna,omitempty"`
	CalibrationDate                 interface{}    `json:"calibrationDate,omitempty"`
	DataSamplingInterval            string         `json:"dataSamplingInterval,omitempty"`
	AccuracyPercentRelativeHumidity string         `json:"accuracyPercentRelativeHumidity,omitempty"`
	Aspiration                      string         `json:"aspiration,omitempty"`
	InputFrequency                  string         `json:"inputFrequency,omitempty"`
} */

type Self struct {
	Href string `json:"href"`
}
type SiteLog struct {
	Href      string `json:"href"`
	Templated bool   `json:"templated"`
}
type Links struct {
	Self    Self    `json:"self"`
	SiteLog SiteLog `json:"siteLog"`
}

// use a single instance of Validate, it caches struct info
var validate *validator.Validate

// Validate validates the site data.
// As often having lousy input, the values are cleaned as much as possible before, missing fields e.g. dates are set if possible.
func (site *Site) Validate() error {
	err := site.cleanReceivers()
	if err != nil {
		return err
	}

	err = site.cleanAntennas()
	if err != nil {
		return err
	}

	validate = validator.New()
	return validate.Struct(site)
}

func (site *Site) cleanReceivers() error {
	// Dates
	item := "receiver"
	list := site.Receivers
	nReceivers := len(list)
	for i, curr := range site.Receivers {
		n := i + 1 // receiver number

		prev := func() *Receiver {
			if i-1 >= 0 {
				return list[i-1]
			}
			return nil
		}
		next := func() *Receiver {
			if n+1 <= nReceivers {
				return list[i+1]
			}
			return nil
		}

		// check date installed
		if curr.DateInstalled.IsZero() {
			site.Warnings = append(site.Warnings, fmt.Errorf("%s %d with empty %q", item, n, "Date Installed"))
			if prev() == nil { // first one
				return fmt.Errorf("%s %d with empty %q", item, n, "Date Installed")
			}

			if prev().DateRemoved.IsZero() {
				return fmt.Errorf("Empty %q from %s %d could not be corrected", "Date Installed", item, n)
			}

			curr.DateInstalled = prev().DateRemoved.Add(timeShift)
		}

		// check date removed
		if curr.DateRemoved.IsZero() && next() != nil {
			site.Warnings = append(site.Warnings, fmt.Errorf("%s %d with empty %q", item, n, "Date Removed"))
			nextRecv := next()
			if nextRecv.DateInstalled.IsZero() {
				return fmt.Errorf("Empty %q from %s %d could not be corrected", "Date Removed", item, n)
			}

			curr.DateRemoved = nextRecv.DateInstalled.Add(timeShift * -1)
		}

		if prev() != nil {
			prevRecv := prev()
			if prevRecv.DateRemoved.After(curr.DateInstalled) {
				return fmt.Errorf("%s %d and %d are not chronological", item, n-1, n)
			} else if prevRecv.DateRemoved.Equal(curr.DateInstalled) {
				// dates must be unique, so we introduce a small shift
				prevRecv.DateRemoved = prevRecv.DateRemoved.Add(timeShift * -1)
			}
		}
	}

	// Other checks
	/* 	for i, recv := range site.Receivers {
		if err := recv.validate(); err != nil {
			return fmt.Errorf("Block 3.%d: %v", i+1, err)
		}
	} */

	return nil
}

func (site *Site) cleanAntennas() error {
	item := "antenna"
	list := site.Antennas
	nAntennas := len(list)
	for i, curr := range site.Antennas {
		n := i + 1 // antenna number

		prev := func() *Antenna {
			if i-1 >= 0 {
				return list[i-1]
			}
			return nil
		}
		next := func() *Antenna {
			if n+1 <= nAntennas {
				return list[i+1]
			}
			return nil
		}

		// ANT TYPE should be 20 char long
		if len(curr.Type) != 20 {
			parts := strings.Fields(curr.Type)
			if len(parts) == 2 && len(parts[1]) == 4 {
				curr.Type = fmt.Sprintf("%-15s %4s", parts[0], parts[1])
				if curr.Radome == "" {
					curr.Radome = parts[1]
				} else {
					if curr.Radome != parts[1] {
						return fmt.Errorf("%s %d Antenna Radome Type %q differs from Antenna Type %q", item, n, curr.Radome, curr.Type)
					}
				}
			} else if len(parts) == 1 && curr.Radome != "" {
				curr.Type = fmt.Sprintf("%-15s %4s", parts[0], curr.Radome)
			}
		}

		// check date installed
		if curr.DateInstalled.IsZero() {
			site.Warnings = append(site.Warnings, fmt.Errorf("%s %d with empty %q", item, n, "Date Installed"))
			if prev() == nil { // first one
				return fmt.Errorf("%s %d with empty %q", item, n, "Date Installed")
			}

			if prev().DateRemoved.IsZero() {
				return fmt.Errorf("Empty %q from %s %d could not be corrected", "Date Installed", item, n)
			}

			curr.DateInstalled = prev().DateRemoved.Add(timeShift)
		}

		// check date removed
		if curr.DateRemoved.IsZero() && next() != nil {
			site.Warnings = append(site.Warnings, fmt.Errorf("%s %d with empty %q", item, n, "Date Removed"))
			nextRecv := next()
			if nextRecv.DateInstalled.IsZero() {
				return fmt.Errorf("Empty %q from %s %d could not be corrected", "Date Removed", item, n)
			}

			curr.DateRemoved = nextRecv.DateInstalled.Add(timeShift * -1)
		}

		if prev() != nil {
			prevRecv := prev()
			if prevRecv.DateRemoved.After(curr.DateInstalled) {
				return fmt.Errorf("%s %d and %d are not chronological", item, n-1, n)
			} else if prevRecv.DateRemoved.Equal(curr.DateInstalled) {
				// dates must be unique, so we introduce a small shift
				prevRecv.DateRemoved = prevRecv.DateRemoved.Add(timeShift * -1)
			}
		}
	}

	return nil
}
