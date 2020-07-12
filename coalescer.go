package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	peopleDirFlagName = "peopledir"
	picsDirFlagName   = "picsdir"
)

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
		log.Fatalln(msg)
	}

	// Let's run the application.
	if err := run(conf); err != nil {
		log.Fatalln(err)
	}
}

func run(c *config) error {
	// Let's walk through the people dir and get the people picture
	// that we are going to recognize.
	err := filepath.Walk(c.PeopleDir, func(path string, info os.FileInfo, err error) error {
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
	return nil
}
