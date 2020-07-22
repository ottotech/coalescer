package main

import (
	"fmt"
	"strings"
	"testing"
)

const (
	testPeopleDir = "people_dir"
	testPicsDir   = "pics_dir"
)

func TestConfig_Transform(t *testing.T) {
	c, err := newConfig()
	if err != nil {
		t.Fatal(err)
	}

	// These are all the options that config.Transform will transform.
	c.Confidence = 70
	c.Combine = "pepe,julia"
	c.MatchMultiple = true

	c.Transform()

	expectedConfidence := float64(70) / float64(100)
	if c.Confidence != expectedConfidence {
		t.Errorf("expected a confidenßce value of %v got instead %f", expectedConfidence, c.Confidence)
	}

	for _, x := range []string{"pepe", "julia"} {
		exists := true
		for _, y := range c.PeopleCombined {
			if x == y {
				exists = true
				break
			}
		}
		if !exists {
			t.Errorf("expected person (%s) in PeopleCombined", x)
		}
	}

	if !c.MatchMultiple {
		t.Errorf("expected option MatchMultiple to be true got %t instead.", c.MatchMultiple)
	}

	expectedDirName := strings.Join(c.PeopleCombined, "_")
	if c.PeopleCombinedDirName != expectedDirName {
		t.Errorf("expected PeopleCombinedDirName to be %s; got %q instead", expectedDirName, c.PeopleCombinedDirName)
	}
}

func TestConfig_Validate(t *testing.T) {
	scenarios := []struct {
		desc       string
		msg        string
		getConf    func() *config
		shouldFail bool
	}{
		{
			desc: "happy path",
			getConf: func() *config {
				c, err := newConfig()
				if err != nil {
					t.Fatal(err)
				}
				c.FaceboxUrl = "http://localhost:8080"
				c.Confidence = 70
				c.Combine = "pepe,julia"
				c.MatchMultiple = true
				c.PicsDir = testPicsDir
				c.PeopleDir = testPeopleDir
				return c
			},
			shouldFail: false,
		},
		{
			desc: "conf without PeopleDir field should be invalid",
			getConf: func() *config {
				c, err := newConfig()
				if err != nil {
					t.Fatal(err)
				}
				c.FaceboxUrl = "http://localhost:8080"
				c.Confidence = 70
				c.Combine = "pepe,julia"
				c.MatchMultiple = true
				c.PicsDir = testPicsDir
				return c
			},
			shouldFail: true,
		},
		{
			desc: "conf without PicsDir field should be invalid",
			getConf: func() *config {
				c, err := newConfig()
				if err != nil {
					t.Fatal(err)
				}
				c.FaceboxUrl = "http://localhost:8080"
				c.Confidence = 70
				c.Combine = "pepe,julia"
				c.MatchMultiple = true
				c.PeopleDir = testPeopleDir
				return c
			},
			shouldFail: true,
		},
		{
			desc: "conf with the PicsDir and PeopleDir values should be invalid",
			getConf: func() *config {
				c, err := newConfig()
				if err != nil {
					t.Fatal(err)
				}
				c.FaceboxUrl = "http://localhost:8080"
				c.Confidence = 70
				c.Combine = "pepe,julia"
				c.MatchMultiple = true
				c.PeopleDir = "same_dir"
				c.PicsDir = "same_dir"
				return c
			},
			shouldFail: true,
		},
		{
			desc: "conf with nonexistent PeopleDir should be invalid",
			getConf: func() *config {
				c, err := newConfig()
				if err != nil {
					t.Fatal(err)
				}
				c.FaceboxUrl = "http://localhost:8080"
				c.Confidence = 70
				c.Combine = "pepe,julia"
				c.MatchMultiple = true
				c.PicsDir = testPicsDir
				c.PeopleDir = "nonexistent"
				return c
			},
			shouldFail: true,
		},
		{
			desc: "conf with nonexistent PicsDir should be invalid",
			getConf: func() *config {
				c, err := newConfig()
				if err != nil {
					t.Fatal(err)
				}
				c.FaceboxUrl = "http://localhost:8080"
				c.Confidence = 70
				c.Combine = "pepe,julia"
				c.MatchMultiple = true
				c.PicsDir = "nonexistent"
				c.PeopleDir = testPeopleDir
				return c
			},
			shouldFail: true,
		},
		{
			desc: "conf only one person to combine should be invalid",
			getConf: func() *config {
				c, err := newConfig()
				if err != nil {
					t.Fatal(err)
				}
				c.FaceboxUrl = "http://localhost:8080"
				c.Confidence = 70
				c.Combine = "pepe"
				c.MatchMultiple = true
				c.PicsDir = testPicsDir
				c.PeopleDir = testPeopleDir
				return c
			},
			shouldFail: true,
		},
	}

	for _, scenario := range scenarios {
		if scenario.shouldFail {
			c := scenario.getConf()
			if ok, _ := c.Validate(); ok {
				t.Errorf("conf should be invalid when testing scenario (%s)", scenario.desc)
			}
		} else {
			c := scenario.getConf()
			if ok, _ := c.Validate(); !ok {
				t.Errorf("conf should be valid when testing scenario (%s)", scenario.desc)
			}
		}
	}
}

func TestConfig_CheckPeopleCombination(t *testing.T) {
	c, err := newConfig()
	if err != nil {
		t.Fatal(err)
	}

	// In order to test the conf method CheckPeopleCombination we just need to reference some fields
	// in the config struct: PeopleCombined and People. So we will just manually set those values for
	// testing purposes.
	c.PeopleCombined = []string{"pepe", "julia"}
	p := make(PeopleToIdentify, 0)
	p["pepe"] = append(p["pepe"], "/some-path")
	p["julia"] = append(p["pepe"], "/some-path")
	c.People = p

	// Because the people stored in the map field People are also map in the field PeopleCombined
	// CheckPeopleCombination shouldn't fail.
	if success := c.CheckPeopleCombination(); !success {
		t.Errorf("CheckPeopleCombination should have returned true got %t instead", success)
	}

	// Now let's try to remove pepe from the field PeopleCombined.
	c.PeopleCombined = c.PeopleCombined[1:]

	fmt.Println(c.PeopleCombined)
	// Because we modified the field PeopleCombined with line from above  CheckPeopleCombination
	// should fail now.
	if success := c.CheckPeopleCombination(); success {
		t.Errorf("CheckPeopleCombination should have returned false got %t instead", success)
	}
}
