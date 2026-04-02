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
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.FontSize = 24
					s.LineHeight = 33.599999999999994
					s.FontWeight = 700
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      15,
				ContentY:      15,
				ContentWidth:  565.28,
				ContentHeight: 33.599999999999994,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontSize = 24
							s.LineHeight = 33.599999999999994
							s.FontWeight = 700
						}),
						Text:          "Chapter One",
						ContentX:      15,
						ContentY:      15,
						ContentWidth:  151.171875,
						ContentHeight: 33.599999999999994,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      15,
				ContentY:      63.599999999999994,
				ContentWidth:  565.28,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
						}),
						Text:          "Introduction paragraph.",
						ContentX:      15,
						ContentY:      63.599999999999994,
						ContentWidth:  136.0078125,
						ContentHeight: 16.799999999999997,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.FontSize = 18
					s.LineHeight = 25.2
					s.FontWeight = 700
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      15,
				ContentY:      95.39999999999999,
				ContentWidth:  565.28,
				ContentHeight: 25.2,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontSize = 18
							s.LineHeight = 25.2
							s.FontWeight = 700
						}),
						Text:          "Section 1.1",
						ContentX:      15,
						ContentY:      95.39999999999999,
						ContentWidth:  96.43359375,
						ContentHeight: 25.2,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      15,
				ContentY:      135.6,
				ContentWidth:  565.28,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
						}),
						Text:          "First section content.",
						ContentX:      15,
						ContentY:      135.6,
						ContentWidth:  117.50390625,
						ContentHeight: 16.799999999999997,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.FontSize = 18
					s.LineHeight = 25.2
					s.FontWeight = 700
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      15,
				ContentY:      167.39999999999998,
				ContentWidth:  565.28,
				ContentHeight: 25.2,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontSize = 18
							s.LineHeight = 25.2
							s.FontWeight = 700
						}),
						Text:          "Section 1.2",
						ContentX:      15,
						ContentY:      167.39999999999998,
						ContentWidth:  96.43359375,
						ContentHeight: 25.2,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      15,
				ContentY:      207.59999999999997,
				ContentWidth:  565.28,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
						}),
						Text:          "Second section content.",
						ContentX:      15,
						ContentY:      207.59999999999997,
						ContentWidth:  134.34375,
						ContentHeight: 16.799999999999997,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.FontSize = 14.04
					s.LineHeight = 19.656
					s.FontWeight = 700
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      15,
				ContentY:      239.39999999999998,
				ContentWidth:  565.28,
				ContentHeight: 19.656,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontSize = 14.04
							s.LineHeight = 19.656
							s.FontWeight = 700
						}),
						Text:          "Subsection 1.2.1",
						ContentX:      15,
						ContentY:      239.39999999999998,
						ContentWidth:  113.9296875,
						ContentHeight: 19.656,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      15,
				ContentY:      274.056,
				ContentWidth:  565.28,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
						}),
						Text:          "Subsection content.",
						ContentX:      15,
						ContentY:      274.056,
						ContentWidth:  111.421875,
						ContentHeight: 16.799999999999997,
					},
				},
			},
		},
	}
}()
