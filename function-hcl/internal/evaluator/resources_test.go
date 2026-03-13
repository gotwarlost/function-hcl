package evaluator

import (
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// createTestEvaluator provides a minimal evaluator for testing resource processing.
func createTestEvaluator(t *testing.T) *Evaluator {
	evaluator, err := New(Options{})
	require.NoError(t, err)
	return evaluator
}

// createTestEvalContext creates a test context with typical variables for resource processing.
func createTestEvalContext() *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"req": cty.ObjectVal(map[string]cty.Value{
				"composite": cty.ObjectVal(map[string]cty.Value{
					"metadata": cty.ObjectVal(map[string]cty.Value{
						"name":      cty.StringVal("my-composite"),
						"namespace": cty.StringVal("default"),
					}),
					"spec": cty.ObjectVal(map[string]cty.Value{
						"enabled":     cty.BoolVal(true),
						"environment": cty.StringVal("production"),
						"image":       cty.StringVal("nginx:latest"),
						"region":      cty.StringVal("us-west-2"),
						"replicas":    cty.NumberIntVal(3),
					}),
				}),
			}),
			"self": cty.ObjectVal(map[string]cty.Value{
				"name":     cty.StringVal("test-resource"),
				"basename": cty.StringVal("test-base"),
			}),
		},
	}
}

// parseHCL parses HCL content for testing.
func parseHCL(t *testing.T, evaluator *Evaluator, content string, filename string) *hcl.BodyContent {
	t.Helper()
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(content), filename)
	evaluator.files[filename] = file
	require.False(t, diags.HasErrors(), "failed to parse HCL: %s", diags)

	schema := topLevelSchema()
	contentBody, diags := file.Body.Content(schema)
	require.False(t, diags.HasErrors(), "failed to get content: %s", diags)

	return contentBody
}

