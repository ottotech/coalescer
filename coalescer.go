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

var _logger *log.Logger
var fbox *facebox.Client

func main() {
	// Let's configure the logger.
	logFile, err := os.OpenFile("./coalescer.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	_logger = log.New(logFile, "Coalescer Logger:\t", log.Ldate|log.Ltime|log.Lshortfile)
	defer logFile.Close()

	// Let's parse the flags.
	conf, output, err := parseFlags(os.Args[0], os.Args[1:])
	if err == flag.ErrHelp {
		fmt.Println("output:\n", output)
		os.Exit(2)
	} else if err != nil {
		fmt.Println("output:\n", output)
		os.Exit(1)
	}

	// Let's validate the configuration.
	if ok, msg := conf.Validate(); !ok {
		log.Fatalln(msg)
	}

	// Let's connect to facebox and instantiate our fbox global variable.
	fbox = facebox.New(conf.FaceboxUrl)

	// Let's test the connection with facebox.
	_, err = fbox.Info()
	if err != nil {
		log.Fatalln(err)
	}

	// Let's run the application.
	if err := run(conf); err != nil {
		log.Fatalln(err)
	}
}

// run runs our main program logic with the given config options.
func run(c *config) error {
	// Let's collect the people's pictures that we want to recognize.
	err := collectPeoplePics(c)
	if err != nil {
		return err
	}

	// We need to check if the client of the app wants to do multiple matches on each picture.
	// If so, we need to check first if the passed names in config.People match the ones in
	// config.PeopleCombination.
	if c.MatchMultiple {
		if success := c.CheckPeopleCombination(); !success {
			return fmt.Errorf("there is a mismatch with the names of the people defined " +
				"in peopledir and the flag combine")
		}
	}

	// Let's create the folders for the pictures of the people we want to filter out.
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

	reClassifier := make(map[string][]result)
	const success = "success"
	const fail = "fail"
	for re := range ch {
		if re.err == nil {
			reClassifier[success] = append(reClassifier[success], re)
		} else {
			reClassifier[fail] = append(reClassifier[fail], re)
		}
	}

	for _, positiveResult := range reClassifier[success] {
		_logger.Printf("Success to recognize people in file %s", positiveResult.path)
	}

	for _, failure := range reClassifier[fail] {
		_logger.Printf("Failed to recognize people in file %s; got error %s", failure.path, failure.err)
	}

	if err := <-errc; err != nil {
		return fmt.Errorf("we couldn't check all the picture in picsdir; got err %s", err)
	}

	return nil
}

// collectPeoplePics walks through the people's dir and get the people's pictures that we want
// to recognize, and stores the peoples' names and files paths in config.People map. Where each
// key of the map will be the name of a person and its value a slice with the paths of the pictures
// of that person.
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

// createFoldersForPeople will create folders in the current dir where we are going to store
// the pictures of the people we want to filter out from picsDirFlag. If config.MatchMultiple
// option is true instead of creating multiple folders for each person that we are going to recognize,
// createFoldersForPeople will create one folder with the name defined in config.PeopleCombinedDirName.
func createFoldersForPeople(c *config) error {
	if c.MatchMultiple {
		path := filepath.Join(c.WorkingDir, c.PeopleCombinedDirName)
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
		return nil
	}

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

	if c.CoolDownPeriod {
		fmt.Println("There would be a cooldown period of 5 seconds, please wait...")
		time.Sleep(time.Second * 5)
	}

	return nil
}

// result represents a result of trying to recognize people from a picture in a particular path.
type result struct {
	path string
	err  error
}

// walkFiles starts a goroutine to walk the directory tree at root and send the
// path of each regular file on the string channel. It send the result of the
// walk on the error channel. If done is closed, walkFiles abandons its work.
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

// digester reads path names from paths and sends digests of the corresponding
// files on c until either paths or done is closed.
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

// recognizeAndCopy tries to recognize people in a picture located in the given path.
// If it succeeds to do so recognizeAndCopy will copy the picture in the corresponding
// path for all recognized pictures.
// TODO: It might be a good idea to factor out the logic when conf.MatchMultiple is True or False.
// TODO: I need to put more thoughts on this, but for now it works :)
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

	match := false

	if conf.MatchMultiple {
		matchesCount := make([]bool, 0)
		for _, name := range conf.PeopleCombined {
			itMatches := false
			for _, face := range faces {
				if face.Matched && face.Name == name {
					if face.Confidence < conf.Confidence {
						continue
					}
					itMatches = true
					break
				}
			}
			matchesCount = append(matchesCount, itMatches)
		}
		if allTrue(matchesCount) {
			nfPath := filepath.Join(conf.WorkingDir, conf.PeopleCombinedDirName, filepath.Base(path))
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
			match = true
		}
	} else {
		for _, face := range faces {
			if face.Matched && conf.People.exists(face.Name) {
				if face.Confidence < conf.Confidence {
					continue
				}
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
				// We need to rewind the file so it can be read in the next iteration.
				_, err = file.Seek(0, io.SeekStart)
				if err != nil {
					return fmt.Errorf("we couldn't rewind the file in the loop; got error %s", err)
				}
				match = true
			}
		}
	}

	if !match {
		return fmt.Errorf("there is no match with a confidence %.2f", conf.Confidence)
	}

	return nil
}

// allTrue checks whether all booleans in the given slice are True or not.
func allTrue(sl []bool) bool {
	for _, b := range sl {
		if b == false {
			return false
		}
	}
	return true
}
