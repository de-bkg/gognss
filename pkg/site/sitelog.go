package site

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var (
	// main block. e.g. '1.   Site Identification of the GNSS Monument'
	blockPattern = regexp.MustCompile(`^(\d+)\.\s+([\w\s]+)`)

	// sub block. e.g. '4.x  Antenna Type : (A20, from rcvr_ant.tab; see instructions)'
	//8.1.x Humidity Sensor Model   :
	dummyBlockPattern = regexp.MustCompile(`^(\d+\.[xX])\s+(.*)`)

	// satSysMap is a help map for checking satellite systems
	satSysMap = map[string]int{"GPS": 1, "GLO": 1, "GAL": 1, "BDS": 1, "IRNSS": 1, "QZSS": 1, "SBAS": 1}

	// timeShift used if chronological items e.g. receivers have identical start/end time.
	timeShift = time.Second

	// use a single instance of Validate, it caches struct info
	validate *validator.Validate
)

// A SiteDecoder decodes GNSS site information.
// type SiteDecoder interface {
// 	Decode(site *Site) error
// }

// A SitelogDecoder reads and decodes site information from an IGS sitelog input stream.
type SitelogDecoder struct {
	r   io.Reader
	err error

	/* 	// see  gin.Context.go for adding errors!!!!
	//
		//Error represents a error's specification.
	type Error struct {
		Err  error
		Type ErrorType
		Meta interface{}
	}

	type errorMsgs []*Error
		// Errors is a list of errors attached to all the handlers/middlewares who used this context.
		Errors errorMsgs */
}

