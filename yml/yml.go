package yml

import (
	"github.com/ghodss/yaml"
	"io/ioutil"
	"path/filepath"
)

func Read(path string, value interface{}) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if err = yaml.Unmarshal(data, &value); err != nil {
		return err
	}
	return nil
}

func Write(path string, value interface{}) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(path, data, 0644); err != nil {
		return err
	}
	return nil
}
