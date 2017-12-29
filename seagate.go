package main

import (
	"encoding/json"
	"fmt"
	"path"

	"os"

	"path/filepath"

	"github.com/pkg/errors"
	"github.com/scritch007/go-ra-seagate"
)

// SyncPlugin interface
type SeagatePlugin struct {
	Login      string
	Password   string
	Path       string
	DeviceID   string
	connection *raseagate.Connection
}

func init() {
	registerPlugin(syncPluginRegistration{
		Name:      "seagate",
		NewMethod: newSeagateSyncPlugin,
	})
}

func newSeagateSyncPlugin(config string) (SyncPlugin, error) {
	var c *raseagate.Client
	var err error
	s := SeagatePlugin{}
	if len(config) == 0 {

		fmt.Println("Enter login")
		_, _ = fmt.Scanln(&s.Login)
		fmt.Println("Enter password")
		_, _ = fmt.Scanln(&s.Password)
		c, err = raseagate.NewClient(s.Login, s.Password)
		if err != nil {
			return nil, err
		}
		dList, err := c.GetDeviceList()
		if err != nil {
			return nil, err
		}
		count := 0
		for _, d := range dList {
			if d.IsAvailable() {
				fmt.Println(d.FriendlyName)
				count++
			}
		}
		if count == 0 {
			return nil, fmt.Errorf("No device available")
		}
		fmt.Println("Select device")
		var fName string
		_, _ = fmt.Scanln(&fName)
		for _, d := range dList {
			if d.FriendlyName == fName {
				s.DeviceID = d.DeviceID
				break
			}
		}
		fmt.Println("Enter path on device")
		_, _ = fmt.Scanln(&s.Path)

	} else {
		err := json.Unmarshal([]byte(config), &s)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read config")
		}
		fmt.Printf("=>%s %s\n", s.Login, s.Password)
		c, err = raseagate.NewClient(s.Login, s.Password)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to instantiate new client")
		}
	}
	if len(s.DeviceID) == 0 {
		return nil, fmt.Errorf("Unknown deviceID")
	}

	fmt.Printf("=>%v", s)
	fmt.Println("Connecting to server")
	dList, err := c.GetDeviceList()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list devices")
	}
	for _, d := range dList {
		if d.DeviceID == s.DeviceID {
			if !d.IsAvailable() {
				return nil, fmt.Errorf("Device not available")
			}
			s.connection, err = c.Connect(&d)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to connect to device")
			}
			break
		}
	}
	return &s, nil
}

// Name should return the name if the plugin, this is for debug purpose only
func (s *SeagatePlugin) Name() string {
	return "seagate"
}

// BrowseFolder should only return the list of files, and not subfolders
func (s *SeagatePlugin) BrowseFolder(folder string) ([]SyncResourceInfo, error) {
	files, err := s.connection.Browse(path.Join(s.Path, folder))
	if err != nil {
		return nil, err
	}

	var res []SyncResourceInfo
	for _, f := range files {
		res = append(res, SyncResourceInfo{
			Path:   path.Join(folder, f.Name),
			Name:   f.Name,
			IsDir:  f.IsDir,
			Parent: folder,
		})
	}
	return res, nil
}

// RemoveResource is not used Yet
func (s *SeagatePlugin) RemoveResource(r SyncResourceInfo) error {
	return fmt.Errorf("Not implemented")
}

// AddResource create a new Folder or a new File
func (s *SeagatePlugin) AddResource(r *SyncResourceInfo) error {
	if r.IsDir {
		return s.connection.CreateFolder(path.Join(s.Path, r.Parent, r.Name))
	}
	f, err := os.Open(r.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	err = s.connection.UploadFile(path.Join(s.Path, r.Parent, r.Name), f)
	return err
}

// HasFolder check if folder already exists
func (s *SeagatePlugin) HasFolder(folder string) bool {
	has, _ := s.connection.HasFile(path.Join(s.Path, folder))
	return has
}

// DownloadResource will download the resource
func (s *SeagatePlugin) DownloadResource(r *SyncResourceInfo) error {
	f, err := os.Create(r.Path)
	if err != nil {
		return errors.Wrap(err, "Failed to create file")
	}
	return s.connection.DownloadFile(path.Join(s.Path, r.Parent, r.Name), f)
}

// GetResourceInfo return information about the resource
func (s *SeagatePlugin) GetResourceInfo(folder string) (SyncResourceInfo, error) {
	var name, parent string
	if folder == "/" {
		name = "/"
		parent = "/"
	} else {
		name = filepath.Base(folder)
		parent = filepath.Dir(folder)
	}
	return SyncResourceInfo{Name: name, Parent: parent, Path: folder, IsDir: true}, nil
}

func (s *SeagatePlugin) SyncOnlyMedia() bool {
	return false
}
