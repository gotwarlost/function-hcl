package crds

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/resource/loader"
	types "github.com/crossplane-contrib/function-hcl/function-hcl-ls/types/v1"
	"github.com/ghodss/yaml"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

func writeCacheFile(dir string, image string, objects []runtime.Object) (filename string, finalErr error) {
	ref, err := name.ParseReference(image, name.WithDefaultTag("latest"))
	if err != nil {
		return "", fmt.Errorf("invalid image reference %q: %w", image, err)
	}
	var (
		repo    string
		version string
	)
	switch r := ref.(type) {
	case name.Tag:
		// e.g. gcr.io/repo/img:tag
		repo = r.RepositoryStr()
		version = r.TagStr()
	case name.Digest:
		// e.g. gcr.io/repo/img@sha256:abcdef...
		repo = r.RepositoryStr()
		// Replace ':' in digest to keep filename filesystem-safe.
		version = r.DigestStr()
	default:
		repo = ref.Context().RepositoryStr()
		version = ref.Identifier()
	}
	base := path.Base(repo)
	version = strings.ReplaceAll(version, ":", "-")

	file := filepath.Join(dir, fmt.Sprintf("%s-%s.yaml", base, version))
	f, err := os.Create(file)
	if err != nil {
		return file, err
	}
	defer func() {
		err := f.Close()
		if err != nil && finalErr == nil {
			finalErr = err
		}
	}()
	_, _ = io.WriteString(f, strings.TrimSpace(fmt.Sprintf(`
# GENERATED FILE DO NOT EDIT
# CRDs and XRDs downloaded from %s
`, image)))
	_, _ = io.WriteString(f, "\n")

	for i, obj := range objects {
		if i > 0 {
			if _, err = io.WriteString(f, "\n---\n"); err != nil {
				return file, err
			}
		}
		var b []byte
		b, err = yaml.Marshal(obj)
		if err != nil {
			return file, err
		}
		if _, err = f.Write(b); err != nil {
			return file, err
		}
	}
	return file, nil
}

func downloadCRDs(f string, deleteCache bool) (finalErr error) {
	logger := log.New(os.Stderr, "", 0)
	start := time.Now()
	defer func() {
		if finalErr == nil {
			logger.Printf("completed in %s", time.Since(start).Round(time.Second))
		}
	}()
	sourcesFile, err := filepath.Abs(f)
	if err != nil {
		return errors.Wrap(err, "get absolute path")
	}

	logger.Printf("* processing locations from: %s ...", sourcesFile)
	src, err := readSource(sourcesFile)
	if err != nil {
		return err
	}

	cd := src.Offline.CacheDir
	if cd == "" {
		return fmt.Errorf("no cache dir specified in %s", sourcesFile)
	}
	if len(src.Offline.Images) == 0 {
		return fmt.Errorf("no offline images specified in %s", sourcesFile)
	}

	baseDir := filepath.Dir(sourcesFile)
	cacheDir := filepath.Clean(filepath.Join(baseDir, cd))

	if deleteCache {
		logger.Printf("* deleting cache dir: %s ...", cacheDir)
		if err := os.RemoveAll(cacheDir); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	logger.Printf("* writing CRDs to dir: %s ...", cacheDir)
	logger.Printf("* downloading CRDs from %d images ...", len(src.Offline.Images))

	for imageIndex, image := range src.Offline.Images {
		l := loader.NewCrossplanePackage(image)
		objects, err := l.ExtractObjects()
		if err != nil {
			return errors.Wrapf(err, "extract objects from %s", image)
		}
		outFile, err := writeCacheFile(cacheDir, image, objects)
		if err != nil {
			return errors.Wrapf(err, "write CRD file %s", outFile)
		}
		logger.Printf("\t%3d. %s (%d objects)", imageIndex+1, filepath.Base(outFile), len(objects))
	}
	return nil
}

func readSource(filename string) (*types.CRDSource, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var ret types.CRDSource
	err = yaml.Unmarshal(b, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal CRD source")
	}
	return &ret, nil
}
