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
					s.BackgroundColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
					s.Height = layouter_domain.DimensionPt(112.5)
					s.Width = layouter_domain.DimensionPt(225)
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.BorderTopLeftRadius = 11.25
					s.BorderTopRightRadius = 11.25
					s.BorderBottomRightRadius = 11.25
					s.BorderBottomLeftRadius = 11.25
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      15,
				ContentY:      15,
				ContentWidth:  225,
				ContentHeight: 112.5,
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.BackgroundColour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
					s.Height = layouter_domain.DimensionPt(75)
					s.Width = layouter_domain.DimensionPt(150)
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.BorderTopLeftRadius = 22.5
					s.BorderTopRightRadius = 7.5
					s.BorderBottomRightRadius = 22.5
					s.BorderBottomLeftRadius = 7.5
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      15,
				ContentY:      142.5,
				ContentWidth:  150,
				ContentHeight: 75,
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.BorderTopColour = layouter_domain.NewRGBA(0.15294117647058825, 0.6823529411764706, 0.3764705882352941, 1)
					s.BorderRightColour = layouter_domain.NewRGBA(0.15294117647058825, 0.6823529411764706, 0.3764705882352941, 1)
					s.BackgroundColour = layouter_domain.NewRGBA(0.1803921568627451, 0.8, 0.44313725490196076, 1)
					s.BorderBottomColour = layouter_domain.NewRGBA(0.15294117647058825, 0.6823529411764706, 0.3764705882352941, 1)
					s.BorderLeftColour = layouter_domain.NewRGBA(0.15294117647058825, 0.6823529411764706, 0.3764705882352941, 1)
					s.Height = layouter_domain.DimensionPt(60)
					s.Width = layouter_domain.DimensionPt(187.5)
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.BorderTopWidth = 2.25
					s.BorderRightWidth = 2.25
					s.BorderBottomWidth = 2.25
					s.BorderLeftWidth = 2.25
					s.BorderTopLeftRadius = 9
					s.BorderTopRightRadius = 9
					s.BorderBottomRightRadius = 9
					s.BorderBottomLeftRadius = 9
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
					s.BorderTopStyle = layouter_domain.BorderStyleSolid
					s.BorderRightStyle = layouter_domain.BorderStyleSolid
					s.BorderBottomStyle = layouter_domain.BorderStyleSolid
					s.BorderLeftStyle = layouter_domain.BorderStyleSolid
				}),
				Border: layouter_domain.BoxEdges{
					Top:    2.25,
					Right:  2.25,
					Bottom: 2.25,
					Left:   2.25,
				},
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      17.25,
				ContentY:      234.75,
				ContentWidth:  183,
				ContentHeight: 55.5,
			},
		},
	}
}()
