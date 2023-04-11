// google functions
//

package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/streetviewpublish/v1"
)

var svc *streetviewpublish.Service
var client *http.Client

func uploadGoogleMaps(clientID *string, clientIDFile *string, secret *string, secretFile *string, apikey *string, apiKeyFile *string, cacheToken *bool, skipConnections *bool, placeId *string, filenames []string) {
	startOauth(clientID, clientIDFile, secret, secretFile, cacheToken)

	var photosIds []string
	var gpxFiles []string
	hasTracks := false

	// process gpx files first
	//
	for _, imageFilename := range filenames {
		if filepath.Ext(imageFilename) == ".gpx" {
			gpxFiles = append(gpxFiles, imageFilename)
			hasTracks = true
		}
	}
	file, err := os.CreateTemp("", "tracks.*.gpx")
	if err != nil {
		log.Printf("Unable to create tracks.gpx file - %v", err)
		os.Exit(1)
	}
	defer os.Remove(file.Name())
	err = mergeGPX(gpxFiles, file.Name())
	if err != nil {
		log.Printf("Unable to create tracks.gpx file - %v", err)
		os.Exit(1)
	}

	for _, imageFilename := range filenames {

		if filepath.Ext(imageFilename) == ".jpg" || filepath.Ext(imageFilename) == ".JPG" {

			// only support 360 images
			//
			if !is360(imageFilename) {
				log.Printf("%s: Donesn't seem to be a 360 picture, skipping picture", imageFilename)
				continue
			}

			// get photo metadata
			//
			timestamp, lat, long, altitude, err := getMetadata(imageFilename)
			if err != nil || lat != lat || long != long {
				if hasTracks {
					lat, long, altitude, err = getMetadataFromGPX(timestamp, file.Name())
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

			// get upload url
			//
			uploadUrl, err := getUploadUrl()
			if err != nil {
				log.Printf("Unable to StartUpload: %v, skipping picture\n", err)
				continue
			}

			// upload file
			//
			uploadFile(imageFilename, uploadUrl)
			if err != nil {
				log.Printf("Unable to upload file: %v, skipping picture\n", err)
				continue
			}
			log.Printf("%s: Uploaded\n", imageFilename)

			// create meta data
			//
			photoId, err := createPhoto(uploadUrl, lat, long, altitude, timestamp, *placeId)
			if err != nil {
				log.Printf("Unable to Upload metadata: %v, skipping metadata\n", err)
				continue
			}
			log.Printf("%s: Created metadata with id %s\n", imageFilename, photoId)

			photosIds = append(photosIds, photoId)
		}
	}

	// wait for index complete
	//
	for _, photoId := range photosIds {
		waitPhotoUploaded(photoId)
	}

	// fix metadata by adding connections and bearings
	//
	if !*skipConnections {
		addConnections(photosIds)
	}
}

func startOauth(clientID *string, clientIDFile *string, secret *string, secretFile *string, cacheToken *bool) {
	config := &oauth2.Config{
		ClientID:     valueOrFileContents(*clientID, *clientIDFile),
		ClientSecret: valueOrFileContents(*secret, *secretFile),
		Endpoint:     google.Endpoint,
		Scopes:       []string{streetviewpublish.StreetviewpublishScope},
	}

	ctx := context.Background()
	client = newOAuthClient(cacheToken, ctx, config)
	var err error
	svc, err = streetviewpublish.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create StreetViewPublish service: %v", err)
	}
}

func getUploadUrl() (string, error) {
	uploadRef, err := svc.Photo.StartUpload(&streetviewpublish.Empty{}).Do()
	if err != nil {
		return "", err
	}
	return uploadRef.UploadUrl, nil
}

func uploadFile(file string, uploadUrl string) error {
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

func createPhoto(uploadUrl string, latitude float64, longitude float64, altitude float64, timestamp time.Time, placeId string) (string, error) {
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
		return "", err
	}
	return resp.PhotoId.Id, nil
}

type result struct {
	Name    string `json:"name"`
	PlaceId string `json:"place_id"`
}

type response struct {
	Results []result `json:"results"`
}

func listPois(apikey *string, apiKeyFile *string, imageFilenames []string) {

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

func waitPhotoUploaded(photoId string) {

	log.Printf("%s: Waiting to be published\n", photoId)
	for {
		_, err := svc.Photo.Get(photoId).Do()
		if err != nil {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
}

func addConnections(photoIds []string) {
	// collect array of photos, then add connections
	//
	// 	1st -> 2nd
	// 	2nd -> 1st
	//  2nd -> 3rd
	//	...
	//  last -> n-1
	//

	var photos []*streetviewpublish.Photo

	// get list of photos
	//
	for _, photoId := range photoIds {
		photo, err := svc.Photo.Get(photoId).Do()
		if err != nil {
			log.Printf("Unable to get photo: %v", err)
			continue
		}
		photos = append(photos, photo)
	}

	if len(photos) > 1 {
		for count, photo := range photos {

			if count == 0 && count < (len(photos)-1) {
				// only connect to next
				next := streetviewpublish.Connection{Target: &streetviewpublish.PhotoId{Id: photos[count+1].PhotoId.Id}}
				photo.Connections = []*streetviewpublish.Connection{&next}
				bearing := getBearing(photo.Pose.LatLngPair.Latitude, photo.Pose.LatLngPair.Longitude, photos[count+1].Pose.LatLngPair.Latitude, photos[count+1].Pose.LatLngPair.Longitude)
				photo.Pose = &streetviewpublish.Pose{LatLngPair: &streetviewpublish.LatLng{Latitude: photo.Pose.LatLngPair.Latitude, Longitude: photo.Pose.LatLngPair.Longitude}, Altitude: bearing}
				log.Printf("%s: Connect to next %s, bearing %f\n", photo.PhotoId.Id, photos[count+1].PhotoId.Id, bearing)
			} else if count < (len(photos) - 1) {
				// connect to previous and next
				previous := streetviewpublish.Connection{Target: &streetviewpublish.PhotoId{Id: photos[count-1].PhotoId.Id}}
				next := streetviewpublish.Connection{Target: &streetviewpublish.PhotoId{Id: photos[count+1].PhotoId.Id}}
				photo.Connections = []*streetviewpublish.Connection{&previous, &next}
				bearing := getBearing(photo.Pose.LatLngPair.Latitude, photo.Pose.LatLngPair.Longitude, photos[count+1].Pose.LatLngPair.Latitude, photos[count+1].Pose.LatLngPair.Longitude)
				photo.Pose = &streetviewpublish.Pose{LatLngPair: &streetviewpublish.LatLng{Latitude: photo.Pose.LatLngPair.Latitude, Longitude: photo.Pose.LatLngPair.Longitude}, Altitude: bearing}
				log.Printf("%s: Connect to previous %s and next %s, bearing %f\n", photo.PhotoId.Id, photos[count-1].PhotoId.Id, photos[count+1].PhotoId.Id, bearing)
			} else {
				// only connect to previous
				previous := streetviewpublish.Connection{Target: &streetviewpublish.PhotoId{Id: photos[count-1].PhotoId.Id}}
				photo.Connections = []*streetviewpublish.Connection{&previous}
				bearing := getBearing(photos[count-1].Pose.LatLngPair.Latitude, photos[count-1].Pose.LatLngPair.Longitude, photo.Pose.LatLngPair.Latitude, photo.Pose.LatLngPair.Longitude)
				photo.Pose = &streetviewpublish.Pose{LatLngPair: &streetviewpublish.LatLng{Latitude: photo.Pose.LatLngPair.Latitude, Longitude: photo.Pose.LatLngPair.Longitude}, Altitude: bearing}
				log.Printf("%s: Connect to previous %s, assumed bearing %f\n", photo.PhotoId.Id, photos[count-1].PhotoId.Id, bearing)
			}

			_, err := svc.Photo.Update(photo.PhotoId.Id, photo).UpdateMask("connections,pose.heading").Do()
			if err != nil {
				log.Printf("Unable to Update metadata: %v", err)
				continue
			}
		}
	}
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

func tokenFromFile(cacheToken *bool, file string) (*oauth2.Token, error) {
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

func newOAuthClient(cacheToken *bool, ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile := tokenCacheFile(config)
	token, err := tokenFromFile(cacheToken, cacheFile)
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
