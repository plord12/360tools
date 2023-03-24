// Set of 360 photo utilities
//
// See Readme.md

// see https://github.com/googleapis/google-api-go-client

package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Flags
var (
	clientID     = flag.String("clientid", "", "OAuth 2.0 Client ID.  If non-empty, overrides --clientid_file")
	clientIDFile = flag.String("clientid-file", "clientid.dat",
		"Name of a file containing just the project's OAuth 2.0 Client ID from https://developers.google.com/console.")
	secret     = flag.String("secret", "", "OAuth 2.0 Client Secret.  If non-empty, overrides --secret_file")
	secretFile = flag.String("secret-file", "clientsecret.dat",
		"Name of a file containing just the project's OAuth 2.0 Client Secret from https://developers.google.com/console.")
	apikey     = flag.String("apikey", "", "API key.  If non-empty, overrides --apikey_file")
	apiKeyFile = flag.String("apikey-file", "apikey.dat",
		"Name of a file containing just the project's API key from https://developers.google.com/console.")
	cacheToken = flag.Bool("cachetoken", true, "cache the OAuth 2.0 token")
	pois       = flag.Bool("pois", false, "only list the nearest points of interest - requires api token")
	placeId    = flag.String("placeid", "", "place id (from --pois output) to add to upload")
	gpxFile    = flag.String("gpx-file", "",
		"Name of a file containing gpx track recorded at the same time as the photo was taken.")
)

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		log.Println("No jpgs supplied")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *pois {
		listPois(flag.Args())
		os.Exit(0)
	}

	startOauth()

	var photosIds []string

	for _, imageFilename := range flag.Args() {

		// get photo metadata
		//
		timestamp, lat, long, altitude, err := getMetadata(imageFilename)
		if err != nil {
			if len(*gpxFile) > 0 {
				lat, long, altitude, err = getMetadataFromGPX(timestamp, *gpxFile)
				if err != nil {
					log.Printf("%s: Unable to get metadata from gpx: %v\n", imageFilename, err)
					continue
				}
			} else {
				log.Printf("%s: Unable to get metadata: %v\n", imageFilename, err)

				continue
			}
		}
		log.Printf("%s: Timestamp %s\n", imageFilename, timestamp)
		log.Printf("%s: Latitude %f, Longitude %f\n", imageFilename, lat, long)
		log.Printf("%s: Altitude %f\n", imageFilename, altitude)

		// get upload url
		//
		uploadUrl, err := getUploadUrl()
		if err != nil {
			log.Printf("Unable to StartUpload: %v\n", err)
			continue
		}

		// upload file
		//
		uploadFile(imageFilename, uploadUrl)
		if err != nil {
			log.Printf("Unable to upload file: %v\n", err)
			continue
		}
		log.Printf("%s: Uploaded\n", imageFilename)

		// create meta data
		//
		photoId, err := createPhoto(uploadUrl, lat, long, altitude, timestamp, *placeId)
		if err != nil {
			log.Printf("Unable to Upload metadata: %v\n", err)
			continue
		}
		log.Printf("%s: Created metadata with id %s\n", imageFilename, photoId)

		photosIds = append(photosIds, photoId)
	}

	// wait for index complete
	//
	for _, photoId := range photosIds {
		waitPhotoUploaded(photoId)
	}

	// fix metadata by adding connections and bearings
	//
	addConnections(photosIds)
}

func openURL(url string) {
	try := []string{"xdg-open", "google-chrome", "open"}
	for _, bin := range try {
		err := exec.Command(bin, url).Run()
		if err == nil {
			return
		}
	}
	log.Printf("Error opening URL in browser.")
}

func valueOrFileContents(value string, filename string) string {
	if value != "" {
		return value
	}
	slurp, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading %q: %v", filename, err)
	}
	return strings.TrimSpace(string(slurp))
}
