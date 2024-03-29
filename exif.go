// exif functions
//

package main

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"time"

	"github.com/evanoberholster/imagemeta"
	"github.com/evanoberholster/imagemeta/jpeg"
)

func getMetadata(file string) (time.Time, float64, float64, float64, error) {
	// get lat, long, altitude, timestamp from jpgs
	//
	jpg, err := os.Open(file)
	if err != nil {
		return time.Time{}, 0.0, 0.0, 0.0, err
	}
	defer jpg.Close()

	metadata, err := imagemeta.Decode(jpg)
	if err != nil {
		return time.Time{}, 0.0, 0.0, 0.0, err
	}

	timestamp := metadata.DateTimeOriginal()
	lat := metadata.GPS.Latitude()
	long := metadata.GPS.Longitude()
	altitude := float64(metadata.GPS.Altitude())

	if lat == 0.0 && long == 0.0 && altitude == 0 {
		return timestamp, 0.0, 0.0, 0.0, errors.New("no GPS data")
	}

	return timestamp, lat, long, altitude, nil
}

type data struct {
	Data string `xml:",chardata"`
}

type panoData struct {
	Data string `xml:"ProjectionType,attr"`
}

// FIX THIS - is there a better way ?
var equirectangular bool

func is360(file string) bool {

	equirectangular = false

	// read xmp data to check if this is a 360 image
	//

	reader := func(r io.Reader) error {
		d := xml.NewDecoder(r)
		for {
			tok, err := d.Token()
			if tok == nil || err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			switch ty := tok.(type) {
			case xml.StartElement:
				if ty.Name.Local == "ProjectionType" {
					var projectionType data
					err = d.DecodeElement(&projectionType, &ty)
					if err != nil {
						return err
					}
					if projectionType.Data == "equirectangular" {
						equirectangular = true
					}
				}
			}
		}
		return nil
	}

	panoReader := func(r io.Reader) error {
		println("pano start")
		d := xml.NewDecoder(r)
		for {
			tok, err := d.Token()
			if tok == nil || err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			switch ty := tok.(type) {
			case xml.StartElement:
				if ty.Name.Local == "Description" {
					var projectionType panoData
					err = d.DecodeElement(&projectionType, &ty)
					if err != nil {
						println("pano Description error " + err.Error())
						return err
					}
					if projectionType.Data == "equirectangular" {
						equirectangular = true
					}
				}
			}
		}
		return nil
	}

	jpg, err := os.Open(file)
	if err != nil {
		return false
	}
	jpeg.ScanJPEG(jpg, nil, reader)
	jpg.Close()

	jpg, err = os.Open(file)
	if err != nil {
		return false
	}
	jpeg.ScanJPEG(jpg, nil, panoReader)
	jpg.Close()

	return equirectangular
}
