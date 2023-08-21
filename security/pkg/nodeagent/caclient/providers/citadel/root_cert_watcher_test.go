// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package caclient

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"istio.io/istio/pkg/test/util/assert"
)

func TestWatcherIsNotCreatedWhenEmptyPath(t *testing.T) {
	watcher := newRootCertWatcher("")
	assert.Equal(t, watcher, nil)
}

func TestExistsMethodOnNotSpawnedWatcher(t *testing.T) {
	watcher := newRootCertWatcher("")
	assert.Equal(t, watcher, nil)
	assert.Equal(t, watcher.Exists(), false)
}

func TestChangedMethodOnNotSpawnedWatcher(t *testing.T) {
	watcher := newRootCertWatcher("")
	assert.Equal(t, watcher, nil)
	assert.Equal(t, watcher.Changed(), false)
}

func TestCloseMethodOnNotSpawnedWatcher(t *testing.T) {
	watcher := newRootCertWatcher("")
	assert.Equal(t, watcher, nil)
	watcher.Close()
}

func TestNoChangesAfterTheWatcherWasJustCreated(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	assert.Equal(t, watcher.Changed(), false, "The file didn't change in the meantime")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestExistsMethodForNewlyCreatedWatcherAndExistingCert(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	assert.Equal(t, watcher.Exists(), true)

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestCreatingWatcherForNonExistingCert(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)
	deleteCert(t, fakeCertPath)

	watcher := createWatcher(t, fakeCertPath)

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestCheckingPresenceForCertificateNotPresentYet(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)
	deleteCert(t, fakeCertPath)

	watcher := createWatcher(t, fakeCertPath)

	assert.Equal(t, watcher.Exists(), false)

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestDelayedCertCreation(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)
	deleteCert(t, fakeCertPath)

	watcher := createWatcher(t, fakeCertPath)

	createCertInDirectory(t, testCaseDir)
	assert.Equal(t, watcher.Changed(), true)

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestCheckingPresenceForCertificateCreatedShortlyAfterWatcher(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)
	deleteCert(t, fakeCertPath)

	watcher := createWatcher(t, fakeCertPath)

	createCertInDirectory(t, testCaseDir)
	assert.Equal(t, watcher.Exists(), true)

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestDetectingCertUpdate(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	updateCert(t, fakeCertPath)

	assert.Equal(t, watcher.Changed(), true, "The file change was not detected")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestCheckingPresenceOfUpdatedCert(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	updateCert(t, fakeCertPath)

	assert.Equal(t, watcher.Exists(), true, "The file is not detected")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestFileChangeReportedOnlyOnce(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	assert.Equal(t, watcher.Changed(), false, "The file didn't change in the meantime")

	updateCert(t, fakeCertPath)

	assert.Equal(t, watcher.Changed(), true, "The file change was not detected")
	assert.Equal(t, watcher.Changed(), false, "Unexpected file change detected. Did the watcher clear previous state?")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestDetectingCertDelete(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	deleteCert(t, fakeCertPath)

	assert.Equal(t, watcher.Changed(), true, "The file change was not detected")
	assert.Equal(t, watcher.Changed(), false, "Unexpected file change detected. Did the watcher clear previous state?")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestCheckingPresenceForDeletedCert(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	deleteCert(t, fakeCertPath)

	assert.Equal(t, watcher.Exists(), false, "The file was deleted but still detected")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestDetectingCertRestoration(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	assert.Equal(t, watcher.Changed(), false, "The file didn't change in the meantime")

	deleteCert(t, fakeCertPath)

	assert.Equal(t, watcher.Changed(), true, "The file change was not detected")
	assert.Equal(t, watcher.Changed(), false, "Unexpected file change detected. Did the watcher clear previous state?")

	createCertInDirectory(t, testCaseDir)
	assert.Equal(t, watcher.Changed(), true, "The file was restored but the watcher didn't observe that")
	assert.Equal(t, watcher.Changed(), false, "Unexpected file change detected. Did the watcher clear previous state?")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestMultipleChangesBeforeCallingChangedMethod(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	updateCertWithContent(t, fakeCertPath, "A")
	updateCertWithContent(t, fakeCertPath, "B")
	updateCertWithContent(t, fakeCertPath, "C")

	assert.Equal(t, watcher.Changed(), true, "The file change was not detected")
	assert.Equal(t, watcher.Changed(), false, "Unexpected file change detected. Did the watcher clear previous state? Or did it collected all events separately?")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestMultipleChangesSeparatedByChangedMethod(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	updateCertWithContent(t, fakeCertPath, "A")
	assert.Equal(t, watcher.Changed(), true, "The file change was not detected")

	updateCertWithContent(t, fakeCertPath, "B")
	assert.Equal(t, watcher.Changed(), true, "The file change was not detected")

	updateCertWithContent(t, fakeCertPath, "C")
	assert.Equal(t, watcher.Changed(), true, "The file change was not detected")

	assert.Equal(t, watcher.Changed(), false, "Unexpected file change detected.")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestCheckingPresenceForRestoredCert(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	deleteCert(t, fakeCertPath)
	createCertInDirectory(t, testCaseDir)

	assert.Equal(t, watcher.Exists(), true, "The file is not detected even though it was recreated")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestClosingWatcherMultipleTimes(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	watcher.Close()
	watcher.Close()

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestChangedMethodForClosedWatcher(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	watcher.Close()

	updateCert(t, fakeCertPath)

	assert.Equal(t, watcher.Changed(), false)

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestExistsMethodForClosedWatcher(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	watcher := createWatcher(t, fakeCertPath)

	watcher.Close()

	assert.Equal(t, watcher.Exists(), false)

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestAllWatchersObtainTheFileEvent(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)

	w1 := createWatcher(t, fakeCertPath)
	w2 := createWatcher(t, fakeCertPath)

	updateCert(t, fakeCertPath)

	assert.Equal(t, w1.Changed(), true, "The file change was not detected")
	assert.Equal(t, w2.Changed(), true, "The file change was not detected")

	assert.Equal(t, w1.Changed(), false, "Unexpected change detected")
	assert.Equal(t, w2.Changed(), false, "Unexpected change detected")

	clean(t, []*rootCertWatcher{w1, w2}, testCaseDir)
}

func TestExistenceForCertThroughASymlink(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)
	symlinkPath := createSymlinkToCert(t, fakeCertPath)

	watcher := createWatcher(t, symlinkPath)

	assert.Equal(t, watcher.Exists(), true, "The cert exists")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestExistenceForNonPresentCertThroughASymlink(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)
	symlinkPath := createSymlinkToCert(t, fakeCertPath)
	deleteCert(t, fakeCertPath)

	watcher := createWatcher(t, symlinkPath)

	assert.Equal(t, watcher.Exists(), false, "The actual cert does not exist")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestDetectingCertChangeThroughASymlink(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)
	symlinkPath := createSymlinkToCert(t, fakeCertPath)

	watcher := createWatcher(t, symlinkPath)

	assert.Equal(t, watcher.Changed(), false, "Detected unexpected change")

	updateCert(t, fakeCertPath)

	assert.Equal(t, watcher.Changed(), true, "The change was not detected. Problem with symlink?")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func TestDetectingCertDeletionThroughASymlink(t *testing.T) {
	testCaseDir, fakeCertPath := createCert(t)
	symlinkPath := createSymlinkToCert(t, fakeCertPath)

	watcher := createWatcher(t, symlinkPath)

	assert.Equal(t, watcher.Changed(), false, "Detected unexpected change")

	deleteCert(t, fakeCertPath)

	assert.Equal(t, watcher.Changed(), true, "The change was not detected. Problem with symlink?")

	clean(t, []*rootCertWatcher{watcher}, testCaseDir)
}

func createWatcher(t *testing.T, certPath string) *rootCertWatcher {
	watcher := newRootCertWatcher(certPath)
	if watcher == nil {
		t.Fatalf("Watcher was not created")
	}
	return watcher
}

func createCert(t *testing.T) (string, string) {
	testCaseDir, err := os.MkdirTemp("", "istio_test_citadel_client")
	assert.NoError(t, err)
	fakeCertPath := createCertInDirectory(t, testCaseDir)
	return testCaseDir, fakeCertPath
}

func createCertInDirectory(t *testing.T, directoryPath string) string {
	fakeCertPath := directoryPath + "/root-cert.pem"
	fakeCertData := []byte("Hello, it's me, Mittens. Trust me!")

	err := os.WriteFile(fakeCertPath, fakeCertData, 0644)
	assert.NoError(t, err)
	// The filewatcher needs a moment to register a change.
	time.Sleep(time.Millisecond * 100)
	return fakeCertPath
}

func createSymlinkToCert(t *testing.T, certPath string) string {
	dir := filepath.Dir(certPath)

	symlinkPath := dir + "/symlink-cert.pem"
	err := os.Symlink(certPath, symlinkPath)
	assert.NoError(t, err)

	return symlinkPath
}

func updateCert(t *testing.T, filepath string) {
	updateCertWithContent(t, filepath, "Okay, I lied")
}

func updateCertWithContent(t *testing.T, filepath, content string) {
	updated_data := []byte(content)
	err := os.WriteFile(filepath, updated_data, 0644)
	assert.NoError(t, err)
	output, err := os.ReadFile(filepath)
	assert.NoError(t, err)
	assert.Equal(t, output, updated_data, "File was not updated")
	// The filewatcher needs a moment to register a change.
	time.Sleep(time.Millisecond * 100)
}

func deleteCert(t *testing.T, filepath string) {
	err := os.Remove(filepath)
	assert.NoError(t, err)
	// The filewatcher needs a moment to register a change.
	time.Sleep(time.Millisecond * 100)
}

func clean(t *testing.T, watchers []*rootCertWatcher, directory string) {
	for _, w := range watchers {
		w.Close()
	}
	err := os.RemoveAll(directory)
	assert.NoError(t, err)
}
