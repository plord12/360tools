// gpx functions
//

package main

import (
	"errors"
	"os"
	"time"

	"github.com/tkrajina/gpxgo/gpx"
)

func getMetadataFromGPX(timestamp time.Time, gpxFilename string) (float64, float64, float64, error) {
	// get lat, long, altitude from gpx
	//

	gpxBytes, err := os.ReadFile(gpxFilename)
	if err != nil {
		return 0.0, 0.0, 0.0, err
	}

	gpxFile, err := gpx.ParseBytes(gpxBytes)
	if err != nil {
		return 0.0, 0.0, 0.0, err
	}

	var lastPoint gpx.GPXPoint

	for _, track := range gpxFile.Tracks {
		for _, segment := range track.Segments {
			for _, point := range segment.Points {
				if (timestamp.Equal(lastPoint.Timestamp) || timestamp.After(lastPoint.Timestamp)) &&
					(timestamp.Equal(point.Timestamp) || timestamp.Before(point.Timestamp)) {

					// same point so no need to interpolate
					//
					if point.Timestamp.Unix() == lastPoint.Timestamp.Unix() {
						alt := 0.0
						if lastPoint.Elevation.NotNull() {
							alt = lastPoint.Elevation.Value()
						}
						return lastPoint.Latitude, lastPoint.Longitude, alt, nil
					}

					// get distance between the points
					//
					x, y := getDisplament(lastPoint.Latitude, lastPoint.Longitude, point.Latitude, point.Longitude)

					// proportion distance base on time difference
					//
					diff := float64(timestamp.Unix()-lastPoint.Timestamp.Unix()) / float64(point.Timestamp.Unix()-lastPoint.Timestamp.Unix())
					lat, lon := getLocation(lastPoint.Latitude, lastPoint.Longitude, x*diff, y*diff)
					alt := 0.0
					if lastPoint.Elevation.NotNull() && point.Elevation.NotNull() {
						alt = lastPoint.Elevation.Value() + (point.Elevation.Value()-lastPoint.Elevation.Value())*diff
					}

					return lat, lon, alt, nil
				}
				lastPoint = point
			}
		}
	}

	// not found
	//
	return 0.0, 0.0, 0.0, errors.New("Timestamp " + timestamp.String() + " not found in GPX")
}

func mergeGPX(gpxFilenames []string, gpxOutputFilename string) error {

	outputGpxFile := new(gpx.GPX)

	for _, gpxFilename := range gpxFilenames {
		gpxBytes, err := os.ReadFile(gpxFilename)
		if err != nil {
			return err
		}

		gpxFile, err := gpx.ParseBytes(gpxBytes)
		if err != nil {
			return err
		}

		outputGpxFile.Tracks = append(outputGpxFile.Tracks, gpxFile.Tracks...)
	}

	output, err := os.Create(gpxOutputFilename)
	if err != nil {
		return err
	}
	defer output.Close()

	xmlBytes, err := outputGpxFile.ToXml(gpx.ToXmlParams{Version: "1.1", Indent: true})
	if err != nil {
		return err
	}
	output.Write(xmlBytes)

	return nil
}
