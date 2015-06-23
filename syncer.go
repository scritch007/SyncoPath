package main

import (
	//	"io/ioutil"
	"github.com/jmcvetta/randutil"
	"mime"
	"path"
	"path/filepath"
	"strings"
	//	"math"
	"os"
)

type SyncResourceInfo struct {
	Name      string
	Path      string
	Parent    string
	IsDir     bool
	MimeType  string
	ExtraInfo string
}

func (s *SyncResourceInfo) GetPath() string {
	return path.Join(s.Parent, s.Name)
}

func (s *SyncResourceInfo) String() string {
	return s.GetPath()
}

// SyncPlugin interface
// Name() should return the name if the plugin, this is for debug purpose only
// BrowseFolder should only return the list of files, and not subfolders
// RemoveResource is not used Yet
// AddResource create a new Folder or a new File
// HasFolder check if folder already exists
type SyncPlugin interface {
	Name() string
	BrowseFolder(f string) (error, []SyncResourceInfo)
	RemoveResource(r SyncResourceInfo) error
	AddResource(r *SyncResourceInfo) error
	HasFolder(folder string) bool
	DownloadResource(r *SyncResourceInfo) error
	GetResourceInfo(folder string) (error, SyncResourceInfo)
}

type Syncer struct {
	// list of entries that have been encountered
	browseEntryList map[string]*BrowseEntry
}

func NewSyncer() *Syncer {
	var syncer = new(Syncer)
	syncer.browseEntryList = make(map[string]*BrowseEntry)
	return syncer
}

// Call the syncer with it's SyncPlugin
func (s *Syncer) Sync(src, dst SyncPlugin) {
	s.syncLocal(src, dst)
}

func (s *Syncer) AddNewJob(e *BrowseEntry) {
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
	err, fileList := src.BrowseFolder("/")
	if nil != err {
		ERROR.Print("Something went wrong ", err)
		return
	}

	DEBUG.Printf("Syncing from %s to %s\n", src.Name(), dst.Name())
	err, main := src.GetResourceInfo("/")
	if nil != err {
		ERROR.Print("Something went wrong ", err)
		return
	}
	main_entry := &BrowseEntry{status: 0, me: &main}
	s.browseEntryList[main.GetPath()] = main_entry
	for _, file := range fileList {
		if file.IsDir {
			var entry = new(BrowseEntry)
			entry.status = 0
			entry.me = new(SyncResourceInfo)
			(*entry.me) = file
			entry.parent = main_entry
			s.AddNewJob(entry)
		}
	}
	nbWorkers := 4
	nbRunningJobs := 0
	i := 0
	jobChan := make(chan *BrowseEntry, nbWorkers)
	resultChan := make(chan *BrowseEntry)
	for i < nbWorkers {
		i = i + 1
		go syncWorker(src, dst, jobChan, resultChan)
		_, entry := s.hasPendingJobs()
		if nil != entry {
			//Set entry status to 1 for PENDING
			entry.status = 1
			INFO.Printf("Pushing new job %p %s\n", entry, entry)
			jobChan <- entry
			nbRunningJobs = nbRunningJobs + 1
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
		nbRunningJobs = nbRunningJobs - 1
		//Now that we have a job done, try look if we can start some new workers

		for {
			pending, nextEntry := s.hasPendingJobs()
			if nil != nextEntry && nbRunningJobs < nbWorkers {

				INFO.Printf("Pushing new job %s\n", nextEntry)
				nextEntry.status = 1
				DEBUG.Printf("%s\n", s.browseEntryList)
				jobChan <- nextEntry
				nbRunningJobs = nbRunningJobs + 1
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

type BrowseEntry struct {
	status int
	me     *SyncResourceInfo
	parent *BrowseEntry
}

func (b *BrowseEntry) String() string {
	return b.me.String()
}

func (s *Syncer) hasPendingJobs() (pending bool, next *BrowseEntry) {
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

func syncWorker(src, dst SyncPlugin, jobChan <-chan *BrowseEntry, newJobChan chan<- *BrowseEntry) {
	for job := range jobChan {
		folder := job.me.GetPath()
		//Check if this folder exists, other wise create it
		//Exclude the / folder which shouldn't be created
		if !dst.HasFolder(folder) {
			//base := filepath.Base(folder)
			//parent := filepath.Dir(folder)
			//TODO create a temporary file for the download
			//err := src.DownloadResource(job.me)
			//if nil != err {
			//	job.status = 3
			//	newJobChan <- job
			//	return
			//}
			err := dst.AddResource(job.me)
			if nil != err {
				ERROR.Print("Failed to add resource ", err)
				job.status = 3
				newJobChan <- job
				return
			}
		}

		err, fileList := src.BrowseFolder(folder)
		if nil != err {
			ERROR.Print("Something went wrong ", err)
			job.status = 3
			newJobChan <- job
			return
		}
		err, entries := dst.BrowseFolder(folder)
		if err != nil {
			ERROR.Println("Failed to browse entries ", err)
			job.status = 3
			newJobChan <- job
			return
		}
		//os.Exit(1)
		for _, file := range fileList {
			//Lookup for file and folder on the file system
			extension := filepath.Ext(file.Name)

			if file.IsDir {
				//Entry is a directory, create the directory remotely
				folder_path := file.Parent
				if file.Parent != "/" {
					//Special Case of the local file
					folder_path += "/"
				}
				folder_path += file.Name
				if !dst.HasFolder(folder_path) {
					INFO.Printf("Creating new folder %s", folder_path)
					err = dst.AddResource(&file)
					if err != nil {
						ERROR.Println("Failed to create Resource folder ", folder_path, " with error ", err)
						//Skip this folder, since we couldn't create the folder itself
						continue
					}
				} else {
					INFO.Printf("Folder already exists adding it to the list of sub folder to handle")
				}
				browseEntry := BrowseEntry{status: 0, me: &file, parent: job}
				INFO.Printf("Pushing %s\n", browseEntry.me)
				//Send message that a new folder has been encountered
				newJobChan <- &browseEntry
			} else {
				mimeType := mime.TypeByExtension(extension)
				if !(strings.HasPrefix(mimeType, "image") || strings.HasPrefix(mimeType, "video")) {
					INFO.Printf("Found %s that can't be uploaded", file.Name)
					continue
				}
				file.MimeType = mimeType
				var found = false
				for _, existing_entry := range entries {
					//fmt.Printf("Comparing %s with %s \n", existing_entry.Name, entry.Name)
					if existing_entry.Name == file.Name {
						found = true
					}
				}
				if !found {
					//Download the file
					rnd_name, _ := randutil.AlphaString(20)
					tmp_filename := filepath.Join("/tmp", rnd_name)
					file.Path = tmp_filename
					err = src.DownloadResource(&file)
					if nil != err {
						ERROR.Printf("Failed to download resource %s\n", file.GetPath())
						continue
					}
					err = dst.AddResource(&file)
					INFO.Printf("Adding new Resource %s\n", file.Name)
					if err != nil {
						ERROR.Println("Failed to add entry ", file.Name, " with error ", err)
					}
					if tmp_filename == file.Path {
						os.Remove(tmp_filename)
					}
				} else {
					INFO.Printf("%s already exists\n", file.Name)
				}
			}
		}
		INFO.Printf("Done with the work")
		//Notify the syncer that our job is done, and wait now for new inputs
		job.status = 2
		INFO.Printf("Job %s is done sending done message to main loop\n", folder)
		newJobChan <- job
	}
}