// DecodeSitelog reads and parses the sitelog input stream and returns it as a site.
func DecodeSitelog(r io.Reader) (*Site, error) {
	var err error
	site := &Site{}
	formInfo := FormInformation{}
	ident := Identification{}
	location := Location{}
	cartesianPos := NewCartesianPosition()
	geodPos := NewGeodeticPosition()
	freq := FrequencyStandard{}
	recv := &Receiver{}
	ant := &Antenna{}
	lTies := LocalTies{}
	coll := Collocation{}
	humSensor := HumiditySensor{}
	pressSensor := PressureSensor{}
	tempSensor := TemperatureSensor{}
	watervapSensor := WaterVaporSensor{}
	localEpiEff := LocalEpisodicEffect{}
	moreInfo := MoreInformation{}
	party := Party{}

	blocks := make(map[string]interface{})
	i := 0
	blockNumber := -1 // 6
	subBlock := ""    // 6.1
	key, val := "", ""

	parseError := func() error {
		return fmt.Errorf("line %d: Block %d (%s): could not parse %q: %q (%v)", i, blockNumber, subBlock, key, val, err)
	}

	unknownKeyError := func() error {
		return fmt.Errorf("line %d: Block %d: (%s): unknown key %q", i, blockNumber, subBlock, key)
	}

	/* 	assertNotNull := func() error {
		if val == "" {
			return fmt.Errorf("line %d: Block %d: %q is null", i, blockNumber, key)
		}
	} */

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		i++
		//emptyLine := false
		line := scanner.Text()
		if len(strings.TrimSpace(line)) < 1 {
			//emptyLine = true
			key, val = "", ""
		}

		// Block
		if res := blockPattern.FindStringSubmatch(line); res != nil {
			//fmt.Printf("%s\n", res[1])
			blockNumber, err = strconv.Atoi(res[1])
			if err != nil {
				return nil, parseError()
			}
			continue
		}

		// dummy block
		if dummyBlockPattern.MatchString(line) {
			blockNumber = -1
		}
		// if idx := strings.Index(line, ".x"); idx > 0 && idx < 8 {
		// 	blockNumber = -1
		// }

		if blockNumber < 0 {
			continue
		}

		// Sub block
		/* 		if res := subBlockPattern.FindStringSubmatch(line); res != nil {
			fmt.Printf("%s\n", res[1])
			subBlockNumber = res[1]

			continue
		} */

		if idx := strings.Index(line, ":"); idx > 0 {
			if idx < 32 {
				kv := strings.SplitN(line, ":", 2)
				if len(kv) == 2 {
					//fmt.Printf("%q\n", line)
					newKey := strings.TrimSpace(kv[0])
					if newKey != "" { // keep last key in case of multiple lines
						key = newKey
					}
					val = strings.TrimSpace(kv[1])
					//fmt.Printf("%s : %s\n", key, val)
					if val == "" {
						continue
					}
				}
			} else {
				if key == "Additional Information" {
					val = line
				} else {
					fmt.Printf("could not handle line %d: %q\n", i, line)
					continue
				}
			}
		}

		// Form
		if blockNumber == 0 {
			switch key {
			case "Prepared by", "Prepared by (full name)":
				formInfo.PreparedBy = val
			case "Date Prepared":
				if formInfo.DatePrepared, err = parseDate(val); err != nil {
					return nil, parseError()
				}
			case "Report Type":
				formInfo.ReportType = val
			case "Previous Site Log":
			case "Modified/Added Sections":
			case "":
			default:
				return nil, unknownKeyError()
			}
			continue
		} else if blockNumber == 1 {

			// Site Identification

			switch key {
			case "Site Name":
				ident.Name = val
			case "Four Character ID":
				ident.FourCharacterID = val
			case "Monument Inscription":
				ident.MonumentDescription = val
			case "IERS DOMES Number":
				if len(val) != 9 {
					log.Printf("WARN: %q should have format A9: %q", key, val)
				}
				ident.DOMESNumber = val
			case "CDP Number":
				ident.CDPNumber = val
			case "Monument Description":
				ident.MonumentDescription = val
			case "Height of the Monument":
				if ident.HeightOfMonument, err = parseFloat(val); err != nil {
					log.Printf("WARN: %v", parseError())
				}
			case "Monument Foundation":
				ident.MonumentFoundation = val
			case "Foundation Depth":
				if ident.FoundationDepth, err = parseFloat(val); err != nil {
					log.Printf("WARN: %v", parseError())
				}
			case "Marker Description":
				ident.MarkerDescription = val
			case "Date Installed":
				if ident.DateInstalled, err = parseDate(val); err != nil { // CCYY-MM-DDThh:mmZ
					return nil, parseError()
				}
			case "Geologic Characteristic":
				ident.GeologicCharacteristic = val
			case "Bedrock Type":
				ident.BedrockType = val
			case "Bedrock Condition":
				ident.BedrockCondition = val
			case "Fracture Spacing":
				ident.FractureSpacing = val
			case "Fault zones nearby":
				ident.FaultZonesNearby = val
			case "Distance/activity":
				ident.DistanceActivity = ident.DistanceActivity + " " + val // multiple lines
			case "Additional Information":
				ident.Notes = addMultipleLine(ident.Notes, val)
			case "":
			default:
				return nil, unknownKeyError()
			}
			continue
		} else if blockNumber == 2 {

			// Site Location Information

			/* 			Approximate Position (ITRF)
			   			X coordinate (m)       : 4027881.628
			   			Y coordinate (m)       : 306998.537
			   			Z coordinate (m)       : 4919498.984
			   			Latitude (N is +)      : +504753.03
			   			Longitude (E is +)     : +0042130.83
			   			Elevation (m,ellips.)  : 158.3 */

			switch key {
			case "City or Town":
				location.City = val
			case "State or Province":
				location.State = val
			case "Country":
				location.Country = val
			case "Tectonic Plate":
				location.TectonicPlate = val
			case "X coordinate (m)":
				if cartesianPos.Coordinates[0], err = parseFloat(val); err != nil {
					return nil, parseError()
				}
			case "Y coordinate (m)":
				if cartesianPos.Coordinates[1], err = parseFloat(val); err != nil {
					return nil, parseError()
				}
			case "Z coordinate (m)":
				if cartesianPos.Coordinates[2], err = parseFloat(val); err != nil {
					return nil, parseError()
				}
			case "Latitude (N is +)":
				if geodPos.Coordinates[0], err = parseFloat(val); err != nil {
					log.Printf("WARN: %v", parseError())
				}
			case "Longitude (E is +)":
				if geodPos.Coordinates[1], err = parseFloat(val); err != nil {
					log.Printf("WARN: %v", parseError())
				}
			case "Elevation (m,ellips.)":
				if geodPos.Coordinates[2], err = parseFloat(val); err != nil {
					return nil, parseError()
				}
			case "Additional Information":
				location.Notes = addMultipleLine(location.Notes, val)
			case "Approximate Position":
				// normally 'Approximate Position (ITRF)'
			case "":
				// Save positions if empty line
				if cartesianPos.Type != "" && cartesianPos.Coordinates[0] != 0 {
					location.ApproximatePosition.CartesianPosition = cartesianPos
					cartesianPos = CartesianPosition{}
				}
				if geodPos.Type != "" && geodPos.Coordinates[0] != 0 {
					location.ApproximatePosition.GeodeticPosition = geodPos
					geodPos = GeodeticPosition{}
				}
			default:
				return nil, unknownKeyError()
			}
			continue
		} else if blockNumber == 3 {

			// Receivers

			if strings.HasPrefix(key, "3.") {
				recv = &Receiver{Type: val}
				// check if block is unique
				subBlock = strings.Fields(key)[0]
				if _, ok := blocks[subBlock]; ok {
					return nil, fmt.Errorf("receiver block exists twice: %q", subBlock)
				}
				blocks[subBlock] = 1
				continue
			}

			switch key {
			case "Satellite System":
				//assertNotNull()
				if recv.SatSys, err = parseSatSystems(val); err != nil {
					return nil, parseError()
				}
			case "Serial Number":
				recv.SerialNum = val
			case "Firmware Version":
				recv.Firmware = val
			case "Elevation Cutoff Setting":
				if recv.ElevationCutoff, err = parseFloat(val); err != nil {
					return nil, parseError()
				}
			case "Date Installed":
				if recv.DateInstalled, err = parseDate(val); err != nil {
					return nil, parseError()
				}
			case "Date Removed":
				if recv.DateRemoved, err = parseDate(val); err != nil {
					return nil, parseError()
				}
			case "Temperature Stabiliz.":
				recv.TemperatureStabiliz = val
			case "Additional Information":
				recv.Notes = addMultipleLine(recv.Notes, val)
			case "":
				// Save last block if empty line
				if recv.Type != "" {
					site.Receivers = append(site.Receivers, recv)
					recv = &Receiver{}
				}
			default:
				return nil, unknownKeyError()
			}
			continue
		} else if blockNumber == 4 {

			// Antennas

			if strings.HasPrefix(key, "4.") {
				ant = &Antenna{Type: val}
				// check if block is unique
				subBlock = strings.Fields(key)[0]
				if _, ok := blocks[subBlock]; ok {
					return nil, fmt.Errorf("antenna block exists twice: %q", subBlock)
				}
				blocks[subBlock] = 1

				if len(val) != 20 {
					log.Printf("WARN: Antenna Type in %s is not 20 chars long: %q", subBlock, val)
				}
				continue
			}

			switch key {
			case "Serial Number":
				ant.SerialNum = val
			case "Antenna Reference Point":
				ant.ReferencePoint = val
			case "Marker->ARP Up Ecc. (m)":
				if ant.EccUp, err = parseFloat(val); err != nil {
					return nil, parseError()
				}
			case "Marker->ARP North Ecc(m)":
				if ant.EccNorth, err = parseFloat(val); err != nil {
					return nil, parseError()
				}
			case "Marker->ARP East Ecc(m)":
				if ant.EccEast, err = parseFloat(val); err != nil {
					return nil, parseError()
				}
			case "Alignment from True N":
				if ant.AlignmentFromTrueNorth, err = parseFloat(val); err != nil {
					log.Printf("WARN: %v", parseError())
				}
			case "Antenna Radome Type":
				if len(val) != 4 {
					return nil, fmt.Errorf("Antenna Radome Type must be 4 char long: %q", val)
				}
				ant.Radome = val
			case "Radome Serial Number":
				ant.RadomeSerialNum = val
			case "Antenna Cable Type":
				ant.CableType = val
			case "Antenna Cable Length":
				if l, err := parseFloat(val); err == nil {
					ant.CableLength = float32(l)
				} else {
					log.Printf("WARN: %v", parseError())
				}
			case "Date Installed":
				if ant.DateInstalled, err = parseDate(val); err != nil {
					return nil, parseError()
				}
			case "Date Removed":
				if ant.DateRemoved, err = parseDate(val); err != nil {
					return nil, parseError()
				}
			case "Additional Information":
				ant.Notes = addMultipleLine(ant.Notes, val)
			case "":
				if ant.Type != "" {
					site.Antennas = append(site.Antennas, ant)
					ant = &Antenna{}
				}
			default:
				return nil, unknownKeyError()
			}
			continue
		} else if blockNumber == 5 {

			// Surveyed Local Ties

			if strings.HasPrefix(key, "5.") {
				lTies = LocalTies{MarkerName: val}
				// check if block is unique
				subBlock = strings.Fields(key)[0]
				if _, ok := blocks[subBlock]; ok {
					return nil, fmt.Errorf("local ties block exists twice: %q", subBlock)
				}
				blocks[subBlock] = 1
				continue
			}

			switch key {
			case "Tied Marker Usage":
				lTies.MarkerUsage = val
			case "Tied Marker CDP Number":
				lTies.MarkerCDPNumber = val
			case "Tied Marker DOMES Number", "Tied Marker Domes Number":
				lTies.MarkerDomesNumber = val
			case "dx (m)":
				if lTies.DifferentialFromMarker.Dx, err = parseFloat(val); err != nil {
					return nil, parseError()
				}
			case "dy (m)":
				if lTies.DifferentialFromMarker.Dy, err = parseFloat(val); err != nil {
					return nil, parseError()
				}
			case "dz (m)":
				if lTies.DifferentialFromMarker.Dz, err = parseFloat(val); err != nil {
					return nil, parseError()
				}
			case "Accuracy (mm)":
				if lTies.Accuracy, err = parseFloat(val); err != nil {
					log.Printf("WARN: %v", parseError())
				}
			case "Survey method":
				lTies.SurveyMethod = val
			case "Date Measured":
				if lTies.DateMeasured, err = parseDate(val); err != nil {
					log.Printf("WARN: %v", parseError())
				}
			case "Additional Information", "Additional Informations":
				lTies.Notes = addMultipleLine(lTies.Notes, val)
			case "":
				if lTies.MarkerName != "" {
					site.LocalTies = append(site.LocalTies, lTies)
					lTies = LocalTies{}
				}
			default:
				return nil, unknownKeyError()
			}
			continue
		} else if blockNumber == 6 {

			// Frequency Standard

			if strings.HasPrefix(key, "6.") {
				freq = FrequencyStandard{Type: val}
				// check if block is unique
				subBlock = strings.Fields(key)[0]
				if _, ok := blocks[subBlock]; ok {
					return nil, fmt.Errorf("Frequency Standard block exists twice: %q", subBlock)
				}
				blocks[subBlock] = 1
				continue
			}

			switch key {
			case "Input Frequency":
				freq.InputFrequency = val
			case "Effective Dates":
				if freq.EffectiveDates, err = parseEffectiveDates(val); err != nil {
					return nil, parseError()
				}
			case "Notes":
				freq.Notes = addMultipleLine(freq.Notes, val)
			case "":
				if freq.Type != "" {
					site.FrequencyStandards = append(site.FrequencyStandards, freq)
					freq = FrequencyStandard{}
				}
			default:
				return nil, unknownKeyError()
			}
			continue
		} else if blockNumber == 7 {

			// Collocation Information

			if strings.HasPrefix(key, "7.") {
				if val == "NONE" {
					subBlock = ""
					continue
				}
				coll = Collocation{InstrumentType: val}
				// check if block is unique
				subBlock = strings.Fields(key)[0]
				if _, ok := blocks[subBlock]; ok {
					return nil, fmt.Errorf("Collocation Information block exists twice: %q", subBlock)
				}
				blocks[subBlock] = 1
				continue
			}

			if subBlock == "" {
				continue
			}

			switch key {
			case "Status":
				coll.Status = val
			case "Effective Dates":
				if coll.EffectiveDates, err = parseEffectiveDates(val); err != nil {
					return nil, parseError()
				}
			case "Notes":
				coll.Notes = addMultipleLine(coll.Notes, val)
			case "":
				if coll.InstrumentType != "" {
					site.Collocations = append(site.Collocations, coll)
					coll = Collocation{}
				}
			default:
				return nil, unknownKeyError()
			}
			continue
		} else if blockNumber == 8 {

			// Meteorological Instrumentation

			if strings.HasPrefix(line, "8.") {
				subBlock = strings.Fields(key)[0]
				if strings.HasSuffix(subBlock, ".x") {
					continue
				}
				// check if block is unique
				if _, ok := blocks[subBlock]; ok {
					return nil, fmt.Errorf("Meteo Sensor block exists twice: %q", subBlock)
				}
				blocks[subBlock] = 1

				if strings.HasPrefix(subBlock, "8.1") {
					humSensor = HumiditySensor{Type: val}
				} else if strings.HasPrefix(subBlock, "8.2") {
					pressSensor = PressureSensor{Type: val}
				} else if strings.HasPrefix(key, "8.3") {
					tempSensor = TemperatureSensor{Type: val}
				} else if strings.HasPrefix(key, "8.4") {
					watervapSensor = WaterVaporSensor{Type: val}
				} else if strings.HasPrefix(key, "8.5") {
					// TODO Other instrument
				}
				continue
			}

			if strings.HasSuffix(subBlock, ".x") {
				continue
			}

			// Humidity
			if strings.HasPrefix(subBlock, "8.1.") {
				switch key {
				case "Manufacturer":
					humSensor.Manufacturer = val
				case "Serial Number":
					humSensor.SerialNumber = val
				case "Data Sampling Interval":
					if humSensor.DataSamplingInterval, err = parseFloat(val); err != nil {
						//return nil, parseError()
						log.Printf("%v", parseError())
					}
				case "Accuracy (% rel h)":
					if humSensor.Accuracy, err = parseFloat(val); err != nil {
						//return nil, parseError()
						log.Printf("%v", parseError())
					}
				case "Aspiration":
					humSensor.Aspiration = val
				case "Height Diff to Ant":
					if humSensor.HeightDiffToAntenna, err = parseFloat(val); err != nil {
						//return nil, parseError()
						log.Printf("%v", parseError())
					}
				case "Calibration date":
					if humSensor.CalibrationDate, err = parseDate(val); err != nil {
						return nil, parseError()
					}
				case "Effective Dates":
					if humSensor.EffectiveDates, err = parseEffectiveDates(val); err != nil {
						return nil, parseError()
					}
				case "Notes":
					humSensor.Notes = addMultipleLine(humSensor.Notes, val)
				case "":
					if humSensor.Type != "" {
						site.HumiditySensors = append(site.HumiditySensors, humSensor)
						humSensor = HumiditySensor{}
					}
					subBlock = ""
				default:
					return nil, unknownKeyError()
				}
			} else if strings.HasPrefix(subBlock, "8.2.") {

				// Pressure

				switch key {
				case "Manufacturer":
					pressSensor.Manufacturer = val
				case "Serial Number":
					pressSensor.SerialNumber = val
				case "Data Sampling Interval":
					if pressSensor.DataSamplingInterval, err = parseFloat(val); err != nil {
						log.Printf("%v", parseError())
					}
				case "Accuracy":
					if pressSensor.Accuracy, err = parseFloat(val); err != nil {
						log.Printf("%v", parseError())
					}
				case "Height Diff to Ant":
					if pressSensor.HeightDiffToAntenna, err = parseFloat(val); err != nil {
						log.Printf("%v", parseError())
					}
				case "Calibration date":
					if pressSensor.CalibrationDate, err = parseDate(val); err != nil {
						return nil, parseError()
					}
				case "Effective Dates":
					if pressSensor.EffectiveDates, err = parseEffectiveDates(val); err != nil {
						return nil, parseError()
					}
				case "Notes":
					pressSensor.Notes = addMultipleLine(pressSensor.Notes, val)
				case "":
					if pressSensor.Type != "" {
						site.PressureSensors = append(site.PressureSensors, pressSensor)
						pressSensor = PressureSensor{}
					}
					subBlock = ""
				default:
					return nil, unknownKeyError()
				}
			} else if strings.HasPrefix(subBlock, "8.3.") {

				// Temp. Sensor

				switch key {
				case "Manufacturer":
					tempSensor.Manufacturer = val
				case "Serial Number":
					tempSensor.SerialNumber = val
				case "Data Sampling Interval":
					if tempSensor.DataSamplingInterval, err = parseFloat(val); err != nil {
						log.Printf("%v", parseError())
					}
				case "Accuracy":
					if tempSensor.Accuracy, err = parseFloat(val); err != nil {
						log.Printf("%v", parseError())
					}
				case "Aspiration":
					tempSensor.Aspiration = val
				case "Height Diff to Ant":
					if tempSensor.HeightDiffToAntenna, err = parseFloat(val); err != nil {
						log.Printf("%v", parseError())
					}
				case "Calibration date":
					if tempSensor.CalibrationDate, err = parseDate(val); err != nil {
						return nil, parseError()
					}
				case "Effective Dates":
					if tempSensor.EffectiveDates, err = parseEffectiveDates(val); err != nil {
						return nil, parseError()
					}
				case "Notes":
					tempSensor.Notes = addMultipleLine(tempSensor.Notes, val)
				case "":
					if tempSensor.Type != "" {
						site.TemperatureSensors = append(site.TemperatureSensors, tempSensor)
						tempSensor = TemperatureSensor{}
					}
					subBlock = ""
				default:
					return nil, unknownKeyError()
				}
			} else if strings.HasPrefix(subBlock, "8.4.") {

				// Water Vapor Radiometer

				switch key {
				case "Manufacturer":
					watervapSensor.Manufacturer = val
				case "Serial Number":
					watervapSensor.SerialNumber = val
				case "Distance to Antenna":
					if watervapSensor.DistanceToAntenna, err = parseFloat(val); err != nil {
						return nil, parseError()
					}
				case "Height Diff to Ant":
					if watervapSensor.HeightDiffToAntenna, err = parseFloat(val); err != nil {
						log.Printf("%v", parseError())
					}
				case "Calibration date":
					if watervapSensor.CalibrationDate, err = parseDate(val); err != nil {
						return nil, parseError()
					}
				case "Effective Dates":
					if watervapSensor.EffectiveDates, err = parseEffectiveDates(val); err != nil {
						return nil, parseError()
					}
				case "Notes":
					watervapSensor.Notes = addMultipleLine(watervapSensor.Notes, val)
				case "":
					if watervapSensor.Type != "" {
						site.WaterVaporSensors = append(site.WaterVaporSensors, watervapSensor)
						watervapSensor = WaterVaporSensor{}
					}
					subBlock = ""
				default:
					return nil, unknownKeyError()
				}
			}
		} else if blockNumber == 10 {

			// Local Episodic Effects

			if strings.HasPrefix(key, "10.") {
				localEpiEff = LocalEpisodicEffect{}
				if localEpiEff.EffectiveDates, err = parseEffectiveDates(val); err != nil {
					return nil, parseError()
				}
				// check if block is unique
				subBlock = strings.Fields(key)[0]
				if _, ok := blocks[subBlock]; ok {
					return nil, fmt.Errorf("Local Episodic Effect block exists twice: %q", subBlock)
				}
				blocks[subBlock] = 1
				continue
			}

			switch key {
			case "Event":
				localEpiEff.Event = val
			case "":
				if localEpiEff.Event != "" {
					site.LocalEpisodicEffectLogItems = append(site.LocalEpisodicEffectLogItems, localEpiEff)
					localEpiEff = LocalEpisodicEffect{}
				}
			default:
				return nil, unknownKeyError()
			}
			continue
		} else if blockNumber == 11 || blockNumber == 12 {

			// 11. On-Site, Point of Contact Agency Information
			// 12. Responsible Agency (if different from 11.)

			if strings.HasPrefix(key, "11.") || strings.HasPrefix(key, "12.") {
				/* 				localEpiEff = LocalEpisodicEffect{}
				   				if localEpiEff.EffectiveDates, err = parseEffectiveDates(val); err != nil {
				   					return nil, parseError()
				   				}
				   				// check if block is unique
				   				subBlock = strings.Fields(key)[0]
				   				if _, ok := blocks[subBlock]; ok {
				   					return nil, fmt.Errorf("Local Episodic Effect block exists twice: %q", subBlock)
				   				}
				   				blocks[subBlock] = 1 */
				continue
			}

			if strings.Contains(line, "Secondary Contact") {
				// store Primary Contact
				if blockNumber == 11 {
					site.ResponsibleParties = append(site.ResponsibleParties, ResponsibleParty{Party: party})
				} else { // 12
					site.Contacts = append(site.Contacts, Contact{Party: party})
				}

				// reset fields from primary contact
				party.IndividualName = ""
				party.ContactInfo.Phone.Voices = []string{}
				party.ContactInfo.Phone.Facsimiles = []string{}
				party.ContactInfo.Address.EmailAddresses = []string{}
				continue
			}

			switch key {
			case "Agency":
				party.OrganisationName = val
			case "Preferred Abbreviation":
				party.Abbreviation = val
			case "Mailing Address":
				party.ContactInfo.Address.DeliveryPoints = append(party.ContactInfo.Address.DeliveryPoints, val)
			case "Contact Name":
				party.IndividualName = val
			case "Telephone (primary)":
				party.ContactInfo.Phone.Voices = append(party.ContactInfo.Phone.Voices, val)
			case "Telephone (secondary)":
				party.ContactInfo.Phone.Voices = append(party.ContactInfo.Phone.Voices, val)
			case "Fax":
				party.ContactInfo.Phone.Facsimiles = append(party.ContactInfo.Phone.Facsimiles, val)
			case "E-mail":
				party.ContactInfo.Address.EmailAddresses = append(party.ContactInfo.Address.EmailAddresses, val)
			case "":
				if party.IndividualName != "" {
					// store secondary contact
					if blockNumber == 11 {
						site.ResponsibleParties = append(site.ResponsibleParties, ResponsibleParty{Party: party})
					} else { // 12
						site.Contacts = append(site.Contacts, Contact{Party: party})
					}

					party = Party{}
				}
			default:
				//return nil, unknownKeyError()
			}
			continue
		} else if blockNumber == 13 {

			// More Information

			if strings.TrimSpace(line) == "Antenna Graphics with Dimensions" {
				// get out here
				blockNumber = -1
			}

			switch key {
			case "Primary Data Center":
				moreInfo.PrimaryDataCenter = val
			case "Secondary Data Center":
				moreInfo.SecondaryDataCenter = val
			case "URL for More Information":
				moreInfo.URLForMoreInformation = val
			case "Site Map":
				moreInfo.SiteMap = val
			case "Site Diagram":
				moreInfo.SiteDiagram = val
			case "Horizon Mask":
				moreInfo.HorizonMask = val
			case "Monument Description":
				moreInfo.MonumentDescription = val
			case "Site Pictures":
				moreInfo.SitePictures = val
			case "Additional Information":
				// Do not store the antenna graphics
				moreInfo.Notes = addMultipleLine(moreInfo.Notes, val)
			case "":
			default:
				//return nil, unknownKeyError()
			}
			continue
		}

	}
	err = scanner.Err()

	site.FormInfo = formInfo
	site.Identification = ident
	site.Location = location
	site.MoreInformation = moreInfo
	return site, nil
}

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
			log.Printf("WARN: %s %d with empty %q", item, n, "Date Installed")
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
			log.Printf("WARN: %s %d with empty %q", item, n, "Date Removed")
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
	// Dates
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

		// check date installed
		if curr.DateInstalled.IsZero() {
			log.Printf("WARN: %s %d with empty %q", item, n, "Date Installed")
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
			log.Printf("WARN: %s %d with empty %q", item, n, "Date Removed")
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
	for i, ant := range site.Antennas {
		radome1 := ""
		if len(ant.Type) == 20 {
			radome1 = ant.Type[16:]
			if radome1 != ant.Radome {
				log.Printf("WARN: Block 4.%d: Antenna Radome Type %q differs from Antenna Type: %q", i+1, ant.Radome, ant.Type)
			}
		}
	}

	return nil
}

