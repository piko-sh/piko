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
				ContentHeight: 277.5,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BackgroundColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.Height = layouter_domain.DimensionPt(60)
							s.Width = layouter_domain.DimensionPt(150)
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      15,
						ContentWidth:  150,
						ContentHeight: 60,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Visible box",
								ContentX:      15,
								ContentY:      15,
								ContentWidth:  60.2578125,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BackgroundColour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
							s.Height = layouter_domain.DimensionPt(60)
							s.Width = layouter_domain.DimensionPt(150)
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.LineHeight = 16.799999999999997
							s.Visibility = layouter_domain.VisibilityHidden
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      82.5,
						ContentWidth:  150,
						ContentHeight: 60,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
									s.Visibility = layouter_domain.VisibilityHidden
								}),
								Text:          "Hidden box",
								ContentX:      15,
								ContentY:      82.5,
								ContentWidth:  64.828125,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BackgroundColour = layouter_domain.NewRGBA(0.1803921568627451, 0.8, 0.44313725490196076, 1)
							s.Height = layouter_domain.DimensionPt(60)
							s.Width = layouter_domain.DimensionPt(150)
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      150,
						ContentWidth:  150,
						ContentHeight: 60,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Visible after hidden",
								ContentX:      15,
								ContentY:      150,
								ContentWidth:  108.99609375,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.LineHeight = 16.799999999999997
							s.Visibility = layouter_domain.VisibilityHidden
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      217.5,
						ContentWidth:  565.28,
						ContentHeight: 60,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BackgroundColour = layouter_domain.NewRGBA(0.6078431372549019, 0.34901960784313724, 0.7137254901960784, 1)
									s.Height = layouter_domain.DimensionPt(60)
									s.Width = layouter_domain.DimensionPt(150)
									s.LineHeight = 16.799999999999997
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								ContentX:      15,
								ContentY:      217.5,
								ContentWidth:  150,
								ContentHeight: 60,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.LineHeight = 16.799999999999997
										}),
										Text:          "Visible child of hidden",
										ContentX:      15,
										ContentY:      217.5,
										ContentWidth:  123.33984375,
										ContentHeight: 16.799999999999997,
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.LineHeight = 16.799999999999997
										}),
										Text:          "parent",
										ContentX:      15,
										ContentY:      234.3,
										ContentWidth:  37.359375,
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
