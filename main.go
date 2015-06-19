package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

func main() {
	var help = false
	var debug string
	var verbose bool
	var verboseTooltip = "Enable the debug mode"
	flag.BoolVar(&help, "help", false, "Display Help")

	flag.BoolVar(&verbose, "verbose", false, verboseTooltip)
	flag.BoolVar(&verbose, "v", false, verboseTooltip)

	flag.StringVar(&debug, "debug", "", "Use the debug mode")
	flag.StringVar(&debug, "d", "", "Use the debug mode")

	var picasa string
	flag.StringVar(&picasa, "picasa", "", "picasa plugin configuration path")
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
	} else {
		fmt.Println("What you didn't provide any backend")
		os.Exit(1)
	}
	if nil != err {
		fmt.Println("Failed to instantiate plugin with error ", err)
		os.Exit(1)
	}

	var debugIO io.Writer
	if verbose {
		debugIO = os.Stdout
	} else {
		debugIO = ioutil.Discard
	}

	LogInit(debugIO, os.Stdout, os.Stdout, os.Stderr)

	var syncer = NewSyncer(local_folder)
	syncer.Sync(syncPlugin)

}
