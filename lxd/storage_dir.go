package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lxc/lxd/shared"

	log "gopkg.in/inconshreveable/log15.v2"
)

type storageDir struct {
	d *Daemon

	storageShared
}

func (s *storageDir) Init(config map[string]interface{}) (storage, error) {
	s.sType = storageTypeDir
	s.sTypeName = storageTypeToString(s.sType)
	if err := s.initShared(); err != nil {
		return s, err
	}

	return s, nil
}

func (s *storageDir) ContainerCreate(container container) error {
	cPath := container.Path("")
	if err := os.MkdirAll(cPath, 0755); err != nil {
		return fmt.Errorf("Error creating containers directory")
	}

	if container.IsPrivileged() {
		if err := os.Chmod(cPath, 0700); err != nil {
			return err
		}
	}

	return container.TemplateApply("create")
}

func (s *storageDir) ContainerCreateFromImage(
	container container, imageFingerprint string) error {

	rootfsPath := container.RootfsPath()
	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		return fmt.Errorf("Error creating rootfs directory")
	}

	if container.IsPrivileged() {
		if err := os.Chmod(container.Path(""), 0700); err != nil {
			return err
		}
	}

	imagePath := shared.VarPath("images", imageFingerprint)
	if err := untarImage(imagePath, container.Path("")); err != nil {
		os.RemoveAll(rootfsPath)
		return err
	}

	if !container.IsPrivileged() {
		if err := s.shiftRootfs(container); err != nil {
			s.ContainerDelete(container)
			return err
		}
	}

	return container.TemplateApply("create")
}

func (s *storageDir) ContainerDelete(container container) error {
	cPath := container.Path("")

	err := os.RemoveAll(cPath)
	if err != nil {
		s.log.Error("ContainerDelete: failed", log.Ctx{"cPath": cPath, "err": err})
		return fmt.Errorf("Error cleaning up %s: %s", cPath, err)
	}

	return nil
}

func (s *storageDir) ContainerCopy(
	container container, sourceContainer container) error {

	oldPath := sourceContainer.RootfsPath()
	newPath := container.RootfsPath()

	/*
	 * Copy by using rsync
	 */
	output, err := storageRsyncCopy(oldPath, newPath)
	if err != nil {
		s.ContainerDelete(container)
		s.log.Error("ContainerCopy: rsync failed", log.Ctx{"output": string(output)})
		return fmt.Errorf("rsync failed: %s", string(output))
	}

	err = s.setUnprivUserAcl(sourceContainer, container.Path(""))
	if err != nil {
		return err
	}

	return container.TemplateApply("copy")
}

func (s *storageDir) ContainerStart(container container) error {
	return nil
}

func (s *storageDir) ContainerStop(container container) error {
	return nil
}

func (s *storageDir) ContainerRename(
	container container, newName string) error {

	oldPath := container.Path("")
	newPath := container.Path(newName)

	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}

	// TODO: No TemplateApply here?
	return nil
}

func (s *storageDir) ContainerRestore(
	container container, sourceContainer container) error {

	targetPath := container.Path("")
	sourcePath := sourceContainer.Path("")

	// Restore using rsync
	output, err := storageRsyncCopy(
		sourcePath,
		targetPath)

	if err != nil {
		s.log.Error(
			"ContainerRestore: rsync failed",
			log.Ctx{"output": string(output)})

		return err
	}

	// Now allow unprivileged users to access its data.
	if err := s.setUnprivUserAcl(sourceContainer, targetPath); err != nil {
		return err
	}

	return nil
}

func (s *storageDir) ContainerSnapshotCreate(
	snapshotContainer container, sourceContainer container) error {

	oldPath := sourceContainer.Path("")
	newPath := snapshotContainer.Path("")

	/*
	 * Copy by using rsync
	 */
	output, err := storageRsyncCopy(oldPath, newPath)
	if err != nil {
		s.ContainerDelete(snapshotContainer)
		s.log.Error("ContainerSnapshotCreate: rsync failed",
			log.Ctx{"output": string(output)})

		return fmt.Errorf("rsync failed: %s", string(output))
	}

	return nil
}
func (s *storageDir) ContainerSnapshotDelete(
	snapshotContainer container) error {
	err := s.ContainerDelete(snapshotContainer)
	if err != nil {
		return fmt.Errorf("Error deleting snapshot %s: %s", snapshotContainer.Name(), err)
	}

	oldPathParent := filepath.Dir(snapshotContainer.Path(""))
	if ok, _ := shared.PathIsEmpty(oldPathParent); ok {
		os.Remove(oldPathParent)
	}

	return nil
}

func (s *storageDir) ContainerSnapshotRename(
	snapshotContainer container, newName string) error {

	oldPath := snapshotContainer.Path("")
	newPath := snapshotContainer.Path(newName)

	// Create the new parent.
	if strings.Contains(snapshotContainer.Name(), "/") {
		if !shared.PathExists(filepath.Dir(newPath)) {
			os.MkdirAll(filepath.Dir(newPath), 0700)
		}
	}

	// Now rename the snapshot.
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}

	// Remove the old parent (on container rename) if its empty.
	if strings.Contains(snapshotContainer.Name(), "/") {
		if ok, _ := shared.PathIsEmpty(filepath.Dir(oldPath)); ok {
			os.Remove(filepath.Dir(oldPath))
		}
	}

	return nil
}

func (s *storageDir) ImageCreate(fingerprint string) error {
	return nil
}

func (s *storageDir) ImageDelete(fingerprint string) error {
	return nil
}
