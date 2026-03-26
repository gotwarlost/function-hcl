// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"bytes"
	"log"
	"math"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/document/source"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang/semtok"
)

type TokenEncoder struct {
	types               map[semtok.TokenType]int
	mods                map[semtok.TokenModifier]int
	lines               []source.Line
	tokens              []semtok.SemanticToken
	lastEncodedTokenIdx int
}

func NewTokenEncoder(tokens []semtok.SemanticToken, lines []source.Line) *TokenEncoder {
	types := map[semtok.TokenType]int{}
	mods := map[semtok.TokenModifier]int{}
	for i, tt := range TokenTypesLegend() {
		types[tt] = i
	}
	for i, tm := range TokenModifiersLegend() {
		mods[tm] = i
	}
	return &TokenEncoder{
		types:  types,
		mods:   mods,
		tokens: tokens,
		lines:  lines,
	}
}

func (te *TokenEncoder) Encode() []uint32 {
	var data []uint32
	for i := range te.tokens {
		data = append(data, te.encodeTokenOfIndex(i)...)
	}
	return data
}

func computeBitmask(mapping map[semtok.TokenModifier]int, values semtok.TokenModifiers) int {
	bitMask := 0b0
	for _, modifier := range values {
		index, ok := mapping[modifier]
		if !ok {
			log.Println("no mapping for token modifier:", modifier)
		}
		bitMask |= int(math.Pow(2, float64(index)))
	}
	return bitMask
}

func (te *TokenEncoder) encodeTokenOfIndex(i int) []uint32 {
	var data []uint32

	token := te.tokens[i]
	tokenType := token.Type
	tokenTypeIdx, ok := te.types[tokenType]
	if !ok {
		log.Println("no token type index for:", tokenType)
		return data
	}
	modifierBitMask := computeBitmask(te.mods, token.Modifiers)

	// Client may not support multiline tokens which would be indicated
	// via lsp.SemanticTokensCapabilities.MultilineTokenSupport
	// once it becomes available in gopls LSP structs.
	//
	// For now, we just safely assume client does *not* support it.

	tokenLineDelta := token.Range.End.Line - token.Range.Start.Line

	previousLine := 0
	previousStartChar := 0
	if i > 0 {
		previousLine = te.tokens[te.lastEncodedTokenIdx].Range.End.Line - 1
		currentLine := te.tokens[i].Range.End.Line - 1
		if currentLine == previousLine {
			previousStartChar = te.tokens[te.lastEncodedTokenIdx].Range.Start.Column - 1
		}
	}

	if tokenLineDelta == 0 {
		deltaLine := token.Range.Start.Line - 1 - previousLine
		tokenLength := token.Range.End.Byte - token.Range.Start.Byte
		deltaStartChar := token.Range.Start.Column - 1 - previousStartChar

		data = append(data, []uint32{
			uint32(deltaLine),
			uint32(deltaStartChar),
			uint32(tokenLength),
			uint32(tokenTypeIdx),
			uint32(modifierBitMask),
		}...)
	} else {
		// Add entry for each line of a multiline token
		for tokenLine := token.Range.Start.Line - 1; tokenLine <= token.Range.End.Line-1; tokenLine++ {
			deltaLine := tokenLine - previousLine

			deltaStartChar := 0
			if tokenLine == token.Range.Start.Line-1 {
				deltaStartChar = token.Range.Start.Column - 1 - previousStartChar
			}

			lineBytes := bytes.TrimRight(te.lines[tokenLine].Bytes, "\n\r")
			length := len(lineBytes)

			if tokenLine == token.Range.End.Line-1 {
				length = token.Range.End.Column - 1
			}

			data = append(data, []uint32{
				uint32(deltaLine),
				uint32(deltaStartChar),
				uint32(length),
				uint32(tokenTypeIdx),
				uint32(modifierBitMask),
			}...)

			previousLine = tokenLine
		}
	}

	te.lastEncodedTokenIdx = i
	return data
}
