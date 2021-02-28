package pkg

import (
	"fmt"
	"github.com/jphastings/vm-power/pkg/buttons"
	"github.com/jphastings/vm-power/pkg/led"
	"github.com/jphastings/vm-power/pkg/modules"
	_ "github.com/jphastings/vm-power/pkg/modules/demo"
	_ "github.com/jphastings/vm-power/pkg/modules/virt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"sync"
)

type yamlConfig struct {
	Modules map[string]map[string]interface{}
	Buttons []map[string]interface{}
}

type Config []interact
type interact struct {
	buttonPress chan<- modules.Press
	leds        <-chan led.State
}

func Load(path string) (*Config, error) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var yc yamlConfig
	if err := yaml.Unmarshal(yamlFile, &yc); err != nil {
		return nil, err
	}

	return configureModules(yc)
}

func configureModules(yc yamlConfig) (*Config, error) {
	instances := make(map[string]modules.Module)

	var c Config
	for _, modDef := range yc.Buttons {
		modName, err := extractKey(modDef, "module")
		if err != nil {
			return nil, err
		}

		instance, err := findInstance(instances, modName, yc)
		if err != nil {
			return nil, err
		}

		leds, bttn, err := instance.Configure(modDef)
		if err != nil {
			return nil, err
		}
		c = append(c, interact{
			buttonPress: bttn,
			leds:        leds,
		})
	}
	return &c, nil
}

func findInstance(instances map[string]modules.Module, modName string, yc yamlConfig) (modules.Module, error) {
	instance, ok := instances[modName]
	if ok {
		return instance, nil
	}

	pkgDef, ok := yc.Modules[modName]
	if !ok {
		return nil, fmt.Errorf("no module config of the name '%s'", modName)
	}
	pkgName, err := extractKey(pkgDef, "pkg")
	if err != nil {
		return nil, err
	}
	instance, err = modules.Instantiate(pkgName, pkgDef)
	if err != nil {
		return nil, err
	}
	instances[modName] = instance
	return instance, nil
}

func extractKey(theMap map[string]interface{}, key string) (string, error) {
	val, ok := theMap[key]
	if !ok {
		return "", fmt.Errorf("the '%s' key isn't present", key)
	}
	valStr, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("the '%s' key value (%v) isn't a string", key, val)
	}
	delete(theMap, key)
	return valStr, nil
}

func (c Config) Run() error {
	port, err := buttons.GetPort()
	if err != nil {
		return err
	}

	log.Println("Connected and ready")

	var mu sync.Mutex
	for idx, v := range c {
		led := led.New(uint8(idx), port, &mu)
		go led.DisplayChanges(v.leds)
	}

	presses := buttons.AwaitPress(port)
	for index := range presses {
		c[index].buttonPress <- modules.Press{}
	}

	return nil
}
