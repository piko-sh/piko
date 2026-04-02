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
				ContentHeight: 151.35,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginBottom = layouter_domain.DimensionPt(11.25)
							s.FontSize = 10.5
							s.LineHeight = 14.7
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 11.25,
						},
						ContentX:      15,
						ContentY:      15,
						ContentWidth:  270,
						ContentHeight: 29.4,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxListItem,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 10.5
									s.LineHeight = 14.7
									s.Display = layouter_domain.DisplayListItem
									s.BoxSizing = border - box
								}),
								ContentX:      15,
								ContentY:      15,
								ContentWidth:  270,
								ContentHeight: 14.7,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxListMarker,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.BoxSizing = border - box
										}),
										Text:          "• ",
										ContentX:      8.3203125,
										ContentY:      15,
										ContentWidth:  6.6796875,
										ContentHeight: 14.7,
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
										}),
										Text:          "Disc item one",
										ContentX:      15,
										ContentY:      15,
										ContentWidth:  66.890625,
										ContentHeight: 14.7,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxListItem,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 10.5
									s.LineHeight = 14.7
									s.Display = layouter_domain.DisplayListItem
									s.BoxSizing = border - box
								}),
								ContentX:      15,
								ContentY:      29.7,
								ContentWidth:  270,
								ContentHeight: 14.7,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxListMarker,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.BoxSizing = border - box
										}),
										Text:          "• ",
										ContentX:      8.3203125,
										ContentY:      29.7,
										ContentWidth:  6.6796875,
										ContentHeight: 14.7,
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
										}),
										Text:          "Disc item two",
										ContentX:      15,
										ContentY:      29.7,
										ContentWidth:  66.515625,
										ContentHeight: 14.7,
									},
								},
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginBottom = layouter_domain.DimensionPt(11.25)
							s.FontSize = 10.5
							s.LineHeight = 14.7
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.ListStyleType = layouter_domain.ListStyleTypeCircle
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 11.25,
						},
						ContentX:      15,
						ContentY:      55.65,
						ContentWidth:  270,
						ContentHeight: 29.4,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxListItem,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 10.5
									s.LineHeight = 14.7
									s.Display = layouter_domain.DisplayListItem
									s.BoxSizing = border - box
									s.ListStyleType = layouter_domain.ListStyleTypeCircle
								}),
								ContentX:      15,
								ContentY:      55.65,
								ContentWidth:  270,
								ContentHeight: 14.7,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxListMarker,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.BoxSizing = border - box
											s.ListStyleType = layouter_domain.ListStyleTypeCircle
										}),
										Text:          "◦ ",
										ContentX:      5.96484375,
										ContentY:      55.65,
										ContentWidth:  9.03515625,
										ContentHeight: 14.7,
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.ListStyleType = layouter_domain.ListStyleTypeCircle
										}),
										Text:          "Circle item one",
										ContentX:      15,
										ContentY:      55.65,
										ContentWidth:  73.59375,
										ContentHeight: 14.7,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxListItem,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 10.5
									s.LineHeight = 14.7
									s.Display = layouter_domain.DisplayListItem
									s.BoxSizing = border - box
									s.ListStyleType = layouter_domain.ListStyleTypeCircle
								}),
								ContentX:      15,
								ContentY:      70.35,
								ContentWidth:  270,
								ContentHeight: 14.7,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxListMarker,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.BoxSizing = border - box
											s.ListStyleType = layouter_domain.ListStyleTypeCircle
										}),
										Text:          "◦ ",
										ContentX:      5.96484375,
										ContentY:      70.35,
										ContentWidth:  9.03515625,
										ContentHeight: 14.7,
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.ListStyleType = layouter_domain.ListStyleTypeCircle
										}),
										Text:          "Circle item two",
										ContentX:      15,
										ContentY:      70.35,
										ContentWidth:  73.21875,
										ContentHeight: 14.7,
									},
								},
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginBottom = layouter_domain.DimensionPt(11.25)
							s.FontSize = 10.5
							s.LineHeight = 14.7
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.ListStyleType = layouter_domain.ListStyleTypeSquare
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 11.25,
						},
						ContentX:      15,
						ContentY:      96.3,
						ContentWidth:  270,
						ContentHeight: 29.4,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxListItem,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 10.5
									s.LineHeight = 14.7
									s.Display = layouter_domain.DisplayListItem
									s.BoxSizing = border - box
									s.ListStyleType = layouter_domain.ListStyleTypeSquare
								}),
								ContentX:      15,
								ContentY:      96.3,
								ContentWidth:  270,
								ContentHeight: 14.7,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxListMarker,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.BoxSizing = border - box
											s.ListStyleType = layouter_domain.ListStyleTypeSquare
										}),
										Text:          "▪ ",
										ContentX:      5.96484375,
										ContentY:      96.3,
										ContentWidth:  9.03515625,
										ContentHeight: 14.7,
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.ListStyleType = layouter_domain.ListStyleTypeSquare
										}),
										Text:          "Square item one",
										ContentX:      15,
										ContentY:      96.3,
										ContentWidth:  81.1171875,
										ContentHeight: 14.7,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxListItem,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 10.5
									s.LineHeight = 14.7
									s.Display = layouter_domain.DisplayListItem
									s.BoxSizing = border - box
									s.ListStyleType = layouter_domain.ListStyleTypeSquare
								}),
								ContentX:      15,
								ContentY:      111,
								ContentWidth:  270,
								ContentHeight: 14.7,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxListMarker,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.BoxSizing = border - box
											s.ListStyleType = layouter_domain.ListStyleTypeSquare
										}),
										Text:          "▪ ",
										ContentX:      5.96484375,
										ContentY:      111,
										ContentWidth:  9.03515625,
										ContentHeight: 14.7,
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.ListStyleType = layouter_domain.ListStyleTypeSquare
										}),
										Text:          "Square item two",
										ContentX:      15,
										ContentY:      111,
										ContentWidth:  80.7421875,
										ContentHeight: 14.7,
									},
								},
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontSize = 10.5
							s.LineHeight = 14.7
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.ListStyleType = layouter_domain.ListStyleTypeDecimal
						}),
						ContentX:      15,
						ContentY:      136.95,
						ContentWidth:  270,
						ContentHeight: 29.4,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxListItem,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 10.5
									s.LineHeight = 14.7
									s.Display = layouter_domain.DisplayListItem
									s.BoxSizing = border - box
									s.ListStyleType = layouter_domain.ListStyleTypeDecimal
								}),
								ContentX:      15,
								ContentY:      136.95,
								ContentWidth:  270,
								ContentHeight: 14.7,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxListMarker,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.BoxSizing = border - box
											s.ListStyleType = layouter_domain.ListStyleTypeDecimal
										}),
										Text:          "1. ",
										ContentX:      3.4453125,
										ContentY:      136.95,
										ContentWidth:  11.5546875,
										ContentHeight: 14.7,
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.ListStyleType = layouter_domain.ListStyleTypeDecimal
										}),
										Text:          "Decimal item one",
										ContentX:      15,
										ContentY:      136.95,
										ContentWidth:  86.203125,
										ContentHeight: 14.7,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxListItem,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontSize = 10.5
									s.LineHeight = 14.7
									s.Display = layouter_domain.DisplayListItem
									s.BoxSizing = border - box
									s.ListStyleType = layouter_domain.ListStyleTypeDecimal
								}),
								ContentX:      15,
								ContentY:      151.64999999999998,
								ContentWidth:  270,
								ContentHeight: 14.7,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxListMarker,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.BoxSizing = border - box
											s.ListStyleType = layouter_domain.ListStyleTypeDecimal
										}),
										Text:          "2. ",
										ContentX:      3.4453125,
										ContentY:      151.64999999999998,
										ContentWidth:  11.5546875,
										ContentHeight: 14.7,
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 10.5
											s.LineHeight = 14.7
											s.ListStyleType = layouter_domain.ListStyleTypeDecimal
										}),
										Text:          "Decimal item two",
										ContentX:      15,
										ContentY:      151.64999999999998,
										ContentWidth:  85.828125,
										ContentHeight: 14.7,
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
