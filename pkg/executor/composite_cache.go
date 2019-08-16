/*
Copyright 2018 Google LLC

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

package executor

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/commands"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// NewCompositeCache returns an initialized composite cache object.
func NewCompositeCache(initial ...string) *CompositeCache {
	c := CompositeCache{
		keys: initial,
	}
	return &c
}

// CompositeCache is a type that generates a cache key from a series of keys.
type CompositeCache struct {
	keys []string
}

func (s *CompositeCache) AddCommand(command commands.DockerCommand, args *dockerfile.BuildArgs, config *v1.Config) error {
	s.AddKey(command.String())
	// If the command uses files from the context, add them.
	files, err := command.FilesUsedFromContext(config, args)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := s.AddPath(f); err != nil {
			return err
		}
	}
	return nil
}

// AddKey adds the specified key to the sequence.
func (s *CompositeCache) AddKey(k ...string) {
	s.keys = append(s.keys, k...)
}

// Key returns the human readable composite key as a string.
func (s *CompositeCache) Key() string {
	return strings.Join(s.keys, "-")
}

// Hash returns the composite key in a string SHA256 format.
func (s *CompositeCache) Hash() (string, error) {
	return util.SHA256(strings.NewReader(s.Key()))
}

func (s *CompositeCache) AddPath(p string) error {
	sha := sha256.New()
	fi, err := os.Lstat(p)
	if err != nil {
		return err
	}
	if fi.Mode().IsDir() {
		k, err := HashDir(p)
		if err != nil {
			return err
		}
		s.keys = append(s.keys, k)
		return nil
	}
	fh, err := util.CacheHasher()(p)
	if err != nil {
		return err
	}
	if _, err := sha.Write([]byte(fh)); err != nil {
		return err
	}

	s.keys = append(s.keys, string(sha.Sum(nil)))
	return nil
}

func (s CompositeCache) Copy() *CompositeCache {
	return NewCompositeCache(s.keys...)
}

// HashDir returns a hash of the directory.
func HashDir(p string) (string, error) {
	sha := sha256.New()
	if err := filepath.Walk(p, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fileHash, err := util.CacheHasher()(path)
		if err != nil {
			return err
		}
		if _, err := sha.Write([]byte(fileHash)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return "", err
	}

	return string(sha.Sum(nil)), nil
}
