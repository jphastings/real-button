package modules

import (
	"fmt"
	"strings"
)

type initModule func(map[string]interface{}) (Module, error)

var AvailableModules map[string]initModule

func init() {
	AvailableModules = make(map[string]initModule)
}

func Instantiate(name string, config map[string]interface{}) (Module, error) {
	newModule, ok := AvailableModules[name]
	if !ok {
		var modNames []string
		for modName := range AvailableModules {
			modNames = append(modNames, modName)
		}
		return nil, fmt.Errorf("no loaded module of the name '%s' (only %v)", name, strings.Join(modNames, ", "))
	}
	return newModule(config)
}
