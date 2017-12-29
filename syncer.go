package main

import (
	//	"io/ioutil"
	"errors"
	"mime"
	"path"
	"path/filepath"
	"strings"

	"github.com/jmcvetta/randutil"
	//	"math"

	"os"
)

// SyncResourceInfo carry information about the resource
type SyncResourceInfo struct {
	Name      string
	Path      string
	Parent    string
	IsDir     bool
	MimeType  string
	ExtraInfo string
}

// GetPath returns the path of the resource
func (s *SyncResourceInfo) GetPath() string {
	return path.Join(s.Parent, s.Name)
}

func (s *SyncResourceInfo) String() string {
	return s.GetPath()
}

// SyncPlugin interface
type SyncPlugin interface {
	// Name should return the name if the plugin, this is for debug purpose only
	Name() string
	// BrowseFolder should only return the list of files, and not subfolders
	BrowseFolder(f string) ([]SyncResourceInfo, error)
	// RemoveResource is not used Yet
	RemoveResource(r SyncResourceInfo) error
	// AddResource create a new Folder or a new File
	AddResource(r *SyncResourceInfo) error
	// HasFolder check if folder already exists
	HasFolder(folder string) bool
	// DownloadResource will download the resource
	DownloadResource(r *SyncResourceInfo) error
	// GetResourceInfo return information about the resource
	GetResourceInfo(folder string) (SyncResourceInfo, error)
	// SyncOnlyMedia Plugin can only sync medias files
	SyncOnlyMedia() bool
}

var (
	syncPluginsList map[string]syncPluginRegistration
)

func registerPlugin(r syncPluginRegistration) {
	if syncPluginsList == nil {
		syncPluginsList = make(map[string]syncPluginRegistration)
	}
	syncPluginsList[r.Name] = r
}

type syncPluginRegistration struct {
	Name      string
	NewMethod func(config string) (SyncPlugin, error)
}

// Syncer object
type Syncer struct {
	// list of entries that have been encountered
	browseEntryList map[string]*browseEntry
}

// NewSyncer will create a Syncer instance
func NewSyncer() *Syncer {
	var syncer = new(Syncer)
	syncer.browseEntryList = make(map[string]*browseEntry)
	return syncer
}

// Sync calls the syncer with it's SyncPlugin
func (s *Syncer) Sync(src, dst SyncPlugin) {
	s.syncLocal(src, dst)
}

// AddNewJob add new job to do
func (s *Syncer) AddNewJob(e *browseEntry) {
	//We received a new job to do add it to the list
	_, ok := s.browseEntryList[e.me.GetPath()]
	if !ok {
		INFO.Printf("Adding new job to list %s\n", e)
		s.browseEntryList[e.me.GetPath()] = e
	} else {
		DEBUG.Printf("Job %s already in list of jobs to do\n", e)
	}
	DEBUG.Printf("===>%s\n", s.browseEntryList)
}

func (s *Syncer) syncLocal(src, dst SyncPlugin) {
	fileList, err := src.BrowseFolder("/")
	if nil != err {
		ERROR.Print("Something went wrong ", err)
		return
	}

	DEBUG.Printf("Syncing from %s to %s\n", src.Name(), dst.Name())
	main, err := src.GetResourceInfo("/")
	if nil != err {
		ERROR.Print("Something went wrong ", err)
		return
	}
	mainEntry := &browseEntry{status: 0, me: &main}
	s.browseEntryList[main.GetPath()] = mainEntry
	for _, file := range fileList {
		if file.IsDir {
			var entry = new(browseEntry)
			entry.status = 0
			entry.me = new(SyncResourceInfo)
			*entry.me = file
			entry.parent = mainEntry
			s.AddNewJob(entry)
		}
	}
	nbWorkers := 4
	nbRunningJobs := 0
	i := 0
	jobChan := make(chan *browseEntry, nbWorkers)
	resultChan := make(chan *browseEntry)
	for i < nbWorkers {
		i = i + 1
		go syncWorker(src, dst, jobChan, resultChan)
		_, entry := s.hasPendingJobs()
		if nil != entry {
			//Set entry status to 1 for PENDING
			entry.status = 1
			INFO.Printf("Pushing new job %p %s\n", entry, entry)
			jobChan <- entry
			nbRunningJobs++
		}
	}
	for {
		//Read the Job done channel
		result := <-resultChan
		if 0 == result.status {
			s.AddNewJob(result)
			continue
		}
		//This means the job was finished
		nbRunningJobs--
		//Now that we have a job done, try look if we can start some new workers

		for {
			pending, nextEntry := s.hasPendingJobs()
			if nil != nextEntry && nbRunningJobs < nbWorkers {

				INFO.Printf("Pushing new job %s\n", nextEntry)
				nextEntry.status = 1
				DEBUG.Printf("%s\n", s.browseEntryList)
				jobChan <- nextEntry
				nbRunningJobs++
				continue
			} else if !pending {
				close(jobChan)
				return
			} else {
				break
			}
		}

	}
}

