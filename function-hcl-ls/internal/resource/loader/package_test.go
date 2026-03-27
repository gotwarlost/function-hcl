package loader_test

import (
	"log"
	"os"
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/resource/loader"
	"github.com/stretchr/testify/require"
)

func TestPackageLoader(t *testing.T) {
	image := os.Getenv("TEST_PACKAGE_LOADER_IMAGE")
	if image == "" {
		t.Skip("skip: TEST_PACKAGE_LOADER_IMAGE env variable not set")
	}
	p := loader.NewCrossplanePackage(image)
	schemas, err := p.Load()
	require.NoError(t, err)
	keys := schemas.Keys()
	for _, k := range keys {
		log.Println("\t", k)
	}
	log.Println("NUM:", len(keys))
	require.NotNil(t, schemas)
}
