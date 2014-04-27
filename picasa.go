package main

import (
	"net/url"
	"errors"
	"os"
	"fmt"
	"github.com/bitly/go-simplejson"
	"encoding/json"
	"io/ioutil"
  "code.google.com/p/goauth2/oauth"
)

var oauthCfg = &oauth.Config {
        //TODO: put your project's Client Id here.  To be got from https://code.google.com/apis/console
        ClientId: "106373453700-1rbn7j3e4ddvs68lmv7346evp3uif6i9.apps.googleusercontent.com",

        //TODO: put your project's Client Secret value here https://code.google.com/apis/console
        ClientSecret: "1xb5Q4FWDTMxoHBovwPXfWzm",

        //For Google's oauth2 authentication, use this defined URL
        AuthURL: "https://accounts.google.com/o/oauth2/auth",

        //For Google's oauth2 authentication, use this defined URL
        TokenURL: "https://accounts.google.com/o/oauth2/token",

        //To return your oauth2 code, Google will redirect the browser to this page that you have defined
        //TODO: This exact URL should also be added in your Google API console for this project within "API Access"->"Redirect URIs"
        RedirectURL: "urn:ietf:wg:oauth:2.0:oob",

        //This is the 'scope' of the data that you are asking the user's permission to access. For getting user's info, this is the url that Google has defined.
        Scope: "https://www.googleapis.com/auth/userinfo.profile https://picasaweb.google.com/data/",
    }


type picasaAuthStruct struct{
	AccessToken string
	RefreshToken string
}
type PicasaSyncPlugin struct{
	configFile string
	authStruct picasaAuthStruct
	initializationDone bool
	resp *PicasaMainResponse
}

func NewPicasaSyncPlugin(configFile string) (*PicasaSyncPlugin, error){
	if _, err := os.Stat(configFile); os.IsNotExist(err){
		//Now ask the user to go to the correct place
		fmt.Print("Please got to the following url https://accounts.google.com/o/oauth2/auth?scope=", url.QueryEscape(oauthCfg.Scope), "&redirect_uri=", oauthCfg.RedirectURL, "&response_type=code&client_id=", oauthCfg.ClientId, "\n")
		fmt.Print("Enter the code displayed on the website:\n")
		var code string
		_, _ = fmt.Scanln(&code)
		fmt.Print("You just entered ", code)
	  t := &oauth.Transport{Config: oauthCfg}
    // Exchange the received code for a token
    token, err := t.Exchange(code)
    if err != nil{
    	return nil, errors.New("Couldn't get credentials" + err.Error())
    }
    res, _ := json.Marshal(token)
    ioutil.WriteFile(configFile, res, 777)
	}
	file, err := ioutil.ReadFile(configFile)
	if err != nil{
		fmt.Printf("File error: %v\n", err)
		return nil, err
	}
	token, err := simplejson.NewJson(file)
	p := new(PicasaSyncPlugin)
	p.configFile = configFile
	p.authStruct.AccessToken = token.Get("AccessToken").MustString()
	p.authStruct.RefreshToken = token.Get("RefreshToken").MustString()
	p.initializationDone = false
	return p, nil
}

func (p *PicasaSyncPlugin)Name() string{
	return "Picasa"
}

const albumFeedURL = "https://picasaweb.google.com/data/feed/api/user/default"

func (p *PicasaSyncPlugin)BrowseFolder(f string) (error, []SyncResourceInfo){
	if !p.initializationDone && f == "/"{
		t := &oauth.Transport{Config: oauthCfg}
  	t.Token = &oauth.Token{AccessToken: p.authStruct.AccessToken, RefreshToken: p.authStruct.RefreshToken}
  	resp, err := t.Client().Get(albumFeedURL + "?alt=json")
  	if err != nil{
  		fmt.Println("Failed to retrieve information")
  		return err, nil
  	}

  	buf, err := ioutil.ReadAll(resp.Body)
    m, _ := PicasaParse(buf)
    p.resp = m
    p.initializationDone = true
	}
	var result = make([]SyncResourceInfo, len(p.resp.Feed.Entries))
  for i, album := range p.resp.Feed.Entries{
      //alb := chromecasa.Folder{Name:album.Name.Value, Id:album.Id.Value, Icon: album.Media.Icon[0].Url, Display: true, Browse: false}
      result[i] = SyncResourceInfo{Name:album.Name.Value, Path:album.Id.Value, Parent:f, IsDir: true}
  }
	return nil, result
}
func (p *PicasaSyncPlugin)RemoveResource(r SyncResourceInfo) error{
	return errors.New("Not available")
}
func (p *PicasaSyncPlugin)AddResource(r *SyncResourceInfo) error{
	fmt.Println("Adding new resource ", r.Name)
	return errors.New("Not available")
}
func (p *PicasaSyncPlugin)HasFolder(folder string) bool{
	return false
}
