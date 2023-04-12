// umap functions
//
// FIX THIS - add support for tracks
//
//	Combine all *.gpx > tracks.gx ( multiple <trk> tags )

package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"text/template"
)

type PhotoData struct {
	Photo string
}

type UmapData struct {
	Name         string
	WebURL       string
	East         float64
	West         float64
	North        float64
	South        float64
	CenterLat    float64
	CenterLong   float64
	Has360Photos bool
	HasPhotos    bool
	HasTracks    bool
}

//go:embed photo360-html.template
var photo360htmlTemplate string

//go:embed umap.template
var umapTemplate string

func createUmapFiles(outputDirectory *string, webURL *string, filenames []string) error {

	_, err := os.Stat(*outputDirectory)
	if !os.IsNotExist(err) {
		return fmt.Errorf("output directory %s already exists, refusing to overwrite", *outputDirectory)
	}

	err = os.Mkdir(*outputDirectory, 0755)
	if err != nil {
		return fmt.Errorf("unable to create output directory - %v", err)
	}

	// csv files for 360 and non-360 images
	//
	csvPlain, err := os.Create(path.Join(*outputDirectory, "photos.csv"))
	if err != nil {
		return fmt.Errorf("unable to create output file - %v", err)
	}
	defer csvPlain.Close()
	csvPlain.WriteString("photo,lat,lon\n")

	csv360, err := os.Create(path.Join(*outputDirectory, "photos360.csv"))
	if err != nil {
		return fmt.Errorf("unable to create output file - %v", err)
	}
	defer csv360.Close()
	csv360.WriteString("photo,lat,lon\n")

	east := -90.0
	west := 90.0
	north := -90.0
	south := 90.0

	totalLat := 0.0
	totalLong := 0.0
	totalCount := 0

	has360Photos := false
	hasPhotos := false
	hasTracks := false

	var gpxFiles []string

	// process gpx files first
	//
	for _, imageFilename := range filenames {
		if filepath.Ext(imageFilename) == ".gpx" {
			gpxFiles = append(gpxFiles, imageFilename)
			hasTracks = true
		}
	}
	err = mergeGPX(gpxFiles, path.Join(*outputDirectory, "tracks.gpx"))
	if err != nil {
		return fmt.Errorf("unable to create tracks.gpx file - %v", err)
	}

	// process jpgs
	for _, imageFilename := range filenames {

		if filepath.Ext(imageFilename) == ".jpg" || filepath.Ext(imageFilename) == ".JPG" {

			timestamp, lat, long, altitude, err := getMetadata(imageFilename)
			if err != nil || lat != lat || long != long {
				if hasTracks {
					lat, long, _, err = getMetadataFromGPX(timestamp, path.Join(*outputDirectory, "tracks.gpx"))
					if err != nil {
						log.Printf("%s: Unable to get metadata from gpx: %v, skipping picture\n", imageFilename, err)
						continue
					}
				} else {
					log.Printf("%s: Unable to get metadata: %v, skipping picture\n", imageFilename, err)
					continue
				}
			}

			log.Printf("%s: Timestamp %s\n", imageFilename, timestamp)
			log.Printf("%s: Latitude %f, Longitude %f\n", imageFilename, lat, long)
			log.Printf("%s: Altitude %f\n", imageFilename, altitude)

			if long > east {
				east = long
			}
			if long < west {
				west = long
			}
			if lat > north {
				north = lat
			}
			if lat < south {
				south = lat
			}

			totalLat = totalLat + lat
			totalLong = totalLong + long
			totalCount = totalCount + 1

			if is360(imageFilename) {

				has360Photos = true

				// update 360 csv
				//
				csv360.WriteString(fmt.Sprintf("%s,%f,%f\n", path.Base(imageFilename), lat, long))

				// write 360 phto html
				//
				td := PhotoData{Photo: path.Base(imageFilename)}
				t, err := template.New("umap").Parse(photo360htmlTemplate)
				if err != nil {
					if err != nil {
						log.Printf("%s: Unable to get photo360-html.template: %v, skipping picture\n", imageFilename, err)
						continue
					}
				}
				html, err := os.Create(path.Join(*outputDirectory, path.Base(imageFilename)+".html"))
				if err != nil {
					return fmt.Errorf("unable to create output file - %v", err)
				}
				defer html.Close()
				err = t.Execute(html, td)
				if err != nil {
					if err != nil {
						log.Printf("%s: Unable to process photo360-html.template: %v, skipping picture\n", imageFilename, err)
						continue
					}
				}

			} else {

				hasPhotos = true

				// update plain csv
				//
				csvPlain.WriteString(fmt.Sprintf("%s,%f,%f\n", path.Base(imageFilename), lat, long))

			}

			// copy photo
			//
			r, err := os.Open(imageFilename)
			if err != nil {
				log.Printf("%s: Unable to copy photo: %v\n", imageFilename, err)
				continue
			}
			defer r.Close()
			w, err := os.Create(path.Join(*outputDirectory, path.Base(imageFilename)))
			if err != nil {
				log.Printf("%s: Unable to copy photo: %v\n", imageFilename, err)
				continue
			}
			defer w.Close()
			w.ReadFrom(r)
		}
	}

	// umap file
	//
	td := UmapData{Name: path.Base(*webURL),
		WebURL:       *webURL,
		East:         east,
		West:         west,
		North:        north,
		South:        south,
		CenterLat:    totalLat / float64(totalCount),
		CenterLong:   totalLong / float64(totalCount),
		Has360Photos: has360Photos,
		HasPhotos:    hasPhotos,
		HasTracks:    hasTracks}
	t, err := template.New("umap").Parse(umapTemplate)
	if err != nil {
		if err != nil {
			return fmt.Errorf("unable to get umap.template: %v", err)
		}
	}
	umap, err := os.Create(path.Join(*outputDirectory, "photos.umap"))
	if err != nil {
		return fmt.Errorf("unable to create output file - %v", err)
	}
	defer umap.Close()
	err = t.Execute(umap, td)
	if err != nil {
		if err != nil {
			return fmt.Errorf("unable to process umap.template: %v", err)
		}
	}

	log.Printf("uMap files have been generated in %s directory\n", *outputDirectory)
	log.Printf("To use in uMap :\n")
	log.Printf("1. Copy photos, html pages and csv files to %s\n", *webURL)
	log.Printf("2. On uMap server, click \"Create a map\"\n")
	log.Printf("3. Click \"Edit map settings\" ( cog wheel ), \"Advanced actions\" then \"Empty\"\n")
	log.Printf("4. Click \"Import data\" ( up arrow ), browse and upload photos.umap then \"Import\"\n")
	log.Printf("5. Click \"Save\" and \"Disable editing\"\n")

	return nil
}
