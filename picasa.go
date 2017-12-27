package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	//"net/http/httputil"
	"context"

	"github.com/scritch007/go-simplejson"
)

var oauthCfg = &oauth2.Config{
	//TODO: put your project's Client ID here.  To be got from https://code.google.com/apis/console
	ClientID: "106373453700-1rbn7j3e4ddvs68lmv7346evp3uif6i9.apps.googleusercontent.com",

	//TODO: put your project's Client Secret value here https://code.google.com/apis/console
	ClientSecret: "1xb5Q4FWDTMxoHBovwPXfWzm",

	Endpoint: google.Endpoint,

	//To return your oauth2 code, Google will redirect the browser to this page that you have defined
	//TODO: This exact URL should also be added in your Google API console for this project within "API Access"->"Redirect URIs"
	RedirectURL: "urn:ietf:wg:oauth:2.0:oob",

	//This is the 'scope' of the data that you are asking the user's permission to access. For getting user's info, this is the url that Google has defined.
	Scopes: []string{
		"https://www.googleapis.com/auth/userinfo.profile",
		"https://picasaweb.google.com/data/",
	},
}

const albumFeedURL = "https://picasaweb.google.com/data/feed/api/user/default"

// PicasaSyncPlugin structure
type PicasaSyncPlugin struct {
	initializationDone bool
	resp               *picasaMainResponse
	lock               sync.Mutex
	AuthStruct         oauth2.Token
}

// NewPicasaSyncPlugin instantiate pluging
func NewPicasaSyncPlugin(config string) (*PicasaSyncPlugin, error) {

	if 0 == len(config) {
		//Now ask the user to go to the correct place
		fmt.Print("Please got to the following url ", oauthCfg.AuthCodeURL("state"), "\n")
		fmt.Print("Enter the code displayed on the website:\n")
		var code string
		_, _ = fmt.Scanln(&code)
		fmt.Print("You just entered ", code)
		tok, err := oauthCfg.Exchange(context.Background(), code)

		if err != nil {
			return nil, errors.New("Couldn't get credentials" + err.Error())
		}
		tmp, _ := json.Marshal(PicasaSyncPlugin{AuthStruct: *tok})
		config = string(tmp)
	}

	token, err := simplejson.NewJson([]byte(config))
	if nil != err {
		return nil, err
	}
	p := new(PicasaSyncPlugin)

	fmt.Println(token.Get("AuthStruct").Get("access_token").MustString())
	p.AuthStruct.AccessToken = token.Get("AuthStruct").Get("access_token").MustString()
	p.AuthStruct.RefreshToken = token.Get("AuthStruct").Get("refresh_token").MustString()
	p.AuthStruct.TokenType = token.Get("AuthStruct").Get("token_type").MustString()
	//Force refresh token now
	p.AuthStruct.Expiry = time.Now()
	p.initializationDone = false

	return p, nil
}

// Name ...
func (p *PicasaSyncPlugin) Name() string {
	return "Picasa"
}

// Lock ...
func (p *PicasaSyncPlugin) Lock() {
	DEBUG.Println("Locking")
	p.lock.Lock()
}

// Unlock ...
func (p *PicasaSyncPlugin) Unlock() {
	p.lock.Unlock()
	DEBUG.Println("Unlocked")
}

