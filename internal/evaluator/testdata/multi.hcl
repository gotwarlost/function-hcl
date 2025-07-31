locals {
  compName = req.composite.metadata.name
  params   = req.composite.spec.parameters
  name     = "${compName}-cm"
}

resources config-map {
  for_each = params.namespaces
  name = "cm-${self.basename}-${each.value}"
  template {
    body = {
      apiVersion = "kubernetes.crossplane.io/v1alpha2"
      kind = "Object"
      metadata = {
        name : self.name
      }
      spec = {
        forProvider : {
          manifest : {
            apiVersion : "v1"
            kind : "ConfigMap"
            metadata : {
              namespace : each.value
              name : name
              labels : req.composite.metadata.labels
            }
            data : params.data
          }
        }
      }
    }
  }
}

resources more {
  for_each = params.namespaces
  condition = req.context["example.com/testing"] != "true"
  name = "cm-${self.basename}-${each.value}"
  template {
    body = {
      apiVersion = "kubernetes.crossplane.io/v1alpha2"
      kind = "Object"
      metadata = {
        name : self.name
      }
      spec = {
        forProvider : {
          manifest : {
            apiVersion : "v1"
            kind : "ConfigMap"
            metadata : {
              namespace : each.value
              name : name
              labels : req.composite.metadata.labels
            }
            data : params.data
          }
        }
      }
    }
  }
}
