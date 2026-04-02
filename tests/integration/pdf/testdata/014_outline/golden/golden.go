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
					s.Width = layouter_domain.DimensionPt(300)
					s.PaddingTop = 22.5
					s.PaddingRight = 22.5
					s.PaddingBottom = 22.5
					s.PaddingLeft = 22.5
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    22.5,
					Right:  22.5,
					Bottom: 22.5,
					Left:   22.5,
				},
				ContentX:      22.5,
				ContentY:      22.5,
				ContentWidth:  255,
				ContentHeight: 142.5,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BackgroundColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.OutlineColour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
							s.Height = layouter_domain.DimensionPt(37.5)
							s.Width = layouter_domain.DimensionPt(112.5)
							s.MarginBottom = layouter_domain.DimensionPt(15)
							s.LineHeight = 16.799999999999997
							s.OutlineWidth = 2.25
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.OutlineStyle = layouter_domain.BorderStyleSolid
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 15,
						},
						ContentX:      22.5,
						ContentY:      22.5,
						ContentWidth:  112.5,
						ContentHeight: 37.5,
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BackgroundColour = layouter_domain.NewRGBA(0.1803921568627451, 0.8, 0.44313725490196076, 1)
							s.OutlineColour = layouter_domain.NewRGBA(0.9529411764705882, 0.611764705882353, 0.07058823529411765, 1)
							s.Height = layouter_domain.DimensionPt(37.5)
							s.Width = layouter_domain.DimensionPt(112.5)
							s.MarginBottom = layouter_domain.DimensionPt(15)
							s.LineHeight = 16.799999999999997
							s.OutlineWidth = 2.25
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.OutlineStyle = layouter_domain.BorderStyleDashed
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 15,
						},
						ContentX:      22.5,
						ContentY:      75,
						ContentWidth:  112.5,
						ContentHeight: 37.5,
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BackgroundColour = layouter_domain.NewRGBA(0.6078431372549019, 0.34901960784313724, 0.7137254901960784, 1)
							s.OutlineColour = layouter_domain.NewRGBA(0.10196078431372549, 0.7372549019607844, 0.611764705882353, 1)
							s.Height = layouter_domain.DimensionPt(37.5)
							s.Width = layouter_domain.DimensionPt(112.5)
							s.LineHeight = 16.799999999999997
							s.OutlineWidth = 2.25
							s.OutlineOffset = 3.75
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.OutlineStyle = layouter_domain.BorderStyleSolid
						}),
						ContentX:      22.5,
						ContentY:      127.5,
						ContentWidth:  112.5,
						ContentHeight: 37.5,
					},
				},
			},
		},
	}
}()
