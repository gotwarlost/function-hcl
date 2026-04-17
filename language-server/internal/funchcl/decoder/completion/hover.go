package completion

import (
	"fmt"
	"strings"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func (c *Completer) doHover(filename string, pos hcl.Pos) (*lang.HoverData, error) {
	f, err := c.fileByName(filename)
	if err != nil {
		return nil, err
	}
	rootBody, err := c.bodyForFileAndPos(filename, f, pos)
	if err != nil {
		return nil, err
	}
	// log.Printf("using source:\n%s\n", writer.NodeToSource(rootBody))
	data, err := c.hoverAtPos(rootBody, schema.NewBlockStack(), pos)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Completer) hoverAtPos(body *hclsyntax.Body, bs schema.BlockStack, pos hcl.Pos) (*lang.HoverData, error) {
	if body == nil {
		return nil, fmt.Errorf("hoverAtPos: body is nil")
	}
	filename := body.Range().Filename

	// process position inside an attribute
	for name, attr := range body.Attributes {
		if !attr.Range().ContainsPos(pos) {
			continue
		}
		aSchema := c.ctx.AttributeSchema(bs, attr.Name)
		if attr.NameRange.ContainsPos(pos) {
			impliedSchema := c.ctx.ImpliedAttributeSchema(bs, attr.Name)
			if impliedSchema == nil {
				impliedSchema = aSchema
			}
			return &lang.HoverData{
				Content: hoverContentForAttribute(name, impliedSchema),
				Range:   attr.Range(),
			}, nil
		}
		if attr.Expr.Range().ContainsPos(pos) {
			// return newExpr(attr.Expr, 0, aSchema.Constraint).HoverAtPos(c.ctx, pos), nil
			eh := newExpressionHover(c.ctx, pos)
			return eh.hover(aSchema, attr.Expr), nil
		}
	}

	for _, block := range body.Blocks {
		if !block.Range().ContainsPos(pos) {
			continue
		}
		parentSchema := c.ctx.BodySchema(bs)
		bs.Push(block)
		labelSchemas := c.ctx.LabelSchema(bs)
		bodySchema := c.ctx.BodySchema(bs)
		if bodySchema == nil {
			return nil, fmt.Errorf("unknown block type %q", block.Type)
		}
		if block.TypeRange.ContainsPos(pos) {
			blockSchema := parentSchema.NestedBlocks[block.Type]
			if blockSchema == nil {
				return nil, fmt.Errorf("unknown block type %q", bs.Peek(1).Type)
			}
			return &lang.HoverData{
				Content: c.hoverContentForBlock(block.Type, blockSchema),
				Range:   block.TypeRange,
			}, nil
		}
		for i, labelRange := range block.LabelRanges {
			if labelRange.ContainsPos(pos) || posEqual(labelRange.End, pos) {
				if i+1 > len(labelSchemas) {
					return nil, &positionalError{
						filename: filename,
						pos:      pos,
						msg:      fmt.Sprintf("unexpected label (%d) %q", i, block.Labels[i]),
					}
				}
				return &lang.HoverData{
					Content: c.hoverContentForLabel(labelSchemas[i], block.Labels[i]),
					Range:   labelRange,
				}, nil
			}
		}
		if isPosOutsideBody(block, pos) {
			return nil, &positionalError{
				filename: filename,
				pos:      pos,
				msg:      fmt.Sprintf("position outside of %q body", block.Type),
			}
		}
		return c.hoverAtPos(block.Body, bs, pos)
	}

	// Position outside any attribute or block
	return nil, &positionalError{
		filename: filename,
		pos:      pos,
		msg:      "position outside of any attribute name, value or block",
	}
}

func (c *Completer) hoverContentForBlock(bType string, schema *schema.BasicBlockSchema) lang.MarkupContent {
	value := fmt.Sprintf("**%s** _%s_%s", bType, detailForBlock(schema), schema.Description.AsDetail())
	return lang.NewMarkup(lang.MarkdownKind, value)
}

func (c *Completer) hoverContentForLabel(labelSchema *schema.LabelSchema, value string) lang.MarkupContent {
	content := fmt.Sprintf("%q", value)
	if labelSchema.Name != "" {
		content += fmt.Sprintf(" (%s)", labelSchema.Name)
	}
	content = strings.TrimSpace(content)
	content += labelSchema.Description.AsDetail()
	return lang.Markdown(content)
}
