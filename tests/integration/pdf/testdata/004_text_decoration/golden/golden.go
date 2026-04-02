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
				ContentHeight: 79.19999999999999,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
							s.MarginBottom = layouter_domain.DimensionPt(11.25)
							s.FontSize = 13.5
							s.LineHeight = 18.9
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.TextDecoration = layouter_domain.TextDecorationUnderline
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 11.25,
						},
						ContentX:      15,
						ContentY:      15,
						ContentWidth:  565.28,
						ContentHeight: 18.9,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
									s.FontSize = 13.5
									s.LineHeight = 18.9
									s.TextDecoration = layouter_domain.TextDecorationUnderline
								}),
								Text:          "This text has an underline",
								ContentX:      15,
								ContentY:      15,
								ContentWidth:  163.5,
								ContentHeight: 18.9,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
							s.MarginBottom = layouter_domain.DimensionPt(11.25)
							s.FontSize = 13.5
							s.LineHeight = 18.9
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.TextDecoration = layouter_domain.TextDecorationLineThrough
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 11.25,
						},
						ContentX:      15,
						ContentY:      45.15,
						ContentWidth:  565.28,
						ContentHeight: 18.9,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
									s.FontSize = 13.5
									s.LineHeight = 18.9
									s.TextDecoration = layouter_domain.TextDecorationLineThrough
								}),
								Text:          "This text has a strikethrough",
								ContentX:      15,
								ContentY:      45.15,
								ContentWidth:  180.9609375,
								ContentHeight: 18.9,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.FontSize = 13.5
							s.LineHeight = 18.9
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.TextDecoration = layouter_domain.TextDecorationLineThrough
						}),
						ContentX:      15,
						ContentY:      75.3,
						ContentWidth:  565.28,
						ContentHeight: 18.9,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.FontSize = 13.5
									s.LineHeight = 18.9
									s.TextDecoration = layouter_domain.TextDecorationLineThrough
								}),
								Text:          "This text has both",
								ContentX:      15,
								ContentY:      75.3,
								ContentWidth:  112.65234375,
								ContentHeight: 18.9,
							},
						},
					},
				},
			},
		},
	}
}()
