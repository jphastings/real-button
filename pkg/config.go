package pkg

import (
	"fmt"
	"io/ioutil"
	"log"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/jphastings/real-button/pkg/buttons"
	"github.com/jphastings/real-button/pkg/device"
	"github.com/jphastings/real-button/pkg/led"
	"github.com/jphastings/real-button/pkg/modules"
	_ "github.com/jphastings/real-button/pkg/modules/demo"
	_ "github.com/jphastings/real-button/pkg/modules/virt"
)

type yamlConfig struct {
	Modules map[string]map[string]interface{}
	Buttons []map[string]interface{}
}

type Config []modules.Configured

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

		configured, err := instance.Configure(modDef)
		if err != nil {
			return nil, err
		}
		c = append(c, configured)
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
	port, buttonCount, err := device.GetPort()
	if err != nil {
		return err
	}

	if len(c) != buttonCount {
		return fmt.Errorf("%d buttons configured but %s buttons available\n", len(c), buttonCount)
	}

	log.Println("Connected to buttons, and ready")

	var writeMutex sync.Mutex

	for idx, instance := range c {
		l := led.New(uint8(idx), port, &writeMutex)
		go l.DisplayChanges(instance.LEDState)
		go runInstance(instance)
	}

	presses := buttons.AwaitPress(port)
	for index := range presses {
		if index >= len(c) {
			log.Println("Received invalid pushbutton number:", index)
			continue
		}
		c[index].ButtonPress <-modules.Press{}
	}

	return nil
}

func runInstance(instance modules.Configured) {
	log.Fatal(instance.Run())
}
