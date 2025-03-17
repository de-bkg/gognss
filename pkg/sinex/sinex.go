// Package sinex for reading SINEX files.
// Format description is available at https://www.iers.org/IERS/EN/Organization/AnalysisCoordinator/SinexFormat/sinex.html.

package sinex

import (
	"iter"
	"time"

	"github.com/de-bkg/gognss/pkg/gnss"
)

type Blockname = string

const (
	BlockFileReference     Blockname = "FILE/REFERENCE"                  // Information on the Organization, point of contact, etc.
	BlockFileComment       Blockname = "FILE/COMMENT"                    // General comments about the SINEX data file.
	BlockInputHistory      Blockname = "INPUT/HISTORY"                   // Information about the source of the information used to create the current SINEX file.
	BlockInputFiles        Blockname = "INPUT/FILES"                     // Source data files.
	BlockInputAck          Blockname = "INPUT/ACKNOWLEDGEMENTS"          // Defines the agency codes contributing to the SINEX file.
	BlockNutationData      Blockname = "NUTATION/DATA"                   // VLBI: contains the nutation model used in the analysis procedure.
	BlockPrecessionData    Blockname = "PRECESSION/DATA"                 // VLBI: contains the precession model used in the analysis procedure.
	BlockSourceID          Blockname = "SOURCE/ID"                       // VLBI: contains information about the radio sources estimated in the analysis.
	BlockSiteID            Blockname = "SITE/ID"                         // General information for each site containing estimated parameters.
	BlockSiteData          Blockname = "SITE/DATA"                       // Relationship between the estimated station parameters and in the input files.
	BlockSiteReceiver      Blockname = "SITE/RECEIVER"                   // GNSS: The receiver used at each site during the observation period.
	BlockSiteAntenna       Blockname = "SITE/ANTENNA"                    // GNSS: The antennas used at each site during the observation period.
	BlockSiteGPSPhaseCen   Blockname = "SITE/GPS_PHASE_CENTER"           // GPS: phase center offsets for the antennas.
	BlockSiteGalPhaseCen   Blockname = "SITE/GAL_PHASE_CENTER"           // Galileo: phase center offsets for the antennas.
	BlockSiteEcc           Blockname = "SITE/ECCENTRICITY"               // Antenna eccentricities from the Marker to the Antenna Reference Point (ARP).
	BlockSatelliteID       Blockname = "SATELLITE/ID"                    // List of GNSS satellites used.
	BlockSatellitePhaseCen Blockname = "SATELLITE/PHASE_CENTER"          // GNSS satellite antenna phase center corrections.
	BlockSolEpochs         Blockname = "SOLUTION/EPOCHS"                 // List of observation timespan for each solution, site and point for which parameters have been estimated.
	BlockBiasEpochs        Blockname = "BIAS/EPOCHS"                     // List of epochs of bias parameters for each Site Code/Point Code/Solution Number/Bias Type (SPNB) combination
	BlockSolStatistics     Blockname = "SOLUTION/STATISTICS"             // Statistical information about the solution.
	BlockSolEstimate       Blockname = "SOLUTION/ESTIMATE"               // Estimated values and standard deviations of all solution parameters.
	BlockSolApriori        Blockname = "SOLUTION/APRIORI"                // Apriori information for estimated parameters.
	BlockSolMatrixEst      Blockname = "SOLUTION/MATRIX_ESTIMATE"        // The estimate matrix.
	BlockSolMatrixApr      Blockname = "SOLUTION/MATRIX_APRIORI"         // The apriori matrix.
	BlockSolNormalEquVec   Blockname = "SOLUTION/NORMAL_EQUATION_VECTOR" // Vector of the right hand side of the unconstrained (reduced) normal equation.

	// Inofficial
	BlockSolDiscontinuity Blockname = "SOLUTION/DISCONTINUITY" // Solution discontinuities.
)

// SiteCode is the site identifier, usually the FourCharID.
type SiteCode = string

