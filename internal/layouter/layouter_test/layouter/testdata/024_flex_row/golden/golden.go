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
					s.Display = layouter_domain.DisplayFlex
				}),
				ContentWidth:  225,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxFlexItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FlexBasis = layouter_domain.DimensionPt(0)
							s.LineHeight = 16.799999999999997
							s.FlexGrow = 1
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentWidth:  56.25,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FlexBasis = layouter_domain.DimensionPt(0)
									s.LineHeight = 16.799999999999997
									s.FlexGrow = 1
								}),
								Text:          "A",
								ContentWidth:  6,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxFlexItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FlexBasis = layouter_domain.DimensionPt(0)
							s.LineHeight = 16.799999999999997
							s.FlexGrow = 2
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentX:      56.25,
						ContentWidth:  112.5,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FlexBasis = layouter_domain.DimensionPt(0)
									s.LineHeight = 16.799999999999997
									s.FlexGrow = 2
								}),
								Text:          "B",
								ContentX:      56.25,
								ContentWidth:  6,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxFlexItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FlexBasis = layouter_domain.DimensionPt(0)
							s.LineHeight = 16.799999999999997
							s.FlexGrow = 1
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentX:      168.75,
						ContentWidth:  56.25,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FlexBasis = layouter_domain.DimensionPt(0)
									s.LineHeight = 16.799999999999997
									s.FlexGrow = 1
								}),
								Text:          "C",
								ContentX:      168.75,
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
