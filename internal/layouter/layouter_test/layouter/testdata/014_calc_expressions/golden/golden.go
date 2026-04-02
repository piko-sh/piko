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
					s.Width = layouter_domain.DimensionPt(565.28)
					s.PaddingTop = 19.5
					s.PaddingRight = 19.5
					s.PaddingBottom = 19.5
					s.PaddingLeft = 19.5
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    19.5,
					Right:  19.5,
					Bottom: 19.5,
					Left:   19.5,
				},
				ContentX:      19.5,
				ContentY:      19.5,
				ContentWidth:  565.28,
				ContentHeight: 52.8,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginTop = layouter_domain.DimensionPt(12)
							s.MarginRight = layouter_domain.DimensionPt(12)
							s.MarginBottom = layouter_domain.DimensionPt(12)
							s.MarginLeft = layouter_domain.DimensionPt(12)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						Margin: layouter_domain.BoxEdges{
							Right:  12,
							Bottom: 12,
							Left:   12,
						},
						ContentX:      31.5,
						ContentY:      31.5,
						ContentWidth:  541.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Calc content",
								ContentX:      31.5,
								ContentY:      31.5,
								ContentWidth:  72,
								ContentHeight: 16.799999999999997,
							},
						},
					},
				},
			},
		},
	}
}()
