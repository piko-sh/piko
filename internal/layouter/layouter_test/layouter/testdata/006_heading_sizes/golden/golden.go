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
					s.MarginTop = layouter_domain.DimensionPt(8.040000000000001)
					s.MarginBottom = layouter_domain.DimensionPt(8.040000000000001)
					s.FontSize = 24
					s.LineHeight = 33.599999999999994
					s.FontWeight = 700
					s.Display = layouter_domain.DisplayBlock
				}),
				Margin: layouter_domain.BoxEdges{
					Bottom: 8.040000000000001,
				},
				ContentY:      8.040000000000001,
				ContentWidth:  595.28,
				ContentHeight: 33.599999999999994,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontSize = 24
							s.LineHeight = 33.599999999999994
							s.FontWeight = 700
						}),
						Text:          "Title",
						ContentY:      8.040000000000001,
						ContentWidth:  60,
						ContentHeight: 33.599999999999994,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.MarginTop = layouter_domain.DimensionPt(9.959999999999999)
					s.MarginBottom = layouter_domain.DimensionPt(9.959999999999999)
					s.FontSize = 18
					s.LineHeight = 25.2
					s.FontWeight = 700
					s.Display = layouter_domain.DisplayBlock
				}),
				Margin: layouter_domain.BoxEdges{
					Bottom: 9.959999999999999,
				},
				ContentY:      51.599999999999994,
				ContentWidth:  595.28,
				ContentHeight: 25.2,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontSize = 18
							s.LineHeight = 25.2
							s.FontWeight = 700
						}),
						Text:          "Subtitle",
						ContentY:      51.599999999999994,
						ContentWidth:  72,
						ContentHeight: 25.2,
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
				ContentY:      88.8,
				ContentWidth:  595.28,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
						}),
						Text:          "Body text",
						ContentY:      88.8,
						ContentWidth:  54,
						ContentHeight: 16.799999999999997,
					},
				},
			},
		},
	}
}()
