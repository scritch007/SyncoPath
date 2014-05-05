package main

import (
	//"io/ioutil"
	"flag"
	"fmt"
	"os"
)

func main() {
	var help = false
	var configFile = ".config"
	var debug string
	flag.BoolVar(&help, "help", false, "Display Help")
	flag.StringVar(&configFile, "config", ".config", "Configuration file to use")
	flag.StringVar(&debug, "debug", "", "Use the debug mode")
	flag.StringVar(&debug, "d", "", "Use the debug mode")

	var picasa string
	flag.StringVar(&picasa, "picasa", "", "Use picasa as destination")
	flag.StringVar(&picasa, "p", "", "Use picasa as destination")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Mandatory argument:\n  sync_folder\n")
	}
	flag.Parse()
	if 1 != flag.NArg() {
		flag.Usage()
		os.Exit(0)
	}

	var local_folder = flag.Arg(0)

	if help {
		flag.Usage()
		os.Exit(0)
	}

	var syncPlugin SyncPlugin

	var err error
	if debug != "" {
		syncPlugin, err = NewDebugSyncPlugin(debug)
	} else if picasa != "" {
		syncPlugin, err = NewPicasaSyncPlugin(picasa)
	}
	if nil != err {
		fmt.Println("Failed to instantiate plugin with error ", err)
		os.Exit(1)
	}

	var syncer = NewSyncer(local_folder)
	syncer.Sync(syncPlugin)

}
