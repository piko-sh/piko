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
				Type: layouter_domain.BoxTable,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.Width = layouter_domain.DimensionPt(225)
					s.LineHeight = 16.799999999999997
					s.BorderSpacing = 1.5
					s.Display = layouter_domain.DisplayTable
					s.TableLayout = layouter_domain.TableLayoutFixed
				}),
				ContentWidth:  225,
				ContentHeight: 21.299999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTableRow,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayTableRow
						}),
						ContentX:      1.5,
						ContentY:      1.5,
						ContentWidth:  222,
						ContentHeight: 18.299999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTableCell,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.PaddingTop = 0.75
									s.PaddingRight = 0.75
									s.PaddingBottom = 0.75
									s.PaddingLeft = 0.75
									s.LineHeight = 16.799999999999997
									s.Display = layouter_domain.DisplayTableCell
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    0.75,
									Right:  0.75,
									Bottom: 0.75,
									Left:   0.75,
								},
								ContentX:      2.25,
								ContentY:      2.25,
								ContentWidth:  108.75,
								ContentHeight: 16.799999999999997,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.LineHeight = 16.799999999999997
										}),
										Text:          "Wide content",
										ContentX:      2.25,
										ContentY:      2.25,
										ContentWidth:  72,
										ContentHeight: 16.799999999999997,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTableCell,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.PaddingTop = 0.75
									s.PaddingRight = 0.75
									s.PaddingBottom = 0.75
									s.PaddingLeft = 0.75
									s.LineHeight = 16.799999999999997
									s.Display = layouter_domain.DisplayTableCell
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    0.75,
									Right:  0.75,
									Bottom: 0.75,
									Left:   0.75,
								},
								ContentX:      114,
								ContentY:      2.25,
								ContentWidth:  108.75,
								ContentHeight: 16.799999999999997,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.LineHeight = 16.799999999999997
										}),
										Text:          "Short",
										ContentX:      114,
										ContentY:      2.25,
										ContentWidth:  30,
										ContentHeight: 16.799999999999997,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}()