func (p *PicasaSyncPlugin) browseAlbum(url string) (*picasaMainResponse, error) {
	client := oauthCfg.Client(context.Background(), &p.AuthStruct)
	resp, err := client.Get(url)
	DEBUG.Printf("Accessing this URL %s\n", url)
	if err != nil {
		return nil, err
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	DEBUG.Printf("Received %s\n", buf)
	return picasaParse(buf)
}

// BrowseFolder ...
func (p *PicasaSyncPlugin) BrowseFolder(f string) ([]SyncResourceInfo, error) {
	var parsedResp *picasaMainResponse
	var url string
	p.Lock()
	if !p.initializationDone {
		var err error
		url = albumFeedURL + "?alt=json&imgmax=d"
		parsedResp, err = p.browseAlbum(url)
		if err != nil {
			return nil, err
		}
		p.resp = parsedResp
	}
	p.Unlock()

	if f == "" {
		return make([]SyncResourceInfo, 0), nil
	}
	//Now we're browsing a folder
	albumName := p.buildFolderName(f)
	album := p.getFolder(albumName)
	if nil == album {
		return make([]SyncResourceInfo, 0), nil
	}

	url = albumFeedURL + "/albumid/" + album.ID.Value + "?alt=json&kind=photo&imgmax=d"
	parsedResp, err := p.browseAlbum(url)
	if err != nil {
		return nil, nil
	}

	resultSize := len(parsedResp.Feed.Entries)
	if "/" == f {
		//Special case we need to return both the images and the albums
		resultSize += len(p.resp.Feed.Entries)
	}

	var result = make([]SyncResourceInfo, resultSize)

	for i, album := range parsedResp.Feed.Entries {
		name := album.Title.Value
		result[i] = SyncResourceInfo{Name: name, Path: album.ID.Value, Parent: f}
		if album.Category[0].Term == "http://schemas.google.com/photos/2007#photo" {
			result[i].IsDir = false
			result[i].ExtraInfo = album.Media.Content[0].URL
			INFO.Printf("YESS!!! %s => %s\n", name, result[i].ExtraInfo)
		} else {
			result[i].IsDir = true
		}
	}
	if "/" == f {
		for i, album := range p.resp.Feed.Entries {
			name := album.Title.Value
			result[i+len(parsedResp.Feed.Entries)] = SyncResourceInfo{Name: name, Path: album.ID.Value, Parent: f, IsDir: true}
		}
	}
	INFO.Printf("%v", result)
	return result, nil

}

// RemoveResource ...
func (p *PicasaSyncPlugin) RemoveResource(r SyncResourceInfo) error {
	return errors.New("Not available")
}

func downloadFromURL(url, path string) error {
	INFO.Println("Downloading", url, "to", path)

	// TODO: check file existence first with io.IsExist
	output, err := os.Create(path)
	if err != nil {
		ERROR.Println("Error while creating", path, "-", err)
		return err
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		ERROR.Println("Error while downloading", url, "-", err)
		return err
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		ERROR.Println("Error while downloading", url, "-", err)
		return err
	}

	DEBUG.Println(n, "bytes downloaded.")
	return nil
}

// DownloadResource ...
func (p *PicasaSyncPlugin) DownloadResource(r *SyncResourceInfo) error {
	INFO.Printf("Now downloading %s", r)
	return downloadFromURL(r.ExtraInfo, r.Path)
}

type category struct {
	XMLName xml.Name `xml:"category"`
	Scheme  string   `xml:"scheme,attr"`
	Term    string   `xml:"term,attr"`
}

type xmlAlbum struct {
	XMLName     xml.Name `xml:"entry"`
	Xmlns       string   `xml:"xmlns,attr"`
	XmlnsMedia  string   `xml:"xmlns:media,attr"`
	XmlnsGPhoto string   `xml:"xmlns:gphoto,attr"`
	Title       string   `xml:"title"`
	Category    category `xml:"category"`
	Access      string   `xml:"access"`
	Summary     string   `xml:"summary"`
	Comment     string   `xml:",comment"`
}

type xmlPhoto struct {
	XMLName  xml.Name `xml:"entry"`
	Xmlns    string   `xml:"xmlns,attr"`
	Title    string   `xml:"title"`
	Category category `xml:"category"`
	Summary  string   `xml:"summary"`
	Comment  string   `xml:",comment"`
}

func (p *PicasaSyncPlugin) createAlbum(albumName string) error {
	var album xmlAlbum
	album.Xmlns = "http://www.w3.org/2005/Atom"
	album.XmlnsMedia = "http://search.yahoo.com/mrss/"
	album.XmlnsGPhoto = "http://schemas.google.com/photos/2007"
	album.Category.Term = "http://schemas.google.com/photos/2007#album"
	album.Category.Scheme = "http://schemas.google.com/g/2005#kind"
	album.Access = "private"
	album.Title = albumName
	output, _ := xml.MarshalIndent(&album, "  ", "    ")
	//TODO Make the call to the Google API...

	client := oauthCfg.Client(context.Background(), &p.AuthStruct)
	resp, err := client.Post("https://picasaweb.google.com/data/feed/api/user/default?alt=json", "application/atom+xml", bytes.NewReader(output))
	if nil != err {
		ERROR.Println("Failed to create folder with error ", err)
		buf, _ := ioutil.ReadAll(resp.Body)
		ERROR.Println("Got this ", string(buf))
	} else {
		buf, _ := ioutil.ReadAll(resp.Body)
		DEBUG.Println("Got this ", string(buf))
	}
	return nil
}

func (p *PicasaSyncPlugin) uploadPhoto(inAlbum *picasaEntry, r *SyncResourceInfo) error {
	bodyBuf := bytes.NewBufferString("")
	bodyWriter := multipart.NewWriter(bodyBuf)

	boundary := bodyWriter.Boundary()

	/* Create a completely custom Form Part (or in this case, a file) */
	// http://golang.org/src/pkg/mime/multipart/writer.go?s=2274:2352#L86
	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Type", "application/atom+xml")
	partWriter, err := bodyWriter.CreatePart(mh)
	//TODO create the xml content for the file
	if nil != err {
		panic(err.Error())
	}

	var photo xmlPhoto
	photo.Xmlns = "http://www.w3.org/2005/Atom"
	photo.Title = r.Name
	photo.Category.Term = "http://schemas.google.com/photos/2007#photo"
	photo.Category.Scheme = "http://schemas.google.com/g/2005#kind"
	output, _ := xml.MarshalIndent(&photo, "  ", "    ")
	io.Copy(partWriter, bytes.NewBuffer(output))

	mh2 := make(textproto.MIMEHeader)
	mh2.Set("Content-Type", r.MimeType)
	//open the file
	file, err := os.Open(r.Path)
	if err != nil {
		return errors.New("Couldn't read file from the filesystem")
	}
	defer file.Close()

	fileWriter, err := bodyWriter.CreatePart(mh2)
	if nil != err {
		panic(err.Error())
	}

	io.Copy(fileWriter, file)

	/* Close the body and send the request */
	bodyWriter.Close()

	client := oauthCfg.Client(context.Background(), &p.AuthStruct)

	requestBody, err := ioutil.ReadAll(bodyBuf)
	//DO call
	uri := albumFeedURL + "/albumid/" + inAlbum.ID.Value
	request, err := http.NewRequest("POST", uri, bytes.NewBuffer(requestBody))
	contentType := "multipart/related; boundary=\"" + boundary + "\""
	request.Header.Set("Content-Type", contentType)
	//request.Header.Set("Content-Length", len(body_buf))
	if nil != err {
		return errors.New("Failed to create new Request")
	}
	//dump, err := httputil.DumpRequest(request, false)
	//fmt.Println(string(dump))

	//resp, err := t.Client().Post(uri, content_type, body_buf)
	resp, err := client.Do(request)
	if nil != err {
		if nil != resp {
			body, _ := ioutil.ReadAll(resp.Body)
			DEBUG.Println(body)
		}
		panic(err.Error())
	}

	/* Handle the response */
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		ERROR.Println("Whouch got and error but nothing printed", err, string(body))
		return err
	}
	return nil
}

