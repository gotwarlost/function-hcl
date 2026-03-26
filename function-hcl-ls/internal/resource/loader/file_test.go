package loader_test

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/resource"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/resource/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileLoadWithCRDDocuments(t *testing.T) {
	l := loader.NewFile("testdata/crds.yaml")
	s1, err := l.Load()
	require.NoError(t, err)
	assert.Equal(t, 3, len(s1.Keys()))
	l = loader.NewFile("testdata/xrds.yaml")
	s2, err := l.Load()
	require.NoError(t, err)
	assert.Equal(t, 1, len(s2.Keys()))
	s3 := resource.Compose(s1, s2)
	assert.Equal(t, 4, len(s3.Keys()))
}
