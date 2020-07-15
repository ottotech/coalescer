package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/machinebox/sdk-go/facebox"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// constant variables that represent the names of the flags that we are going to
// user in config and throughout the entire program.
const (
	peopleDirFlag      = "peopledir"
	picsDirFlag        = "picsdir"
	faceboxUrlFlag     = "faceboxurl"
	coolDownPeriodFlag = "cooldown"
)

var _logger = log.New(os.Stdout, "logger: ", log.Llongfile)
var fbox *facebox.Client

func main() {
	// Let's parse the flags.
	conf, output, err := parseFlags(os.Args[0], os.Args[1:])
	if err == flag.ErrHelp {
		fmt.Println(output)
		os.Exit(2)
	} else if err != nil {
		fmt.Println("output:\n", output)
		os.Exit(1)
	}

	// Let's validate the configuration.
	if ok, msg := conf.Validate(); !ok {
		_logger.Fatalln(msg)
	}

	// Let's connect to facebox and instantiate our fbox global variable.
	fbox = facebox.New(conf.FaceboxUrl)

	// Let's test the connection.
	_, err = fbox.Info()
	if err != nil {
		_logger.Fatalln(err)
	}

	// Let's run the application.
	if err := run(conf); err != nil {
		_logger.Fatalln(err)
	}
}

func run(c *config) error {
	// Let's collect the people's pictures that we want to recognize.
	err := collectPeoplePics(c)
	if err != nil {
		return err
	}

	// Let's create the folders for the pictures of the people we want to filter.
	err = createFoldersForPeople(c)
	if err != nil {
		return err
	}

	// Let's teach facebox about the people we want to recognize.
	err = teachFacebox(c)
	if err != nil {
		return err
	}

	done := make(chan struct{})
	defer close(done)

	paths, errc := walkFiles(done, c.PicsDir)
	ch := make(chan result)
	var wg sync.WaitGroup
	const numDigesters = 20
	wg.Add(numDigesters)
	for i := 0; i < numDigesters; i++ {
		go func() {
			digester(c, done, paths, ch)
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for r := range ch {
		fmt.Println(r.path)
		fmt.Println(r.err)
	}

	if err := <-errc; err != nil {
		return err
	}

	return nil
}

// collectPeoplePics walks through the people's dir and get the people's pictures that we want
// to recognize, and stores the people's name and file path in the *config.People map. Where
// the key is the name of a person and the value a slice with the pictures filename of that person.
func collectPeoplePics(c *config) error {
	err := filepath.Walk(c.PeopleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Name() == c.PeopleDir {
			return nil
		}

		if info.IsDir() && info.Name() != c.PeopleDir {
			return filepath.SkipDir
		}

		file, errFile := os.Open(path)
		if errFile != nil {
			return errFile
		}
		defer file.Close()

		_, format, imageErr := image.DecodeConfig(file)
		if imageErr != nil {
			return imageErr
		}
		if format != "jpeg" && format != "png" {
			return nil
		}

		idx := strings.Index(info.Name(), "_")
		var name string

		if idx == -1 || idx == 0 {
			return fmt.Errorf("incorrect file name %s in path (%s)", info.Name(), path)
		}

		name = info.Name()[:idx]
		c.People[name] = append(c.People[name], info.Name())

		return nil
	})

	return err
}

// createFoldersForPeople will create folders in current dir
// where we are going to store the pictures of the people we want
// to filter out from picsDirFlag.
func createFoldersForPeople(c *config) error {
	for k, _ := range c.People {
		path := filepath.Join(c.WorkingDir, k)
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

// teachFacebox will teach facebox instance about the people we want to recognize.
// If the coolDownPeriodFlag is true, we will wait five seconds to give enough
// time to facebox to assimilate the pictures.
func teachFacebox(c *config) error {
	for name, paths := range c.People {
		for _, p := range paths {
			fullPath := filepath.Join(c.WorkingDir, c.PeopleDir, p)
			img, err := os.Open(fullPath)
			if err != nil {
				return err
			}
			filename := filepath.Base(p)
			err = fbox.Teach(img, filename, name)
			if err != nil {
				return err
			}
			img.Close()
		}
	}

	// if user wants to have the cooldown period, we sleep for 5 secs.
	if c.CoolDownPeriod {
		fmt.Println("There would be a cool down period of 5 seconds, please wait...")
		time.Sleep(time.Second * 5)
	}

	return nil
}

type result struct {
	path string
	err  error
}

func walkFiles(done <-chan struct{}, root string) (<-chan string, <-chan error) {
	paths := make(chan string)
	errc := make(chan error, 1)
	go func() {
		defer close(paths)
		errc <- filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}

			select {
			case paths <- path:
			case <-done:
				return errors.New("walk canceled")
			}
			return nil
		})
	}()
	return paths, errc
}

func digester(conf *config, done <-chan struct{}, paths <-chan string, c chan<- result) {
	for path := range paths {
		err := recognizeAndCopy(conf, path)
		select {
		case c <- result{path, err}:
		case <-done:
			return
		}
	}
}

func recognizeAndCopy(conf *config, path string) error {
	fullPath := filepath.Join(conf.WorkingDir, path)
	file, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, format, err := image.DecodeConfig(file)
	if err != nil {
		return err
	}

	if format != "jpeg" && format != "png" {
		return fmt.Errorf("file is not of type jpeg nor png")
	}

	// We need to rewind the file so it can be read in other functions.
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	// Let's get the faces in the photo.
	faces, err := fbox.Check(file)
	if err != nil {
		return err
	}

	// We need to rewind the file so it can be read in other functions.
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	// We will loop through the recognized faces and check if there is a match.
	for _, face := range faces {
		if face.Matched && conf.People.exists(face.Name) {
			nfPath := filepath.Join(conf.WorkingDir, face.Name, filepath.Base(path))
			nf, err := os.Create(nfPath)
			if err != nil {
				return err
			}
			_, err = io.Copy(nf, file)
			if err != nil {
				nf.Close()
				return err
			}
			nf.Close()
		}
	}
	return nil
}
