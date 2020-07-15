package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
)

type PeopleToIdentify map[string][]string

func (p PeopleToIdentify) exists(name string) bool {
	_, exists := p[name]
	return exists
}

type config struct {
	PeopleDir  string
	PicsDir    string
	People     PeopleToIdentify
	WorkingDir string
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

func (c *config) Validate() (ok bool, msg string) {
	ok = true
	msg += "\n"

	if c.PeopleDir == "" {
		ok = false
		msg += fmt.Sprintf("%s flag is not defined.\n", peopleDirFlagName)
	}
	if c.PicsDir == "" {
		ok = false
		msg += fmt.Sprintf("%s flag is not defined.\n", picsDirFlagName)
	}
	if c.PeopleDir == c.PicsDir && c.PeopleDir != "" && c.PicsDir != "" {
		ok = false
		msg += fmt.Sprintf("the %s and %s flags cannot point to the same directory.\n", peopleDirFlagName, picsDirFlagName)
	}
	if info, err := os.Stat(c.PeopleDir); os.IsNotExist(err) {
		ok = false
		msg += fmt.Sprintf("directory %s specified by the flag %s does not exist.\n", c.PeopleDir, peopleDirFlagName)
	} else if !info.IsDir() {
		ok = false
		msg += fmt.Sprintf("directory %s specified by the flag %s is not a directory.\n", c.PeopleDir, peopleDirFlagName)
	}
	if info, err := os.Stat(c.PicsDir); os.IsNotExist(err) {
		ok = false
		msg += fmt.Sprintf("directory %s specified by the flag %s does not exist.\n", c.PicsDir, picsDirFlagName)
	} else if !info.IsDir() {
		ok = false
		msg += fmt.Sprintf("directory %s specified by the flag %s is not a directory.\n", c.PicsDir, picsDirFlagName)
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

	flags.StringVar(&c.PeopleDir, peopleDirFlagName, "", "directory where we can find the photos of the people we want to recognize")
	flags.StringVar(&c.PicsDir, picsDirFlagName, "", "directory where we can find all the photos we want to filter based on the people we want to recognize in peopledir")

	err = flags.Parse(args)
	if err != nil {
		return nil, buf.String(), err
	}
	return c, buf.String(), nil
}
