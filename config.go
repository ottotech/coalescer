package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/url"
	"os"
)

type PeopleToIdentify map[string][]string

func (p PeopleToIdentify) exists(name string) bool {
	_, exists := p[name]
	return exists
}

type config struct {
	PeopleDir      string
	PicsDir        string
	People         PeopleToIdentify
	WorkingDir     string
	FaceboxUrl     string
	CoolDownPeriod bool
	Confidence     float64
}

func newConfig() (*config, error) {
	wdir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c := &config{
		People:     make(PeopleToIdentify),
		WorkingDir: wdir,
	}
	return c, nil
}

func (c *config) Initiate() {
	// The confidence about a match of each picture should be a float64 value
	// between 50 and 99, inclusive. (This value represents a percentage)
	if c.Confidence < 50 || c.Confidence > 99 {
		c.Confidence = 50 / 100
	} else {
		c.Confidence = c.Confidence / 100
	}
}

func (c *config) Validate() (ok bool, msg string) {
	ok = true
	msg += "\n"

	// Let's initiate any default value or behaviour for the fields in config.
	c.Initiate()

	if c.PeopleDir == "" {
		ok = false
		msg += fmt.Sprintf("%s flag is not defined.\n", peopleDirFlag)
	}
	if c.PicsDir == "" {
		ok = false
		msg += fmt.Sprintf("%s flag is not defined.\n", picsDirFlag)
	}
	if c.PeopleDir == c.PicsDir && c.PeopleDir != "" && c.PicsDir != "" {
		ok = false
		msg += fmt.Sprintf("the %s and %s flags cannot point to the same directory.\n", peopleDirFlag, picsDirFlag)
	}
	if info, err := os.Stat(c.PeopleDir); os.IsNotExist(err) {
		ok = false
		msg += fmt.Sprintf("directory %s specified by the flag %s does not exist.\n", c.PeopleDir, peopleDirFlag)
	} else if !info.IsDir() {
		ok = false
		msg += fmt.Sprintf("directory %s specified by the flag %s is not a directory.\n", c.PeopleDir, peopleDirFlag)
	}
	if info, err := os.Stat(c.PicsDir); os.IsNotExist(err) {
		ok = false
		msg += fmt.Sprintf("directory %s specified by the flag %s does not exist.\n", c.PicsDir, picsDirFlag)
	} else if !info.IsDir() {
		ok = false
		msg += fmt.Sprintf("directory %s specified by the flag %s is not a directory.\n", c.PicsDir, picsDirFlag)
	}
	if u, err := url.Parse(c.FaceboxUrl); err != nil {
		ok = false
		msg += fmt.Sprintf("got this error while parsing the FaceboxUrl: %s", err)
	} else {
		if u.Scheme == "" || u.Host == "" {
			ok = false
			msg += fmt.Sprint("malformed FaceboxUrl. try something like: http://localhost:8080")
		}
	}
	if !ok {
		msg += "For more information about the flags of this program, please run ./coalescer -h"
	}
	return
}

func parseFlags(programName string, args []string) (conf *config, output string, err error) {
	flags := flag.NewFlagSet(programName, flag.ContinueOnError)
	var buf bytes.Buffer
	flags.SetOutput(&buf)

	c, err := newConfig()
	if err != nil {
		return nil, buf.String(), err
	}

	flags.StringVar(&c.PeopleDir, peopleDirFlag, "", "directory where we can find the photos of the people we want to recognize")
	flags.StringVar(&c.PicsDir, picsDirFlag, "", "directory where we can find all the photos we want to filter based on the people we want to recognize in peopledir")
	flags.StringVar(&c.FaceboxUrl, faceboxUrlFlag, "", "url pointing to your facebox machine instance")
	flags.BoolVar(&c.CoolDownPeriod, coolDownPeriodFlag, false, "if cooldown is true, coalescer will could down 5 seconds to let facebox assimilate the people's pictures")
	flags.Float64Var(&c.Confidence, confidenceFlag, 50, "Determines how confident coalescer is about the match of each picture. It should be a value between 50 and 99, otherwise it will default to 50.")

	err = flags.Parse(args)
	if err != nil {
		return nil, buf.String(), err
	}
	return c, buf.String(), nil
}
