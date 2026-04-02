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
					s.PaddingTop = 30
					s.PaddingRight = 30
					s.PaddingBottom = 30
					s.PaddingLeft = 30
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    30,
					Right:  30,
					Bottom: 30,
					Left:   30,
				},
				ContentX:      30,
				ContentY:      30,
				ContentWidth:  535.28,
				ContentHeight: 345,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BoxShadow = []layouter_domain.BoxShadowValue{
								layouter_domain.BoxShadowValue{
									OffsetX:    3.75,
									OffsetY:    3.75,
									BlurRadius: 7.5,
									Colour: layouter_domain.NewRGBA(0,
										0,
										0,
										0.3),
								},
							}
							s.BackgroundColour = layouter_domain.ColourWhite
							s.Height = layouter_domain.DimensionPt(75)
							s.Width = layouter_domain.DimensionPt(150)
							s.MarginBottom = layouter_domain.DimensionPt(30)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 30,
						},
						ContentX:      30,
						ContentY:      30,
						ContentWidth:  150,
						ContentHeight: 75,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Outer shadow",
								ContentX:      30,
								ContentY:      30,
								ContentWidth:  79.7109375,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BoxShadow = []layouter_domain.BoxShadowValue{
								layouter_domain.BoxShadowValue{
									OffsetX:    2.25,
									OffsetY:    2.25,
									BlurRadius: 6,
									Colour: layouter_domain.NewRGBA(0,
										0,
										0,
										0.4),
									Inset: true,
								},
							}
							s.BackgroundColour = layouter_domain.ColourWhite
							s.Height = layouter_domain.DimensionPt(75)
							s.Width = layouter_domain.DimensionPt(150)
							s.MarginBottom = layouter_domain.DimensionPt(30)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 30,
						},
						ContentX:      30,
						ContentY:      135,
						ContentWidth:  150,
						ContentHeight: 75,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Inset shadow",
								ContentX:      30,
								ContentY:      135,
								ContentWidth:  75.1875,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BoxShadow = []layouter_domain.BoxShadowValue{
								layouter_domain.BoxShadowValue{
									SpreadRadius: 2.25,
									Colour: layouter_domain.NewRGBA(0.20392156862745098,
										0.596078431372549,
										0.8588235294117647,
										1),
								},
							}
							s.BackgroundColour = layouter_domain.ColourWhite
							s.Height = layouter_domain.DimensionPt(75)
							s.Width = layouter_domain.DimensionPt(150)
							s.MarginBottom = layouter_domain.DimensionPt(30)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 30,
						},
						ContentX:      30,
						ContentY:      240,
						ContentWidth:  150,
						ContentHeight: 75,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.LineHeight = 16.799999999999997
								}),
								Text:          "Spread only (no blur)",
								ContentX:      30,
								ContentY:      240,
								ContentWidth:  117.55078125,
								ContentHeight: 16.799999999999997,
							},
						},
					},
				},
			},
		},
	}
}()
