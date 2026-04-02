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
					s.PaddingTop = 15
					s.PaddingRight = 15
					s.PaddingBottom = 15
					s.PaddingLeft = 15
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    15,
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      15,
				ContentY:      15,
				ContentWidth:  565.28,
				ContentHeight: 83.99999999999999,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						ContentX:      15,
						ContentY:      15,
						ContentWidth:  565.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Normal weight, normal style",
								ContentX:      15,
								ContentY:      15,
								ContentWidth:  160.11328125,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.FontWeight = 700
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						ContentX:      15,
						ContentY:      31.799999999999997,
						ContentWidth:  565.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
									s.FontWeight = 700
								}),
								Text:          "Synthetic bold text",
								ContentX:      15,
								ContentY:      31.799999999999997,
								ContentWidth:  104.765625,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.FontStyle = layouter_domain.FontStyleItalic
						}),
						ContentX:      15,
						ContentY:      48.599999999999994,
						ContentWidth:  565.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
									s.FontStyle = layouter_domain.FontStyleItalic
								}),
								Text:          "Synthetic italic text",
								ContentX:      15,
								ContentY:      48.599999999999994,
								ContentWidth:  105.75,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.FontWeight = 700
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.FontStyle = layouter_domain.FontStyleItalic
						}),
						ContentX:      15,
						ContentY:      65.39999999999999,
						ContentWidth:  565.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
									s.FontWeight = 700
									s.FontStyle = layouter_domain.FontStyleItalic
								}),
								Text:          "Synthetic bold italic text",
								ContentX:      15,
								ContentY:      65.39999999999999,
								ContentWidth:  133.9921875,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.FontWeight = 300
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						ContentX:      15,
						ContentY:      82.19999999999999,
						ContentWidth:  565.28,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
									s.FontWeight = 300
								}),
								Text:          "Light weight (no synthesis, renders as regular)",
								ContentX:      15,
								ContentY:      82.19999999999999,
								ContentWidth:  260.44921875,
								ContentHeight: 16.799999999999997,
							},
						},
					},
				},
			},
		},
	}
}()
