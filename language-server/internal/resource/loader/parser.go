/*
Copyright 2020 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package loader

import (
	"context"
	"fmt"
	"io"

	"github.com/crossplane/crossplane-runtime/v2/pkg/parser"
	v1 "github.com/crossplane/crossplane/v2/apis/apiextensions/v1"
	"github.com/crossplane/crossplane/v2/apis/apiextensions/v1alpha1"
	v2 "github.com/crossplane/crossplane/v2/apis/apiextensions/v2"
	opsv1alpha1 "github.com/crossplane/crossplane/v2/apis/ops/v1alpha1"
	pkgmetav1 "github.com/crossplane/crossplane/v2/apis/pkg/meta/v1"
	pkgmetav1alpha1 "github.com/crossplane/crossplane/v2/apis/pkg/meta/v1alpha1"
	pkgmetav1beta1 "github.com/crossplane/crossplane/v2/apis/pkg/meta/v1beta1"
	admv1 "k8s.io/api/admissionregistration/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

// buildMetaScheme builds the default scheme used for identifying metadata in a
// Crossplane package.
func buildMetaScheme() (*runtime.Scheme, error) {
	metaScheme := runtime.NewScheme()
	if err := pkgmetav1alpha1.SchemeBuilder.AddToScheme(metaScheme); err != nil {
		return nil, err
	}
	if err := pkgmetav1beta1.SchemeBuilder.AddToScheme(metaScheme); err != nil {
		return nil, err
	}
	if err := pkgmetav1.SchemeBuilder.AddToScheme(metaScheme); err != nil {
		return nil, err
	}
	return metaScheme, nil
}

// buildObjectScheme builds the default scheme used for identifying objects in a
// Crossplane package.
func buildObjectScheme() (*runtime.Scheme, error) {
	objScheme := runtime.NewScheme()
	if err := v1.AddToScheme(objScheme); err != nil {
		return nil, err
	}
	if err := v1alpha1.AddToScheme(objScheme); err != nil {
		return nil, err
	}
	if err := opsv1alpha1.AddToScheme(objScheme); err != nil {
		return nil, err
	}
	if err := v2.AddToScheme(objScheme); err != nil {
		return nil, err
	}
	if err := extv1beta1.AddToScheme(objScheme); err != nil {
		return nil, err
	}
	if err := extv1.AddToScheme(objScheme); err != nil {
		return nil, err
	}
	if err := admv1.AddToScheme(objScheme); err != nil {
		return nil, err
	}
	return objScheme, nil
}

func extractObjects(rc io.ReadCloser) ([]runtime.Object, error) {
	metaScheme, err := buildMetaScheme()
	if err != nil {
		return nil, err
	}
	objScheme, err := buildObjectScheme()
	if err != nil {
		return nil, err
	}
	// parse package using Crossplane's parser
	pkg, err := parser.New(metaScheme, objScheme).Parse(context.Background(), rc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse package: %w", err)
	}
	return pkg.GetObjects(), nil
}
