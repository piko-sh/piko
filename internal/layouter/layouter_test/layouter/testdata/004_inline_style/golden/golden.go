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
					s.Width = layouter_domain.DimensionPt(150)
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.PaddingTop = 7.5
					s.PaddingRight = 7.5
					s.PaddingBottom = 7.5
					s.PaddingLeft = 7.5
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    7.5,
					Right:  7.5,
					Bottom: 7.5,
					Left:   7.5,
				},
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      22.5,
				ContentY:      22.5,
				ContentWidth:  150,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
						}),
						Text:          "Content",
						ContentX:      22.5,
						ContentY:      22.5,
						ContentWidth:  42,
						ContentHeight: 16.799999999999997,
					},
				},
			},
		},
	}
}()
