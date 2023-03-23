// Set of 360 photo utilities
//
// See Readme.md

// see https://github.com/googleapis/google-api-go-client

package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/StefanSchroeder/Golang-Ellipsoid/ellipsoid"
	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/streetviewpublish/v1"
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

	svc, client := startOauth()

	var photos []*streetviewpublish.Photo

	for _, imageFilename := range flag.Args() {

		// get photo metadata
		//
		timestamp, lat, long, altitude, err := getMetadata(imageFilename)
		if err != nil {
			log.Printf("%s: Unable to get metadata: %v\n", imageFilename, err)
			continue
		}
		log.Printf("%s: Timestamp %s\n", imageFilename, timestamp)
		log.Printf("%s: Latitude %f, Longitude %f\n", imageFilename, lat, long)
		log.Printf("%s: Altitude %f\n", imageFilename, altitude)

		// get upload url
		//
		uploadUrl, err := getUploadUrl(svc)
		if err != nil {
			log.Printf("Unable to StartUpload: %v\n", err)
			continue
		}

		// upload file
		//
		uploadFile(client, imageFilename, uploadUrl)
		if err != nil {
			log.Printf("Unable to upload file: %v\n", err)
			continue
		}
		log.Printf("%s: Uploaded\n", imageFilename)

		// create meta data
		//
		photo, err := createPhoto(svc, uploadUrl, lat, long, altitude, timestamp, *placeId)
		if err != nil {
			log.Printf("Unable to Upload metadata: %v\n", err)
			continue
		}
		log.Printf("%s: Created metadata with id %s\n", imageFilename, photo.PhotoId.Id)

		photos = append(photos, photo)
	}

	// wait for index complete
	//
	for _, photo := range photos {
		log.Printf("%s: Waiting to be published\n", photo.PhotoId.Id)
		for {
			_, err := svc.Photo.Get(photo.PhotoId.Id).Do()
			if err != nil {
				time.Sleep(1 * time.Second)
			} else {
				break
			}
		}
	}

	// collect array of photos, then add connections
	//
	// 	1st -> 2nd
	// 	2nd -> 1st
	//  2nd -> 3rd
	//	...
	//  last -> n-1
	//
	for count, photo := range photos {

		if count == 0 && count < (len(photos)-1) {
			// only connect to next
			next := streetviewpublish.Connection{Target: &streetviewpublish.PhotoId{Id: photos[count+1].PhotoId.Id}}
			photo.Connections = []*streetviewpublish.Connection{&next}
			bearing := getBearing(photo.Pose.LatLngPair.Latitude, photo.Pose.LatLngPair.Longitude, photos[count+1].Pose.LatLngPair.Latitude, photos[count+1].Pose.LatLngPair.Longitude)
			photo.Pose.Heading = bearing
			log.Printf("%s: Connect to next %s, bearing %f\n", photo.PhotoId.Id, photos[count+1].PhotoId.Id, bearing)
		} else if count > 0 && count < (len(photos)-1) {
			// connect to previous and next
			previous := streetviewpublish.Connection{Target: &streetviewpublish.PhotoId{Id: photos[count-1].PhotoId.Id}}
			next := streetviewpublish.Connection{Target: &streetviewpublish.PhotoId{Id: photos[count+1].PhotoId.Id}}
			photo.Connections = []*streetviewpublish.Connection{&previous, &next}
			bearing := getBearing(photo.Pose.LatLngPair.Latitude, photo.Pose.LatLngPair.Longitude, photos[count+1].Pose.LatLngPair.Latitude, photos[count+1].Pose.LatLngPair.Longitude)
			photo.Pose.Heading = bearing
			log.Printf("%s: Connect to previous %s and next %s, bearing %f\n", photo.PhotoId.Id, photos[count-1].PhotoId.Id, photos[count+1].PhotoId.Id, bearing)
		} else {
			// only connect to previous
			previous := streetviewpublish.Connection{Target: &streetviewpublish.PhotoId{Id: photos[count-1].PhotoId.Id}}
			photo.Connections = []*streetviewpublish.Connection{&previous}
			bearing := getBearing(photos[count-1].Pose.LatLngPair.Latitude, photos[count-1].Pose.LatLngPair.Longitude, photo.Pose.LatLngPair.Latitude, photo.Pose.LatLngPair.Longitude)
			photo.Pose.Heading = bearing
			log.Printf("%s: Connect to previous %s, assumed bearing %f\n", photo.PhotoId.Id, photos[count-1].PhotoId.Id, bearing)
		}

		_, err := svc.Photo.Update(photo.PhotoId.Id, photo).UpdateMask("connections,pose.heading").Do()
		if err != nil {
			log.Printf("Unable to Update metadata: %v", err)
			continue
		}
	}
}

func startOauth() (*streetviewpublish.Service, *http.Client) {
	config := &oauth2.Config{
		ClientID:     valueOrFileContents(*clientID, *clientIDFile),
		ClientSecret: valueOrFileContents(*secret, *secretFile),
		Endpoint:     google.Endpoint,
		Scopes:       []string{streetviewpublish.StreetviewpublishScope},
	}

	ctx := context.Background()
	client := newOAuthClient(ctx, config)
	svc, err := streetviewpublish.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create StreetViewPublish service: %v", err)
	}
	return svc, client
}

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

func getUploadUrl(svc *streetviewpublish.Service) (string, error) {
	uploadRef, err := svc.Photo.StartUpload(&streetviewpublish.Empty{}).Do()
	if err != nil {
		return "", err
	}
	return uploadRef.UploadUrl, nil
}

