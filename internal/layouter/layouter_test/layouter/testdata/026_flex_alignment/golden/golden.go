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
					s.Height = layouter_domain.DimensionPt(150)
					s.Width = layouter_domain.DimensionPt(300)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayFlex
					s.JustifyContent = layouter_domain.JustifySpaceBetween
					s.AlignItems = layouter_domain.AlignItemsCentre
				}),
				ContentWidth:  300,
				ContentHeight: 150,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxFlexItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(22.5)
							s.Width = layouter_domain.DimensionPt(37.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentY:      63.75,
						ContentWidth:  37.5,
						ContentHeight: 22.5,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Small",
								ContentY:      63.75,
								ContentWidth:  30,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxFlexItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(112.5)
							s.Width = layouter_domain.DimensionPt(37.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentX:      131.25,
						ContentY:      18.75,
						ContentWidth:  37.5,
						ContentHeight: 112.5,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Tall",
								ContentX:      131.25,
								ContentY:      18.75,
								ContentWidth:  24,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxFlexItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(60)
							s.Width = layouter_domain.DimensionPt(37.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.AlignSelf = layouter_domain.AlignSelfFlexEnd
						}),
						ContentX:      262.5,
						ContentY:      90,
						ContentWidth:  37.5,
						ContentHeight: 60,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
									s.AlignSelf = layouter_domain.AlignSelfFlexEnd
								}),
								Text:          "Medium",
								ContentX:      262.5,
								ContentY:      90,
								ContentWidth:  36,
								ContentHeight: 16.799999999999997,
							},
						},
					},
				},
			},
		},
	}
}()
