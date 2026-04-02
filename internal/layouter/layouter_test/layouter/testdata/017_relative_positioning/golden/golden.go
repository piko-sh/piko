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
				ContentHeight: 74.39999999999999,
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
								Text:          "Normal flow",
								ContentY:      12,
								ContentWidth:  66,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Top = layouter_domain.DimensionPt(7.5)
							s.MarginTop = layouter_domain.DimensionPt(12)
							s.MarginBottom = layouter_domain.DimensionPt(12)
							s.Left = layouter_domain.DimensionPt(15)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.Position = layouter_domain.PositionRelative
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 12,
						},
						ContentX:      15,
						ContentY:      48.3,
						ContentWidth:  595.28,
						ContentHeight: 16.799999999999997,
						OffsetX:       15,
						OffsetY:       7.5,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Top = layouter_domain.DimensionPt(7.5)
									s.Left = layouter_domain.DimensionPt(15)
									s.LineHeight = 16.799999999999997
									s.Position = layouter_domain.PositionRelative
								}),
								Text:          "Offset paragraph",
								ContentX:      30,
								ContentY:      55.8,
								ContentWidth:  96,
								ContentHeight: 16.799999999999997,
								OffsetX:       15,
								OffsetY:       7.5,
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
						ContentY:      69.6,
						ContentWidth:  595.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Unaffected by offset",
								ContentY:      69.6,
								ContentWidth:  120,
								ContentHeight: 16.799999999999997,
							},
						},
					},
				},
			},
		},
	}
}()
