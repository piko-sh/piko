package test

import "piko.sh/piko/internal/layouter/layouter_domain"

var GeneratedLayoutBox = func() *layouter_domain.LayoutBox {
	withStyle := func(overrides func(*layouter_domain.ComputedStyle)) layouter_domain.ComputedStyle {
		style := layouter_domain.DefaultComputedStyle()
		overrides(&style)
		return style
	}
	_ = withStyle
	return &layouter_domain.LayoutBox{
		Type: layouter_domain.BoxBlock,
		Style: withStyle(func(s *layouter_domain.ComputedStyle) {
			s.Width = layouter_domain.DimensionPt(595.28)
			s.Display = layouter_domain.DisplayBlock
			s.OverflowX = layouter_domain.OverflowHidden
			s.OverflowY = layouter_domain.OverflowHidden
		}),
		ContentWidth:  595.28,
		ContentHeight: 841.89,
		Children: []*layouter_domain.LayoutBox{
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.BgClip = "padding-box"
					s.BorderTopColour = layouter_domain.NewRGBA(0, 0, 0, 0.3)
					s.BorderRightColour = layouter_domain.NewRGBA(0, 0, 0, 0.3)
					s.BackgroundColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
					s.BorderBottomColour = layouter_domain.NewRGBA(0, 0, 0, 0.3)
					s.BorderLeftColour = layouter_domain.NewRGBA(0, 0, 0, 0.3)
					s.Height = layouter_domain.DimensionPt(75)
					s.Width = layouter_domain.DimensionPt(150)
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.PaddingTop = 15
					s.PaddingRight = 15
					s.PaddingBottom = 15
					s.PaddingLeft = 15
					s.BorderTopWidth = 7.5
					s.BorderRightWidth = 7.5
					s.BorderBottomWidth = 7.5
					s.BorderLeftWidth = 7.5
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
					s.BorderTopStyle = layouter_domain.BorderStyleSolid
					s.BorderRightStyle = layouter_domain.BorderStyleSolid
					s.BorderBottomStyle = layouter_domain.BorderStyleSolid
					s.BorderLeftStyle = layouter_domain.BorderStyleSolid
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    15,
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				Border: layouter_domain.BoxEdges{
					Top:    7.5,
					Right:  7.5,
					Bottom: 7.5,
					Left:   7.5,
				},
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      37.5,
				ContentY:      37.5,
				ContentWidth:  105,
				ContentHeight: 30,
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.BgClip = "content-box"
					s.BorderTopColour = layouter_domain.NewRGBA(0, 0, 0, 0.3)
					s.BorderRightColour = layouter_domain.NewRGBA(0, 0, 0, 0.3)
					s.BackgroundColour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
					s.BorderBottomColour = layouter_domain.NewRGBA(0, 0, 0, 0.3)
					s.BorderLeftColour = layouter_domain.NewRGBA(0, 0, 0, 0.3)
					s.Height = layouter_domain.DimensionPt(75)
					s.Width = layouter_domain.DimensionPt(150)
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.PaddingTop = 15
					s.PaddingRight = 15
					s.PaddingBottom = 15
					s.PaddingLeft = 15
					s.BorderTopWidth = 7.5
					s.BorderRightWidth = 7.5
					s.BorderBottomWidth = 7.5
					s.BorderLeftWidth = 7.5
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
					s.BorderTopStyle = layouter_domain.BorderStyleSolid
					s.BorderRightStyle = layouter_domain.BorderStyleSolid
					s.BorderBottomStyle = layouter_domain.BorderStyleSolid
					s.BorderLeftStyle = layouter_domain.BorderStyleSolid
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    15,
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				Border: layouter_domain.BoxEdges{
					Top:    7.5,
					Right:  7.5,
					Bottom: 7.5,
					Left:   7.5,
				},
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      37.5,
				ContentY:      127.5,
				ContentWidth:  105,
				ContentHeight: 30,
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.BgClip = "border-box"
					s.BorderTopColour = layouter_domain.NewRGBA(0, 0, 0, 0.3)
					s.BorderRightColour = layouter_domain.NewRGBA(0, 0, 0, 0.3)
					s.BackgroundColour = layouter_domain.NewRGBA(0.1803921568627451, 0.8, 0.44313725490196076, 1)
					s.BorderBottomColour = layouter_domain.NewRGBA(0, 0, 0, 0.3)
					s.BorderLeftColour = layouter_domain.NewRGBA(0, 0, 0, 0.3)
					s.Height = layouter_domain.DimensionPt(75)
					s.Width = layouter_domain.DimensionPt(150)
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.PaddingTop = 15
					s.PaddingRight = 15
					s.PaddingBottom = 15
					s.PaddingLeft = 15
					s.BorderTopWidth = 7.5
					s.BorderRightWidth = 7.5
					s.BorderBottomWidth = 7.5
					s.BorderLeftWidth = 7.5
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
					s.BorderTopStyle = layouter_domain.BorderStyleSolid
					s.BorderRightStyle = layouter_domain.BorderStyleSolid
					s.BorderBottomStyle = layouter_domain.BorderStyleSolid
					s.BorderLeftStyle = layouter_domain.BorderStyleSolid
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    15,
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				Border: layouter_domain.BoxEdges{
					Top:    7.5,
					Right:  7.5,
					Bottom: 7.5,
					Left:   7.5,
				},
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      37.5,
				ContentY:      217.5,
				ContentWidth:  105,
				ContentHeight: 30,
			},
		},
	}
}()
