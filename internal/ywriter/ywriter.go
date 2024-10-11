package yaml

import (
	"fmt"
	"log"
	"os"
	"time"

	y "gopkg.in/yaml.v3"
)

type Version struct {
	Version string    `yaml:"version"`
	Major   int       `yaml:"major"`
	Minor   int       `yaml:"minor"`
	Patch   int       `yaml:"patch"`
	Last    time.Time `yaml:"last"`
	Kind    string    `yaml:"kind"`
}

func Write(path string, major, minor, patch int, last time.Time, kind string) Version {
	version := Version{
		Version: fmt.Sprintf("v%d.%d.%d", major, minor, patch),
		Major:   major,
		Minor:   minor,
		Patch:   patch,
		Last:    last,
		Kind:    kind}
	yamlBytes, err := y.Marshal(version)

	if err != nil {
		panic(err)
	}

	f, err := os.Create(path + ".lazyver.yaml")
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.Write(yamlBytes)
	if err != nil {
		log.Fatal(err)
	}
	return version
}

func Read(path string) (*Version, error) {
	data, err := os.ReadFile(path + ".lazyver.yaml")
	if err != nil {
		return nil, err
	}

	version := new(Version)
	err = y.Unmarshal(data, version)

	if err != nil {
		return nil, err
	}

	return version, nil
}
