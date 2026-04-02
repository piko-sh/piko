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
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
				}),
				Margin: layouter_domain.BoxEdges{
					Bottom: 12,
				},
				ContentY:      12,
				ContentWidth:  595.28,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginTop = layouter_domain.DimensionPt(12)
							s.MarginBottom = layouter_domain.DimensionPt(12)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 12,
						},
						ContentY:      12,
						ContentWidth:  595.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Variables",
								ContentY:      12,
								ContentWidth:  54,
								ContentHeight: 16.799999999999997,
							},
						},
					},
				},
			},
		},
	}
}()
