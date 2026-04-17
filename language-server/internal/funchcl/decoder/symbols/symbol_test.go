// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package symbols

var (
	_ Symbol = &AttributeSymbol{}
	_ Symbol = &BlockSymbol{}
	_ Symbol = &ExprSymbol{}
)
