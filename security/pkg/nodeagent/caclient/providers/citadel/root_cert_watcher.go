package caclient

import (
	"os"

	"istio.io/pkg/filewatcher"
)

// rootCertWatcher observes a single certificate file
// and holds an information if the file had changed in the
// meantime.
//
// The cert watcher watches for file events in the background
// and holds the information whether the file was changed
// till calling Changed() method, which will reset the collected
// information and start watching for newer events.
//
// WARNING: This struct is NOT supposed to be shared by more than
// one owner as the first owner who call Changed() method will
// clear information about the event and so the second owner would
// miss informaion about file change. If needed, Each owner should
// have its own instance of rootCertWatcher.
//
// WARNING: The watcher needs to be Closed at the end.
type rootCertWatcher struct {
	fileWatcher    filewatcher.FileWatcher
	filePath       string
	changed        chan bool
	stop           chan<- bool
	spawnFailed    bool
	alreadyStopped bool
}

func newRootCertWatcher(filepath string) *rootCertWatcher {
	if filepath == "" {
		citadelClientLog.Warn("Cannot start Root Cert Watcher as the Root Cert path is empty")
		return nil
	}

	fileChangedChan := make(chan bool, 1)
	stopChan := make(chan bool)
	spawnedChan := make(chan bool)

	watcher := &rootCertWatcher{
		fileWatcher:    filewatcher.NewWatcher(),
		filePath:       filepath,
		changed:        fileChangedChan,
		stop:           stopChan,
		alreadyStopped: false,
	}

	go startWatchingForFileEvents(filepath, fileChangedChan, stopChan, spawnedChan)
	watcher.spawnFailed = !(<-spawnedChan)
	close(spawnedChan)

	if watcher.spawnFailed {
		citadelClientLog.Error("Failed to start Root Cert Watcher")
		watcher.Close()
		return nil
	}

	return watcher
}

// Changed returns true if there was any event detected regarding
// the certificate file. It may be either deletion of the file,
// modification or creation if the file was deleted previously and
// then recreated. It also registers changes to the indirect files
// such as symlinks.
//
// The Changed() method doesn't return any details regarding the
// events.
//
// After Changed() returns true, the root cert watcher will clear
// it's internal state and continue observing the file from that
// point, so if there are no more changes, the next call for Changed()
// will return false.
func (r *rootCertWatcher) Changed() bool {
	if r == nil {
		citadelClientLog.Info("Checking the state of not initialized Root Certificate Watcher")
		return false
	}
	if r.alreadyStopped {
		citadelClientLog.Errorf("Trying to read from already closed Root Cert Watcher '%s'", r.filePath)
		return false
	}
	select {
	case <-r.changed:
		return true
	default:
		return false
	}
}

// Exists() checks if the file is present in the filesystem.
//
// If the file is a symlink, it will check if the actual file,
// indicated by that symlink, exists.
//
// WARNING: Doesn't handle windows shortcuts.
func (r *rootCertWatcher) Exists() bool {
	if r == nil {
		citadelClientLog.Info("Checking the presence of Root Certificate with unitialized Watcher")
		return false
	}
	if r.alreadyStopped {
		citadelClientLog.Errorf(
			"Trying to check the presence of Root Certificate from already closed Root Cert Watcher '%s'", r.filePath)
		return false
	}
	fileInfo, err := os.Lstat(r.filePath)
	if err != nil {
		citadelClientLog.Warnf(
			"Cannot check details for root cert file '%s': %v", r.filePath, err)
		return false
	}
	if os.IsNotExist(err) {
		return false
	}
	if fileInfo.Mode()&os.ModeSymlink == 0 {
		return true
	}
	// The file is a symlink. Looking for an actual file.
	_, linkErr := os.Stat(r.filePath)

	if os.IsNotExist(linkErr) {
		return false
	}
	if linkErr != nil {
		citadelClientLog.Warnf(
			"Cannot check details for root cert symlink file '%s': %v", r.filePath, linkErr)
		return false
	}
	return true
}

// Close() stops the watching goroutines and synchronization
// channels.
func (r *rootCertWatcher) Close() {
	if r == nil {
		citadelClientLog.Info("Skipping closing Root Cert Watcher as it was not initialized")
		return
	}
	if r.alreadyStopped {
		citadelClientLog.Info("Skipping closing Root Cert Watcher as it was already closed")
		return
	}
	if !r.spawnFailed {
		citadelClientLog.Infof("Closing Watcher for Root Cert at: %s", r.filePath)
		r.stop <- true
	}
	close(r.changed)
	close(r.stop)
	r.alreadyStopped = true
	citadelClientLog.Infof("Root Cert Watcher closed")
}

func startWatchingForFileEvents(filePath string, fileChanged chan<- bool, stop <-chan bool, spawned chan<- bool) {
	watcher, ok := prepareFileWatcher(filePath)
	spawned <- ok
	if !ok {
		return
	}

	for {
		select {
		case gotEvent := <-watcher.Events(filePath):
			citadelClientLog.Debugf("Receive file %s event %v", filePath, gotEvent)
			notifyCertUpdate(fileChanged)
		case err := <-watcher.Errors(filePath):
			citadelClientLog.Warnf("Watch file %s error: %v", filePath, err)
		case <-stop:
			if err := watcher.Close(); err != nil {
				citadelClientLog.Errorf("Failed to close Root Certificate channel: %v", err)
			}
			return
		}
	}
}

func prepareFileWatcher(filePath string) (filewatcher.FileWatcher, bool) {
	watcher := filewatcher.NewWatcher()

	if err := watcher.Add(filePath); err != nil {
		citadelClientLog.Errorf(
			"Cannot start Root Cert Watcher as the Root Cert path '%s', could not be added: %v", filePath, err,
		)
		if closeErr := watcher.Close(); closeErr != nil {
			citadelClientLog.Errorf("Closing watcher failed: %v", closeErr)
		}
		return nil, false
	}
	return watcher, true
}

func notifyCertUpdate(fileChanged chan<- bool) {
	select {
	case fileChanged <- true:
	default:
		citadelClientLog.Debugf("Ignoring file event as there is already one in queue")
	}
}
