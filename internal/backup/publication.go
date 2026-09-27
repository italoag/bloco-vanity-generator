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
	syncDir func(string) error
}

var errPublicationUnconfirmed = errors.New("backup publication not confirmed")

func (s *Store) publicationOperations() publicationOps {
	ops := publicationOps{link: os.Link, lstat: os.Lstat, syncDir: syncDir}
	if s.publication != nil {
		if s.publication.link != nil {
			ops.link = s.publication.link
		}
		if s.publication.lstat != nil {
			ops.lstat = s.publication.lstat
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
		return fmt.Errorf("backup publish failed: %w", err)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, errPublicationUnconfirmed)
		}
	}()
	verify := func() error {
		for _, path := range []string{staged, final} {
			current, err := ops.lstat(path)
			if err != nil || !current.Mode().IsRegular() || current.Mode().Perm() != 0600 || !os.SameFile(expected, current) {
				return fmt.Errorf("backup verification failed")
			}
		}
		return nil
	}
	if err := verify(); err != nil {
		return err
	}
	if err := ops.syncDir(filepath.Dir(final)); err != nil {
		return fmt.Errorf("backup durability check failed: %w", err)
	}
	return verify()
}
