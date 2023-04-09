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
	cacheToken      = flag.Bool("cachetoken", true, "cache the OAuth 2.0 token")
	pois            = flag.Bool("pois", false, "only list the nearest points of interest - requires api token")
	placeId         = flag.String("placeid", "", "place id (from --pois output) to add to upload")
	mapType         = flag.String("map-type", "google", "Map type - google, for Google Street View, umap for OpenStreetMap uMap.")
	outputDirectory = flag.String("output-dir", "umap", "Output directory for uMap files.")
	webURL          = flag.String("web-url", "", "URL of web server that hosts photos for uMap server.")
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

	if *mapType == "google" {
		uploadGoogleMaps()
	} else if *mapType == "umap" {
		if len(*webURL) == 0 {
			log.Println("Web URL must be provided")
			flag.PrintDefaults()
			os.Exit(1)
		}
		createUmapFiles()
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
