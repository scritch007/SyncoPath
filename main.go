package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/scritch007/go-simplejson"
)

type configSync struct {
	Name   string
	Syncer SyncPlugin
}

func configure() ([2]configSync, error) {
	var command string
	var id = 0
	res := [2]configSync{}
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
		for k := range syncPluginsList {
			fmt.Printf("%s ", k)
		}
		fmt.Println("")
		fmt.Scanln(&command)
		found := false
		for k, r := range syncPluginsList {
			if k == command {
				tmp, err := r.NewMethod("")
				if nil != err {
					return res, err
				}
				res[id] = configSync{Name: command, Syncer: tmp}
				id++
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("Incorrect input %s\n", command)
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

	var res [2]configSync

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
	logInit(debugIO, os.Stdout, os.Stdout, os.Stderr)

	//Now read stuff...
	if _, err := os.Stat(config); os.IsNotExist(err) {
		res, err = configure()
		if nil != err {
			log.Fatalf("Failed to create configuration with error %s\n", err)
		} else {
			tmp, _ := json.Marshal(res)
			ioutil.WriteFile(config, tmp, 777)
		}

	} else {

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
		tmp, err := syncPluginsList[srcName].NewMethod(string(srcConfig))
		if err != nil {
			log.Fatalf("Failed to instantiate source plugin %s : %v", srcName, err)
		}
		res[0] = configSync{
			Name:   srcName,
			Syncer: tmp,
		}
		tmp, err = syncPluginsList[dstName].NewMethod(string(dstConfig))
		if err != nil {
			log.Fatalf("Failed to instantiate destination plugin %s : %v", dstName, err)
		}
		res[1] = configSync{
			Name:   dstName,
			Syncer: tmp,
		}

	}

	if help {
		flag.Usage()
		os.Exit(0)
	}

	var syncer = NewSyncer()
	syncer.Sync(res[0].Syncer, res[1].Syncer)

}
