/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gce_pd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/cloudprovider"
	"k8s.io/kubernetes/pkg/cloudprovider/providers/gce"
	"k8s.io/kubernetes/pkg/util"
	"k8s.io/kubernetes/pkg/util/exec"
	"k8s.io/kubernetes/pkg/util/operationmanager"
	"k8s.io/kubernetes/pkg/util/sets"
)

const (
	diskByIdPath         = "/dev/disk/by-id/"
	diskGooglePrefix     = "google-"
	diskScsiGooglePrefix = "scsi-0Google_PersistentDisk_"
	diskPartitionSuffix  = "-part"
	diskSDPath           = "/dev/sd"
	diskSDPattern        = "/dev/sd*"
	maxChecks            = 60
	maxRetries           = 10
	checkSleepDuration   = time.Second
	errorSleepDuration   = 5 * time.Second
)

// Singleton operation manager for managing detach clean up go routines
var detachCleanupManager = operationmanager.NewOperationManager()

type GCEDiskUtil struct{}

// Attaches a disk specified by a volume.GCEPersistentDisk to the current kubelet.
// Mounts the disk to it's global path.
func (diskUtil *GCEDiskUtil) AttachAndMountDisk(b *gcePersistentDiskBuilder, globalPDPath string) error {
	glog.V(5).Infof("AttachAndMountDisk(b, %q) where b is %#v\r\n", globalPDPath, b)

	// Block execution until any pending detach goroutines for this pd have completed
	detachCleanupManager.Send(b.pdName, true)

	sdBefore, err := filepath.Glob(diskSDPattern)
	if err != nil {
		glog.Errorf("Error filepath.Glob(\"%s\"): %v\r\n", diskSDPattern, err)
	}
	sdBeforeSet := sets.NewString(sdBefore...)

	devicePath, err := attachDiskAndVerify(b, sdBeforeSet)
	if err != nil {
		return err
	}

	// Only mount the PD globally once.
	notMnt, err := b.mounter.IsLikelyNotMountPoint(globalPDPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(globalPDPath, 0750); err != nil {
				return err
			}
			notMnt = true
		} else {
			return err
		}
	}
	options := []string{}
	if b.readOnly {
		options = append(options, "ro")
	}
	if notMnt {
		err = b.diskMounter.Mount(devicePath, globalPDPath, b.fsType, options)
		if err != nil {
			os.Remove(globalPDPath)
			return err
		}
	}
	return nil
}

// Unmounts the device and detaches the disk from the kubelet's host machine.
func (util *GCEDiskUtil) DetachDisk(c *gcePersistentDiskCleaner) error {
	// Unmount the global PD mount, which should be the only one.
	globalPDPath := makeGlobalPDName(c.plugin.host, c.pdName)
	glog.V(5).Infof("DetachDisk(c) where c is %#v and the globalPDPath is %q\r\n", c, globalPDPath)

	if err := c.mounter.Unmount(globalPDPath); err != nil {
		return err
	}
	if err := os.Remove(globalPDPath); err != nil {
		return err
	}

	if detachCleanupManager.Exists(c.pdName) {
		glog.Warningf("Terminating new DetachDisk call for GCE PD %q. A previous detach call for this PD is still pending.", c.pdName)
		return nil

	}

	// Detach disk, retry if needed.
	go detachDiskAndVerify(c)
	return nil
}

