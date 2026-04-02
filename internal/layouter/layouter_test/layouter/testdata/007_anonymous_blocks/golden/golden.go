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
				ContentWidth:  595.28,
				ContentHeight: 74.39999999999999,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxAnonymousBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentWidth:  595.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Text before ",
								ContentWidth:  72,
								ContentHeight: 16.799999999999997,
							},
						},
					},
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
						ContentY:      28.799999999999997,
						ContentWidth:  595.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Block paragraph",
								ContentY:      28.799999999999997,
								ContentWidth:  90,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxAnonymousBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentY:      57.599999999999994,
						ContentWidth:  595.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "text after",
								ContentY:      57.599999999999994,
								ContentWidth:  60,
								ContentHeight: 16.799999999999997,
							},
						},
					},
				},
			},
		},
	}
}()
