package debug

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"os"
	"strings"
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

//go:embed testdata/request.yaml
var runFunctionRequest string

//go:embed testdata/expected-request-dump.txt
var runFunctionRequestExpectedOutput string

//go:embed testdata/response.yaml
var runFunctionResponse string

//go:embed testdata/expected-response-dump.txt
var runFunctionResponseExpectedOutput string

func loadYamlInto(t *testing.T, yamlStr string, target proto.Message) {
	var data any
	err := yaml.Unmarshal([]byte(yamlStr), &data)
	require.NoError(t, err)
	b, err := json.Marshal(data)
	require.NoError(t, err)
	err = protojson.Unmarshal(b, target)
	require.NoError(t, err)
}

func loadRequest(t *testing.T) *fnv1.RunFunctionRequest {
	var req fnv1.RunFunctionRequest
	loadYamlInto(t, runFunctionRequest, &req)
	return &req
}

func loadResponse(t *testing.T) *fnv1.RunFunctionResponse {
	var res fnv1.RunFunctionResponse
	loadYamlInto(t, runFunctionResponse, &res)
	return &res
}

func TestRequestExample(t *testing.T) {
	req := loadRequest(t)
	buf := bytes.NewBuffer(nil)
	outputWriter = buf
	defer func() {
		outputWriter = os.Stderr
	}()

	p := New(Options{})
	err := p.Request(req)
	require.NoError(t, err)
	// log.Println(buf.String())
	assert.Equal(t, strings.TrimSpace(buf.String()), strings.TrimSpace(runFunctionRequestExpectedOutput))
}

func TestResponseExample(t *testing.T) {
	req := loadRequest(t)
	res := loadResponse(t)
	buf := bytes.NewBuffer(nil)
	outputWriter = buf
	defer func() {
		outputWriter = os.Stderr
	}()

	p := New(Options{})
	err := p.Response(req, res)
	require.NoError(t, err)
	// log.Println(buf.String())
	assert.Equal(t, strings.TrimSpace(buf.String()), strings.TrimSpace(runFunctionResponseExpectedOutput))
}
