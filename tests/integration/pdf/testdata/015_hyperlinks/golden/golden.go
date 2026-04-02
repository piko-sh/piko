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
					s.FontSize = 10.5
					s.LineHeight = 14.7
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
				ContentHeight: 44.849999999999994,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxAnonymousBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontSize = 10.5
							s.LineHeight = 14.7
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentX:      15,
						ContentY:      15,
						ContentWidth:  270,
						ContentHeight: 14.7,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxInline,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.FontSize = 10.5
									s.LineHeight = 14.7
									s.BoxSizing = border - box
									s.TextDecoration = layouter_domain.TextDecorationUnderline
								}),
								ContentX:      15,
								ContentY:      15,
								ContentWidth:  142.453125,
								ContentHeight: 14.7,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.TextDecoration = layouter_domain.TextDecorationUnderline
										}),
										Text:          "External link to example.com",
										ContentX:      15,
										ContentY:      15,
										ContentWidth:  142.453125,
										ContentHeight: 14.7,
									},
								},
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginTop = layouter_domain.DimensionPt(11.25)
							s.FontSize = 10.5
							s.LineHeight = 14.7
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						ContentX:      15,
						ContentY:      40.95,
						ContentWidth:  270,
						ContentHeight: 18.9,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxInline,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
									s.FontSize = 13.5
									s.LineHeight = 18.9
									s.BoxSizing = border - box
								}),
								ContentX:      15,
								ContentY:      40.95,
								ContentWidth:  92.34375,
								ContentHeight: 18.9,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
											s.FontSize = 13.5
											s.LineHeight = 18.9
										}),
										Text:          "Styled link text",
										ContentX:      15,
										ContentY:      40.95,
										ContentWidth:  92.34375,
										ContentHeight: 18.9,
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
