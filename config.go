package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Constant variables that represent the names of the flags that we are going to
// use in the config struct and throughout the entire program.
const (
	peopleDirFlag      = "peopledir"
	picsDirFlag        = "picsdir"
	faceboxUrlFlag     = "faceboxurl"
	coolDownPeriodFlag = "cooldown"
	confidenceFlag     = "confidence"
	combineFlag        = "combine"
	rigidFlag          = "rigid"
)

type PeopleToIdentify map[string][]string

func (p PeopleToIdentify) exists(name string) bool {
	_, exists := p[name]
	return exists
}

type PeopleCombination []string

func (pc PeopleCombination) exists(name string) bool {
	for _, s := range pc {
		if name == s {
			return true
		}
	}
	return false
}

type config struct {
	// fields that represent the flags used by this program.
	PeopleDir      string
	PicsDir        string
	CoolDownPeriod bool
	FaceboxUrl     string
	WorkingDir     string
	Combine        string
	Confidence     float64
	Rigid          bool

	// custom fields.
	People                PeopleToIdentify
	PeopleCombined        PeopleCombination
	PeopleCombinedDirName string
	MatchMultiple         bool
}

// newConfig initializes a ready-to-use config struct.
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

// Transform will transform some fields in config.
// We need to be smart when and where to use Transform. See config.Validate().
func (c *config) Transform() {
	// The confidence about a match of each picture should be a float64 value
	// between 1 and 99, inclusive. (This value would be represented as a percentage)
	if c.Confidence < 1 || c.Confidence > 99 {
		c.Confidence = 50 / 100
	} else {
		c.Confidence = c.Confidence / 100
	}

	// Let's get the names of the people the user wants to combine when checking faces in each picture.
	c.PeopleCombined = strings.Split(c.Combine, ",")

	// If there are people names in config.PeopleCombined we can then set config.MatchMultiple = true.
	// We do this because we can implicitly understand that the user wants to recognize multiple people in each picture.
	// So our program needs to read config.MatchMultiple so it can change its behaviour later when
	// checking for faces.
	if len(c.PeopleCombined) > 1 {
		c.MatchMultiple = true
	}

	// If the user wants to recognize multiple people in each picture, we need to create a custom directory so that
	// coalescer can store the filtered pictures there.
	if c.MatchMultiple {
		c.PeopleCombinedDirName = strings.Join(c.PeopleCombined, "_")
	}
}

// Validate validates the config fields.
func (c *config) Validate() (ok bool, msg string) {
	ok = true
	msg += "\n"

	// Let's transform any default value or behaviour for our custom fields in config.
	c.Transform()

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
		msg += fmt.Sprintf("got this error while parsing the facebox url: %s", err)
	} else {
		if u.Scheme == "" || u.Host == "" {
			ok = false
			msg += fmt.Sprint("malformed facebox url. try something like: http://localhost:8080")
		}
	}
	if c.Combine != "" && len(c.PeopleCombined) == 1 {
		msg += "If you want to match multiple people in each picture you need to at least define two names " +
			"in the combine flag."
	}
	if !ok {
		msg += "For more information about the flags of this program, please run ./coalescer -h"
	}
	return
}

// CheckPeopleCombination checks whether the people defined in config.PeopleCombined can be
// recognized. It does the checking by comparing the peoples' names from config.PeopleCombined
// and config.People.
func (c *config) CheckPeopleCombination() (success bool) {
	for _, name := range c.PeopleCombined {
		exists := false
		for s := range c.People {
			if name == s {
				exists = true
				break
			}
		}
		if !exists {
			return false
		}
	}
	return true
}

func parseFlags(programName string, args []string) (conf *config, output string, err error) {
	flags := flag.NewFlagSet(programName, flag.ContinueOnError)
	var buf bytes.Buffer
	flags.SetOutput(&buf)

	c, err := newConfig()
	if err != nil {
		return nil, buf.String(), err
	}

	flags.StringVar(&c.PeopleDir, peopleDirFlag, "", "Represents the dir where coalescer can find the photos of the people you want to recognize.")
	flags.StringVar(&c.PicsDir, picsDirFlag, "", "Represents the dir where coalescer can find all the photos you want to filter out based on the people you want to recognize in peopledir.")
	flags.StringVar(&c.FaceboxUrl, faceboxUrlFlag, "", "Represents the url of the facebox machine instance.")
	flags.BoolVar(&c.CoolDownPeriod, coolDownPeriodFlag, true, "Represents duration of the cooldown period needed to let facebox assimilate the people's pictures.")
	flags.Float64Var(&c.Confidence, confidenceFlag, 50, "Determines how confident coalescer is about the match of each picture. It should be a value between 1 and 99.")
	flags.StringVar(&c.Combine, combineFlag, "", "Specifies the names of the people you want to recognize in each picture. Use this if you want to do a multiple match.")
	flags.BoolVar(&c.Rigid, rigidFlag, false, "Specifies that in order to have a valid match all faces should appear in each picture exclusively.")

	err = flags.Parse(args)
	if err != nil {
		return nil, buf.String(), err
	}
	return c, buf.String(), nil
}
