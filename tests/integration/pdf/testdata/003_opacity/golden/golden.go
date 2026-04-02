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
					s.Height = layouter_domain.DimensionPt(75)
					s.Width = layouter_domain.DimensionPt(150)
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
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
				ContentWidth:  150,
				ContentHeight: 75,
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.BackgroundColour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
					s.Height = layouter_domain.DimensionPt(75)
					s.Width = layouter_domain.DimensionPt(150)
					s.MarginTop = layouter_domain.DimensionPt(-45)
					s.MarginLeft = layouter_domain.DimensionPt(30)
					s.Opacity = 0.5
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Margin: layouter_domain.BoxEdges{
					Left: 30,
				},
				ContentX:      30,
				ContentY:      60,
				ContentWidth:  150,
				ContentHeight: 75,
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.BackgroundColour = layouter_domain.NewRGBA(0.1803921568627451, 0.8, 0.44313725490196076, 1)
					s.Height = layouter_domain.DimensionPt(60)
					s.Width = layouter_domain.DimensionPt(112.5)
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.Opacity = 0.3
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
				ContentY:      150,
				ContentWidth:  112.5,
				ContentHeight: 60,
			},
		},
	}
}()
