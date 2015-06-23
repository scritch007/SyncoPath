package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/scritch007/go-simplejson"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"sync"
	"time"
	//"net/http/httputil"
)

var oauthCfg = &oauth2.Config{
	//TODO: put your project's Client Id here.  To be got from https://code.google.com/apis/console
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

type PicasaSyncPlugin struct {
	initializationDone bool
	resp               *PicasaMainResponse
	lock               sync.Mutex
	AuthStruct         oauth2.Token
}

func NewPicasaSyncPlugin(config string) (*PicasaSyncPlugin, error) {

	if 0 == len(config) {
		//Now ask the user to go to the correct place
		fmt.Print("Please got to the following url ", oauthCfg.AuthCodeURL("state"), "\n")
		fmt.Print("Enter the code displayed on the website:\n")
		var code string
		_, _ = fmt.Scanln(&code)
		fmt.Print("You just entered ", code)
		tok, err := oauthCfg.Exchange(oauth2.NoContext, code)

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

func (p *PicasaSyncPlugin) Name() string {
	return "Picasa"
}

func (p *PicasaSyncPlugin) Lock() {
	DEBUG.Println("Locking")
	p.lock.Lock()
}

func (p *PicasaSyncPlugin) Unlock() {
	p.lock.Unlock()
	DEBUG.Println("Unlocked")
}

func (p *PicasaSyncPlugin) browseAlbum(url string) (error, *PicasaMainResponse) {
	client := oauthCfg.Client(oauth2.NoContext, &p.AuthStruct)
	resp, err := client.Get(url)
	DEBUG.Printf("Accessing this URL %s\n", url)
	if err != nil {
		return err, nil
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err, nil
	}
	DEBUG.Printf("Received %s\n", buf)
	parsedResp, _ := PicasaParse(buf)
	if err != nil {
		return err, nil
	}
	return nil, parsedResp
}

func (p *PicasaSyncPlugin) BrowseFolder(f string) (error, []SyncResourceInfo) {
	var parsedResp *PicasaMainResponse
	var url string
	p.Lock()
	if !p.initializationDone {
		url = albumFeedURL + "?alt=json&imgmax=d"
		err, parsedResp := p.browseAlbum(url)
		if err != nil {
			return err, nil
		}
		p.resp = parsedResp
	}
	p.Unlock()

	if f == "" {
		return nil, make([]SyncResourceInfo, 0)
	}
	//Now we're browsing a folder
	albumName := p.buildFolderName(f)
	album := p.getFolder(albumName)
	if nil == album {
		return nil, make([]SyncResourceInfo, 0)
	}

	url = albumFeedURL + "/albumid/" + album.Id.Value + "?alt=json&kind=photo&imgmax=d"
	err, parsedResp := p.browseAlbum(url)
	if err != nil {
		return err, nil
	}

	result_size := len(parsedResp.Feed.Entries)
	if "/" == f {
		//Special case we need to return both the images and the albums
		result_size += len(p.resp.Feed.Entries)
	}

	var result = make([]SyncResourceInfo, result_size)

	for i, album := range parsedResp.Feed.Entries {
		name := album.Title.Value
		result[i] = SyncResourceInfo{Name: name, Path: album.Id.Value, Parent: f}
		if album.Category[0].Term == "http://schemas.google.com/photos/2007#photo" {
			result[i].IsDir = false
			result[i].ExtraInfo = album.Media.Content[0].Url
			INFO.Printf("YESS!!! %s => %s\n", name, result[i].ExtraInfo)
		} else {
			result[i].IsDir = true
		}
	}
	if "/" == f {
		for i, album := range p.resp.Feed.Entries {
			name := album.Title.Value
			result[i+len(parsedResp.Feed.Entries)] = SyncResourceInfo{Name: name, Path: album.Id.Value, Parent: f, IsDir: true}
		}
	}
	INFO.Printf("%s", result)
	return nil, result

}

func (p *PicasaSyncPlugin) RemoveResource(r SyncResourceInfo) error {
	return errors.New("Not available")
}

func downloadFromUrl(url, path string) error {
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

func (p *PicasaSyncPlugin) DownloadResource(r *SyncResourceInfo) error {
	INFO.Printf("Now downloading %s", r)
	return downloadFromUrl(r.ExtraInfo, r.Path)
}

type Category struct {
	XMLName xml.Name `xml:"category"`
	Scheme  string   `xml:"scheme,attr"`
	Term    string   `xml:"term,attr"`
}

type XmlAlbum struct {
	XMLName     xml.Name `xml:"entry"`
	Xmlns       string   `xml:"xmlns,attr"`
	XmlnsMedia  string   `xml:"xmlns:media,attr"`
	XmlnsGPhoto string   `xml:"xmlns:gphoto,attr"`
	Title       string   `xml:"title"`
	Category    Category `xml:"category"`
	Access      string   `xml:"access"`
	Summary     string   `xml:"summary"`
	Comment     string   `xml:",comment"`
}

type XmlPhoto struct {
	XMLName  xml.Name `xml:"entry"`
	Xmlns    string   `xml:"xmlns,attr"`
	Title    string   `xml:"title"`
	Category Category `xml:"category"`
	Summary  string   `xml:"summary"`
	Comment  string   `xml:",comment"`
}

func (p *PicasaSyncPlugin) createAlbum(albumName string) error {
	var album XmlAlbum
	album.Xmlns = "http://www.w3.org/2005/Atom"
	album.XmlnsMedia = "http://search.yahoo.com/mrss/"
	album.XmlnsGPhoto = "http://schemas.google.com/photos/2007"
	album.Category.Term = "http://schemas.google.com/photos/2007#album"
	album.Category.Scheme = "http://schemas.google.com/g/2005#kind"
	album.Access = "private"
	album.Title = albumName
	output, _ := xml.MarshalIndent(&album, "  ", "    ")
	//TODO Make the call to the Google API...

	client := oauthCfg.Client(oauth2.NoContext, &p.AuthStruct)
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

func (p *PicasaSyncPlugin) uploadPhoto(in_album *PicasaEntry, r *SyncResourceInfo) error {
	body_buf := bytes.NewBufferString("")
	body_writer := multipart.NewWriter(body_buf)

	boundary := body_writer.Boundary()

	/* Create a completely custom Form Part (or in this case, a file) */
	// http://golang.org/src/pkg/mime/multipart/writer.go?s=2274:2352#L86
	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Type", "application/atom+xml")
	part_writer, err := body_writer.CreatePart(mh)
	//TODO create the xml content for the file
	if nil != err {
		panic(err.Error())
	}

	var photo XmlPhoto
	photo.Xmlns = "http://www.w3.org/2005/Atom"
	photo.Title = r.Name
	photo.Category.Term = "http://schemas.google.com/photos/2007#photo"
	photo.Category.Scheme = "http://schemas.google.com/g/2005#kind"
	output, _ := xml.MarshalIndent(&photo, "  ", "    ")
	io.Copy(part_writer, bytes.NewBuffer(output))

	mh2 := make(textproto.MIMEHeader)
	mh2.Set("Content-Type", r.MimeType)
	//open the file
	file, err := os.Open(r.Path)
	if err != nil {
		return errors.New("Couldn't read file from the filesystem")
	}
	defer file.Close()

	file_writer, err := body_writer.CreatePart(mh2)
	if nil != err {
		panic(err.Error())
	}
	buff, _ := ioutil.ReadAll(file)
	io.Copy(file_writer, bytes.NewBuffer(buff))

	/* Close the body and send the request */
	body_writer.Close()

	client := oauthCfg.Client(oauth2.NoContext, &p.AuthStruct)

	request_body, err := ioutil.ReadAll(body_buf)
	//DO call
	uri := albumFeedURL + "/albumid/" + in_album.Id.Value
	request, err := http.NewRequest("POST", uri, bytes.NewBuffer(request_body))
	content_type := "multipart/related; boundary=\"" + boundary + "\""
	request.Header.Set("Content-Type", content_type)
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
	body, _ := ioutil.ReadAll(resp.Body)
	if nil != err {
		ERROR.Println("Whouch got and error but nothing printed", err, string(body))
		return err
	}
	return nil
}

func (p *PicasaSyncPlugin) AddResource(r *SyncResourceInfo) error {
	DEBUG.Printf("Adding new resource %s\n", r.Name)
	if r.IsDir {
		//We need to create the new repository
		var folder_name string
		if r.Parent == "/" {
			folder_name = "/" + r.Name
		} else {
			folder_name = r.Parent + "/" + r.Name
		}
		album_name := p.buildFolderName(folder_name)
		album := p.getFolder(album_name)
		if album != nil {
			return errors.New("This album already exist, cannot create it")
		}
		p.createAlbum(album_name)
		//Update our information now, so reset the initializationDone flag...
		p.initializationDone = false
		p.BrowseFolder("")
		return nil
	} else {
		album_name := p.buildFolderName(r.Parent)
		album := p.getFolder(album_name)
		if album == nil {
			return errors.New("Can't add entry " + r.Name + "the folder " + album_name + " doesn't exist")
		}
		return p.uploadPhoto(album, r)
	}
	return errors.New("Not available")
}

func (p *PicasaSyncPlugin) buildFolderName(folder string) string {
	if folder == "/" {
		return "NOT_SORTED_REPOSITORY"
	}
	splits := strings.Split(folder, "/")
	folder_name := splits[1]
	if len(splits) > 2 {
		folder_name += " ("
		folder_name += strings.Join(splits[2:], ", ")
		folder_name += ")"
	}
	//fmt.Printf("Hey you \n############################\n%s=>\n%s\n => %s#######################\n", folder, splits, folder_name)
	return folder_name
}

//Check if the album name is in our list
func (p *PicasaSyncPlugin) getFolder(album_name string) *PicasaEntry {
	p.Lock()
	defer p.Unlock()
	for _, album := range p.resp.Feed.Entries {
		//fmt.Printf("\n@@@@@@@@@@@@@@@@@@@@@@\n%s vs %s\n@@@@@@@@@@@@@@@@@@@@@@@@@@\n", album.Title.Value, album_name)
		if album.Title.Value == album_name {
			return &album
		}
	}
	return nil
}

func (p *PicasaSyncPlugin) HasFolder(folder string) bool {
	//Since Picasa doesn't handle "Sub Folder" construction, we'll use some rewriting name
	if !p.initializationDone {
		err, _ := p.BrowseFolder(folder)
		if nil != err {
			ERROR.Printf("Failed to browse for initialisation with error %s\n", err.Error())
			return false
		}
	}
	folder_name := p.buildFolderName(folder)
	return nil != p.getFolder(folder_name)
}

func (p *PicasaSyncPlugin) GetResourceInfo(folder string) (error, SyncResourceInfo) {
	return nil, SyncResourceInfo{}
}
