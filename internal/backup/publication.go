package backup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type publicationOps struct {
	link    func(string, string) error
	lstat   func(string) (os.FileInfo, error)
	remove  func(string) error
	syncDir func(string) error
}

var errPublicationRollback = errors.New("backup final-link cleanup not confirmed")

func (s *Store) publicationOperations() publicationOps {
	ops := publicationOps{
		link:    os.Link,
		lstat:   os.Lstat,
		remove:  os.Remove,
		syncDir: syncDir,
	}
	if s.publication != nil {
		if s.publication.link != nil {
			ops.link = s.publication.link
		}
		if s.publication.lstat != nil {
			ops.lstat = s.publication.lstat
		}
		if s.publication.remove != nil {
			ops.remove = s.publication.remove
		}
		if s.publication.syncDir != nil {
			ops.syncDir = s.publication.syncDir
		}
	}
	return ops
}

func (s *Store) publishArtifact(staged, final string) (err error) {
	ops := s.publicationOperations()
	expected, err := ops.lstat(staged)
	if err != nil || !expected.Mode().IsRegular() || expected.Mode().Perm() != 0600 {
		return fmt.Errorf("backup publish failed")
	}
	if err := ops.link(staged, final); err != nil {
		return fmt.Errorf("backup publish failed")
	}
	defer func() {
		if err != nil {
			if rbErr := rollbackOwnedPublication(staged, final, expected, ops); rbErr != nil {
				err = errors.Join(err, rbErr)
			}
		}
	}()
	info, err := ops.lstat(final)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return fmt.Errorf("backup verification failed")
	}
	if !os.SameFile(expected, info) {
		return fmt.Errorf("backup verification failed")
	}
	after, err := ops.lstat(staged)
	if err != nil || !os.SameFile(expected, after) {
		return fmt.Errorf("backup verification failed")
	}
	if err := ops.syncDir(filepath.Dir(final)); err != nil {
		return fmt.Errorf("backup durability check failed")
	}
	return nil
}

func rollbackOwnedPublication(staged, final string, expected os.FileInfo, ops publicationOps) error {
	currentFinal, err := ops.lstat(final)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errPublicationRollback
	}
	currentStaged, err := ops.lstat(staged)
	if err != nil || !os.SameFile(expected, currentStaged) {
		return errPublicationRollback
	}
	if !os.SameFile(expected, currentFinal) {
		return errPublicationRollback
	}
	if err := ops.remove(final); err != nil {
		return errors.Join(errPublicationRollback, fmt.Errorf("backup publish cleanup failed: %w", err))
	}
	if err := ops.syncDir(filepath.Dir(final)); err != nil {
		return errPublicationRollback
	}
	return nil
}
