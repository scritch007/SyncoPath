package main

import (
  //"io/ioutil"
  "os"
  "fmt"
  "net/url"
  "flag"
)

var clientId = "106373453700.apps.googleusercontent.com"
var clientSecret = "x_1Ebngp5sfvKkB-vqN-Q260"
var scope = "https://www.googleapis.com/auth/userinfo.profile https://picasaweb.google.com/data/"

func main() {
	var help = false
  var configFile = ".config"
  var debug string
	flag.BoolVar(&help, "help", false, "Display Help")
	flag.StringVar(&configFile, "config", ".config", "Configuration file to use")
	flag.StringVar(&debug, "debug", "", "Use the debug mode")
	flag.StringVar(&debug, "d", "", "Use the debug mode")

	flag.Usage = func(){
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
    flag.PrintDefaults()
    fmt.Fprintf(os.Stderr, "Mandatory argument:\n  sync_folder\n")
	}
	flag.Parse()
	if 1 != flag.NArg(){
		flag.Usage()
		os.Exit(0)
	}

	var local_folder = flag.Arg(0)

  if help {
  	flag.Usage()
  	os.Exit(0)
  }
  if debug == "" {
		if _, err := os.Stat(configFile); os.IsNotExist(err){
			//Now ask the user to go to the correct place
			fmt.Print("Please got to the following url https://accounts.google.com/o/oauth2/auth?scope=", url.QueryEscape(scope), "&redirect_uri=urn:ietf:wg:oauth:2.0:oob&response_type=code&client_id=", clientId, "\n")
			fmt.Print("Enter the code displayed on the website:\n")
			var code string
			_, _ = fmt.Scanln(&code)
			fmt.Print("You just entered ", code)
		}
	}

	var syncer = NewSyncer(local_folder)
	syncer.Sync()

}