// AddResource ...
func (p *PicasaSyncPlugin) AddResource(r *SyncResourceInfo) error {
	DEBUG.Printf("Adding new resource %s\n", r.Name)
	if r.IsDir {
		//We need to create the new repository
		var folderName string
		if r.Parent == "/" {
			folderName = "/" + r.Name
		} else {
			folderName = r.Parent + "/" + r.Name
		}
		albumName := p.buildFolderName(folderName)
		album := p.getFolder(albumName)
		if album != nil {
			return errors.New("This album already exist, cannot create it")
		}
		p.createAlbum(albumName)
		//Update our information now, so reset the initializationDone flag...
		p.initializationDone = false
		p.BrowseFolder("")
		return nil
	}
	albumName := p.buildFolderName(r.Parent)
	album := p.getFolder(albumName)
	if album == nil {
		return errors.New("Can't add entry " + r.Name + "the folder " + albumName + " doesn't exist")
	}
	return p.uploadPhoto(album, r)

}

func (p *PicasaSyncPlugin) buildFolderName(folder string) string {
	if folder == "/" {
		return "NOT_SORTED_REPOSITORY"
	}
	splits := strings.Split(folder, "/")
	folderName := splits[1]
	if len(splits) > 2 {
		folderName += " ("
		folderName += strings.Join(splits[2:], ", ")
		folderName += ")"
	}
	//fmt.Printf("Hey you \n############################\n%s=>\n%s\n => %s#######################\n", folder, splits, folder_name)
	return folderName
}

//Check if the album name is in our list
func (p *PicasaSyncPlugin) getFolder(albumName string) *picasaEntry {
	p.Lock()
	defer p.Unlock()
	for _, album := range p.resp.Feed.Entries {
		//fmt.Printf("\n@@@@@@@@@@@@@@@@@@@@@@\n%s vs %s\n@@@@@@@@@@@@@@@@@@@@@@@@@@\n", album.Title.Value, album_name)
		if album.Title.Value == albumName {
			return &album
		}
	}
	return nil
}

// HasFolder ...
func (p *PicasaSyncPlugin) HasFolder(folder string) bool {
	//Since Picasa doesn't handle "Sub Folder" construction, we'll use some rewriting name
	if !p.initializationDone {
		_, err := p.BrowseFolder(folder)
		if nil != err {
			ERROR.Printf("Failed to browse for initialisation with error %s\n", err.Error())
			return false
		}
	}
	folderName := p.buildFolderName(folder)
	return nil != p.getFolder(folderName)
}

// GetResourceInfo ...
func (p *PicasaSyncPlugin) GetResourceInfo(folder string) (SyncResourceInfo, error) {
	return SyncResourceInfo{}, nil
}