func TestEvaluator_ProcessResource_Basic(t *testing.T) {
	hclContent := `
resource "test-deployment" {
  body = {
    apiVersion = "apps/v1"
    kind       = "Deployment"
    metadata = {
      name = "test-app"
    }
    spec = {
      replicas = 3
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "main.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// check that resource was added to desired resources
	assert.Contains(t, evaluator.desiredResources, "test-deployment")

	resource := evaluator.desiredResources["test-deployment"]
	resourceMap := resource.AsMap()

	assert.Equal(t, "apps/v1", resourceMap["apiVersion"])
	assert.Equal(t, "Deployment", resourceMap["kind"])

	metadata, ok := resourceMap["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-app", metadata["name"])

	spec, ok := resourceMap["spec"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(3), spec["replicas"])
}

func TestEvaluator_ProcessResource_WithLocals(t *testing.T) {
	hclContent := `
resource "test-service" {
  locals {
    app_name = "my-app"
    port     = 8080
  }
  
  body = {
    apiVersion = "v1"
    kind       = "Service"
    metadata = {
      name = app_name
    }
    spec = {
      ports = [{
        port = port
      }]
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	assert.Contains(t, evaluator.desiredResources, "test-service")

	resource := evaluator.desiredResources["test-service"]
	resourceMap := resource.AsMap()

	metadata, ok := resourceMap["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "my-app", metadata["name"])

	spec, ok := resourceMap["spec"].(map[string]interface{})
	require.True(t, ok)
	ports, ok := spec["ports"].([]interface{})
	require.True(t, ok)
	port, ok := ports[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(8080), port["port"])
}

func TestEvaluator_ProcessResource_WithCondition(t *testing.T) {
	hclContent := `
resource "conditional-resource" {
  condition = req.composite.spec.replicas > 0
  
  body = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name = "test-config"
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// resource should be created since replicas = 3 > 0
	assert.Contains(t, evaluator.desiredResources, "conditional-resource")
}

func TestEvaluator_ProcessResource_ConditionFalse(t *testing.T) {
	hclContent := `
resource "conditional-resource" {
  condition = req.composite.spec.replicas < 0
  
  body = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name = "test-config"
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// resource should not be created since replicas = 3 is not < 0
	assert.NotContains(t, evaluator.desiredResources, "conditional-resource")

	// should have a discard entry for user condition
	assert.Len(t, evaluator.discards, 1)
	assert.Equal(t, discardReasonUserCondition, evaluator.discards[0].Reason)
}

func TestEvaluator_ProcessResource_Duplicate(t *testing.T) {
	hclContent := `
resource "duplicate-name" {
  body = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name = "config1"
    }
  }
}

resource "duplicate-name" {
  body = {
    apiVersion = "v1"
    kind       = "Secret"
    metadata = {
      name = "secret1"
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	err := evaluator.processGroup(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate resource")
}

func TestEvaluator_ProcessResources_ForEach(t *testing.T) {
	hclContent := `
resources "databases" {
  for_each = {
    "primary" = {
      size = "large"
      backup = true
    }
    "secondary" = {
      size = "small"
      backup = false
    }
  }
  
  template {
    body = {
      apiVersion = "postgresql.cnpg.io/v1"
      kind       = "Cluster"
      metadata = {
        name = "${self.basename}-${each.key}"
      }
      spec = {
        instances = each.value.size == "large" ? 3 : 1
        backup = {
          enabled = each.value.backup
        }
      }
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// check that both resources were created with correct names
	// self.basename gets set to the resources block label "databases"
	assert.Contains(t, evaluator.desiredResources, "databases-primary")
	assert.Contains(t, evaluator.desiredResources, "databases-secondary")

	// verify primary database configuration
	primaryResource := evaluator.desiredResources["databases-primary"]
	primaryMap := primaryResource.AsMap()

	primaryMetadata, ok := primaryMap["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "databases-primary", primaryMetadata["name"])

	primarySpec, ok := primaryMap["spec"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(3), primarySpec["instances"]) // large = 3 instances

	primaryBackup, ok := primarySpec["backup"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, primaryBackup["enabled"])

	// verify secondary database configuration
	secondaryResource := evaluator.desiredResources["databases-secondary"]
	secondaryMap := secondaryResource.AsMap()

	secondarySpec, ok := secondaryMap["spec"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), secondarySpec["instances"]) // small = 1 instance

	secondaryBackup, ok := secondarySpec["backup"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, false, secondaryBackup["enabled"])
}

func TestEvaluator_ProcessResources_ForEachList(t *testing.T) {
	hclContent := `
resources "workers" {
  for_each = ["worker-1", "worker-2", "worker-3"]
  
  template {
    body = {
      apiVersion = "v1"
      kind       = "Pod"
      metadata = {
        name = "${self.basename}-${each.key}"
        labels = {
          worker_name = each.value
        }
      }
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// check that all three resources were created (list indices 0, 1, 2)
	// self.basename gets set to the resources block label "workers"
	assert.Contains(t, evaluator.desiredResources, "workers-0")
	assert.Contains(t, evaluator.desiredResources, "workers-1")
	assert.Contains(t, evaluator.desiredResources, "workers-2")

	// verify worker-1 (index 0)
	worker0 := evaluator.desiredResources["workers-0"]
	worker0Map := worker0.AsMap()

	worker0Metadata, ok := worker0Map["metadata"].(map[string]interface{})
	require.True(t, ok)
	worker0Labels, ok := worker0Metadata["labels"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "worker-1", worker0Labels["worker_name"])
}

func TestEvaluator_ProcessResources_CustomName(t *testing.T) {
	hclContent := `
resources "apps" {
  for_each = ["frontend", "backend"]
  name     = "${each.value}-service"
  
  template {
    body = {
      apiVersion = "v1"
      kind       = "Service"
      metadata = {
        name = each.value
      }
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// check that resources use custom names instead of default self.basename-each.key
	assert.Contains(t, evaluator.desiredResources, "frontend-service")
	assert.Contains(t, evaluator.desiredResources, "backend-service")
	assert.NotContains(t, evaluator.desiredResources, "apps-0")
	assert.NotContains(t, evaluator.desiredResources, "apps-1")
}

func TestEvaluator_ProcessResources_WithCondition(t *testing.T) {
	hclContent := `
resources "conditional-apps" {
  condition = req.composite.spec.replicas > 1
  for_each  = ["app1", "app2"]
  
  template {
    body = {
      apiVersion = "apps/v1"
      kind       = "Deployment"
      metadata = {
        name = each.value
      }
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// resources should be created since replicas = 3 > 1
	// self.basename gets set to "conditional-apps"
	assert.Contains(t, evaluator.desiredResources, "conditional-apps-0")
	assert.Contains(t, evaluator.desiredResources, "conditional-apps-1")
}

func TestEvaluator_ProcessResources_ConditionFalse(t *testing.T) {
	hclContent := `
resources "conditional-apps" {
  condition = req.composite.spec.replicas > 10
  for_each  = ["app1", "app2"]
  
  template {
    body = {
      apiVersion = "apps/v1"
      kind       = "Deployment"
      metadata = {
        name = each.value
      }
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// no resources should be created since replicas = 3 is not > 10
	assert.Empty(t, evaluator.desiredResources)
}

func TestEvaluator_ProcessResources_NoTemplate(t *testing.T) {
	hclContent := `
resources "missing-template" {
  for_each = ["item1", "item2"]
  
  # no template block - should error
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	err := evaluator.processGroup(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no template block")
}

func TestEvaluator_ProcessResources_MultipleTemplates(t *testing.T) {
	hclContent := `
resources "multiple-templates" {
  for_each = ["item1"]
  
  template {
    body = {
      kind = "ConfigMap"
    }
  }
  
  template {
    body = {
      kind = "Secret"
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	err := evaluator.processGroup(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple template blocks")
}

func TestEvaluator_ProcessResource_WithReady(t *testing.T) {
	hclContent := `
resource "ready-resource" {
  body = {
    apiVersion = "v1"
    kind       = "Pod"
    metadata = {
      name = "test-pod"
    }
  }
  
  ready {
    value = "READY_TRUE"
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// check that resource was created
	assert.Contains(t, evaluator.desiredResources, "ready-resource")

	// check that ready state was set
	assert.Contains(t, evaluator.ready, "ready-resource")
	assert.Equal(t, fnv1.Ready_READY_TRUE, fnv1.Ready(evaluator.ready["ready-resource"]))
}

func TestEvaluator_ProcessResource_InvalidReadyValue(t *testing.T) {
	hclContent := `
resource "invalid-ready" {
  body = {
    apiVersion = "v1"
    kind       = "Pod"
    metadata = {
      name = "test-pod"
    }
  }
  
  ready {
    value = "INVALID_READY_VALUE"
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	err := evaluator.processGroup(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not have a valid value")
}

func TestEvaluator_ProcessResource_IncompleteBody(t *testing.T) {
	hclContent := `
resource "incomplete-resource" {
  body = {
    apiVersion = "v1"
    kind       = "Pod"
    metadata = {
      name = req.nonexistent_field
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags.Errs())

	// resource should not be in desired resources due to incomplete evaluation
	assert.NotContains(t, evaluator.desiredResources, "incomplete-resource")

	// should have a discard entry for incomplete resource
	assert.Len(t, evaluator.discards, 1)
	assert.Equal(t, discardReasonIncomplete, evaluator.discards[0].Reason)
	assert.Equal(t, discardTypeResource, evaluator.discards[0].Type)
}

func TestEvaluator_ProcessResource_IncompleteNestedLocal(t *testing.T) {
	hclContent := `
resource "incomplete-resource" {
  locals {
    manifest = {
      name = {
	  	foo = [{
			bar = {
				label_1 = "value_1"
				label_2 = self.resource.status.nonexistent
			}
		}]
	  }
    }
  }

  body = {
    apiVersion = "v1"
    kind       = "Pod"
    metadata = {
      labels = manifest
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags.Errs())

	// resource should not be in desired resources due to incomplete evaluation
	assert.NotContains(t, evaluator.desiredResources, "incomplete-resource")

	expectedDiagnosticMessagePart := "test.hcl:9,28-35: Attempt to get attribute from null value; This value is null, so it does not have any attributes"
	assert.Contains(t, diags.Error(), expectedDiagnosticMessagePart)

	// should have a discard entry for incomplete resource
	assert.Len(t, evaluator.discards, 1)
	assert.Equal(t, discardReasonIncomplete, evaluator.discards[0].Reason)
	assert.Equal(t, discardTypeResource, evaluator.discards[0].Type)
	assert.Len(t, evaluator.discards[0].Context, 1)
	assert.Equal(t, evaluator.discards[0].Context[0], "unknown values: manifest.name.foo[0].bar.label_2")
}

func TestEvaluator_ProcessResources_EmptyForEach(t *testing.T) {
	hclContent := `
resources "empty-collection" {
  for_each = []
  
  template {
    body = {
      apiVersion = "v1"
      kind       = "ConfigMap"
      metadata = {
        name = "should-not-exist"
      }
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// no resources should be created from empty for_each
	assert.Empty(t, evaluator.desiredResources)
}

func TestEvaluator_ProcessResources_WithResourceLocals(t *testing.T) {
	hclContent := `
resources "apps-with-locals" {
  for_each = ["api", "worker"]
  
  locals {
    port_map = {
      "api"    = 8080
      "worker" = 9090
    }
    base_config = {
      replicas = 3
      image    = "alpine:latest"
    }
  }
  
  template {
    locals {
      app_type = each.value
      selected_port = port_map[app_type]
    }
    
    body = {
      apiVersion = "v1"
      kind       = "Service"
      metadata = {
        name = "${self.basename}-${app_type}"
      }
      spec = {
        ports = [{
          port = selected_port
        }]
        replicas = base_config.replicas
      }
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// check that both resources were created
	// self.basename gets set to "apps-with-locals"
	assert.Contains(t, evaluator.desiredResources, "apps-with-locals-0")
	assert.Contains(t, evaluator.desiredResources, "apps-with-locals-1")

	// verify api service (index 0)
	apiResource := evaluator.desiredResources["apps-with-locals-0"]
	apiMap := apiResource.AsMap()

	apiMetadata, ok := apiMap["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "apps-with-locals-api", apiMetadata["name"])

	apiSpec, ok := apiMap["spec"].(map[string]interface{})
	require.True(t, ok)
	apiPorts, ok := apiSpec["ports"].([]interface{})
	require.True(t, ok)
	apiPort, ok := apiPorts[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(8080), apiPort["port"])
	assert.Equal(t, float64(3), apiSpec["replicas"]) // from resources-level locals

	// verify worker service (index 1)
	workerResource := evaluator.desiredResources["apps-with-locals-1"]
	workerMap := workerResource.AsMap()

	workerMetadata, ok := workerMap["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "apps-with-locals-worker", workerMetadata["name"])

	workerSpec, ok := workerMap["spec"].(map[string]interface{})
	require.True(t, ok)
	workerPorts, ok := workerSpec["ports"].([]interface{})
	require.True(t, ok)
	workerPort, ok := workerPorts[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(9090), workerPort["port"])
	assert.Equal(t, float64(3), workerSpec["replicas"]) // from resources-level locals
}

func TestEvaluator_ProcessGroup_Basic(t *testing.T) {
	hclContent := `
group {
  resource "app-deployment" {
    body = {
      apiVersion = "apps/v1"
      kind       = "Deployment"
      metadata = {
        name = "app"
      }
    }
  }
  
  resource "app-service" {
    body = {
      apiVersion = "v1"
      kind       = "Service"
      metadata = {
        name = "app-svc"
      }
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// check that both resources from the group were created
	assert.Contains(t, evaluator.desiredResources, "app-deployment")
	assert.Contains(t, evaluator.desiredResources, "app-service")

	// verify deployment
	deployment := evaluator.desiredResources["app-deployment"]
	deploymentMap := deployment.AsMap()
	assert.Equal(t, "apps/v1", deploymentMap["apiVersion"])
	assert.Equal(t, "Deployment", deploymentMap["kind"])

	// verify service
	service := evaluator.desiredResources["app-service"]
	serviceMap := service.AsMap()
	assert.Equal(t, "v1", serviceMap["apiVersion"])
	assert.Equal(t, "Service", serviceMap["kind"])
}

func TestEvaluator_ProcessGroup_WithLocals(t *testing.T) {
	hclContent := `
group {
  locals {
    app_name = "my-application"
    namespace = "production"
  }
  
  resource "deployment" {
    body = {
      apiVersion = "apps/v1"
      kind       = "Deployment"
      metadata = {
        name      = app_name
        namespace = namespace
      }
    }
  }
  
  resource "service" {
    body = {
      apiVersion = "v1"
      kind       = "Service"
      metadata = {
        name      = "${app_name}-svc"
        namespace = namespace
      }
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify that group locals are shared across resources
	deployment := evaluator.desiredResources["deployment"]
	deploymentMap := deployment.AsMap()
	deploymentMetadata, ok := deploymentMap["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "my-application", deploymentMetadata["name"])
	assert.Equal(t, "production", deploymentMetadata["namespace"])

	service := evaluator.desiredResources["service"]
	serviceMap := service.AsMap()
	serviceMetadata, ok := serviceMap["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "my-application-svc", serviceMetadata["name"])
	assert.Equal(t, "production", serviceMetadata["namespace"])
}

func TestEvaluator_ProcessGroup_WithCondition(t *testing.T) {
	hclContent := `
group {
  condition = req.composite.spec.environment == "production"
  
  resource "prod-resource" {
    body = {
      apiVersion = "v1"
      kind       = "ConfigMap"
      metadata = {
        name = "production-config"
      }
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// resource should be created since environment = "production"
	assert.Contains(t, evaluator.desiredResources, "prod-resource")
}

func TestEvaluator_ProcessGroup_ConditionFalse(t *testing.T) {
	hclContent := `
group {
  condition = req.composite.spec.environment == "development"
  
  resource "dev-resource" {
    body = {
      apiVersion = "v1"
      kind       = "ConfigMap"
      metadata = {
        name = "development-config"
      }
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// resource should not be created since environment = "production" != "development"
	assert.NotContains(t, evaluator.desiredResources, "dev-resource")
}

func TestEvaluator_ProcessGroup_Nested(t *testing.T) {
	hclContent := `
group {
  locals {
    base_name = "app"
  }
  
  group {
    locals {
      component = "frontend"
    }
    
    resource "frontend-deployment" {
      body = {
        apiVersion = "apps/v1"
        kind       = "Deployment"
        metadata = {
          name = "${base_name}-${component}"
        }
      }
    }
  }
  
  group {
    locals {
      component = "backend"
    }
    
    resource "backend-deployment" {
      body = {
        apiVersion = "apps/v1"
        kind       = "Deployment"
        metadata = {
          name = "${base_name}-${component}"
        }
      }
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify nested groups created resources with proper variable scoping
	assert.Contains(t, evaluator.desiredResources, "frontend-deployment")
	assert.Contains(t, evaluator.desiredResources, "backend-deployment")

	frontend := evaluator.desiredResources["frontend-deployment"]
	frontendMap := frontend.AsMap()
	frontendMetadata, ok := frontendMap["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "app-frontend", frontendMetadata["name"])

	backend := evaluator.desiredResources["backend-deployment"]
	backendMap := backend.AsMap()
	backendMetadata, ok := backendMap["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "app-backend", backendMetadata["name"])
}
