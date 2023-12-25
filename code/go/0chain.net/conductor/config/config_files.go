package config

import (
	"fmt"
	"io"
	"log"
	"os"

	"0chain.net/conductor/utils"
)

type ConfigFileChanges struct {
	FileName string `json:"file_name" yaml:"file_name" mapstructure:"file_name"`
	Changes []ConfigChange `json:"changes" yaml:"changes" mapstructure:"changes"`
}

type ConfigChange struct {
	Key string `json:"key" yaml:"key" mapstructure:"key"`
	Value interface{} `json:"value" yaml:"value" mapstructure:"value"`
}

type ConfigFile struct {
	Name string `json:"name" yaml:"name" mapstructure:"name"`
	Path string `json:"path" yaml:"path" mapstructure:"path"`
}

func (cf *ConfigFile) Update(changes []ConfigChange) (error) {
	// Read config file and back it up
	
	log.Printf("reading config file: %v\n", cf.Path)
	
	file, err := os.Open(cf.Path)
	if err != nil {
		return fmt.Errorf("opening config file (%s): %v", cf.Path, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return  fmt.Errorf("reading config file (%s): %v", cf.Path, err)
	}

	err = os.WriteFile(getBackupFilePath(cf.Path), content, 0644)
	if err != nil {
		return fmt.Errorf("creating backup file (%s): %v", cf.Path + ".bak", err)
	}

	log.Printf("content %v\n", content)

	reader, err := utils.NewYamlReader(content)
	if err != nil {
		return fmt.Errorf("creating yaml reader for config file (%s): %v", cf.Path, err)
	}

	// Update config file
	for _, change := range changes {
		if err := reader.ValidateValue(change.Value); err != nil {
			log.Printf("[ERR] invalid value for key %s: %v\n", change.Key, err)
		}
		
		err = reader.SetKey(change.Key, change.Value)
		if err != nil {
			return fmt.Errorf("updating config file (%s): %v", cf.Path, err)
		}
	}

	// Write updated config file
	err = os.WriteFile(cf.Path, []byte(reader.String()), 0644)
	if err != nil {
		return fmt.Errorf("writing config file (%s): %v", cf.Path, err)
	}

	return nil
}

func (cf *ConfigFile) Restore() error {
	// Restore config file from backup
	err := os.Rename(getBackupFilePath(cf.Path), cf.Path)
	if err != nil {
		return fmt.Errorf("restoring config file (%s): %v", cf.Path, err)
	}

	return nil
}

func getBackupFilePath(configFile string) (string) {
	return configFile + ".bak"
}