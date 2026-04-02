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
					s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
					s.Width = layouter_domain.DimensionPt(375)
					s.PaddingTop = 15
					s.PaddingRight = 15
					s.PaddingBottom = 15
					s.PaddingLeft = 15
					s.FontSize = 9
					s.LineHeight = 12.6
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
				ContentWidth:  345,
				ContentHeight: 408.90000000000003,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
							s.MarginBottom = layouter_domain.DimensionPt(12)
							s.FontSize = 13.5
							s.LineHeight = 18.9
							s.FontWeight = 700
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 12,
						},
						ContentX:      15,
						ContentY:      15,
						ContentWidth:  345,
						ContentHeight: 18.9,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.FontSize = 13.5
									s.LineHeight = 18.9
									s.FontWeight = 700
								}),
								Text:          "Registration Form",
								ContentX:      15,
								ContentY:      15,
								ContentWidth:  121.62890625,
								ContentHeight: 18.9,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.FontSize = 9
							s.LineHeight = 12.6
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      45.9,
						ContentWidth:  345,
						ContentHeight: 44.1,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.MarginBottom = layouter_domain.DimensionPt(3)
									s.FontSize = 9
									s.LineHeight = 12.6
									s.FontWeight = 700
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 3,
								},
								ContentX:      15,
								ContentY:      45.9,
								ContentWidth:  345,
								ContentHeight: 12.6,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.FontWeight = 700
										}),
										Text:          "Full Name",
										ContentX:      15,
										ContentY:      45.9,
										ContentWidth:  45.3515625,
										ContentHeight: 12.6,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxReplaced,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.Width = layouter_domain.DimensionPt(187.5)
									s.PaddingTop = 3
									s.PaddingRight = 3
									s.PaddingBottom = 3
									s.PaddingLeft = 3
									s.BorderTopWidth = 0.75
									s.BorderRightWidth = 0.75
									s.BorderBottomWidth = 0.75
									s.BorderLeftWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.BorderRightStyle = layouter_domain.BorderStyleSolid
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
									s.BorderLeftStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    3,
									Right:  3,
									Bottom: 3,
									Left:   3,
								},
								Border: layouter_domain.BoxEdges{
									Top:    0.75,
									Right:  0.75,
									Bottom: 0.75,
									Left:   0.75,
								},
								ContentX:        18.75,
								ContentY:        65.25,
								ContentWidth:    180,
								ContentHeight:   21,
								IntrinsicWidth:  127.5,
								IntrinsicHeight: 21,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.FontSize = 9
							s.LineHeight = 12.6
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      97.5,
						ContentWidth:  345,
						ContentHeight: 44.1,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.MarginBottom = layouter_domain.DimensionPt(3)
									s.FontSize = 9
									s.LineHeight = 12.6
									s.FontWeight = 700
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 3,
								},
								ContentX:      15,
								ContentY:      97.5,
								ContentWidth:  345,
								ContentHeight: 12.6,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.FontWeight = 700
										}),
										Text:          "Password",
										ContentX:      15,
										ContentY:      97.5,
										ContentWidth:  42.87890625,
										ContentHeight: 12.6,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxReplaced,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.Width = layouter_domain.DimensionPt(187.5)
									s.PaddingTop = 3
									s.PaddingRight = 3
									s.PaddingBottom = 3
									s.PaddingLeft = 3
									s.BorderTopWidth = 0.75
									s.BorderRightWidth = 0.75
									s.BorderBottomWidth = 0.75
									s.BorderLeftWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.BorderRightStyle = layouter_domain.BorderStyleSolid
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
									s.BorderLeftStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    3,
									Right:  3,
									Bottom: 3,
									Left:   3,
								},
								Border: layouter_domain.BoxEdges{
									Top:    0.75,
									Right:  0.75,
									Bottom: 0.75,
									Left:   0.75,
								},
								ContentX:        18.75,
								ContentY:        116.85,
								ContentWidth:    180,
								ContentHeight:   21,
								IntrinsicWidth:  127.5,
								IntrinsicHeight: 21,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.FontSize = 9
							s.LineHeight = 12.6
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      149.1,
						ContentWidth:  345,
						ContentHeight: 44.1,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.MarginBottom = layouter_domain.DimensionPt(3)
									s.FontSize = 9
									s.LineHeight = 12.6
									s.FontWeight = 700
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 3,
								},
								ContentX:      15,
								ContentY:      149.1,
								ContentWidth:  345,
								ContentHeight: 12.6,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.FontWeight = 700
										}),
										Text:          "Email",
										ContentX:      15,
										ContentY:      149.1,
										ContentWidth:  24.5859375,
										ContentHeight: 12.6,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxReplaced,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BackgroundColour = layouter_domain.NewRGBA(0.9607843137254902, 0.9607843137254902, 0.9607843137254902, 1)
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.Width = layouter_domain.DimensionPt(187.5)
									s.PaddingTop = 3
									s.PaddingRight = 3
									s.PaddingBottom = 3
									s.PaddingLeft = 3
									s.BorderTopWidth = 0.75
									s.BorderRightWidth = 0.75
									s.BorderBottomWidth = 0.75
									s.BorderLeftWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.BorderRightStyle = layouter_domain.BorderStyleSolid
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
									s.BorderLeftStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    3,
									Right:  3,
									Bottom: 3,
									Left:   3,
								},
								Border: layouter_domain.BoxEdges{
									Top:    0.75,
									Right:  0.75,
									Bottom: 0.75,
									Left:   0.75,
								},
								ContentX:        18.75,
								ContentY:        168.45,
								ContentWidth:    180,
								ContentHeight:   21,
								IntrinsicWidth:  127.5,
								IntrinsicHeight: 21,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.FontSize = 9
							s.LineHeight = 12.6
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      200.7,
						ContentWidth:  345,
						ContentHeight: 53.1,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.MarginBottom = layouter_domain.DimensionPt(3)
									s.FontSize = 9
									s.LineHeight = 12.6
									s.FontWeight = 700
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 3,
								},
								ContentX:      15,
								ContentY:      200.7,
								ContentWidth:  345,
								ContentHeight: 12.6,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.FontWeight = 700
										}),
										Text:          "Comments",
										ContentX:      15,
										ContentY:      200.7,
										ContentWidth:  48.59765625,
										ContentHeight: 12.6,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxReplaced,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.Height = layouter_domain.DimensionPt(37.5)
									s.Width = layouter_domain.DimensionPt(262.5)
									s.PaddingTop = 3
									s.PaddingRight = 3
									s.PaddingBottom = 3
									s.PaddingLeft = 3
									s.BorderTopWidth = 0.75
									s.BorderRightWidth = 0.75
									s.BorderBottomWidth = 0.75
									s.BorderLeftWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.BorderRightStyle = layouter_domain.BorderStyleSolid
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
									s.BorderLeftStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    3,
									Right:  3,
									Bottom: 3,
									Left:   3,
								},
								Border: layouter_domain.BoxEdges{
									Top:    0.75,
									Right:  0.75,
									Bottom: 0.75,
									Left:   0.75,
								},
								ContentX:        18.75,
								ContentY:        220.04999999999998,
								ContentWidth:    255,
								ContentHeight:   30,
								IntrinsicWidth:  120,
								IntrinsicHeight: 42,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.BorderTopColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
											s.BorderRightColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.BorderBottomColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.BorderTopStyle = layouter_domain.BorderStyleSolid
											s.BorderRightStyle = layouter_domain.BorderStyleSolid
											s.BorderBottomStyle = layouter_domain.BorderStyleSolid
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
										}),
										Text:          "Initial comments here",
										ContentX:      18.75,
										ContentY:      220.04999999999998,
										ContentWidth:  92.61328125,
										ContentHeight: 12.6,
									},
								},
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.FontSize = 9
							s.LineHeight = 12.6
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      261.29999999999995,
						ContentWidth:  345,
						ContentHeight: 44.1,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.MarginBottom = layouter_domain.DimensionPt(3)
									s.FontSize = 9
									s.LineHeight = 12.6
									s.FontWeight = 700
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 3,
								},
								ContentX:      15,
								ContentY:      261.29999999999995,
								ContentWidth:  345,
								ContentHeight: 12.6,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.FontWeight = 700
										}),
										Text:          "Country",
										ContentX:      15,
										ContentY:      261.29999999999995,
										ContentWidth:  36.1640625,
										ContentHeight: 12.6,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxReplaced,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.6, 0.6, 0.6, 1)
									s.Width = layouter_domain.DimensionPt(150)
									s.PaddingTop = 3
									s.PaddingRight = 3
									s.PaddingBottom = 3
									s.PaddingLeft = 3
									s.BorderTopWidth = 0.75
									s.BorderRightWidth = 0.75
									s.BorderBottomWidth = 0.75
									s.BorderLeftWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.BorderRightStyle = layouter_domain.BorderStyleSolid
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
									s.BorderLeftStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    3,
									Right:  3,
									Bottom: 3,
									Left:   3,
								},
								Border: layouter_domain.BoxEdges{
									Top:    0.75,
									Right:  0.75,
									Bottom: 0.75,
									Left:   0.75,
								},
								ContentX:        18.75,
								ContentY:        280.65,
								ContentWidth:    142.5,
								ContentHeight:   21,
								IntrinsicWidth:  127.5,
								IntrinsicHeight: 21,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.FontSize = 9
							s.LineHeight = 12.6
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      312.9,
						ContentWidth:  345,
						ContentHeight: 12.6,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxReplaced,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.MarginRight = layouter_domain.DimensionPt(3)
									s.FontSize = 9
									s.LineHeight = 12.6
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Right: 3,
								},
								ContentX:        15,
								ContentY:        312.9,
								ContentWidth:    10.5,
								ContentHeight:   10.5,
								IntrinsicWidth:  10.5,
								IntrinsicHeight: 10.5,
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxInline,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.FontSize = 9
									s.LineHeight = 12.6
									s.BoxSizing = border - box
								}),
								ContentX:      28.5,
								ContentY:      312.9,
								ContentWidth:  148.95703125,
								ContentHeight: 12.6,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
										}),
										Text:          "I agree to the terms and conditions",
										ContentX:      28.5,
										ContentY:      312.9,
										ContentWidth:  148.95703125,
										ContentHeight: 12.6,
									},
								},
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.FontSize = 9
							s.LineHeight = 12.6
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      333,
						ContentWidth:  345,
						ContentHeight: 12.6,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxReplaced,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.MarginRight = layouter_domain.DimensionPt(3)
									s.FontSize = 9
									s.LineHeight = 12.6
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Right: 3,
								},
								ContentX:        15,
								ContentY:        333,
								ContentWidth:    10.5,
								ContentHeight:   10.5,
								IntrinsicWidth:  10.5,
								IntrinsicHeight: 10.5,
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxInline,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.FontSize = 9
									s.LineHeight = 12.6
									s.BoxSizing = border - box
								}),
								ContentX:      28.5,
								ContentY:      333,
								ContentWidth:  99.234375,
								ContentHeight: 12.6,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
										}),
										Text:          "Subscribe to newsletter",
										ContentX:      28.5,
										ContentY:      333,
										ContentWidth:  99.234375,
										ContentHeight: 12.6,
									},
								},
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
							s.MarginBottom = layouter_domain.DimensionPt(7.5)
							s.FontSize = 9
							s.LineHeight = 12.6
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 7.5,
						},
						ContentX:      15,
						ContentY:      353.1,
						ContentWidth:  345,
						ContentHeight: 28.2,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.MarginBottom = layouter_domain.DimensionPt(3)
									s.FontSize = 9
									s.LineHeight = 12.6
									s.FontWeight = 700
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 3,
								},
								ContentX:      15,
								ContentY:      353.1,
								ContentWidth:  345,
								ContentHeight: 12.6,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.FontWeight = 700
										}),
										Text:          "Plan",
										ContentX:      15,
										ContentY:      353.1,
										ContentWidth:  19.51171875,
										ContentHeight: 12.6,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								ContentX:      15,
								ContentY:      368.70000000000005,
								ContentWidth:  345,
								ContentHeight: 12.6,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxReplaced,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.MarginRight = layouter_domain.DimensionPt(3)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Right: 3,
										},
										ContentX:        15,
										ContentY:        368.70000000000005,
										ContentWidth:    10.5,
										ContentHeight:   10.5,
										IntrinsicWidth:  10.5,
										IntrinsicHeight: 10.5,
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxInline,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.MarginRight = layouter_domain.DimensionPt(9)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Right: 9,
										},
										ContentX:      28.5,
										ContentY:      368.70000000000005,
										ContentWidth:  18.36328125,
										ContentHeight: 12.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          "Free",
												ContentX:      28.5,
												ContentY:      368.70000000000005,
												ContentWidth:  18.36328125,
												ContentHeight: 12.6,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxReplaced,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.MarginRight = layouter_domain.DimensionPt(3)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Right: 3,
										},
										ContentX:        55.86328125,
										ContentY:        368.70000000000005,
										ContentWidth:    10.5,
										ContentHeight:   10.5,
										IntrinsicWidth:  10.5,
										IntrinsicHeight: 10.5,
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxInline,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.MarginRight = layouter_domain.DimensionPt(9)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Right: 9,
										},
										ContentX:      69.36328125,
										ContentY:      368.70000000000005,
										ContentWidth:  14.4375,
										ContentHeight: 12.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          "Pro",
												ContentX:      69.36328125,
												ContentY:      368.70000000000005,
												ContentWidth:  14.4375,
												ContentHeight: 12.6,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxReplaced,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.MarginRight = layouter_domain.DimensionPt(3)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Right: 3,
										},
										ContentX:        92.80078125,
										ContentY:        368.70000000000005,
										ContentWidth:    10.5,
										ContentHeight:   10.5,
										IntrinsicWidth:  10.5,
										IntrinsicHeight: 10.5,
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxInline,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.BoxSizing = border - box
										}),
										ContentX:      106.30078125,
										ContentY:      368.70000000000005,
										ContentWidth:  43.55859375,
										ContentHeight: 12.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          "Enterprise",
												ContentX:      106.30078125,
												ContentY:      368.70000000000005,
												ContentWidth:  43.55859375,
												ContentHeight: 12.6,
											},
										},
									},
								},
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxReplaced,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
							s.FontSize = 9
							s.LineHeight = 12.6
							s.BoxSizing = border - box
						}),
						ContentX:     15,
						ContentY:     388.8,
						ContentWidth: 345,
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
							s.MarginTop = layouter_domain.DimensionPt(12)
							s.FontSize = 9
							s.LineHeight = 12.6
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						ContentX:      15,
						ContentY:      400.8,
						ContentWidth:  345,
						ContentHeight: 23.1,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxReplaced,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.BackgroundColour = layouter_domain.NewRGBA(0.26666666666666666, 0.4666666666666667, 0.6666666666666666, 1)
									s.Colour = layouter_domain.ColourWhite
									s.BorderBottomColour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
									s.PaddingTop = 4.5
									s.PaddingRight = 12
									s.PaddingBottom = 4.5
									s.PaddingLeft = 12
									s.BorderTopWidth = 0.75
									s.BorderRightWidth = 0.75
									s.BorderBottomWidth = 0.75
									s.BorderLeftWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.BorderRightStyle = layouter_domain.BorderStyleSolid
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
									s.BorderLeftStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    4.5,
									Right:  12,
									Bottom: 4.5,
									Left:   12,
								},
								Border: layouter_domain.BoxEdges{
									Top:    0.75,
									Right:  0.75,
									Bottom: 0.75,
									Left:   0.75,
								},
								ContentX:        27.75,
								ContentY:        406.05,
								ContentWidth:    37.5,
								ContentHeight:   12.6,
								IntrinsicWidth:  37.5,
								IntrinsicHeight: 21,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.BorderTopColour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.BorderRightColour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.Colour = layouter_domain.ColourWhite
											s.BorderBottomColour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.2, 0.2, 0.2, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.BorderTopStyle = layouter_domain.BorderStyleSolid
											s.BorderRightStyle = layouter_domain.BorderStyleSolid
											s.BorderBottomStyle = layouter_domain.BorderStyleSolid
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
										}),
										Text:          "Submit",
										ContentX:      27.75,
										ContentY:      406.05,
										ContentWidth:  30.0234375,
										ContentHeight: 12.6,
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
