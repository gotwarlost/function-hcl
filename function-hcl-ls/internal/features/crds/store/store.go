package store

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/resource"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/utils/queue"
	types "github.com/crossplane-contrib/function-hcl/function-hcl-ls/types/v1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

func shouldProcessFile(name string) bool {
	return filepath.Ext(name) == ".yaml"
}

var emptySchema = resource.ToSchemas()

func discoveryKeyForRegisteredDir(dir string) queue.Key {
	return queue.Key(fmt.Sprintf("dir-discovery:%s", dir))
}

func loadSourceKey(dir string) queue.Key {
	return queue.Key(fmt.Sprintf("load-source:%s", dir))
}

func loadSchemaKey(filePath string) queue.Key {
	return queue.Key(fmt.Sprintf("load-schema:%s", filePath))
}

func (s *Store) registerOpenDir(path string) {
	s.seenLock.Lock()
	defer s.seenLock.Unlock()
	if _, ok := s.seenDirs[path]; ok {
		return
	}
	s.seenDirs[path] = ""
	s.queue.Enqueue(discoveryKeyForRegisteredDir(path), func() error {
		return s.discoverSourceStore(path, false)
	})
}

func (s *Store) reprocess() {
	s.seenLock.Lock()
	defer s.seenLock.Unlock()
	for moduleDir := range s.seenDirs {
		s.queue.Enqueue(discoveryKeyForRegisteredDir(moduleDir), func() error {
			return s.discoverSourceStore(moduleDir, true)
		})
	}
}

func findAncestor(pathToSearch string, fileToFind string, expectDir bool) (string, bool) {
	parent := filepath.Dir(pathToSearch)
	if parent == pathToSearch {
		return "", false
	}
	testPath := filepath.Join(pathToSearch, fileToFind)
	st, err := os.Stat(testPath)
	if err != nil {
		return findAncestor(parent, fileToFind, expectDir)
	}
	if st.IsDir() != expectDir {
		return findAncestor(parent, fileToFind, expectDir)
	}
	return pathToSearch, true
}

func (s *Store) discoverSourceStore(dir string, reprocessing bool) error {
	foundDir, found := findAncestor(dir, types.StandardSourcesFile, false)
	if !found {
		foundDir, found = findAncestor(dir, types.DefaultSourcesDir, true)
	}
	if !found {
		if !reprocessing && s.onNoCRDSources != nil {
			s.onNoCRDSources(dir)
		}
		return nil
	}

	s.seenLock.Lock()
	s.seenDirs[dir] = foundDir
	defer s.seenLock.Unlock()

	store := s.sources.get(foundDir)
	if store == nil || reprocessing {
		s.queue.Enqueue(loadSourceKey(foundDir), func() error {
			return s.loadSourceStoreAt(foundDir)
		})
	}
	return nil
}

func (s *Store) getSourceInfo(dir string) (ret types.CRDSource, _ error) {
	var src types.CRDSource

	filePath := filepath.Join(dir, types.StandardSourcesFile)
	st, err := os.Stat(filePath)
	if err == nil && !st.IsDir() {
		sourceFilePath := filepath.Join(dir, types.StandardSourcesFile)
		b, err := os.ReadFile(sourceFilePath)
		if err != nil {
			return ret, err
		}
		err = yaml.Unmarshal(b, &src)
		if err != nil {
			return ret, errors.Wrapf(err, "unmarshal source file %s", sourceFilePath)
		}
		return src, nil
	}

	filePath = filepath.Join(dir, types.DefaultSourcesDir)
	_, err = os.Stat(filePath)
	if err != nil {
		return ret, fmt.Errorf("load source store at %s: no sources file or default directory", filePath)
	}
	return types.CRDSource{
		Scope: types.ScopeBoth,
		Paths: []string{
			filepath.Join(types.DefaultSourcesDir, "*.yaml"),
		},
	}, nil
}

func (s *Store) loadSourceStoreAt(dir string) error {
	src, err := s.getSourceInfo(dir)
	if err != nil {
		return err
	}
	foundFiles := map[string]bool{}
	for _, p := range src.Paths {
		var pattern string
		if filepath.IsAbs(p) {
			pattern = p
		} else {
			pattern = filepath.Clean(filepath.Join(dir, p))
		}
		matches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			log.Printf("filepath glob err, ignore: %s : %s", pattern, err)
			continue
		}
		for _, match := range matches {
			st, err := os.Stat(match)
			if err != nil {
				log.Printf("stat err, ignore: %s : %s", match, err)
				continue
			}
			if st.IsDir() {
				log.Printf("skip dir: %s", match)
				continue
			}
			if !shouldProcessFile(match) {
				log.Printf("skip non-yaml file: %s", match)
				continue
			}
			foundFiles[match] = true
		}
	}
	ss := s.sources.get(dir)
	if ss == nil { // never before seen
		ss = &sourceInfo{
			sourcePath:    dir,
			source:        &src,
			expandedFiles: foundFiles,
			schema:        emptySchema,
		}
	} else {
		// but leave the last known schema for this one alone
		ss.source = &src
		ss.expandedFiles = foundFiles
	}
	s.sources.put(ss)

	var files []string
	for ff := range foundFiles {
		files = append(files, ff)
	}
	sort.Strings(files)
	for _, file := range files {
		s.queue.Enqueue(loadSchemaKey(file), func() error {
			err := s.files.add(file)
			if err != nil && !errors.Is(err, errNoChanges) {
				return err
			}
			if errors.Is(err, errNoChanges) {
				return nil
			}
			return s.propagateSchemaForFile(file)
		})
	}
	return nil
}

func (s *Store) propagateSchemaForFile(file string) error {
	stores := s.sources.list()
	for _, storePath := range stores {
		ss := s.sources.get(storePath)
		if !ss.expandedFiles[file] {
			continue
		}
		var schemas []*resource.Schemas
		for f := range ss.expandedFiles {
			schemas = append(schemas, s.files.getSchema(f))
		}
		updated := resource.Compose(schemas...).FilterScope(ss.source.Scope)
		ss.schema = updated
		s.sources.put(ss)
	}
	return nil
}
