package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/bitly/go-simplejson"

	"github.com/spf13/cobra"
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

var rootCmd = cobra.Command{
	Use:   "SyncoPath",
	Short: "Sync things with various backend",
	Long:  `[Warning] Picasa doesn't work as destination because folder creation is forbidden now`,
	Run:   mainFunc,
}

var (
	fromremote bool
	help       bool
	verbose    bool
	config     string
)

func init() {
	rootCmd.PersistentFlags().BoolVar(&fromremote, "from_remote", false, "Inverse configuration")
	rootCmd.PersistentFlags().BoolVar(&help, "help", false, "Show help")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose mode")
	rootCmd.PersistentFlags().StringVar(&config, "config", "", "config file (default is $HOME/.cobra.yaml)")
}
func main() {
	rootCmd.Execute()
}

func mainFunc(cmd *cobra.Command, args []string) {
	var res [2]configSync

	if 0 != len(args) || 0 == len(config) {
		cmd.Usage()
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