// ObservationTechnique used to arrive at the solutions obtained in this SINEX file, e.g. SLR, GPS, VLBI.
// It should be consistent with the IERS convention.
type ObservationTechnique int

// Observation techniques.
const (
	ObsTechCombined ObservationTechnique = iota + 1
	ObsTechDORIS
	ObsTechSLR
	ObsTechLLR
	ObsTechGPS
	ObsTechVLBI
)

func (techn ObservationTechnique) String() string {
	return [...]string{"", "Combined", "DORIS", "SLR", "LLR", "GPS", "VLBI"}[techn]
}

// ParameterType identifies the type of parameter.
type ParameterType string

const (
	ParameterTypeSTAX ParameterType = "STAX" // Station X coordinate in m.
	ParameterTypeSTAY ParameterType = "STAY" // Station Y coordinate in m.
	ParameterTypeSTAZ ParameterType = "STAZ" // Station Z coordinate in m.

	// TODO: extend
)

// DiscontinuityType identifies the type of discontinuity.
type DiscontinuityType string

const (
	DiscontinuityTypePos        DiscontinuityType = "P" // discontinuity for position.
	DiscontinuityTypeVel        DiscontinuityType = "V" // discontinuity for velocity.
	DiscontinuityTypeAnnual     DiscontinuityType = "A" // discontinuity for annnual.
	DiscontinuityTypeSemiAnnual DiscontinuityType = "S" // discontinuity for semmi Annual.
	DiscontinuityTypeExpPSD     DiscontinuityType = "E" // discontinuity for Exponential Post-sesmic Relaxation.
)

// Header containes the information from the SINEX Header line.
type Header struct {
	Version            string               // Format version.
	Agency             string               // Agency creating the file.
	AgencyDataProvider string               // Agency providing the data in the file.
	CreationTime       time.Time            // Creation time of the file.
	StartTime          time.Time            // Start time of the data.
	EndTime            time.Time            // End time of the data.
	ObsTech            ObservationTechnique // Technique(s) used to generate the SINEX solution.
	NumEstimates       int                  // parameters estimated
	ConstraintCode     int                  // Single digit indicating the constraints:  0-fixed/tight constraints, 1-significant constraints, 2-unconstrained.
	SolutionTypes      []string             // Solution types contained in this SINEX file. Each character in this field may be one of the following:
	/* 	S - all station parameters, i.e. station coordinates, station velocities, biases, geocenter
	    O - Orbits
		  E - Earth Orientation Parameter
		  T - Troposphere
		  C - Celestial Reference Frame
	BLANK */

	//warnings []string
}

// FileReference provides information on the Organization, point of contact, the software and hardware involved in the creation of the file.
type FileReference struct {
	Description string // Organization(s).
	Output      string // File contents.
	Contact     string // Contact information.
	Software    string // SW used to generate the file.
	Hardware    string // Hardware on which above software was run.
	Input       string // Input used to generate this solution.
}

// Site provides general information for each site.
// See also site.Site{}
type Site struct {
	Code        SiteCode // 4-charID site code, e.g. WTZR.
	PointCode   string   // A 2-char code identifying physical monument within a site.
	DOMESNumber string
	ObsTech     ObservationTechnique // Technique(s) used to generate the SINEX solution.
	Description string               // Site description, e.g. city.
	Lon         string               // Longitude
	Lat         string               // Latitude
	Height      float64

	Receivers []*gnss.Receiver
	Antennas  []*gnss.Antenna
}

// Antenna for GNSS.
type Antenna struct {
	SiteCode  SiteCode             // 4-char site code, e.g. WTZR.
	PointCode string               // A 2-char code identifying physical monument within a site.
	SolID     string               // Solution ID at a Site/Point code for which the parameter is estimated.
	ObsTech   ObservationTechnique // Technique(s) used to generate the SINEX solution.
	*gnss.Antenna
}

// Receiver for GNSS.
type Receiver struct {
	SiteCode  SiteCode             // 4-char site code, e.g. WTZR.
	PointCode string               // A 2-char code identifying physical monument within a site.
	SolID     string               // Solution ID at a Site/Point code for which the parameter is estimated.
	ObsTech   ObservationTechnique // Technique(s) used to generate the SINEX solution.
	*gnss.Receiver
}

