package main

import (
	"crypto/sha1"
	"fmt"
	"github.com/machinebox/sdk-go/boxutil"
	"github.com/machinebox/sdk-go/facebox"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	testingLogger := log.New(os.Stdout, "testingLogger", log.Lshortfile)
	_logger = testingLogger
	code := m.Run()
	os.Exit(code)
}

var testFilesMapSha = map[string]string{
	"598fe17e22744b1a4ec6c053677a8e686c71beac": "bill_and_steve.jpg",
	"b13cba3f6478673bee294c3e99f6e0196616e1d7": "mark_and_bill.jpg",
}

type mockNOCombination struct {
}

func (c *mockNOCombination) Info() (*boxutil.Info, error) {
	return &boxutil.Info{
		Name:    "xx",
		Version: 0,
		Build:   "xx",
		Status:  "xx",
	}, nil
}

func (c *mockNOCombination) Teach(image io.Reader, id string, name string) error {
	return nil
}

func (c *mockNOCombination) Check(image io.Reader) ([]facebox.Face, error) {
	hash := sha1.New()

	_, err := io.Copy(hash, image)
	if err != nil {
		return nil, err
	}

	fileSha := fmt.Sprintf("%x", hash.Sum(nil))

	val, exists := testFilesMapSha[fileSha]
	if !exists {
		return nil, fmt.Errorf("an unknown image was given to mockCombination while testing")
	}

	// In this image facebox should recognize 1 face.
	if val == "bill_and_steve.jpg" {
		faces := []facebox.Face{
			{
				Rect:       facebox.Rect{},
				ID:         "1",
				Name:       "bill",
				Matched:    true,
				Confidence: 70,
				Faceprint:  "",
			},
		}
		return faces, nil
	}

	// In this image facebox should recognize 2 faces.
	if val == "mark_and_bill.jpg" {
		faces := []facebox.Face{
			{
				Rect:       facebox.Rect{},
				ID:         "1",
				Name:       "bill",
				Matched:    true,
				Confidence: 70,
				Faceprint:  "",
			},
			{
				Rect:       facebox.Rect{},
				ID:         "2",
				Name:       "mark",
				Matched:    true,
				Confidence: 70,
				Faceprint:  "",
			},
		}
		return faces, nil
	}

	return nil, fmt.Errorf("error while testing and using Check method in mockNOCombination")
}

func Test_run_normal_usage_without_combination(t *testing.T) {
	conf, output, err := parseFlags("coalescer",
		[]string{"-faceboxurl=http://localhost:8080", "-peopledir=people_dir", "-picsdir=pics_dir", "-confidence=50"})

	if err != nil {
		t.Fatalf("got error (%s) while using parseFlags. Output was: %s", err, output)
	}
	// Let's clean the directory after testing.
	defer func() {
		err := os.RemoveAll("bill")
		if err != nil {
			log.Println(err)
		}
		err = os.RemoveAll("mark")
		if err != nil {
			log.Println(err)
		}
	}()

	conf.FaceboxUrl = "http://localhost:8080"
	conf.PicsDir = "pics_dir"
	conf.PeopleDir = "people_dir"
	conf.Confidence = 50

	originalFacebox := fbox
	fbox = &mockNOCombination{}
	defer func(original recognizer) {
		fbox = original
	}(originalFacebox)

	err = run(conf)
	if err != nil {
		t.Errorf("run shouldn't fail; got this err %s", err)
	}

	dirs := []string{
		"bill",
		"mark",
	}

	pathPics := map[string][]string{
		"bill": {
			"bill_and_steve.jpg",
			"mark_and_bill.jpg",
		},
		"mark": {
			"mark_and_bill.jpg",
		},
	}

	for _, d := range dirs {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			t.Fatalf("directory %s should exist", d)
		}
	}

	for dir, pics := range pathPics {
		for _, pic := range pics {
			path := filepath.Join(dir, pic)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("picture %s should exist in path %s", pic, dir)
			}
		}
	}
}
