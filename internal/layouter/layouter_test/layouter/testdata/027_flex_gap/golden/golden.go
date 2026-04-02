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
				Type: layouter_domain.BoxFlex,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.Width = layouter_domain.DimensionPt(225)
					s.LineHeight = 16.799999999999997
					s.RowGap = 7.5
					s.ColumnGap = 15
					s.Display = layouter_domain.DisplayFlex
				}),
				ContentWidth:  225,
				ContentHeight: 30,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxFlexItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(30)
							s.Width = layouter_domain.DimensionPt(45)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentWidth:  45,
						ContentHeight: 30,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "1",
								ContentWidth:  6,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxFlexItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(30)
							s.Width = layouter_domain.DimensionPt(45)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentX:      60,
						ContentWidth:  45,
						ContentHeight: 30,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "2",
								ContentX:      60,
								ContentWidth:  6,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxFlexItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(30)
							s.Width = layouter_domain.DimensionPt(45)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentX:      120,
						ContentWidth:  45,
						ContentHeight: 30,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "3",
								ContentX:      120,
								ContentWidth:  6,
								ContentHeight: 16.799999999999997,
							},
						},
					},
				},
			},
		},
	}
}()
