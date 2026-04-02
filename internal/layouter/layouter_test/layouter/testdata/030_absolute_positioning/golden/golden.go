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
					s.Height = layouter_domain.DimensionPt(225)
					s.Width = layouter_domain.DimensionPt(300)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.Position = layouter_domain.PositionRelative
				}),
				ContentY:      12,
				ContentWidth:  300,
				ContentHeight: 225,
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
						ContentWidth:  300,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Normal flow content",
								ContentY:      12,
								ContentWidth:  114,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Width = layouter_domain.DimensionPt(75)
							s.Right = layouter_domain.DimensionPt(7.5)
							s.Top = layouter_domain.DimensionPt(15)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.Position = layouter_domain.PositionAbsolute
						}),
						ContentX:      217.5,
						ContentY:      27,
						ContentWidth:  75,
						ContentHeight: 33.599999999999994,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Right = layouter_domain.DimensionPt(7.5)
									s.Top = layouter_domain.DimensionPt(15)
									s.LineHeight = 16.799999999999997
									s.Position = layouter_domain.PositionAbsolute
								}),
								Text:          "Absolutely",
								ContentX:      527.78,
								ContentY:      15,
								ContentWidth:  60,
								ContentHeight: 16.799999999999997,
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Right = layouter_domain.DimensionPt(7.5)
									s.Top = layouter_domain.DimensionPt(15)
									s.LineHeight = 16.799999999999997
									s.Position = layouter_domain.PositionAbsolute
								}),
								Text:          "positioned",
								ContentX:      527.78,
								ContentY:      15,
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
