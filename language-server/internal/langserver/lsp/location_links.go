// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"path/filepath"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/utils/uri"
	"github.com/hashicorp/hcl/v2"
)

func ToLocationLink(path lang.Path, rng hcl.Range) lsp.LocationLink {
	targetUri := uri.FromPath(filepath.Join(path.Path, rng.Filename))
	lspRange := HCLRangeToLSP(rng)

	locLink := lsp.LocationLink{
		OriginSelectionRange: &lspRange,
		TargetURI:            lsp.DocumentURI(targetUri),
		TargetRange:          lspRange,
		TargetSelectionRange: lspRange,
	}
	return locLink
}

func ToLocationLinks(path lang.Path, rng []hcl.Range) []lsp.LocationLink {
	var ret []lsp.LocationLink
	for _, r := range rng {
		ret = append(ret, ToLocationLink(path, r))
	}
	return ret
}

func ToLocations(path lang.Path, rng []hcl.Range) []lsp.Location {
	var ret []lsp.Location
	for _, r := range rng {
		ret = append(ret, ToLocation(path, r))
	}
	return ret
}

func ToLocation(path lang.Path, rng hcl.Range) lsp.Location {
	targetUri := uri.FromPath(filepath.Join(path.Path, rng.Filename))
	lspRange := HCLRangeToLSP(rng)

	locLink := lsp.Location{
		URI:   lsp.DocumentURI(targetUri),
		Range: lspRange,
	}
	return locLink
}
