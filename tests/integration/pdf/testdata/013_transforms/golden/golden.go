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
					s.Width = layouter_domain.DimensionPt(375)
					s.PaddingTop = 30
					s.PaddingRight = 30
					s.PaddingBottom = 30
					s.PaddingLeft = 30
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    30,
					Right:  30,
					Bottom: 30,
					Left:   30,
				},
				ContentX:      30,
				ContentY:      30,
				ContentWidth:  315,
				ContentHeight: 277.5,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.TransformValue = "rotate(15deg)"
							s.BackgroundColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.Height = layouter_domain.DimensionPt(37.5)
							s.Width = layouter_domain.DimensionPt(75)
							s.MarginBottom = layouter_domain.DimensionPt(22.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.HasTransform = true
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 22.5,
						},
						ContentX:      30,
						ContentY:      30,
						ContentWidth:  75,
						ContentHeight: 37.5,
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.TransformValue = "scale(0.8)"
							s.BackgroundColour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
							s.Height = layouter_domain.DimensionPt(37.5)
							s.Width = layouter_domain.DimensionPt(75)
							s.MarginBottom = layouter_domain.DimensionPt(22.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.HasTransform = true
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 22.5,
						},
						ContentX:      30,
						ContentY:      90,
						ContentWidth:  75,
						ContentHeight: 37.5,
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.TransformValue = "translate(20px, 10px)"
							s.BackgroundColour = layouter_domain.NewRGBA(0.1803921568627451, 0.8, 0.44313725490196076, 1)
							s.Height = layouter_domain.DimensionPt(37.5)
							s.Width = layouter_domain.DimensionPt(75)
							s.MarginBottom = layouter_domain.DimensionPt(22.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.HasTransform = true
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 22.5,
						},
						ContentX:      30,
						ContentY:      150,
						ContentWidth:  75,
						ContentHeight: 37.5,
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.TransformValue = "rotate(10deg) scale(1.2)"
							s.BackgroundColour = layouter_domain.NewRGBA(0.9529411764705882, 0.611764705882353, 0.07058823529411765, 1)
							s.Height = layouter_domain.DimensionPt(37.5)
							s.Width = layouter_domain.DimensionPt(75)
							s.MarginBottom = layouter_domain.DimensionPt(22.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.HasTransform = true
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 22.5,
						},
						ContentX:      30,
						ContentY:      210,
						ContentWidth:  75,
						ContentHeight: 37.5,
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.TransformOrigin = "left top"
							s.TransformValue = "rotate(30deg)"
							s.BackgroundColour = layouter_domain.NewRGBA(0.6078431372549019, 0.34901960784313724, 0.7137254901960784, 1)
							s.Height = layouter_domain.DimensionPt(37.5)
							s.Width = layouter_domain.DimensionPt(75)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.HasTransform = true
						}),
						ContentX:      30,
						ContentY:      270,
						ContentWidth:  75,
						ContentHeight: 37.5,
					},
				},
			},
		},
	}
}()