// Estimate stores the estimated solution parameters.
type Estimate struct {
	Idx            int           // Index of estimated parameters, beginning with 1.
	ParType        ParameterType // The type of the parameter.
	SiteCode       SiteCode      // 4-char site code, e.g. WTZR.
	PointCode      string        // A 2-char code identifying physical monument within a site.
	SolID          string        // Solution ID at a Site/Point code for which the parameter is estimated.
	Epoch          time.Time     // Epoch at which the estimated parameter is valid.
	Unit           string        // Units used for the estimates and sigmas.
	ConstraintCode string        // Constraint code applied to the parameter.
	Value          float64       // Estimated value of the parameter.
	Stddev         float64       // Estimated standard deviation for the parameter.
}

// equalMeta returns true if both Estimates have the same metadata (sitecode, epoch, solutionID etc.).
// What about ConstraintCode?
func (est Estimate) equalMeta(est2 Estimate) bool {
	return est.SiteCode == est2.SiteCode && est.PointCode == est2.PointCode && est.SolID == est2.SolID &&
		est.Epoch.Equal(est2.Epoch) && est.Unit == est2.Unit
}

// Discontinuity describes a discontinuity e.g. in the solution. Note this block is not official.
type Discontinuity struct {
	SiteCode  SiteCode          // 4-char site code, e.g. WTZR.
	ParType   ParameterType     // The type of the parameter.
	Idx       int               // soln number, beginning with 1, not identical as soln in estimate.
	Type      DiscontinuityType // Discontinuity type.
	StartTime time.Time         // Start time of the data.
	EndTime   time.Time         // End time of the data.
	Event     string            // Event explaination text, e.g. info for earth quake, equipment changes.
}

type StationCoordinates struct {
	SiteCode       SiteCode   // 4-char site code, e.g. WTZR.
	PointCode      string     // A 2-char code identifying physical monument within a site.
	SolID          string     // Solution ID at a Site/Point code for which the parameter is estimated.
	Epoch          time.Time  // Epoch at which the estimated parameter is valid.
	Unit           string     // Units used for the estimates and sigmas.
	ConstraintCode string     // Constraint code applied to the parameter.
	Values         [3]float64 // The XYZ-coordinates.
	Stddev         [3]float64 // Estimated standard deviation for the coordinates.
}

// AllStationCoordinates returns an iterator over Estimates that yields the coordinates
// for each station and epoch.
func AllStationCoordinates(estimates []Estimate) iter.Seq[StationCoordinates] {
	return func(yield func(StationCoordinates) bool) {
		crd := StationCoordinates{}
		lastEst := Estimate{}
		for _, est := range estimates {
			if est.ParType != ParameterTypeSTAX &&
				est.ParType != ParameterTypeSTAY &&
				est.ParType != ParameterTypeSTAZ {
				continue
			}

			if lastEst.SiteCode != "" && !lastEst.equalMeta(est) {
				if !yield(crd) {
					return
				}
				crd = StationCoordinates{} // clear
			}

			switch est.ParType {
			case ParameterTypeSTAX:
				crd.Values[0] = est.Value
				crd.Stddev[0] = est.Stddev
			case ParameterTypeSTAY:
				crd.Values[1] = est.Value
				crd.Stddev[1] = est.Stddev
			case ParameterTypeSTAZ:
				crd.Values[2] = est.Value
				crd.Stddev[2] = est.Stddev
			default:
				continue
			}

			if !lastEst.equalMeta(est) {
				crd.SiteCode = est.SiteCode
				crd.PointCode = est.PointCode
				crd.SolID = est.SolID
				crd.Epoch = est.Epoch
				crd.Unit = est.Unit
				crd.ConstraintCode = est.ConstraintCode
			}

			lastEst = est
		}

		// Yield the last one.
		yield(crd)
	}
}
