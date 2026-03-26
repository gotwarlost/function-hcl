package resource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	xpv1 "github.com/crossplane/crossplane/v2/apis/apiextensions/v1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestXRDToSchema(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "xeks.yaml"))
	require.NoError(t, err)
	var xrd xpv1.CompositeResourceDefinition
	err = yaml.Unmarshal(b, &xrd)
	require.NoError(t, err)

	b, err = os.ReadFile(filepath.Join("testdata", "s3.yaml"))
	require.NoError(t, err)
	var crd v1.CustomResourceDefinition
	err = yaml.Unmarshal(b, &crd)
	require.NoError(t, err)

	s := ToSchemas(&xrd, &crd)

	keys := s.Keys()
	require.Equal(t, 2, len(keys))
	require.EqualValues(t, Key{ApiVersion: "s3.aws.upbound.io/v1beta1", Kind: "BucketAccelerateConfiguration"}, keys[0])
	require.EqualValues(t, Key{ApiVersion: "aws.xrd.example.com/v1alpha1", Kind: "XEKS"}, keys[1])

	as := s.Schema("aws.xrd.example.com/v1alpha1", "XEKS")
	require.NotNil(t, as)
	cons, ok := as.Constraint.(schema.Object)
	require.True(t, ok)
	require.NotNil(t, cons)

	as2 := cons.Attributes["spec"]
	require.NotNil(t, as)
	assert.True(t, as2.IsRequired)

	as3 := cons.Attributes["status"]
	require.NotNil(t, as3)
	assert.False(t, as3.IsRequired)

	cons2, ok := as2.Constraint.(schema.Object)
	require.True(t, ok)
	require.NotNil(t, cons2)

	as4 := cons2.Attributes["parameters"]
	require.NotNil(t, as4)
	assert.True(t, as4.IsRequired)

	as = s.Schema("s3.aws.upbound.io/v1beta1", "BucketAccelerateConfiguration")
	require.NotNil(t, as)
	cons, ok = as.Constraint.(schema.Object)
	require.True(t, ok)
	require.NotNil(t, cons)

	as2 = cons.Attributes["spec"]
	require.NotNil(t, as)
	assert.True(t, as2.IsRequired)

	cons2, ok = as2.Constraint.(schema.Object)
	require.True(t, ok)
	require.NotNil(t, cons2)

	as3 = cons2.Attributes["forProvider"]
	require.NotNil(t, as3)
	assert.True(t, as3.IsRequired)
}
