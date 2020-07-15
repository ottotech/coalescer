package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/machinebox/sdk-go/facebox"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	peopleDirFlagName = "peopledir"
	picsDirFlagName   = "picsdir"
)

var _logger = log.New(os.Stdout, "logger: ", log.Llongfile)
var fbox *facebox.Client

func main() {
	// Let's parse the flags
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

	// Let's connect to facebox.
	fbox = facebox.New("http://localhost:8080")

	// Let's run the application.
	if err := run(conf); err != nil {
		_logger.Fatalln(err)
	}
}

func run(c *config) error {
	// Let's walk through the people's dir and get the people's pictures
	// that we are going to recognize.
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

	if err != nil {
		return err
	}

	// Let's create the folders for the pictures of the people we want to filter.
	for k, _ := range c.People {
		path := filepath.Join(c.WorkingDir, k)
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}

	// Let's teach facebox how to recognize the people we want to filter out from our pictures directory.
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

	done := make(chan struct{})
	defer close(done)

	paths, errc := walkFiles(done, c.PicsDir)
	ch := make(chan result) // HLc
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

type result struct {
	path string
	err  error
}

func walkFiles(done <-chan struct{}, root string) (<-chan string, <-chan error) {
	paths := make(chan string)
	errc := make(chan error, 1)
	go func() { // HL
		// Close the paths channel after Walk returns.
		defer close(paths) // HL
		// No select needed for this send, since errc is buffered.
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
		err := RecognizeAndMove(conf, path)
		select {
		case c <- result{path, err}:
		case <-done:
			return
		}
	}
}

func RecognizeAndMove(conf *config, path string) error {
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
		return fmt.Errorf("file is not of type jpeg not png")
	}

	//faces, err := fbox.Check(file)
	//if err != nil {
	//	return err
	//}
	faces, err := IdentifyFace(file, filepath.Base(path))
	if err != nil {
		return err
	}

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

func IdentifyFace(img io.Reader, filename string) ([]facebox.Face, error) {
	body := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(body)
	imgWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(imgWriter, img)
	if err != nil {
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	client := http.Client{}
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/facebox/check", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Add("Content-Type", writer.FormDataContentType())
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		resErrData := struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}{}
		sb, err := ioutil.ReadAll(res.Body)
		if err != nil {
			_logger.Println(err)
			return nil, err
		}
		fmt.Println(string(sb))
		err = json.Unmarshal(sb, &resErrData)
		if err != nil {
			_logger.Println(err)
			return nil, err
		}
		return nil, fmt.Errorf(resErrData.Error)
	}

	resData := struct {
		Success   bool           `json:"success"`
		FaceCount int            `json:"faceCount"`
		Faces     []facebox.Face `json:"faces"`
	}{}

	sb, err := ioutil.ReadAll(res.Body)
	if err != nil {
		_logger.Println(err)
		return nil, err
	}
	err = json.Unmarshal(sb, &resData)
	if err != nil {
		_logger.Println(err)
		return nil, err
	}

	return resData.Faces, nil
}
