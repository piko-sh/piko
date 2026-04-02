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
					s.MarginTop = layouter_domain.DimensionPt(12)
					s.MarginBottom = layouter_domain.DimensionPt(12)
					s.PaddingLeft = 30
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
				}),
				Padding: layouter_domain.BoxEdges{
					Left: 30,
				},
				Margin: layouter_domain.BoxEdges{
					Bottom: 12,
				},
				ContentX:      30,
				ContentY:      12,
				ContentWidth:  565.28,
				ContentHeight: 50.39999999999999,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxListItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayListItem
						}),
						ContentX:      30,
						ContentY:      12,
						ContentWidth:  565.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxListMarker,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "• ",
								ContentX:      18,
								ContentY:      12,
								ContentWidth:  12,
								ContentHeight: 16.799999999999997,
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "First item",
								ContentX:      30,
								ContentY:      12,
								ContentWidth:  60,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxListItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayListItem
						}),
						ContentX:      30,
						ContentY:      28.799999999999997,
						ContentWidth:  565.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxListMarker,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "• ",
								ContentX:      18,
								ContentY:      28.799999999999997,
								ContentWidth:  12,
								ContentHeight: 16.799999999999997,
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Second item",
								ContentX:      30,
								ContentY:      28.799999999999997,
								ContentWidth:  66,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxListItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayListItem
						}),
						ContentX:      30,
						ContentY:      45.599999999999994,
						ContentWidth:  565.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxListMarker,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "• ",
								ContentX:      18,
								ContentY:      45.599999999999994,
								ContentWidth:  12,
								ContentHeight: 16.799999999999997,
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Third item",
								ContentX:      30,
								ContentY:      45.599999999999994,
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
