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
					s.Height = layouter_domain.DimensionPt(252.567)
					s.Width = layouter_domain.DimensionPt(297.64)
					s.FontSize = 17.8584
					s.LineHeight = 25.001759999999997
					s.Display = layouter_domain.DisplayBlock
				}),
				ContentWidth:  297.64,
				ContentHeight: 252.567,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontSize = 17.8584
							s.LineHeight = 25.001759999999997
						}),
						Text:          "Viewport-relative sizing ",
						ContentWidth:  223.23,
						ContentHeight: 25.001759999999997,
					},
				},
			},
		},
	}
}()
