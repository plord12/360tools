// umap functions
//

package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"text/template"
)

type PhotoData struct {
	Photo string
}

type UmapData struct {
	WebURL     string
	East       float64
	West       float64
	North      float64
	South      float64
	CenterLat  float64
	CenterLong float64
}

//go:embed photo360-html.template
var photo360htmlTemplate string

//go:embed umap.template
var umapTemplate string

func createUmapFiles() {

	_, err := os.Stat(*outputDirectory)
	if !os.IsNotExist(err) {
		log.Printf("Output directory %s already exists, refusing to overwrite\n", *outputDirectory)
		os.Exit(1)
	}

	err = os.Mkdir(*outputDirectory, 0755)
	if err != nil {
		log.Printf("Unable to create output directory - %v", err)
		os.Exit(1)
	}

	// csv files for 360 and non-360 images
	//
	csvPlain, err := os.Create(path.Join(*outputDirectory, "photos.csv"))
	if err != nil {
		log.Printf("Unable to create output file - %v", err)
		os.Exit(1)
	}
	defer csvPlain.Close()
	csvPlain.WriteString("photo,lat,lon\n")

	csv360, err := os.Create(path.Join(*outputDirectory, "/photos360.csv"))
	if err != nil {
		log.Printf("Unable to create output file - %v", err)
		os.Exit(1)
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

	for _, imageFilename := range flag.Args() {

		timestamp, lat, long, _, err := getMetadata(imageFilename)
		if err != nil {
			if len(*gpxFile) > 0 {
				lat, long, _, err = getMetadataFromGPX(timestamp, *gpxFile)
				if err != nil {
					log.Printf("%s: Unable to get metadata from gpx: %v, skipping picture\n", imageFilename, err)
					continue
				}
			} else {
				log.Printf("%s: Unable to get metadata: %v, skipping picture\n", imageFilename, err)
				continue
			}
		}

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
				log.Printf("Unable to create output file - %v", err)
				os.Exit(1)
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

	// umap file
	//
	td := UmapData{WebURL: *webURL, East: east, West: west, North: north, South: south, CenterLat: totalLat / float64(totalCount), CenterLong: totalLong / float64(totalCount)}
	t, err := template.New("umap").Parse(umapTemplate)
	if err != nil {
		if err != nil {
			log.Printf("Unable to get umap.template: %v\n", err)
			os.Exit(1)
		}
	}
	umap, err := os.Create(path.Join(*outputDirectory, "photos.umap"))
	if err != nil {
		log.Printf("Unable to create output file - %v", err)
		os.Exit(1)
	}
	defer umap.Close()
	err = t.Execute(umap, td)
	if err != nil {
		if err != nil {
			log.Printf("Unable to process umap.template: %v\n", err)
			os.Exit(1)
		}
	}

	log.Printf("uMap files have been generated in %s directory\n", *outputDirectory)
	log.Printf("To use in uMap :\n")
	log.Printf("1. Copy photos, html pages and csv files to %s\n", *webURL)
	log.Printf("2. On uMap server, click \"Create a map\"\n")
	log.Printf("3. Click \"Edit map settings\" ( cog wheel ), \"Advanced actions\" then \"Empty\"\n")
	log.Printf("4. Click \"Import data\" ( up arrow ), browse and upload photos.umap then \"Import\"\n")
	log.Printf("5. Click \"Save\" and \"Disable editing\"\n")
}
