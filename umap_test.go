package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"
)

func TestUmapDirectoryExists(t *testing.T) {

	dir := "."
	server := "http://server"
	err := createUmapFiles(&dir, &server, []string{"testdata/good1.gpx"})
	if err == nil {
		t.Errorf("didn't fail")
	}
}

func TestUmap1(t *testing.T) {
	dir, _ := os.MkdirTemp("", "umap")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)
	server := "http://server"
	err := createUmapFiles(&dir, &server, []string{"testdata/flat1.jpg", "testdata/3601.jpg", "testdata/nolocation.jpg", "testdata/good1.gpx"})
	if err != nil {
		t.Errorf("unexpected fail %v", err)
	}

	// simple line counts for validations

	if lineCount(path.Join(dir, "photos.csv")) != 2 {
		dumpFile(path.Join(dir, "photos.csv"))
		t.Errorf("photos.csv invalid line count %d", lineCount(path.Join(dir, "photos.csv")))
	}

	if lineCount(path.Join(dir, "photos360.csv")) != 3 {
		dumpFile(path.Join(dir, "photos360.csv"))
		t.Errorf("photos360.csv invalid line count %d", lineCount(path.Join(dir, "photos360.csv")))
	}

	if lineCount(path.Join(dir, "3601.jpg.html")) != 19 {
		dumpFile(path.Join(dir, "3601.jpg.html"))
		t.Errorf("3601.jpg.html invalid line count %d", lineCount(path.Join(dir, "3601.jpg.html")))
	}

	if lineCount(path.Join(dir, "tracks.gpx")) != 35 {
		dumpFile(path.Join(dir, "tracks.gpx"))
		t.Errorf("tracks.gpx invalid line count %d", lineCount(path.Join(dir, "tracks.gpx")))
	}

	if lineCount(path.Join(dir, "photos.umap")) != 111 {
		dumpFile(path.Join(dir, "photos.umap"))
		t.Errorf("photos.umap invalid line count %d", lineCount(path.Join(dir, "photos.umap")))
	}
}

func dumpFile(fileName string) {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	b, _ := ioutil.ReadAll(file)
	fmt.Print(string(b[:]))
}

func lineCount(fileName string) int {
	file, _ := os.Open(fileName)
	fileScanner := bufio.NewScanner(file)
	lineCount := 0
	for fileScanner.Scan() {
		lineCount++
	}
	return lineCount
}
