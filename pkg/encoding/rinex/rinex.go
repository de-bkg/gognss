// Package encoding/rinex implements encoding and decoding of RINEX formated GNSS data.
// See RINEX format documentation at
// https://igs.org/formats-and-standards/
package rinex

import (
	"strings"
	"time"
)

const (
	// The Date/Time format in the PGM / RUN BY / DATE header record.
	headerDateFormat string = "20060102 150405"

	// The Date/Time format with time zone in the PGM / RUN BY / DATE header record.
	//
	// Format: "yyyymmdd hhmmss zone" with 3–4 character code for the time zone.
	headerDateWithZoneFormat string = "20060102 150405 MST"

	// The RINEX-2 Date/Time format in the PGM / RUN BY / DATE header record.
	headerDateFormatv2 string = "02-Jan-06 15:04"
)

// Parse the Date/Time in the PGM / RUN BY / DATE header record.
// It is recommended to use UTC as the time zone. Set zone to LCL if an unknown local time was used.
func parseHeaderDate(date string) (time.Time, error) {
	format := headerDateFormat
	if len(date) == 19 || len(date) == 20 {
		format = headerDateWithZoneFormat
	} else if len(date) == 15 && strings.Contains(date, "-") {
		format = headerDateFormatv2
	} else if len(date) == 18 && strings.Contains(date, "-") {
		format = "02-Jan-06 15:04:05" // unofficial!
	} else if len(date) == 17 && strings.Contains(date, "-") {
		format = "02-Jan-2006 15:04" // unofficial!
	} else if len(date) == 16 && strings.Contains(date, "-") {
		format = "2006-01-02 15:04" // unofficial!
	}

	ti, err := time.Parse(format, date)
	if err != nil {
		return time.Time{}, err
	}
	return ti, nil
}
