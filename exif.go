// exif functions
//

package main

import (
	"os"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

func getMetadata(file string) (time.Time, float64, float64, float64, error) {
	// get lat, long, altitude, timestamp from jpgs
	//
	// FIX THIS - verify 360 ( in xmp )
	//

	jpg, err := os.Open(file)
	if err != nil {
		return time.Time{}, 0.0, 0.0, 0.0, err
	}
	metadata, err := exif.Decode(jpg)
	if err != nil {
		return time.Time{}, 0.0, 0.0, 0.0, err
	}
	timestamp, err := metadata.DateTime()
	if err != nil {
		return time.Time{}, 0.0, 0.0, 0.0, err
	}
	lat, long, err := metadata.LatLong()
	if err != nil {
		return timestamp, 0.0, 0.0, 0.0, err
	}
	altTag, err := metadata.Get(exif.GPSAltitude)
	altitude := 0.0
	if err == nil {
		numer, denom, err := altTag.Rat2(0)
		if err == nil {
			altitude = float64(numer) / float64(denom)
		}
	}

	jpg.Close()

	return timestamp, lat, long, altitude, nil
}
