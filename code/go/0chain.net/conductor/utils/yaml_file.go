package utils

import (
	"fmt"
	"log"
	"strings"

	"gopkg.in/yaml.v3"
)

type YamlReader struct {
	content map[string]interface{}
}

func NewYamlReader(content []byte) (*YamlReader, error) {
	yr := new(YamlReader)
	yr.content = make(map[string]interface{})
	err := yaml.Unmarshal(content, &yr.content)
	if err != nil {
		return nil, err
	}

	log.Printf("raw_content: %v, content: %v\n", content, yr.content)
	return yr, nil
}

func (yr *YamlReader) ValidateValue(val interface{}) (error) {
	switch val.(type) {
	case int, int8, int16, int32, int64, uint8, uint16, uint32, uint64, float32, float64, bool, string:
		return nil
	default:
		return fmt.Errorf("invalid val %v (type: %T)", val, val)
	}
}

func (yr *YamlReader) SetKey(key string, val interface{}) (err error) {
	return yr.setKey(key, val)
}

func (yr *YamlReader) setKey(key string, val interface{}) (err error) {
	keyParts := strings.Split(key, ".")

	if len(keyParts) == 1 {
		if _, ok := yr.content[key]; !ok {
			return fmt.Errorf("key %s not found", key)
		}
		yr.content[key] = val
		return
	}

	
	var (
		level = yr.content
		levelValue interface{}
		ok = false
	)
	for i, keyPart := range keyParts {
		log.Printf("current level: %v\n", level)
		if levelValue, ok = level[keyPart]; !ok {
			return fmt.Errorf("key %s not found at level %d (%s)", key, i, strings.Join(keyParts[:i], "."))
		}
		
		if !ok {
			return fmt.Errorf("key %s not found at level %s (couldn't parse to map)", key, strings.Join(keyParts[:i], "."))
		}

		if i == len(keyParts) - 1 {
			level[keyPart] = val
			return
		} else {
			level, ok = levelValue.(map[string]interface{})
			if !ok {
				return fmt.Errorf("key %s not found at level %s (couldn't parse to map)", key, strings.Join(keyParts[:i], "."))
			}
		}
	}

	return
}

func (yr *YamlReader) String() (s string) {
	content, err := yaml.Marshal(yr.content)
	if err != nil {
		return ""
	}
	return string(content)
}