// Attaches the specified persistent disk device to node, verifies that it is attached, and retries if it fails.
func attachDiskAndVerify(b *gcePersistentDiskBuilder, sdBeforeSet sets.String) (string, error) {
	devicePaths := getDiskByIdPaths(b.gcePersistentDisk)
	var gceCloud *gce_cloud.GCECloud
	for numRetries := 0; numRetries < maxRetries; numRetries++ {
		var err error
		if gceCloud == nil {
			gceCloud, err = getCloudProvider()
			if err != nil || gceCloud == nil {
				// Retry on error. See issue #11321
				glog.Errorf("Error getting GCECloudProvider while detaching PD %q: %v", b.pdName, err)
				time.Sleep(errorSleepDuration)
				continue
			}
		}

		if numRetries > 0 {
			glog.Warningf("Timed out waiting for GCE PD %q to attach. Retrying attach.", b.pdName)
		}

		if err := gceCloud.AttachDisk(b.pdName, b.readOnly); err != nil {
			// Retry on error. See issue #11321. Continue and verify if disk is attached, because a
			// previous attach operation may still succeed.
			glog.Errorf("Error attaching PD %q: %v", b.pdName, err)
		}

		for numChecks := 0; numChecks < maxChecks; numChecks++ {
			path, err := verifyDevicePath(devicePaths, sdBeforeSet)
			if err != nil {
				// Log error, if any, and continue checking periodically. See issue #11321
				glog.Errorf("Error verifying GCE PD (%q) is attached: %v", b.pdName, err)
			} else if path != "" {
				// A device path has successfully been created for the PD
				glog.Infof("Successfully attached GCE PD %q.", b.pdName)
				return path, nil
			}

			// Sleep then check again
			glog.V(3).Infof("Waiting for GCE PD %q to attach.", b.pdName)
			time.Sleep(checkSleepDuration)
		}
	}

	return "", fmt.Errorf("Could not attach GCE PD %q. Timeout waiting for mount paths to be created.", b.pdName)
}

// Returns the first path that exists, or empty string if none exist.
func verifyDevicePath(devicePaths []string, sdBeforeSet sets.String) (string, error) {
	if err := udevadmChangeToNewDrives(sdBeforeSet); err != nil {
		// udevadm errors should not block disk detachment, log and continue
		glog.Errorf("udevadmChangeToNewDrives failed with: %v", err)
	}

	for _, path := range devicePaths {
		if pathExists, err := pathExists(path); err != nil {
			return "", fmt.Errorf("Error checking if path exists: %v", err)
		} else if pathExists {
			return path, nil
		}
	}

	return "", nil
}

// Detaches the specified persistent disk device from node, verifies that it is detached, and retries if it fails.
// This function is intended to be called asynchronously as a go routine.
// It starts the detachCleanupManager with the specified pdName so that callers can wait for completion.
func detachDiskAndVerify(c *gcePersistentDiskCleaner) {
	glog.V(5).Infof("detachDiskAndVerify for pd %q.", c.pdName)
	defer util.HandleCrash()

	// Start operation, so that other threads can wait on this detach operation.
	// Set bufferSize to 0 so senders are blocked on send until we receive.
	ch, err := detachCleanupManager.Start(c.pdName, 0 /* bufferSize */)
	if err != nil {
		glog.Errorf("Error adding %q to detachCleanupManager: %v", c.pdName, err)
		return
	}

	defer detachCleanupManager.Close(c.pdName)

	defer func() {
		// Unblock any callers that have been waiting for this detach routine to complete.
		for {
			select {
			case <-ch:
				glog.V(5).Infof("detachDiskAndVerify for pd %q clearing chan.", c.pdName)
			default:
				glog.V(5).Infof("detachDiskAndVerify for pd %q done clearing chans.", c.pdName)
				return
			}
		}
	}()

	devicePaths := getDiskByIdPaths(c.gcePersistentDisk)
	var gceCloud *gce_cloud.GCECloud
	for numRetries := 0; numRetries < maxRetries; numRetries++ {
		var err error
		if gceCloud == nil {
			gceCloud, err = getCloudProvider()
			if err != nil || gceCloud == nil {
				// Retry on error. See issue #11321
				glog.Errorf("Error getting GCECloudProvider while detaching PD %q: %v", c.pdName, err)
				time.Sleep(errorSleepDuration)
				continue
			}
		}

		if numRetries > 0 {
			glog.Warningf("Timed out waiting for GCE PD %q to detach. Retrying detach.", c.pdName)
		}

		if err := gceCloud.DetachDisk(c.pdName); err != nil {
			// Retry on error. See issue #11321. Continue and verify if disk is detached, because a
			// previous detach operation may still succeed.
			glog.Errorf("Error detaching PD %q: %v", c.pdName, err)
		}

		for numChecks := 0; numChecks < maxChecks; numChecks++ {
			allPathsRemoved, err := verifyAllPathsRemoved(devicePaths)
			if err != nil {
				// Log error, if any, and continue checking periodically.
				glog.Errorf("Error verifying GCE PD (%q) is detached: %v", c.pdName, err)
			} else if allPathsRemoved {
				// All paths to the PD have been succefully removed
				glog.Infof("Successfully detached GCE PD %q.", c.pdName)
				return
			}

			// Sleep then check again
			glog.V(3).Infof("Waiting for GCE PD %q to detach.", c.pdName)
			time.Sleep(checkSleepDuration)
		}

	}

	glog.Errorf("Failed to detach GCE PD %q. One or more mount paths was not removed.", c.pdName)
}

