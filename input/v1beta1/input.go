// Package v1beta1 contains the input type for the cue function runner.
// +kubebuilder:object:generate=true
// +groupName=hcl.fn.crossplane.io
// +versionName=v1beta1
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This isn't a custom resource, in the sense that we never install its CRD.
// It is a KRM-like object, so we generate a CRD to describe its schema.

// A ScriptSource is a source from which a script can be loaded.
type ScriptSource string

// Supported script sources.
const (
	// ScriptSourceInline specifies a script inline.
	ScriptSourceInline ScriptSource = "Inline"
)

// HclInput can be used to provide input to the function.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=crossplane
type HclInput struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Source of this script. Currently only Inline is supported.
	// +kubebuilder:validation:Enum=Inline
	// +kubebuilder:default=Inline
	Source ScriptSource `json:"source"`
	// HCL specifies inline hcl. This can be the contents of a single file
	// or a set of files with unique names in txtar format. The actual names of
	// the files are irrelevant and only used for error reporting.
	// +optional
	HCL string `json:"hcl,omitempty"`
	// Debug prints inputs to and outputs of the hcl script for all XRs.
	// Inputs are pre-processed to remove typically irrelevant information like
	// the last applied kubectl annotation, managed fields etc.
	// Objects are displayed in compact cue format. (the equivalent of `cue fmt -s`)
	// When false, individual XRs can still be debugged by annotating them with
	//    "hcl.fn.crossplane.io/debug: "true"
	// +optional
	Debug bool `json:"debug,omitempty"`
	// DebugNew controls whether a new XR that is being processed by the function
	// has debug output. A "new" XR is determined by the request having only an
	// observed composite but no other observed resources. This allows debug output for
	// first-time reconciles of XRs when the user has not yet had the opportunity to
	// annotate them.
	// +optional
	DebugNew bool `json:"debugNew,omitempty"`
}