func uploadFile(client *http.Client, file string, uploadUrl string) error {
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", uploadUrl, bytes.NewBuffer(dat))
	if err != nil {
		return err
	}
	_, err = client.Do(req)
	if err != nil {
		return err
	}
	return nil
}

func createPhoto(svc *streetviewpublish.Service, uploadUrl string, latitude float64, longitude float64, altitude float64, timestamp time.Time, placeId string) (*streetviewpublish.Photo, error) {
	photo := streetviewpublish.Photo{
		UploadReference: &streetviewpublish.UploadRef{UploadUrl: uploadUrl},
		Pose:            &streetviewpublish.Pose{LatLngPair: &streetviewpublish.LatLng{Latitude: latitude, Longitude: longitude}, Altitude: altitude},
		CaptureTime:     timestamp.Format("2006-01-02T15:04:05Z")}
	if len(placeId) > 0 {
		place := streetviewpublish.Place{PlaceId: placeId}
		photo.Places = []*streetviewpublish.Place{&place}
	}
	resp, err := svc.Photo.Create(&photo).Do()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type result struct {
	Name    string `json:"name"`
	PlaceId string `json:"place_id"`
}

type response struct {
	Results []result `json:"results"`
}

func listPois(imageFilenames []string) {

	client := &http.Client{}

	apiKey := valueOrFileContents(*apikey, *apiKeyFile)

	printed := make(map[string]int)

	for _, imageFilename := range imageFilenames {

		_, lat, long, _, err := getMetadata(imageFilename)
		if err != nil {
			// ignore for this file, just see less places
			continue
		}

		placeurl := fmt.Sprintf("https://maps.googleapis.com/maps/api/place/nearbysearch/json?location=%f%%2C%f&key="+apiKey+"&type=point_of_interest&rankby=distance", lat, long)
		req, err := http.NewRequest("GET", placeurl, nil)
		if err != nil {
			continue
		}
		httpresp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer httpresp.Body.Close()
		body, _ := io.ReadAll(httpresp.Body)

		response := response{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			continue
		}

		for _, result := range response.Results {
			_, exists := printed[result.PlaceId]
			if !exists {
				log.Printf("%s: %s\n", result.PlaceId, result.Name)
				printed[result.PlaceId] = 1
			}
		}
	}
}

func getBearing(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {
	geo1 := ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter, ellipsoid.LongitudeIsSymmetric, ellipsoid.BearingIsSymmetric)
	_, bearing := geo1.To(lat1, lon1, lat2, lon2)
	if bearing < 0 {
		bearing = bearing + 360
	}
	return bearing
}

func osUserCacheDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Caches")
	case "linux", "freebsd":
		return filepath.Join(os.Getenv("HOME"), ".cache")
	}
	log.Printf("TODO: osUserCacheDir on GOOS %q", runtime.GOOS)
	return "."
}

func tokenCacheFile(config *oauth2.Config) string {
	hash := fnv.New32a()
	hash.Write([]byte(config.ClientID))
	hash.Write([]byte(config.ClientSecret))
	hash.Write([]byte(strings.Join(config.Scopes, " ")))
	fn := fmt.Sprintf("go-api-demo-tok%v", hash.Sum32())
	return filepath.Join(osUserCacheDir(), url.QueryEscape(fn))
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	if !*cacheToken {
		return nil, errors.New("--cachetoken is false")
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := new(oauth2.Token)
	err = gob.NewDecoder(f).Decode(t)
	return t, err
}

func saveToken(file string, token *oauth2.Token) {
	f, err := os.Create(file)
	if err != nil {
		log.Printf("Warning: failed to cache oauth token: %v", err)
		return
	}
	defer f.Close()
	gob.NewEncoder(f).Encode(token)
}

func newOAuthClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile := tokenCacheFile(config)
	token, err := tokenFromFile(cacheFile)
	if err != nil {
		token = tokenFromWeb(ctx, config)
		saveToken(cacheFile, token)
	} else {
		if token.Expiry.Before(time.Now()) {
			log.Printf("need to renew new access token")
			token = tokenFromWeb(ctx, config)
			saveToken(cacheFile, token)
		} else {
			log.Printf("Using cached token %#v from %q", token, cacheFile)
		}
	}

	return config.Client(ctx, token)
}

func tokenFromWeb(ctx context.Context, config *oauth2.Config) *oauth2.Token {
	ch := make(chan string)
	randState := fmt.Sprintf("st%d", time.Now().UnixNano())
	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/favicon.ico" {
			http.Error(rw, "", 404)
			return
		}
		if req.FormValue("state") != randState {
			log.Printf("State doesn't match: req = %#v", req)
			http.Error(rw, "", 500)
			return
		}
		if code := req.FormValue("code"); code != "" {
			fmt.Fprintf(rw, "<h1>Success</h1>Authorized.")
			rw.(http.Flusher).Flush()
			ch <- code
			return
		}
		log.Printf("no code")
		http.Error(rw, "", 500)
	}))
	defer ts.Close()

	config.RedirectURL = ts.URL
	authURL := config.AuthCodeURL(randState)
	go openURL(authURL)
	log.Printf("Authorize this app at: %s", authURL)
	code := <-ch
	log.Printf("Got code: %s", code)

	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("Token exchange error: %v", err)
	}
	return token
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
	slurp, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading %q: %v", filename, err)
	}
	return strings.TrimSpace(string(slurp))
}
