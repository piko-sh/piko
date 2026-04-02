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
					s.ColumnRuleColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
					s.Width = layouter_domain.DimensionPt(300)
					s.PaddingTop = 15
					s.PaddingRight = 15
					s.PaddingBottom = 15
					s.PaddingLeft = 15
					s.FontSize = 9
					s.LineHeight = 12.6
					s.ColumnRuleWidth = 1.5
					s.ColumnCount = 3
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
					s.ColumnRuleStyle = layouter_domain.BorderStyleSolid
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    15,
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      15,
				ContentY:      15,
				ContentWidth:  270,
				ContentHeight: 42,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontSize = 9
							s.LineHeight = 12.6
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						ContentX:      15,
						ContentY:      15,
						ContentWidth:  84,
						ContentHeight: 50.4,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 9
									s.LineHeight = 12.6
								}),
								Text:          "Column one",
								ContentX:      15,
								ContentY:      15,
								ContentWidth:  51.43359375,
								ContentHeight: 12.6,
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 9
									s.LineHeight = 12.6
								}),
								Text:          "content that fills",
								ContentX:      15,
								ContentY:      27.6,
								ContentWidth:  68.63671875,
								ContentHeight: 12.6,
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 9
									s.LineHeight = 12.6
								}),
								Text:          "the first column",
								ContentX:      15,
								ContentY:      40.2,
								ContentWidth:  66.90234375,
								ContentHeight: 12.6,
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 9
									s.LineHeight = 12.6
								}),
								Text:          "with some text.",
								ContentX:      15,
								ContentY:      52.8,
								ContentWidth:  64.7109375,
								ContentHeight: 12.6,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontSize = 9
							s.LineHeight = 12.6
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						ContentX:      108,
						ContentY:      15,
						ContentWidth:  84,
						ContentHeight: 50.4,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 9
									s.LineHeight = 12.6
								}),
								Text:          "Column two",
								ContentX:      108,
								ContentY:      15,
								ContentWidth:  51.1171875,
								ContentHeight: 12.6,
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 9
									s.LineHeight = 12.6
								}),
								Text:          "content that fills",
								ContentX:      108,
								ContentY:      27.6,
								ContentWidth:  68.63671875,
								ContentHeight: 12.6,
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 9
									s.LineHeight = 12.6
								}),
								Text:          "the second column",
								ContentX:      108,
								ContentY:      40.2,
								ContentWidth:  80.47265625,
								ContentHeight: 12.6,
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 9
									s.LineHeight = 12.6
								}),
								Text:          "with text.",
								ContentX:      108,
								ContentY:      52.8,
								ContentWidth:  39.1171875,
								ContentHeight: 12.6,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontSize = 9
							s.LineHeight = 12.6
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						ContentX:      201,
						ContentY:      15,
						ContentWidth:  84,
						ContentHeight: 25.2,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 9
									s.LineHeight = 12.6
								}),
								Text:          "Column three has",
								ContentX:      201,
								ContentY:      15,
								ContentWidth:  75.1171875,
								ContentHeight: 12.6,
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 9
									s.LineHeight = 12.6
								}),
								Text:          "content too.",
								ContentX:      201,
								ContentY:      27.6,
								ContentWidth:  51.375,
								ContentHeight: 12.6,
							},
						},
					},
				},
			},
		},
	}
}()