func parseFloat(s string) (float64, error) {
	if strings.ToLower(s) == "unknown" {
		return 0, nil
	}
	//r := strings.NewReplacer("(", "",")", "","deg", "","%", "","rel h", "","sec", "","m", "")
	r := strings.NewReplacer(",", ".")
	s = r.Replace(s)

	s = strings.Trim(s, " %()acCdDeEgGhKlmMNOPrstUWw")
	// or better use regex!!
	//s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	fl, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return fl, nil
	}
	return 0, err
}

func parseDate(s string) (t time.Time, err error) {
	if strings.Contains(s, "CCYY") {
		return t, nil
	}
	if strings.Contains(s, "YYYY") {
		return t, nil
	}

	s = strings.TrimSuffix(s, "Thh:mmZ")

	s = strings.Trim(s, " ()NONE")
	if s == "" {
		return t, nil
	}

	if strings.Contains(s, "DD") {
		r := strings.NewReplacer("DD", "01")
		s = r.Replace(s)
	}

	switch len(s) {
	case 4: // 2002
		//log.Printf("DEBUG: malformed date string: %q", s)
		t, err = time.Parse("2006", s)
	case 8, 9, 10:
		t, err = time.Parse("2006-1-2", s)
	case 16:
		t, err = time.Parse("2006-1-2 15:04", s) // 2003-02-01 12:00
	case 20:
		t, err = time.Parse("2006-1-2T15:04:05Z", s)
	default:
		// CCYY-MM-DDThh:mmZ
		t, err = time.Parse("2006-1-2T15:04Z", s)
	}

	return
}

// Effective Dates        : 2018-02-01/CCYY-MM-DD
func parseEffectiveDates(s string) (effDates EffectiveDates, err error) {
	dates := strings.SplitN(s, "/", 2)
	if effDates.From, err = parseDate(dates[0]); err != nil {
		return
	}

	if len(dates) < 2 {
		return
	}
	if dates[1] == "" {
		return
	}
	if effDates.To, err = parseDate(dates[1]); err != nil {
		return
	}
	return
}

func parseSatSystems(s string) (string, error) {
	r := strings.NewReplacer("/", "+", "GLONASS", "GLO", "GALILEO", "GAL", "BEIDOU", "BDS", "NavIC", "IRNSS")
	s = r.Replace(s)

	systems := strings.Split(s, "+")
	for _, sys := range systems {
		if _, exists := satSysMap[sys]; !exists {
			return "", fmt.Errorf("Unknwon Satellite System: %s", sys)
		}
	}

	return s, nil
}

// Notes often have multiple lines
func addMultipleLine(note, newNote string) string {
	if strings.Contains(newNote, "multiple lines") {
		return note
	}

	if note != "" {

		note += " " + newNote
		return note
	}

	return newNote
}
