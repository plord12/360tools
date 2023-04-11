package main

import (
	"math"
	"os"
	"testing"
	"time"
)

func TestNoSuchFile(t *testing.T) {
	time := time.Time{}
	_, _, _, err := getMetadataFromGPX(time, "junk")
	if err == nil {
		t.Errorf("didn't fail")
	}
}

func TestJunkFile(t *testing.T) {
	time := time.Time{}
	_, _, _, err := getMetadataFromGPX(time, "testdata/junk.gpx")
	if err == nil {
		t.Errorf("didn't fail")
	}
}

func TestNotFoundGood1(t *testing.T) {
	time := time.Time{}
	lat, lon, ele, err := getMetadataFromGPX(time, "testdata/good1.gpx")
	if err != nil {
		t.Errorf("unexpected fail %v", err)
	}
	if lat != 0.0 {
		t.Errorf("lat valid %v", lat)
	}
	if lon != 0.0 {
		t.Errorf("lon valid %v", lon)
	}
	if ele != 0.0 {
		t.Errorf("ele valid %v", ele)
	}
}

func TestFoundGood1(t *testing.T) {
	time := time.Date(2022, time.October, 22, 8, 10, 0, 0, time.UTC)
	lat, lon, ele, err := getMetadataFromGPX(time, "testdata/good1.gpx")
	if err != nil {
		t.Errorf("unexpected fail %v", err)
	}
	if math.Abs(lat-51.0) > 1e-9 {
		t.Errorf("lat valid %v", lat)
	}
	if math.Abs(lon - -3) > 1e-9 {
		t.Errorf("lon valid %v", lon)
	}
	if math.Abs(ele-139.05) > 1e-9 {
		t.Errorf("ele valid %v", ele)
	}
}

func TestOverrunGood1(t *testing.T) {
	time := time.Date(2022, time.October, 22, 8, 40, 0, 0, time.UTC)
	_, _, _, err := getMetadataFromGPX(time, "testdata/good1.gpx")
	if err == nil {
		t.Errorf("didn't fail")
	}
}

func TestJunkOutputMerge1(t *testing.T) {
	err := mergeGPX([]string{"testdata/good1.gpx"}, "")
	if err == nil {
		t.Errorf("didn't fail")
	}
}

func TestJunkInputtMerge1(t *testing.T) {
	err := mergeGPX([]string{"junk.gpx"}, "")
	if err == nil {
		t.Errorf("didn't fail")
	}
}

func TestJunkInputtMerge2(t *testing.T) {
	err := mergeGPX([]string{"testdata/junk.gpx"}, "")
	if err == nil {
		t.Errorf("didn't fail")
	}
}

func TestGoodMerge1(t *testing.T) {
	f, _ := os.CreateTemp("", "out.gpx")
	defer os.Remove(f.Name())
	err := mergeGPX([]string{"testdata/good1.gpx"}, f.Name())
	if err != nil {
		t.Errorf("failed %v", err)
	}
	time := time.Date(2022, time.October, 22, 8, 10, 0, 0, time.UTC)
	_, _, _, err = getMetadataFromGPX(time, f.Name())
	if err != nil {
		t.Errorf("couldnt read mergerd file %v", err)
	}
}
