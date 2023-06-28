package entity

import (
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type Warning error

type Entity struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
}

/*
 * parseEntity is a generic function that is used to parse any JSON file
 * and discover the apiVersion and kind of the file.
 */
func parseEntity(fs afero.Fs, filename string) (*Entity, error) {
	filecontent, err := afero.ReadFile(fs, filename)
	if err != nil {
		return nil, err
	}

	yamldata := &Entity{}
	err = yaml.Unmarshal(filecontent, yamldata)
	if err != nil {
		return nil, err
	}

	return yamldata, nil
}
