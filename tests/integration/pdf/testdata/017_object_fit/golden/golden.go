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
				ContentHeight: 675,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginBottom = layouter_domain.DimensionPt(11.25)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 11.25,
						},
						ContentX:      15,
						ContentY:      15,
						ContentWidth:  270,
						ContentHeight: 126,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.MarginBottom = layouter_domain.DimensionPt(3)
									s.FontSize = 7.5
									s.LineHeight = 10.5
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 3,
								},
								ContentX:      15,
								ContentY:      15,
								ContentWidth:  270,
								ContentHeight: 10.5,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 7.5
											s.LineHeight = 10.5
										}),
										Text:          "object-fit: fill (default)",
										ContentX:      15,
										ContentY:      15,
										ContentWidth:  74.953125,
										ContentHeight: 10.5,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxReplaced,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.Height = layouter_domain.DimensionPt(112.5)
									s.MaxHeight = layouter_domain.DimensionPct(100)
									s.Width = layouter_domain.DimensionPt(150)
									s.MaxWidth = layouter_domain.DimensionPct(100)
									s.BorderTopWidth = 0.75
									s.BorderRightWidth = 0.75
									s.BorderBottomWidth = 0.75
									s.BorderLeftWidth = 0.75
									s.LineHeight = 16.799999999999997
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.BorderRightStyle = layouter_domain.BorderStyleSolid
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
									s.BorderLeftStyle = layouter_domain.BorderStyleSolid
								}),
								Border: layouter_domain.BoxEdges{
									Top:    0.75,
									Right:  0.75,
									Bottom: 0.75,
									Left:   0.75,
								},
								ContentX:        15.75,
								ContentY:        29.25,
								ContentWidth:    148.5,
								ContentHeight:   111,
								IntrinsicWidth:  100,
								IntrinsicHeight: 100,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginBottom = layouter_domain.DimensionPt(11.25)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 11.25,
						},
						ContentX:      15,
						ContentY:      152.25,
						ContentWidth:  270,
						ContentHeight: 126,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.MarginBottom = layouter_domain.DimensionPt(3)
									s.FontSize = 7.5
									s.LineHeight = 10.5
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 3,
								},
								ContentX:      15,
								ContentY:      152.25,
								ContentWidth:  270,
								ContentHeight: 10.5,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 7.5
											s.LineHeight = 10.5
										}),
										Text:          "object-fit: contain",
										ContentX:      15,
										ContentY:      152.25,
										ContentWidth:  61.4765625,
										ContentHeight: 10.5,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxReplaced,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.Height = layouter_domain.DimensionPt(112.5)
									s.MaxHeight = layouter_domain.DimensionPct(100)
									s.Width = layouter_domain.DimensionPt(150)
									s.MaxWidth = layouter_domain.DimensionPct(100)
									s.BorderTopWidth = 0.75
									s.BorderRightWidth = 0.75
									s.BorderBottomWidth = 0.75
									s.BorderLeftWidth = 0.75
									s.LineHeight = 16.799999999999997
									s.ObjectFit = layouter_domain.ObjectFitContain
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.BorderRightStyle = layouter_domain.BorderStyleSolid
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
									s.BorderLeftStyle = layouter_domain.BorderStyleSolid
								}),
								Border: layouter_domain.BoxEdges{
									Top:    0.75,
									Right:  0.75,
									Bottom: 0.75,
									Left:   0.75,
								},
								ContentX:        15.75,
								ContentY:        166.5,
								ContentWidth:    148.5,
								ContentHeight:   111,
								IntrinsicWidth:  100,
								IntrinsicHeight: 100,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginBottom = layouter_domain.DimensionPt(11.25)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 11.25,
						},
						ContentX:      15,
						ContentY:      289.5,
						ContentWidth:  270,
						ContentHeight: 126,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.MarginBottom = layouter_domain.DimensionPt(3)
									s.FontSize = 7.5
									s.LineHeight = 10.5
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 3,
								},
								ContentX:      15,
								ContentY:      289.5,
								ContentWidth:  270,
								ContentHeight: 10.5,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 7.5
											s.LineHeight = 10.5
										}),
										Text:          "object-fit: cover",
										ContentX:      15,
										ContentY:      289.5,
										ContentWidth:  54.33984375,
										ContentHeight: 10.5,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxReplaced,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.Height = layouter_domain.DimensionPt(112.5)
									s.MaxHeight = layouter_domain.DimensionPct(100)
									s.Width = layouter_domain.DimensionPt(150)
									s.MaxWidth = layouter_domain.DimensionPct(100)
									s.BorderTopWidth = 0.75
									s.BorderRightWidth = 0.75
									s.BorderBottomWidth = 0.75
									s.BorderLeftWidth = 0.75
									s.LineHeight = 16.799999999999997
									s.ObjectFit = layouter_domain.ObjectFitCover
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.BorderRightStyle = layouter_domain.BorderStyleSolid
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
									s.BorderLeftStyle = layouter_domain.BorderStyleSolid
								}),
								Border: layouter_domain.BoxEdges{
									Top:    0.75,
									Right:  0.75,
									Bottom: 0.75,
									Left:   0.75,
								},
								ContentX:        15.75,
								ContentY:        303.75,
								ContentWidth:    148.5,
								ContentHeight:   111,
								IntrinsicWidth:  100,
								IntrinsicHeight: 100,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.MarginBottom = layouter_domain.DimensionPt(11.25)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 11.25,
						},
						ContentX:      15,
						ContentY:      426.75,
						ContentWidth:  270,
						ContentHeight: 126,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.MarginBottom = layouter_domain.DimensionPt(3)
									s.FontSize = 7.5
									s.LineHeight = 10.5
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 3,
								},
								ContentX:      15,
								ContentY:      426.75,
								ContentWidth:  270,
								ContentHeight: 10.5,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 7.5
											s.LineHeight = 10.5
										}),
										Text:          "object-fit: none",
										ContentX:      15,
										ContentY:      426.75,
										ContentWidth:  53.26171875,
										ContentHeight: 10.5,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxReplaced,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.Height = layouter_domain.DimensionPt(112.5)
									s.MaxHeight = layouter_domain.DimensionPct(100)
									s.Width = layouter_domain.DimensionPt(150)
									s.MaxWidth = layouter_domain.DimensionPct(100)
									s.BorderTopWidth = 0.75
									s.BorderRightWidth = 0.75
									s.BorderBottomWidth = 0.75
									s.BorderLeftWidth = 0.75
									s.LineHeight = 16.799999999999997
									s.ObjectFit = layouter_domain.ObjectFitNone
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.BorderRightStyle = layouter_domain.BorderStyleSolid
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
									s.BorderLeftStyle = layouter_domain.BorderStyleSolid
								}),
								Border: layouter_domain.BoxEdges{
									Top:    0.75,
									Right:  0.75,
									Bottom: 0.75,
									Left:   0.75,
								},
								ContentX:        15.75,
								ContentY:        441,
								ContentWidth:    148.5,
								ContentHeight:   111,
								IntrinsicWidth:  100,
								IntrinsicHeight: 100,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						ContentX:      15,
						ContentY:      564,
						ContentWidth:  270,
						ContentHeight: 126,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.MarginBottom = layouter_domain.DimensionPt(3)
									s.FontSize = 7.5
									s.LineHeight = 10.5
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 3,
								},
								ContentX:      15,
								ContentY:      564,
								ContentWidth:  270,
								ContentHeight: 10.5,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontSize = 7.5
											s.LineHeight = 10.5
										}),
										Text:          "object-fit: scale-down",
										ContentX:      15,
										ContentY:      564,
										ContentWidth:  74.7421875,
										ContentHeight: 10.5,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxReplaced,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.8, 0.8, 0.8, 1)
									s.Height = layouter_domain.DimensionPt(112.5)
									s.MaxHeight = layouter_domain.DimensionPct(100)
									s.Width = layouter_domain.DimensionPt(150)
									s.MaxWidth = layouter_domain.DimensionPct(100)
									s.BorderTopWidth = 0.75
									s.BorderRightWidth = 0.75
									s.BorderBottomWidth = 0.75
									s.BorderLeftWidth = 0.75
									s.LineHeight = 16.799999999999997
									s.ObjectFit = layouter_domain.ObjectFitScaleDown
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.BorderRightStyle = layouter_domain.BorderStyleSolid
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
									s.BorderLeftStyle = layouter_domain.BorderStyleSolid
								}),
								Border: layouter_domain.BoxEdges{
									Top:    0.75,
									Right:  0.75,
									Bottom: 0.75,
									Left:   0.75,
								},
								ContentX:        15.75,
								ContentY:        578.25,
								ContentWidth:    148.5,
								ContentHeight:   111,
								IntrinsicWidth:  100,
								IntrinsicHeight: 100,
							},
						},
					},
				},
			},
		},
	}
}()
