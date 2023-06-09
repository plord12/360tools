// Set of 360 photo utilities
//
// See Readme.md

// see https://github.com/googleapis/google-api-go-client

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// Flags
	var (
		clientID     = flag.String("clientid", "", "Google OAuth 2.0 Client ID.  If non-empty, overrides --clientid_file")
		clientIDFile = flag.String("clientid-file", "clientid.dat",
			"Name of a file containing just the project's Google OAuth 2.0 Client ID from https://developers.google.com/console.")
		secret     = flag.String("secret", "", "Google OAuth 2.0 Client Secret.  If non-empty, overrides --secret_file")
		secretFile = flag.String("secret-file", "clientsecret.dat",
			"Name of a file containing just the project's OAuth 2.0 Client Secret from https://developers.google.com/console.")
		apikey     = flag.String("apikey", "", "Google API key.  If non-empty, overrides --apikey_file")
		apiKeyFile = flag.String("apikey-file", "apikey.dat",
			"Name of a file containing just the project's Google API key from https://developers.google.com/console.")
		cacheToken      = flag.Bool("cachetoken", true, "cache the Google OAuth 2.0 token")
		pois            = flag.Bool("pois", false, "only list the nearest points of interest - requires api token")
		skipConnections = flag.Bool("skip-connections", false, "skip Google Maps connections")
		placeId         = flag.String("placeid", "", "place id (from --pois output) to add to upload")
		mapType         = flag.String("map-type", "google", "Map type - google, for Google Street View, umap for OpenStreetMap uMap.")
		outputDirectory = flag.String("output-dir", "umap", "Output directory for uMap files.")
		webURL          = flag.String("web-url", "", "URL of web server that hosts photos for uMap server.")
	)
	flag.Usage = func() {
		fmt.Printf("Tools to upload 360 images to Google Maps and OpenStreetMap uMap\n\nUsage: %s [flags] [jpg files] [gpx files]\n\nWhere [flags] can be:\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		log.Println("No jpgs supplied")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *pois {
		listPois(apikey, apiKeyFile, flag.Args())
		os.Exit(0)
	}

	if *mapType == "google" {
		uploadGoogleMaps(clientID, clientIDFile, secret, secretFile, apikey, apiKeyFile, cacheToken, skipConnections, placeId, flag.Args())
	} else if *mapType == "umap" {
		if len(*webURL) == 0 {
			log.Println("Web URL must be provided")
			flag.PrintDefaults()
			os.Exit(1)
		}
		err := createUmapFiles(outputDirectory, webURL, flag.Args())
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
	} else {
		log.Println("Invalid map type - must be one of google or umap")
		flag.PrintDefaults()
		os.Exit(1)
	}
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