type browseEntry struct {
	status int
	me     *SyncResourceInfo
	parent *browseEntry
}

func (b *browseEntry) String() string {
	return b.me.String()
}

func (s *Syncer) hasPendingJobs() (pending bool, next *browseEntry) {
	pending = false
	for _, entry := range s.browseEntryList {
		switch entry.status {
		case 0:
			//This entry is not being dealt with
			if nil == entry.parent {
				return true, entry
			} else if 2 == entry.parent.status {
				return true, entry
			} else if 3 == entry.parent.status {
				entry.status = 3
			} else {
				pending = true
			}
		case 1:
			//This entry is being dealt with
			pending = true
		case 2:
		case 3:

		}
	}
	return pending, nil
}

func syncWorker(src, dst SyncPlugin, jobChan <-chan *browseEntry, newJobChan chan<- *browseEntry) {
	for job := range jobChan {
		folder := job.me.GetPath()
		//Check if this folder exists, other wise create it
		//Exclude the / folder which shouldn't be created
		if !dst.HasFolder(folder) {
			err := dst.AddResource(job.me)
			if nil != err {
				ERROR.Print("Failed to add resource ", err)
				job.status = 3
				newJobChan <- job
				return
			}
		}

		fileList, err := src.BrowseFolder(folder)
		if nil != err {
			ERROR.Print("Something went wrong ", err)
			job.status = 3
			newJobChan <- job
			return
		}
		entries, err := dst.BrowseFolder(folder)
		if err != nil {
			ERROR.Println("Failed to browse entries ", err)
			job.status = 3
			newJobChan <- job
			return
		}
		//os.Exit(1)
		newJobChan = doTheWork(fileList, dst, job, newJobChan, entries, src)
		INFO.Printf("Done with the work")
		//Notify the syncer that our job is done, and wait now for new inputs
		job.status = 2
		INFO.Printf("Job %s is done sending done message to main loop\n", folder)
		newJobChan <- job
	}
}
func doTheWork(fileList []SyncResourceInfo, dst SyncPlugin, job *browseEntry, newJobChan chan<- *browseEntry, entries []SyncResourceInfo, src SyncPlugin) chan<- *browseEntry {
	for _, file := range fileList {
		//Lookup for file and folder on the file system

		if file.IsDir {
			if err := createDirJob(file, newJobChan, dst, job); err != nil {
				continue
			}
		} else {
			if err := handleFile(file, entries, src, dst); err != nil {
				continue
			}
		}
	}
	return newJobChan
}

func downloadFile(src, dst SyncPlugin, file SyncResourceInfo) error {
	//Download the file
	rndName, _ := randutil.AlphaString(20)
	tmpFilename := filepath.Join("/tmp", rndName)
	file.Path = tmpFilename
	err := src.DownloadResource(&file)
	if nil != err {
		ERROR.Printf("Failed to download resource %s %v\n", file.GetPath(), err)
		return err
	}
	err = dst.AddResource(&file)
	INFO.Printf("Adding new Resource %s\n", file.Name)
	if err != nil {
		ERROR.Println("Failed to add entry ", file.Name, " with error ", err)
	}
	if tmpFilename == file.Path {
		os.Remove(tmpFilename)
	}
	return err
}

func createDirJob(file SyncResourceInfo, newJobChan chan<- *browseEntry, dst SyncPlugin, job *browseEntry) error {
	//Entry is a directory, create the directory remotely
	folderPath := file.Parent
	if file.Parent != "/" {
		//Special Case of the local file
		folderPath += "/"
	}
	folderPath += file.Name
	if !dst.HasFolder(folderPath) {
		INFO.Printf("Creating new folder %s", folderPath)
		err := dst.AddResource(&file)
		if err != nil {
			ERROR.Println("Failed to create Resource folder ", folderPath, " with error ", err)
			//Skip this folder, since we couldn't create the folder itself
			return err
		}
	} else {
		INFO.Printf("Folder already exists adding it to the list of sub folder to handle")
	}
	bEntry := browseEntry{status: 0, me: &file, parent: job}
	INFO.Printf("Pushing %s\n", bEntry.me)
	//Send message that a new folder has been encountered
	newJobChan <- &bEntry
	return nil
}

func handleFile(file SyncResourceInfo, entries []SyncResourceInfo, src, dst SyncPlugin) error {

	if dst.SyncOnlyMedia() {
		extension := filepath.Ext(file.Name)
		mimeType := mime.TypeByExtension(extension)
		if !(strings.HasPrefix(mimeType, "image") || strings.HasPrefix(mimeType, "video")) {
			INFO.Printf("Found %s that can't be uploaded", file.Name)
			return errors.New("Can't be uploaded")
		}
		file.MimeType = mimeType
	}
	var found = false
	for _, existingEntry := range entries {
		//fmt.Printf("Comparing %s with %s \n", existingEntry.Name, entry.Name)
		if existingEntry.Name == file.Name {
			found = true
		}
	}
	if !found {
		return downloadFile(src, dst, file)
	}
	INFO.Printf("%s already exists\n", file.Name)

	return nil
}
