// Ellipsoid functions
//

package main

import (
	"github.com/StefanSchroeder/Golang-Ellipsoid/ellipsoid"
)

func getBearing(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {
	geo1 := ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter, ellipsoid.LongitudeIsSymmetric, ellipsoid.BearingIsSymmetric)
	_, bearing := geo1.To(lat1, lon1, lat2, lon2)
	if bearing < 0 {
		bearing = bearing + 360
	}
	return bearing
}

func getDisplament(lat1 float64, lon1 float64, lat2 float64, lon2 float64) (float64, float64) {
	geo1 := ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter, ellipsoid.LongitudeIsSymmetric, ellipsoid.BearingIsSymmetric)
	x, y := geo1.Displacement(lat1, lon1, lat2, lon2)
	return x, y
}

func getLocation(lat float64, lon float64, x float64, y float64) (float64, float64) {
	geo1 := ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter, ellipsoid.LongitudeIsSymmetric, ellipsoid.BearingIsSymmetric)
	newLat, newLon := geo1.Location(lat, lon, x, y)
	return newLat, newLon
}
