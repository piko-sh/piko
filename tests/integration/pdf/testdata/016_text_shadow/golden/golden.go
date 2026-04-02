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
					s.BackgroundColour = layouter_domain.ColourWhite
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
				ContentHeight: 160.8,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.TextShadow = []layouter_domain.TextShadowValue{
								layouter_domain.TextShadowValue{
									OffsetX: 1.5,
									OffsetY: 1.5,
									Colour: layouter_domain.NewRGBA(0.8,
										0.8,
										0.8,
										1),
								},
							}
							s.MarginBottom = layouter_domain.DimensionPt(15)
							s.FontSize = 18
							s.LineHeight = 25.2
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 15,
						},
						ContentX:      15,
						ContentY:      15,
						ContentWidth:  270,
						ContentHeight: 25.2,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.TextShadow = []layouter_domain.TextShadowValue{
										layouter_domain.TextShadowValue{
											OffsetX: 1.5,
											OffsetY: 1.5,
											Colour: layouter_domain.NewRGBA(0.8,
												0.8,
												0.8,
												1),
										},
									}
									s.FontSize = 18
									s.LineHeight = 25.2
								}),
								Text:          "Sharp shadow",
								ContentX:      15,
								ContentY:      15,
								ContentWidth:  119.8828125,
								ContentHeight: 25.2,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.TextShadow = []layouter_domain.TextShadowValue{
								layouter_domain.TextShadowValue{
									OffsetX:    1.5,
									OffsetY:    1.5,
									BlurRadius: 3,
									Colour: layouter_domain.NewRGBA(0.6,
										0.6,
										0.6,
										1),
								},
							}
							s.MarginBottom = layouter_domain.DimensionPt(15)
							s.FontSize = 18
							s.LineHeight = 25.2
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 15,
						},
						ContentX:      15,
						ContentY:      55.2,
						ContentWidth:  270,
						ContentHeight: 25.2,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.TextShadow = []layouter_domain.TextShadowValue{
										layouter_domain.TextShadowValue{
											OffsetX:    1.5,
											OffsetY:    1.5,
											BlurRadius: 3,
											Colour: layouter_domain.NewRGBA(0.6,
												0.6,
												0.6,
												1),
										},
									}
									s.FontSize = 18
									s.LineHeight = 25.2
								}),
								Text:          "Blurred shadow",
								ContentX:      15,
								ContentY:      55.2,
								ContentWidth:  133.46484375,
								ContentHeight: 25.2,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.TextShadow = []layouter_domain.TextShadowValue{
								layouter_domain.TextShadowValue{
									OffsetX: 0.75,
									OffsetY: 0.75,
									Colour: layouter_domain.NewRGBA(1,
										0,
										0,
										1),
								},
								layouter_domain.TextShadowValue{
									OffsetX: -0.75,
									OffsetY: -0.75,
									Colour: layouter_domain.NewRGBA(0,
										0,
										1,
										1),
								},
							}
							s.MarginBottom = layouter_domain.DimensionPt(15)
							s.FontSize = 18
							s.LineHeight = 25.2
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 15,
						},
						ContentX:      15,
						ContentY:      95.4,
						ContentWidth:  270,
						ContentHeight: 25.2,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.TextShadow = []layouter_domain.TextShadowValue{
										layouter_domain.TextShadowValue{
											OffsetX: 0.75,
											OffsetY: 0.75,
											Colour: layouter_domain.NewRGBA(1,
												0,
												0,
												1),
										},
										layouter_domain.TextShadowValue{
											OffsetX: -0.75,
											OffsetY: -0.75,
											Colour: layouter_domain.NewRGBA(0,
												0,
												1,
												1),
										},
									}
									s.FontSize = 18
									s.LineHeight = 25.2
								}),
								Text:          "Multiple shadows",
								ContentX:      15,
								ContentY:      95.4,
								ContentWidth:  147.984375,
								ContentHeight: 25.2,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.TextShadow = []layouter_domain.TextShadowValue{
								layouter_domain.TextShadowValue{
									BlurRadius: 6,
									Colour: layouter_domain.NewRGBA(0,
										1,
										0,
										1),
								},
							}
							s.BackgroundColour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
							s.Colour = layouter_domain.ColourWhite
							s.PaddingTop = 7.5
							s.PaddingRight = 7.5
							s.PaddingBottom = 7.5
							s.PaddingLeft = 7.5
							s.FontSize = 18
							s.LineHeight = 25.2
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Padding: layouter_domain.BoxEdges{
							Top:    7.5,
							Right:  7.5,
							Bottom: 7.5,
							Left:   7.5,
						},
						ContentX:      22.5,
						ContentY:      143.10000000000002,
						ContentWidth:  255,
						ContentHeight: 25.2,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.TextShadow = []layouter_domain.TextShadowValue{
										layouter_domain.TextShadowValue{
											BlurRadius: 6,
											Colour: layouter_domain.NewRGBA(0,
												1,
												0,
												1),
										},
									}
									s.Colour = layouter_domain.ColourWhite
									s.FontSize = 18
									s.LineHeight = 25.2
								}),
								Text:          "Glow effect",
								ContentX:      22.5,
								ContentY:      143.10000000000002,
								ContentWidth:  94.91015625,
								ContentHeight: 25.2,
							},
						},
					},
				},
			},
		},
	}
}()