// Returns the first path that exists, or empty string if none exist.
func verifyAllPathsRemoved(devicePaths []string) (bool, error) {
	allPathsRemoved := true
	for _, path := range devicePaths {
		if err := udevadmChangeToDrive(path); err != nil {
			// udevadm errors should not block disk detachment, log and continue
			glog.Errorf("%v", err)
		}
		if exists, err := pathExists(path); err != nil {
			return false, fmt.Errorf("Error checking if path exists: %v", err)
		} else {
			allPathsRemoved = allPathsRemoved && !exists
		}
	}

	return allPathsRemoved, nil
}

// Returns list of all /dev/disk/by-id/* paths for given PD.
func getDiskByIdPaths(pd *gcePersistentDisk) []string {
	devicePaths := []string{
		path.Join(diskByIdPath, diskGooglePrefix+pd.pdName),
		path.Join(diskByIdPath, diskScsiGooglePrefix+pd.pdName),
	}

	if pd.partition != "" {
		for i, path := range devicePaths {
			devicePaths[i] = path + diskPartitionSuffix + pd.partition
		}
	}

	return devicePaths
}

// Checks if the specified path exists
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

// Return cloud provider
func getCloudProvider() (*gce_cloud.GCECloud, error) {
	gceCloudProvider, err := cloudprovider.GetCloudProvider("gce", nil)
	if err != nil || gceCloudProvider == nil {
		return nil, err
	}

	// The conversion must be safe otherwise bug in GetCloudProvider()
	return gceCloudProvider.(*gce_cloud.GCECloud), nil
}

// Calls "udevadm trigger --action=change" for newly created "/dev/sd*" drives (exist only in after set).
// This is workaround for Issue #7972. Once the underlying issue has been resolved, this may be removed.
func udevadmChangeToNewDrives(sdBeforeSet sets.String) error {
	sdAfter, err := filepath.Glob(diskSDPattern)
	if err != nil {
		return fmt.Errorf("Error filepath.Glob(\"%s\"): %v\r\n", diskSDPattern, err)
	}

	for _, sd := range sdAfter {
		if !sdBeforeSet.Has(sd) {
			return udevadmChangeToDrive(sd)
		}
	}

	return nil
}

// Calls "udevadm trigger --action=change" on the specified drive.
// drivePath must be the the block device path to trigger on, in the format "/dev/sd*", or a symlink to it.
// This is workaround for Issue #7972. Once the underlying issue has been resolved, this may be removed.
func udevadmChangeToDrive(drivePath string) error {
	glog.V(5).Infof("udevadmChangeToDrive: drive=%q", drivePath)

	// Evaluate symlink, if any
	drive, err := filepath.EvalSymlinks(drivePath)
	if err != nil {
		return fmt.Errorf("udevadmChangeToDrive: filepath.EvalSymlinks(%q) failed with %v.", drivePath, err)
	}
	glog.V(5).Infof("udevadmChangeToDrive: symlink path is %q", drive)

	// Check to make sure input is "/dev/sd*"
	if !strings.Contains(drive, diskSDPath) {
		return fmt.Errorf("udevadmChangeToDrive: expected input in the form \"%s\" but drive is %q.", diskSDPattern, drive)
	}

	// Call "udevadm trigger --action=change --property-match=DEVNAME=/dev/sd..."
	_, err = exec.New().Command(
		"udevadm",
		"trigger",
		"--action=change",
		fmt.Sprintf("--property-match=DEVNAME=%s", drive)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("udevadmChangeToDrive: udevadm trigger failed for drive %q with %v.", drive, err)
	}
	return nil
}
