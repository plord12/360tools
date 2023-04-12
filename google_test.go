package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

var photoid = 1

func TestGoogle(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		//log.Printf("Got %v\n", req)
		if req.Method == "POST" && strings.HasPrefix(req.RequestURI, "/v1/photo:startUpload") {
			// start upload
			rw.Header().Set("Content-Type", "application/json")
			rw.Write([]byte("{ \"uploadUrl\": \"http://" + req.Host + "/upload/uploadreference\" }"))
		} else if req.Method == "POST" && strings.HasPrefix(req.RequestURI, "/upload/") {
			// upload
		} else if req.Method == "POST" && strings.HasPrefix(req.RequestURI, "/v1/photo?") {
			// create ( metadata )
			rw.Write([]byte("{\"photoId\": { \"id\": \"photoid-" + strconv.Itoa(photoid) + "\" } }"))
			photoid++
		} else if strings.HasPrefix(req.RequestURI, "/v1/photo/photoid-1") {
			// get & update
			id := strings.Split(strings.Split(req.RequestURI, "?")[0], "/")[3]
			rw.Write([]byte("{\"photoId\": { \"id\": \"" + id + "\" }, \"pose\": { \"accuracyMeters\": 0, \"altitude\": 93.180000, \"heading\": 0,	\"latLngPair\": { \"latitude\": 51.427768, \"longitude\": -0.853968 }}}"))
		} else if strings.HasPrefix(req.RequestURI, "/v1/photo/photoid-2") {
			// get & update
			id := strings.Split(strings.Split(req.RequestURI, "?")[0], "/")[3]
			rw.Write([]byte("{\"photoId\": { \"id\": \"" + id + "\" }, \"pose\": { \"accuracyMeters\": 0, \"altitude\": 0, \"heading\": 0,	\"latLngPair\": { \"latitude\": 54.000000, \"longitude\": -6.000000 }}}"))
		} else {
			log.Printf("*** FIX THIS - Unhandled %v\n", req)
		}
	}))
	defer ts.Close()

	// skip oauth stuff
	testServer = ts.URL

	clientID := "xxx"
	clientIDFile := ""
	secret := "xxx"
	secretFile := ""
	apikey := "xxx"
	apiKeyFile := ""
	cacheToken := false
	skipConnections := false
	placeId := ""

	uploadGoogleMaps(&clientID, &clientIDFile, &secret, &secretFile, &apikey, &apiKeyFile, &cacheToken, &skipConnections, &placeId, []string{"testdata/3601.jpg", "testdata/flat.jpg", "testdata/nolocation.jpg", "testdata/good1.gpx"})
}
