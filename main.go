package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/scritch007/go-simplejson"
	"io"
	"io/ioutil"
	"os"
)

type ConfigSync struct {
	Name   string
	Syncer SyncPlugin
}

func instantiatePlugin(name, config string) (*ConfigSync, error) {
	if name == "picasa" {
		fmt.Println("You selected picasa")
		plugin, err := NewPicasaSyncPlugin(config)
		if nil != err {
			return nil, err
		}
		return &ConfigSync{name, plugin}, nil
	} else if name == "local" {
		fmt.Println("You selected local")
		p, err := NewLocalSyncPlugin(config)
		if nil != err {
			return nil, err
		}
		return &ConfigSync{name, p}, nil
	} else if name == "debug" {
		fmt.Println("You selected debug")
		p, err := NewDebugSyncPlugin(config)
		if nil != err {
			return nil, err
		}
		return &ConfigSync{name, p}, nil
	}
	return nil, errors.New("Unknown type")
}

func configure() ([2]ConfigSync, error) {
	var command string
	var id = 0
	res := [2]ConfigSync{}
	for {
		if 2 == id {
			break
		}
		fmt.Printf("Please select what you want to do for ")
		if 0 == id {
			fmt.Println("source")
		} else {
			fmt.Println("destination")
		}
		fmt.Scanln(&command)
		if command == "picasa" || command == "local" || command == "debug" {
			tmp, err := instantiatePlugin(command, "")
			if nil != err {
				return res, nil
			}
			res[id] = *tmp
			id += 1
		} else {
			fmt.Printf("Possible value:\n\t- picasa\n\t- local\n\t- debug\n")
		}
	}
	return res, nil
}
func main() {

	var fromremote = false
	flag.BoolVar(&fromremote, "from_remote", false, "Take remote as source")

	var help = false
	flag.BoolVar(&help, "help", false, "Display Help")

	var verbose bool
	var verboseTooltip = "Enable the debug mode"

	flag.BoolVar(&verbose, "verbose", false, verboseTooltip)
	flag.BoolVar(&verbose, "v", false, verboseTooltip)

	var config string
	flag.StringVar(&config, "config", "", "Provide config File")

	var res [2]ConfigSync

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Mandatory argument:\n  sync_folder\n")
	}
	flag.Parse()
	if 0 != flag.NArg() || 0 == len(config) {
		flag.Usage()
		os.Exit(0)
	}

	var debugIO io.Writer
	if verbose {
		debugIO = os.Stdout
	} else {
		debugIO = ioutil.Discard
	}
	LogInit(debugIO, os.Stdout, os.Stdout, os.Stderr)

	//Now read stuff...
	if _, err := os.Stat(config); os.IsNotExist(err) {
		res, err = configure()
		if nil != err {
			fmt.Println("Failed to create configuration with error %s\n", err)
			os.Exit(1)
		} else {
			tmp, _ := json.Marshal(res)
			ioutil.WriteFile(config, tmp, 777)

		}

	} else {

		//TODO Read it

		file, err := ioutil.ReadFile(config)
		if err != nil {
			fmt.Printf("File error: %v\n", err)
		}

		configuration, err := simplejson.NewJson(file)
		if nil != err {
			os.Exit(2)
		}
		srcName := configuration.GetIndex(0).Get("Name").MustString()
		srcConfig, _ := configuration.GetIndex(0).Get("Syncer").Encode()
		dstName := configuration.GetIndex(1).Get("Name").MustString()
		dstConfig, _ := configuration.GetIndex(1).Get("Syncer").Encode()
		fmt.Printf("%s %s", string(srcConfig), string(dstConfig))
		tmp, err := instantiatePlugin(srcName, string(srcConfig))
		res[0] = *tmp
		tmp, err = instantiatePlugin(dstName, string(dstConfig))
		res[1] = *tmp

	}

	if help {
		flag.Usage()
		os.Exit(0)
	}

	var syncer = NewSyncer()
	syncer.Sync(res[0].Syncer, res[1].Syncer)

}
