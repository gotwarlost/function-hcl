package evaluator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluator_ProcessContext_BasicStringValue(t *testing.T) {
	hclContent := `
context {
  key   = "deployment_status"
  value = "ready"
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify context was added
	assert.Len(t, evaluator.contexts, 1)
	contextObj := evaluator.contexts[0]

	assert.Equal(t, "ready", contextObj["deployment_status"])
}

func TestEvaluator_ProcessContext_NumericValue(t *testing.T) {
	hclContent := `
context {
  key   = "replica_count"
  value = 5
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify context was added with numeric value
	assert.Len(t, evaluator.contexts, 1)
	contextObj := evaluator.contexts[0]

	assert.EqualValues(t, 5, contextObj["replica_count"])
}

func TestEvaluator_ProcessContext_BooleanValue(t *testing.T) {
	hclContent := `
context {
  key   = "backup_enabled"
  value = true
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify context was added with boolean value
	assert.Len(t, evaluator.contexts, 1)
	contextObj := evaluator.contexts[0]

	assert.Equal(t, true, contextObj["backup_enabled"])
}

func TestEvaluator_ProcessContext_ObjectValue(t *testing.T) {
	hclContent := `
context {
  key   = "database_config"
  value = {
    host     = "db.example.com"
    port     = 5432
    ssl_mode = "require"
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify context was added with object value
	assert.Len(t, evaluator.contexts, 1)
	contextObj := evaluator.contexts[0]

	dbConfig, ok := contextObj["database_config"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "db.example.com", dbConfig["host"])
	assert.Equal(t, float64(5432), dbConfig["port"]) // json unmarshaling makes numbers float64
	assert.Equal(t, "require", dbConfig["ssl_mode"])
}

func TestEvaluator_ProcessContext_ListValue(t *testing.T) {
	hclContent := `
context {
  key   = "allowed_regions"
  value = ["us-west-2", "us-east-1", "eu-west-1"]
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify context was added with list value
	assert.Len(t, evaluator.contexts, 1)
	contextObj := evaluator.contexts[0]

	regions, ok := contextObj["allowed_regions"].([]interface{})
	require.True(t, ok)
	assert.Len(t, regions, 3)
	assert.Equal(t, "us-west-2", regions[0])
	assert.Equal(t, "us-east-1", regions[1])
	assert.Equal(t, "eu-west-1", regions[2])
}

func TestEvaluator_ProcessContext_WithVariableReferences(t *testing.T) {
	hclContent := `
context {
  key   = "composite_info"
  value = {
    name        = req.composite.metadata.name
    namespace   = req.composite.metadata.namespace
    environment = req.composite.spec.environment
    region      = req.composite.spec.region
    replicas    = req.composite.spec.replicas
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify context was added with values from variables
	assert.Len(t, evaluator.contexts, 1)
	contextObj := evaluator.contexts[0]

	compositeInfo, ok := contextObj["composite_info"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "my-composite", compositeInfo["name"])
	assert.Equal(t, "default", compositeInfo["namespace"])
	assert.Equal(t, "production", compositeInfo["environment"])
	assert.Equal(t, "us-west-2", compositeInfo["region"])
	assert.Equal(t, float64(3), compositeInfo["replicas"])
}

func TestEvaluator_ProcessContext_WithLocals(t *testing.T) {
	hclContent := `
context {
  locals {
    app_name = req.composite.metadata.name
    env      = req.composite.spec.environment
    full_name = "${app_name}-${env}"
  }
  
  key   = "application_context"
  value = {
    application_name = full_name
    is_production   = env == "production"
    config_path     = "/config/${env}/${app_name}.yaml"
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify context was added with computed values using locals
	assert.Len(t, evaluator.contexts, 1)
	contextObj := evaluator.contexts[0]

	appContext, ok := contextObj["application_context"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "my-composite-production", appContext["application_name"])
	assert.Equal(t, true, appContext["is_production"])
	assert.Equal(t, "/config/production/my-composite.yaml", appContext["config_path"])
}

func TestEvaluator_ProcessContext_MultipleContexts(t *testing.T) {
	hclContent := `
context {
  key   = "environment"
  value = req.composite.spec.environment
}

context {
  key   = "region"
  value = req.composite.spec.region
}

context {
  key   = "metadata"
  value = {
    name      = req.composite.metadata.name
    namespace = req.composite.metadata.namespace
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify all contexts were added
	assert.Len(t, evaluator.contexts, 3)

	// find each context by checking for expected keys
	var envContext, regionContext, metadataContext map[string]interface{}
	for _, ctx := range evaluator.contexts {
		if _, ok := ctx["environment"]; ok {
			envContext = ctx
		}
		if _, ok := ctx["region"]; ok {
			regionContext = ctx
		}
		if _, ok := ctx["metadata"]; ok {
			metadataContext = ctx
		}
	}

	require.NotNil(t, envContext)
	require.NotNil(t, regionContext)
	require.NotNil(t, metadataContext)

	assert.Equal(t, "production", envContext["environment"])
	assert.Equal(t, "us-west-2", regionContext["region"])

	metadata, ok := metadataContext["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "my-composite", metadata["name"])
	assert.Equal(t, "default", metadata["namespace"])
}

func TestEvaluator_ProcessContext_WithinResource(t *testing.T) {
	hclContent := `
resource "database" {
  body = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "Cluster"
    metadata = {
      name = "my-database"
    }
  }
  
  context {
    key   = "database_resource"
    value = {
      name         = self.name
      cluster_name = "my-database"
      ready        = true
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify resource was created
	assert.Contains(t, evaluator.desiredResources, "database")

	// verify context was added from within resource
	require.Len(t, evaluator.contexts, 1)
	contextObj := evaluator.contexts[0]

	dbResource, ok := contextObj["database_resource"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "database", dbResource["name"]) // self.name should be the resource name
	assert.Equal(t, "my-database", dbResource["cluster_name"])
	assert.Equal(t, true, dbResource["ready"])
}

func TestEvaluator_ProcessContext_WithinResourceCollection(t *testing.T) {
	hclContent := `
resources "workers" {
  for_each = ["worker-1", "worker-2"]
  
  template {
    body = {
      apiVersion = "batch/v1"
      kind       = "Job"
      metadata = {
        name = "${self.basename}-${each.key}"
      }
    }
  }
  
  context {
    key   = "worker_collection"
    value = {
      basename      = self.basename
      created_at    = "2024-01-01T00:00:00Z"
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify resources were created
	assert.Contains(t, evaluator.desiredResources, "workers-0")
	assert.Contains(t, evaluator.desiredResources, "workers-1")

	// verify context was added from resource collection
	assert.Len(t, evaluator.contexts, 1)
	contextObj := evaluator.contexts[0]

	workerCollection, ok := contextObj["worker_collection"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "workers", workerCollection["basename"])
	assert.Equal(t, "2024-01-01T00:00:00Z", workerCollection["created_at"])
}

func TestEvaluator_ProcessContext_IncompleteKey(t *testing.T) {
	hclContent := `
context {
  key   = req.composite.spec.nonexistent_field
  value = "test-value"
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	err := evaluator.processGroup(ctx, content)
	require.Error(t, err)
}

func TestEvaluator_ProcessContext_IncompleteValue(t *testing.T) {
	hclContent := `
context {
  key   = "test_key"
  value = req.composite.spec.nonexistent_field
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags.Errs())

	// no context should be added due to incomplete value evaluation
	assert.Empty(t, evaluator.contexts)

	// should have a discard entry for incomplete context
	assert.Len(t, evaluator.discards, 1)
	assert.Equal(t, discardReasonIncomplete, evaluator.discards[0].Reason)
	assert.Equal(t, discardTypeContext, evaluator.discards[0].Type)
}

func TestEvaluator_ProcessContext_NonStringKey(t *testing.T) {
	hclContent := `
context {
  key   = 42
  value = "test-value"
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	err := evaluator.processGroup(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context key was not a string, got number")
}

func TestEvaluator_ProcessContext_NullValue(t *testing.T) {
	hclContent := `
context {
  key   = "null_field"
  value = null
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify context was added with null value
	assert.Len(t, evaluator.contexts, 1)
	contextObj := evaluator.contexts[0]

	assert.Nil(t, contextObj["null_field"])
}

func TestEvaluator_ProcessContext_WithExpressionKey(t *testing.T) {
	hclContent := `
context {
  locals {
    key_prefix = "deployment"
    key_suffix = "status"
  }
  
  key   = "${key_prefix}_${key_suffix}"
  value = "ready"
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify context was added with computed key
	assert.Len(t, evaluator.contexts, 1)
	contextObj := evaluator.contexts[0]

	assert.Equal(t, "ready", contextObj["deployment_status"])
}

func TestEvaluator_ProcessContext_ComplexNestedValue(t *testing.T) {
	hclContent := `
context {
  key   = "complex_config"
  value = {
    database = {
      host = "db.example.com"
      port = 5432
      credentials = {
        username = "admin"
        password_ref = "secret-name"
      }
    }
    cache = {
      enabled = true
      ttl     = 3600
      nodes   = ["cache-1", "cache-2", "cache-3"]
    }
    features = {
      logging   = true
      metrics   = true
      tracing   = false
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify context was added with complex nested structure
	assert.Len(t, evaluator.contexts, 1)
	contextObj := evaluator.contexts[0]

	config, ok := contextObj["complex_config"].(map[string]interface{})
	require.True(t, ok)

	// verify database config
	database, ok := config["database"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "db.example.com", database["host"])
	assert.Equal(t, float64(5432), database["port"])

	credentials, ok := database["credentials"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "admin", credentials["username"])
	assert.Equal(t, "secret-name", credentials["password_ref"])

	// verify cache config
	cache, ok := config["cache"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, cache["enabled"])
	assert.Equal(t, float64(3600), cache["ttl"])

	nodes, ok := cache["nodes"].([]interface{})
	require.True(t, ok)
	assert.Len(t, nodes, 3)
	assert.Equal(t, "cache-1", nodes[0])
	assert.Equal(t, "cache-2", nodes[1])
	assert.Equal(t, "cache-3", nodes[2])

	// verify features config
	features, ok := config["features"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, features["logging"])
	assert.Equal(t, true, features["metrics"])
	assert.Equal(t, false, features["tracing"])
}

func TestEvaluator_ProcessContext_WithinGroup(t *testing.T) {
	hclContent := `
group {
  locals {
    environment = "staging"
    app_name    = "test-app"
  }
  
  resource "deployment" {
    body = {
      apiVersion = "apps/v1"
      kind       = "Deployment"
      metadata = {
        name = app_name
      }
    }
  }
  
  context {
    key   = "group_context"
    value = {
      environment    = environment
      app_name       = app_name
      resources_created = 1
    }
  }
}
`

	evaluator := createTestEvaluator(t)
	ctx := createTestEvalContext()
	content := parseHCL(t, evaluator, hclContent, "test.hcl")

	diags := evaluator.processGroup(ctx, content)
	require.Empty(t, diags)

	// verify resource was created
	assert.Contains(t, evaluator.desiredResources, "deployment")

	// verify context was added from group with shared locals
	assert.Len(t, evaluator.contexts, 1)
	contextObj := evaluator.contexts[0]

	groupContext, ok := contextObj["group_context"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "staging", groupContext["environment"])
	assert.Equal(t, "test-app", groupContext["app_name"])
	assert.Equal(t, float64(1), groupContext["resources_created"])
}
