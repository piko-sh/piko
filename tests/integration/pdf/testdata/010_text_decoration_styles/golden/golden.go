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
					s.Width = layouter_domain.DimensionPt(300)
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
				ContentWidth:  270,
				ContentHeight: 138.29999999999998,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.TextDecoration = layouter_domain.TextDecorationUnderline
							s.TextDecorationStyle = dashed
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      15,
						ContentWidth:  270,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
									s.TextDecoration = layouter_domain.TextDecorationUnderline
									s.TextDecorationStyle = dashed
								}),
								Text:          "Dashed underline",
								ContentX:      15,
								ContentY:      15,
								ContentWidth:  100.25390625,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.TextDecoration = layouter_domain.TextDecorationUnderline
							s.TextDecorationStyle = dotted
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      39.3,
						ContentWidth:  270,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
									s.TextDecoration = layouter_domain.TextDecorationUnderline
									s.TextDecorationStyle = dotted
								}),
								Text:          "Dotted underline",
								ContentX:      15,
								ContentY:      39.3,
								ContentWidth:  96.3046875,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.TextDecoration = layouter_domain.TextDecorationUnderline
							s.TextDecorationStyle = double
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      63.599999999999994,
						ContentWidth:  270,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
									s.TextDecoration = layouter_domain.TextDecorationUnderline
									s.TextDecorationStyle = double
								}),
								Text:          "Double underline",
								ContentX:      15,
								ContentY:      63.599999999999994,
								ContentWidth:  98.14453125,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.TextDecoration = layouter_domain.TextDecorationUnderline
							s.TextDecorationStyle = wavy
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      87.89999999999999,
						ContentWidth:  270,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
									s.TextDecoration = layouter_domain.TextDecorationUnderline
									s.TextDecorationStyle = wavy
								}),
								Text:          "Wavy underline",
								ContentX:      15,
								ContentY:      87.89999999999999,
								ContentWidth:  87.3046875,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.TextDecorationColour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.TextDecoration = layouter_domain.TextDecorationUnderline
							s.TextDecorationColourSet = true
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      112.19999999999999,
						ContentWidth:  270,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.TextDecorationColour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
									s.LineHeight = 16.799999999999997
									s.TextDecoration = layouter_domain.TextDecorationUnderline
									s.TextDecorationColourSet = true
								}),
								Text:          "Coloured underline",
								ContentX:      15,
								ContentY:      112.19999999999999,
								ContentWidth:  108.94921875,
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
							s.TextDecoration = layouter_domain.TextDecorationLineThrough
							s.TextDecorationStyle = dashed
						}),
						ContentX:      15,
						ContentY:      136.5,
						ContentWidth:  270,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
									s.TextDecoration = layouter_domain.TextDecorationLineThrough
									s.TextDecorationStyle = dashed
								}),
								Text:          "Dashed line-through",
								ContentX:      15,
								ContentY:      136.5,
								ContentWidth:  116.1328125,
								ContentHeight: 16.799999999999997,
							},
						},
					},
				},
			},
		},
	}
}()
