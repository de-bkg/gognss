// Package sinex for reading SINEX files.
// Format description is available at https://www.iers.org/IERS/EN/Organization/AnalysisCoordinator/SinexFormat/sinex.html.

package sinex

import (
	"time"

	"github.com/de-bkg/gognss/pkg/site"
)

const (
	BlockFileReference     = "FILE/REFERENCE"                  // Information on the Organization, point of contact, etc.
	BlockFileComment       = "FILE/COMMENT"                    // General comments about the SINEX data file.
	BlockInputHistory      = "INPUT/HISTORY"                   // Information about the source of the information used to create the current SINEX file.
	BlockInputFiles        = "INPUT/FILES"                     // Identify the input files.
	BlockInputAck          = "INPUT/ACKNOWLEDGEMENTS"          // Defines the agency codes contributing to the SINEX file.
	BlockNutationData      = "NUTATION/DATA"                   // VLBI: contains the nutation model used in the analysis procedure.
	BlockPrecessionData    = "PRECESSION/DATA"                 // VLBI: contains the precession model used in the analysis procedure.
	BlockSourceID          = "SOURCE/ID"                       // VLBI: contains information about the radio sources estimated in the analysis.
	BlockSiteID            = "SITE/ID"                         // General information for each site containing estimated parameters.
	BlockSiteData          = "SITE/DATA"                       // Relationship between the estimated station parameters and in the input files.
	BlockSiteReceiver      = "SITE/RECEIVER"                   // GNSS: The receiver used at each site during the observation period.
	BlockSiteAntenna       = "SITE/ANTENNA"                    // GNSS: The antennas used at each site during the observation period.
	BlockSiteGPSPhaseCen   = "SITE/GPS_PHASE_CENTER"           // GPS: phase center offsets for the antennas.
	BlockSiteGalPhaseCen   = "SITE/GAL_PHASE_CENTER"           // Galileo: phase center offsets for the antennas.
	BlockSiteEcc           = "SITE/ECCENTRICITY"               // Antenna eccentricities from the Marker to the Antenna Reference Point (ARP).
	BlockSatelliteID       = "SATELLITE/ID"                    // List of GNSS satellites used.
	BlockSatellitePhaseCen = "SATELLITE/PHASE_CENTER"          // GNSS satellite antenna phase center corrections.
	BlockSolEpochs         = "SOLUTION/EPOCHS"                 // List of solution epoch for each Site Code/Point Code/Solution Number/Observation Code (SPNO) combination.
	BlockBiasEpochs        = "BIAS/EPOCHS"                     // List of epochs of bias parameters for each Site Code/Point Code/Solution Number/Bias Type (SPNB) combination
	BlockSolStatistics     = "SOLUTION/STATISTICS"             // Statistical information about the solution.
	BlockSolEstimate       = "SOLUTION/ESTIMATE"               // The Estimated parameters.
	BlockSolApriori        = "SOLUTION/APRIORI"                // Apriori information for estimated parameters.
	BlockSolMatrixEst      = "SOLUTION/MATRIX_ESTIMATE"        // The estimate matrix.
	BlockSolMatrixApr      = "SOLUTION/MATRIX_APRIORI"         // The apriori matrix.
	BlockSolNormalEquVec   = "SOLUTION/NORMAL_EQUATION_VECTOR" // Vector of the right hand side of the unconstrained (reduced) normal equation.
)

// SiteCode is the site identifier, usually the FourCharID.
type SiteCode string

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

	Receivers []*site.Receiver
	Antennas  []*site.Antenna
}

// Antenna for GNSS.
type Antenna struct {
	SiteCode  SiteCode             // 4-char site code, e.g. WTZR.
	PointCode string               // A 2-char code identifying physical monument within a site.
	SolID     string               // Solution ID at a Site/Point code for which the parameter is estimated.
	ObsTech   ObservationTechnique // Technique(s) used to generate the SINEX solution.
	*site.Antenna
}

// Receiver for GNSS.
type Receiver struct {
	SiteCode  SiteCode             // 4-char site code, e.g. WTZR.
	PointCode string               // A 2-char code identifying physical monument within a site.
	SolID     string               // Solution ID at a Site/Point code for which the parameter is estimated.
	ObsTech   ObservationTechnique // Technique(s) used to generate the SINEX solution.
	*site.Receiver
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
