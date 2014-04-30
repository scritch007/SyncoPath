package main

import (
	"net/url"
	"errors"
	"os"
	"fmt"
	"github.com/bitly/go-simplejson"
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"code.google.com/p/goauth2/oauth"
	"strings"
	"bytes"
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
	var album Album
	album.Xmlns = "http://www.w3.org/2005/Atom"
  album.XmlnsMedia = "http://search.yahoo.com/mrss/"
  album.XmlnsGPhoto = "http://schemas.google.com/photos/2007"
  album.Category.Scheme = "tada"
  album.Category.Term = "titi"
  album.Title = "badada"
  album.Access = "private"
  output, err := xml.MarshalIndent(&album, "  ", "    ")
  if err != nil{
  	fmt.Println(err)
	}else{
		fmt.Println("", string(output))
	}

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
	fmt.Println(p.authStruct)
	p.initializationDone = false

	t := &oauth.Transport{Config: oauthCfg}
	t.Token = &oauth.Token{AccessToken: p.authStruct.AccessToken, RefreshToken: p.authStruct.RefreshToken}
	err = t.Refresh()
	p.authStruct.AccessToken = t.Token.AccessToken
	p.authStruct.RefreshToken = t.Token.RefreshToken
	if err != nil{
		return nil, err
	}
	return p, nil
}

func (p *PicasaSyncPlugin)Name() string{
	return "Picasa"
}

const albumFeedURL = "https://picasaweb.google.com/data/feed/api/user/default"

func (p *PicasaSyncPlugin)BrowseFolder(f string) (error, []SyncResourceInfo){

	if f == "/"{
		if !p.initializationDone{
			t := &oauth.Transport{Config: oauthCfg}
			t.Token = &oauth.Token{AccessToken: p.authStruct.AccessToken, RefreshToken: p.authStruct.RefreshToken}
			resp, err := t.Client().Get(albumFeedURL + "?alt=json")
			if err != nil{
				fmt.Println("Failed to retrieve information")
				return err, nil
			}else{
				fmt.Println("Got some information!!!")
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
	//Now we're browsing a folder
	return errors.New("Not Available"), nil
}
func (p *PicasaSyncPlugin)RemoveResource(r SyncResourceInfo) error{
	return errors.New("Not available")
}

type Category struct{
	XMLName xml.Name `xml:"category"`
	Scheme string `xml:"scheme,attr"`
	Term   string `xml:"term,attr"`
}

type Album struct {
	XMLName      xml.Name    `xml:"entry"`
	Xmlns        string      `xml:"xmlns,attr"`
	XmlnsMedia   string      `xml:"xmlns:media,attr"`
	XmlnsGPhoto  string      `xml:"xmlns:gphoto,attr"`
	Title        string      `xml:"title"`
	Category     Category    `xml:"category"`
	Access       string      `xml:"access"`
	Summary      string      `xml:"summary"`
	Comment      string      `xml:",comment"`
}

func (p *PicasaSyncPlugin)createAlbum(albumName string) error{
	var album Album
	album.Xmlns = "http://www.w3.org/2005/Atom"
  album.XmlnsMedia = "http://search.yahoo.com/mrss/"
  album.XmlnsGPhoto = "http://schemas.google.com/photos/2007"
  album.Category.Term = "http://schemas.google.com/photos/2007#album"
  album.Category.Scheme = "http://schemas.google.com/g/2005#kind"
  album.Access = "private"
  album.Title = albumName
  output, _ := xml.MarshalIndent(&album, "  ", "    ")
  //TODO Make the call to the Google API...
	t := &oauth.Transport{Config: oauthCfg}
	t.Token = &oauth.Token{AccessToken: p.authStruct.AccessToken, RefreshToken: p.authStruct.RefreshToken}
	resp, err := t.Client().Post("https://picasaweb.google.com/data/feed/api/user/default?alt=json", "application/atom+xml", bytes.NewReader(output))
	if nil != err{
		fmt.Println("Failed to create folder with error ", err)
		buf, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("Got this ", string(buf))
	}else{
		buf, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("Got this ", string(buf))
	}
  return nil
}

func (p *PicasaSyncPlugin)AddResource(r *SyncResourceInfo) error{
	fmt.Println("Adding new resource ", r.Name)
	if r.IsDir{
		//We need to create the new repository
		folder_name := r.Parent + "/" + r.Name
		album_name := p.buildFolderName(folder_name)
		album := p.getFolder(album_name)
		if album != nil{
			return errors.New("This album already exist, cannot create it")
		}

		//Update our information now, so reset the initializationDone flag...
		p.initializationDone = false
		p.BrowseFolder("/")
	}else{
		album_name := p.buildFolderName(r.Parent)
		album := p.getFolder(album_name)
		if album == nil{
			return errors.New("Adding file to a now existing folder")
		}
	}
	return errors.New("Not available")
}

func (p *PicasaSyncPlugin)buildFolderName(folder string) string{
	if folder == "/"{
		return "NOT_SORTED_REPOSITORY"
	}
	splits := strings.Split(folder, "/")
	folder_name := splits[1]
	if len(splits) > 2{
		folder_name += " ("
		folder_name += strings.Join(splits[2:], ", ")
		folder_name += ")"
	}
	return folder_name
}

//Check if the album name is in our list
func (p *PicasaSyncPlugin)getFolder(album_name string) *PicasaEntry{
	for _, album := range p.resp.Feed.Entries{
		if album.Title.Value == album_name{
			return &album
		}
	}
	return nil
}

func (p *PicasaSyncPlugin)HasFolder(folder string) bool{
	//Since Picasa doesn't handle "Sub Folder" construction, we'll use some rewriting name
	folder_name := p.buildFolderName(folder)
	return nil != p.getFolder(folder_name)
}
