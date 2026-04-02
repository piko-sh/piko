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
					s.LineHeight = 16.799999999999997
					s.BorderSpacing = 1.5
					s.Display = layouter_domain.DisplayTable
				}),
				ContentWidth:  595.28,
				ContentHeight: 41.099999999999994,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTableRow,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayTableRow
						}),
						ContentX:      1.5,
						ContentY:      1.5,
						ContentWidth:  592.28,
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
								ContentWidth:  293.89,
								ContentHeight: 16.799999999999997,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.LineHeight = 16.799999999999997
										}),
										Text:          "Cell 1",
										ContentX:      2.25,
										ContentY:      2.25,
										ContentWidth:  36,
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
								ContentX:      299.14,
								ContentY:      2.25,
								ContentWidth:  293.89,
								ContentHeight: 16.799999999999997,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.LineHeight = 16.799999999999997
										}),
										Text:          "Cell 2",
										ContentX:      299.14,
										ContentY:      2.25,
										ContentWidth:  36,
										ContentHeight: 16.799999999999997,
									},
								},
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTableRow,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayTableRow
						}),
						ContentX:      1.5,
						ContentY:      21.299999999999997,
						ContentWidth:  592.28,
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
								ContentY:      22.049999999999997,
								ContentWidth:  293.89,
								ContentHeight: 16.799999999999997,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.LineHeight = 16.799999999999997
										}),
										Text:          "Cell 3",
										ContentX:      2.25,
										ContentY:      22.049999999999997,
										ContentWidth:  36,
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
								ContentX:      299.14,
								ContentY:      22.049999999999997,
								ContentWidth:  293.89,
								ContentHeight: 16.799999999999997,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.LineHeight = 16.799999999999997
										}),
										Text:          "Cell 4",
										ContentX:      299.14,
										ContentY:      22.049999999999997,
										ContentWidth:  36,
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
