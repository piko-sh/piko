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
			s.Width = layouter_domain.DimensionPt(510.24062993)
			s.Display = layouter_domain.DisplayBlock
			s.OverflowX = layouter_domain.OverflowHidden
			s.OverflowY = layouter_domain.OverflowHidden
		}),
		ContentWidth:  510.24062993,
		ContentHeight: 762.5199212680001,
		Children: []*layouter_domain.LayoutBox{
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
					s.FontSize = 9.5
					s.LineHeight = 13.775
					s.LineHeightFactor = 1.45
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
					s.LineHeightAuto = false
				}),
				ContentWidth:  510.24062993,
				ContentHeight: 1238.775,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
							s.FontSize = 9.5
							s.LineHeight = 13.775
							s.LineHeightFactor = 1.45
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.LineHeightAuto = false
						}),
						Margin: layouter_domain.BoxEdges{
							Bottom: 10,
						},
						ContentWidth:  510.24062993,
						ContentHeight: 47.675,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
									s.FontSize = 22
									s.LineHeight = 31.9
									s.LineHeightFactor = 1.45
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.PageBreakAfter = layouter_domain.PageBreakAvoid
									s.PageBreakInside = layouter_domain.PageBreakAvoid
									s.LineHeightAuto = false
								}),
								ContentWidth:  510.24062993,
								ContentHeight: 31.9,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.FontSize = 22
											s.LineHeight = 31.9
											s.LineHeightFactor = 1.45
											s.PageBreakAfter = layouter_domain.PageBreakAvoid
											s.PageBreakInside = layouter_domain.PageBreakAvoid
											s.LineHeightAuto = false
										}),
										Text:          "Alex Example",
										ContentWidth:  140.58984375,
										ContentHeight: 31.9,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
									s.MarginTop = layouter_domain.DimensionPt(2)
									s.MarginBottom = layouter_domain.DimensionPt(10)
									s.FontSize = 9.5
									s.LineHeight = 13.775
									s.LineHeightFactor = 1.45
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.LineHeightAuto = false
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 10,
								},
								ContentY:      33.9,
								ContentWidth:  510.24062993,
								ContentHeight: 13.775,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
											s.FontSize = 9.5
											s.LineHeight = 13.775
											s.LineHeightFactor = 1.45
											s.LineHeightAuto = false
										}),
										Text:          "Platform engineer / Go / Kubernetes",
										ContentY:      33.9,
										ContentWidth:  166.59375,
										ContentHeight: 13.775,
									},
								},
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
							s.FontSize = 9.5
							s.LineHeight = 13.775
							s.LineHeightFactor = 1.45
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.LineHeightAuto = false
						}),
						ContentY:      61.675,
						ContentWidth:  510.24062993,
						ContentHeight: 57,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.788235294117647, 0.807843137254902, 0.8392156862745098, 1)
									s.MarginTop = layouter_domain.DimensionPt(14)
									s.MarginBottom = layouter_domain.DimensionPt(6)
									s.PaddingBottom = 3
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 13.049999999999999
									s.LineHeightFactor = 1.45
									s.LetterSpacing = 0.76
									s.FontWeight = 700
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.TextTransform = layouter_domain.TextTransformUppercase
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
									s.PageBreakAfter = layouter_domain.PageBreakAvoid
									s.PageBreakInside = layouter_domain.PageBreakAvoid
									s.LineHeightAuto = false
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 3,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 6,
								},
								ContentY:      61.675,
								ContentWidth:  510.24062993,
								ContentHeight: 13.049999999999999,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderBottomColour = layouter_domain.NewRGBA(0.788235294117647, 0.807843137254902, 0.8392156862745098, 1)
											s.FontSize = 9
											s.LineHeight = 13.049999999999999
											s.LineHeightFactor = 1.45
											s.LetterSpacing = 0.76
											s.FontWeight = 700
											s.TextTransform = layouter_domain.TextTransformUppercase
											s.BorderBottomStyle = layouter_domain.BorderStyleSolid
											s.PageBreakAfter = layouter_domain.PageBreakAvoid
											s.PageBreakInside = layouter_domain.PageBreakAvoid
											s.LineHeightAuto = false
										}),
										Text:          "TOOLKIT",
										ContentY:      61.675,
										ContentWidth:  44.26140625,
										ContentHeight: 13.049999999999999,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxFlex,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
									s.FontSize = 9.5
									s.LineHeight = 13.775
									s.LineHeightFactor = 1.45
									s.RowGap = 4
									s.ColumnGap = 4
									s.Display = layouter_domain.DisplayFlex
									s.BoxSizing = border - box
									s.FlexWrap = layouter_domain.FlexWrapWrap
									s.LineHeightAuto = false
								}),
								ContentY:      84.475,
								ContentWidth:  510.24062993,
								ContentHeight: 34.2,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxFlexItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.PaddingTop = 1
											s.PaddingRight = 6
											s.PaddingBottom = 1
											s.PaddingLeft = 6
											s.BorderTopWidth = 0.75
											s.BorderRightWidth = 0.75
											s.BorderBottomWidth = 0.75
											s.BorderLeftWidth = 0.75
											s.BorderTopLeftRadius = 8
											s.BorderTopRightRadius = 8
											s.BorderBottomRightRadius = 8
											s.BorderBottomLeftRadius = 8
											s.FontSize = 8
											s.LineHeight = 11.6
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayInlineFlex
											s.BoxSizing = border - box
											s.BorderTopStyle = layouter_domain.BorderStyleSolid
											s.BorderRightStyle = layouter_domain.BorderStyleSolid
											s.BorderBottomStyle = layouter_domain.BorderStyleSolid
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Top:    1,
											Right:  6,
											Bottom: 1,
											Left:   6,
										},
										Border: layouter_domain.BoxEdges{
											Top:    0.75,
											Right:  0.75,
											Bottom: 0.75,
											Left:   0.75,
										},
										ContentX:      6.75,
										ContentY:      86.225,
										ContentWidth:  11.00390625,
										ContentHeight: 11.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxAnonymousBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.FontSize = 8
													s.LineHeight = 11.6
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.LineHeightAuto = false
												}),
												ContentX:      6.75,
												ContentY:      86.225,
												ContentWidth:  11.00390625,
												ContentHeight: 11.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.FontSize = 8
															s.LineHeight = 11.6
															s.LineHeightFactor = 1.45
															s.BorderTopStyle = layouter_domain.BorderStyleSolid
															s.BorderRightStyle = layouter_domain.BorderStyleSolid
															s.BorderBottomStyle = layouter_domain.BorderStyleSolid
															s.BorderLeftStyle = layouter_domain.BorderStyleSolid
															s.LineHeightAuto = false
														}),
														Text:          "Go",
														ContentX:      6.75,
														ContentY:      86.225,
														ContentWidth:  11.00390625,
														ContentHeight: 11.6,
													},
												},
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxFlexItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.PaddingTop = 1
											s.PaddingRight = 6
											s.PaddingBottom = 1
											s.PaddingLeft = 6
											s.BorderTopWidth = 0.75
											s.BorderRightWidth = 0.75
											s.BorderBottomWidth = 0.75
											s.BorderLeftWidth = 0.75
											s.BorderTopLeftRadius = 8
											s.BorderTopRightRadius = 8
											s.BorderBottomRightRadius = 8
											s.BorderBottomLeftRadius = 8
											s.FontSize = 8
											s.LineHeight = 11.6
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayInlineFlex
											s.BoxSizing = border - box
											s.BorderTopStyle = layouter_domain.BorderStyleSolid
											s.BorderRightStyle = layouter_domain.BorderStyleSolid
											s.BorderBottomStyle = layouter_domain.BorderStyleSolid
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Top:    1,
											Right:  6,
											Bottom: 1,
											Left:   6,
										},
										Border: layouter_domain.BoxEdges{
											Top:    0.75,
											Right:  0.75,
											Bottom: 0.75,
											Left:   0.75,
										},
										ContentX:      35.25390625,
										ContentY:      86.225,
										ContentWidth:  27.234375,
										ContentHeight: 11.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxAnonymousBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.FontSize = 8
													s.LineHeight = 11.6
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.LineHeightAuto = false
												}),
												ContentX:      35.25390625,
												ContentY:      86.225,
												ContentWidth:  27.234375,
												ContentHeight: 11.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.FontSize = 8
															s.LineHeight = 11.6
															s.LineHeightFactor = 1.45
															s.BorderTopStyle = layouter_domain.BorderStyleSolid
															s.BorderRightStyle = layouter_domain.BorderStyleSolid
															s.BorderBottomStyle = layouter_domain.BorderStyleSolid
															s.BorderLeftStyle = layouter_domain.BorderStyleSolid
															s.LineHeightAuto = false
														}),
														Text:          "C / C++",
														ContentX:      35.25390625,
														ContentY:      86.225,
														ContentWidth:  27.234375,
														ContentHeight: 11.6,
													},
												},
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxFlexItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.PaddingTop = 1
											s.PaddingRight = 6
											s.PaddingBottom = 1
											s.PaddingLeft = 6
											s.BorderTopWidth = 0.75
											s.BorderRightWidth = 0.75
											s.BorderBottomWidth = 0.75
											s.BorderLeftWidth = 0.75
											s.BorderTopLeftRadius = 8
											s.BorderTopRightRadius = 8
											s.BorderBottomRightRadius = 8
											s.BorderBottomLeftRadius = 8
											s.FontSize = 8
											s.LineHeight = 11.6
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayInlineFlex
											s.BoxSizing = border - box
											s.BorderTopStyle = layouter_domain.BorderStyleSolid
											s.BorderRightStyle = layouter_domain.BorderStyleSolid
											s.BorderBottomStyle = layouter_domain.BorderStyleSolid
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Top:    1,
											Right:  6,
											Bottom: 1,
											Left:   6,
										},
										Border: layouter_domain.BoxEdges{
											Top:    0.75,
											Right:  0.75,
											Bottom: 0.75,
											Left:   0.75,
										},
										ContentX:      79.98828125,
										ContentY:      86.225,
										ContentWidth:  77.61328125,
										ContentHeight: 11.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxAnonymousBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.FontSize = 8
													s.LineHeight = 11.6
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.LineHeightAuto = false
												}),
												ContentX:      79.98828125,
												ContentY:      86.225,
												ContentWidth:  77.61328125,
												ContentHeight: 11.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.FontSize = 8
															s.LineHeight = 11.6
															s.LineHeightFactor = 1.45
															s.BorderTopStyle = layouter_domain.BorderStyleSolid
															s.BorderRightStyle = layouter_domain.BorderStyleSolid
															s.BorderBottomStyle = layouter_domain.BorderStyleSolid
															s.BorderLeftStyle = layouter_domain.BorderStyleSolid
															s.LineHeightAuto = false
														}),
														Text:          "Distributed Systems",
														ContentX:      79.98828125,
														ContentY:      86.225,
														ContentWidth:  77.61328125,
														ContentHeight: 11.6,
													},
												},
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxFlexItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.PaddingTop = 1
											s.PaddingRight = 6
											s.PaddingBottom = 1
											s.PaddingLeft = 6
											s.BorderTopWidth = 0.75
											s.BorderRightWidth = 0.75
											s.BorderBottomWidth = 0.75
											s.BorderLeftWidth = 0.75
											s.BorderTopLeftRadius = 8
											s.BorderTopRightRadius = 8
											s.BorderBottomRightRadius = 8
											s.BorderBottomLeftRadius = 8
											s.FontSize = 8
											s.LineHeight = 11.6
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayInlineFlex
											s.BoxSizing = border - box
											s.BorderTopStyle = layouter_domain.BorderStyleSolid
											s.BorderRightStyle = layouter_domain.BorderStyleSolid
											s.BorderBottomStyle = layouter_domain.BorderStyleSolid
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Top:    1,
											Right:  6,
											Bottom: 1,
											Left:   6,
										},
										Border: layouter_domain.BoxEdges{
											Top:    0.75,
											Right:  0.75,
											Bottom: 0.75,
											Left:   0.75,
										},
										ContentX:      175.1015625,
										ContentY:      86.225,
										ContentWidth:  113.1328125,
										ContentHeight: 11.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxAnonymousBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.FontSize = 8
													s.LineHeight = 11.6
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.LineHeightAuto = false
												}),
												ContentX:      175.1015625,
												ContentY:      86.225,
												ContentWidth:  113.1328125,
												ContentHeight: 11.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.FontSize = 8
															s.LineHeight = 11.6
															s.LineHeightFactor = 1.45
															s.BorderTopStyle = layouter_domain.BorderStyleSolid
															s.BorderRightStyle = layouter_domain.BorderStyleSolid
															s.BorderBottomStyle = layouter_domain.BorderStyleSolid
															s.BorderLeftStyle = layouter_domain.BorderStyleSolid
															s.LineHeightAuto = false
														}),
														Text:          "Data Platforms (Kafka, Spark)",
														ContentX:      175.1015625,
														ContentY:      86.225,
														ContentWidth:  113.1328125,
														ContentHeight: 11.6,
													},
												},
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxFlexItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.PaddingTop = 1
											s.PaddingRight = 6
											s.PaddingBottom = 1
											s.PaddingLeft = 6
											s.BorderTopWidth = 0.75
											s.BorderRightWidth = 0.75
											s.BorderBottomWidth = 0.75
											s.BorderLeftWidth = 0.75
											s.BorderTopLeftRadius = 8
											s.BorderTopRightRadius = 8
											s.BorderBottomRightRadius = 8
											s.BorderBottomLeftRadius = 8
											s.FontSize = 8
											s.LineHeight = 11.6
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayInlineFlex
											s.BoxSizing = border - box
											s.BorderTopStyle = layouter_domain.BorderStyleSolid
											s.BorderRightStyle = layouter_domain.BorderStyleSolid
											s.BorderBottomStyle = layouter_domain.BorderStyleSolid
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Top:    1,
											Right:  6,
											Bottom: 1,
											Left:   6,
										},
										Border: layouter_domain.BoxEdges{
											Top:    0.75,
											Right:  0.75,
											Bottom: 0.75,
											Left:   0.75,
										},
										ContentX:      305.734375,
										ContentY:      86.225,
										ContentWidth:  44.671875,
										ContentHeight: 11.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxAnonymousBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.FontSize = 8
													s.LineHeight = 11.6
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.LineHeightAuto = false
												}),
												ContentX:      305.734375,
												ContentY:      86.225,
												ContentWidth:  44.671875,
												ContentHeight: 11.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.FontSize = 8
															s.LineHeight = 11.6
															s.LineHeightFactor = 1.45
															s.BorderTopStyle = layouter_domain.BorderStyleSolid
															s.BorderRightStyle = layouter_domain.BorderStyleSolid
															s.BorderBottomStyle = layouter_domain.BorderStyleSolid
															s.BorderLeftStyle = layouter_domain.BorderStyleSolid
															s.LineHeightAuto = false
														}),
														Text:          "Kubernetes",
														ContentX:      305.734375,
														ContentY:      86.225,
														ContentWidth:  44.671875,
														ContentHeight: 11.6,
													},
												},
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxFlexItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.PaddingTop = 1
											s.PaddingRight = 6
											s.PaddingBottom = 1
											s.PaddingLeft = 6
											s.BorderTopWidth = 0.75
											s.BorderRightWidth = 0.75
											s.BorderBottomWidth = 0.75
											s.BorderLeftWidth = 0.75
											s.BorderTopLeftRadius = 8
											s.BorderTopRightRadius = 8
											s.BorderBottomRightRadius = 8
											s.BorderBottomLeftRadius = 8
											s.FontSize = 8
											s.LineHeight = 11.6
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayInlineFlex
											s.BoxSizing = border - box
											s.BorderTopStyle = layouter_domain.BorderStyleSolid
											s.BorderRightStyle = layouter_domain.BorderStyleSolid
											s.BorderBottomStyle = layouter_domain.BorderStyleSolid
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Top:    1,
											Right:  6,
											Bottom: 1,
											Left:   6,
										},
										Border: layouter_domain.BoxEdges{
											Top:    0.75,
											Right:  0.75,
											Bottom: 0.75,
											Left:   0.75,
										},
										ContentX:      367.90625,
										ContentY:      86.225,
										ContentWidth:  38.89453125,
										ContentHeight: 11.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxAnonymousBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.FontSize = 8
													s.LineHeight = 11.6
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.LineHeightAuto = false
												}),
												ContentX:      367.90625,
												ContentY:      86.225,
												ContentWidth:  38.89453125,
												ContentHeight: 11.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.FontSize = 8
															s.LineHeight = 11.6
															s.LineHeightFactor = 1.45
															s.BorderTopStyle = layouter_domain.BorderStyleSolid
															s.BorderRightStyle = layouter_domain.BorderStyleSolid
															s.BorderBottomStyle = layouter_domain.BorderStyleSolid
															s.BorderLeftStyle = layouter_domain.BorderStyleSolid
															s.LineHeightAuto = false
														}),
														Text:          "Terraform",
														ContentX:      367.90625,
														ContentY:      86.225,
														ContentWidth:  38.89453125,
														ContentHeight: 11.6,
													},
												},
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxFlexItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.PaddingTop = 1
											s.PaddingRight = 6
											s.PaddingBottom = 1
											s.PaddingLeft = 6
											s.BorderTopWidth = 0.75
											s.BorderRightWidth = 0.75
											s.BorderBottomWidth = 0.75
											s.BorderLeftWidth = 0.75
											s.BorderTopLeftRadius = 8
											s.BorderTopRightRadius = 8
											s.BorderBottomRightRadius = 8
											s.BorderBottomLeftRadius = 8
											s.FontSize = 8
											s.LineHeight = 11.6
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayInlineFlex
											s.BoxSizing = border - box
											s.BorderTopStyle = layouter_domain.BorderStyleSolid
											s.BorderRightStyle = layouter_domain.BorderStyleSolid
											s.BorderBottomStyle = layouter_domain.BorderStyleSolid
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Top:    1,
											Right:  6,
											Bottom: 1,
											Left:   6,
										},
										Border: layouter_domain.BoxEdges{
											Top:    0.75,
											Right:  0.75,
											Bottom: 0.75,
											Left:   0.75,
										},
										ContentX:      424.30078125,
										ContentY:      86.225,
										ContentWidth:  40.72265625,
										ContentHeight: 11.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxAnonymousBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.FontSize = 8
													s.LineHeight = 11.6
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.LineHeightAuto = false
												}),
												ContentX:      424.30078125,
												ContentY:      86.225,
												ContentWidth:  40.72265625,
												ContentHeight: 11.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.FontSize = 8
															s.LineHeight = 11.6
															s.LineHeightFactor = 1.45
															s.BorderTopStyle = layouter_domain.BorderStyleSolid
															s.BorderRightStyle = layouter_domain.BorderStyleSolid
															s.BorderBottomStyle = layouter_domain.BorderStyleSolid
															s.BorderLeftStyle = layouter_domain.BorderStyleSolid
															s.LineHeightAuto = false
														}),
														Text:          "AWS / GCP",
														ContentX:      424.30078125,
														ContentY:      86.225,
														ContentWidth:  40.72265625,
														ContentHeight: 11.6,
													},
												},
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxFlexItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
											s.PaddingTop = 1
											s.PaddingRight = 6
											s.PaddingBottom = 1
											s.PaddingLeft = 6
											s.BorderTopWidth = 0.75
											s.BorderRightWidth = 0.75
											s.BorderBottomWidth = 0.75
											s.BorderLeftWidth = 0.75
											s.BorderTopLeftRadius = 8
											s.BorderTopRightRadius = 8
											s.BorderBottomRightRadius = 8
											s.BorderBottomLeftRadius = 8
											s.FontSize = 8
											s.LineHeight = 11.6
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayInlineFlex
											s.BoxSizing = border - box
											s.BorderTopStyle = layouter_domain.BorderStyleSolid
											s.BorderRightStyle = layouter_domain.BorderStyleSolid
											s.BorderBottomStyle = layouter_domain.BorderStyleSolid
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Top:    1,
											Right:  6,
											Bottom: 1,
											Left:   6,
										},
										Border: layouter_domain.BoxEdges{
											Top:    0.75,
											Right:  0.75,
											Bottom: 0.75,
											Left:   0.75,
										},
										ContentX:      6.75,
										ContentY:      105.32499999999999,
										ContentWidth:  72.609375,
										ContentHeight: 11.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxAnonymousBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.FontSize = 8
													s.LineHeight = 11.6
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.LineHeightAuto = false
												}),
												ContentX:      6.75,
												ContentY:      105.32499999999999,
												ContentWidth:  72.609375,
												ContentHeight: 11.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.BorderTopColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderRightColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.BorderBottomColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.BorderLeftColour = layouter_domain.NewRGBA(0.7254901960784313, 0.6588235294117647, 0.8784313725490196, 1)
															s.FontSize = 8
															s.LineHeight = 11.6
															s.LineHeightFactor = 1.45
															s.BorderTopStyle = layouter_domain.BorderStyleSolid
															s.BorderRightStyle = layouter_domain.BorderStyleSolid
															s.BorderBottomStyle = layouter_domain.BorderStyleSolid
															s.BorderLeftStyle = layouter_domain.BorderStyleSolid
															s.LineHeightAuto = false
														}),
														Text:          "Observability / SRE",
														ContentX:      6.75,
														ContentY:      105.32499999999999,
														ContentWidth:  72.609375,
														ContentHeight: 11.6,
													},
												},
											},
										},
									},
								},
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
							s.FontSize = 9.5
							s.LineHeight = 13.775
							s.LineHeightFactor = 1.45
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.LineHeightAuto = false
						}),
						ContentY:      132.675,
						ContentWidth:  510.24062993,
						ContentHeight: 1106.1000000000001,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.788235294117647, 0.807843137254902, 0.8392156862745098, 1)
									s.MarginTop = layouter_domain.DimensionPt(14)
									s.MarginBottom = layouter_domain.DimensionPt(6)
									s.PaddingBottom = 3
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 13.049999999999999
									s.LineHeightFactor = 1.45
									s.LetterSpacing = 0.76
									s.FontWeight = 700
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.TextTransform = layouter_domain.TextTransformUppercase
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
									s.PageBreakAfter = layouter_domain.PageBreakAvoid
									s.PageBreakInside = layouter_domain.PageBreakAvoid
									s.LineHeightAuto = false
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 3,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 6,
								},
								ContentY:      132.675,
								ContentWidth:  510.24062993,
								ContentHeight: 13.049999999999999,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderBottomColour = layouter_domain.NewRGBA(0.788235294117647, 0.807843137254902, 0.8392156862745098, 1)
											s.FontSize = 9
											s.LineHeight = 13.049999999999999
											s.LineHeightFactor = 1.45
											s.LetterSpacing = 0.76
											s.FontWeight = 700
											s.TextTransform = layouter_domain.TextTransformUppercase
											s.BorderBottomStyle = layouter_domain.BorderStyleSolid
											s.PageBreakAfter = layouter_domain.PageBreakAvoid
											s.PageBreakInside = layouter_domain.PageBreakAvoid
											s.LineHeightAuto = false
										}),
										Text:          "EXPERIENCE",
										ContentY:      132.675,
										ContentWidth:  61.88125,
										ContentHeight: 13.049999999999999,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxGrid,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.GridTemplateColumns = []layouter_domain.GridTrack{
										layouter_domain.GridTrack{
											Value: 108,
											Unit:  layouter_domain.GridTrackPoints,
										},
										layouter_domain.GridTrack{
											Value: 1,
											Unit:  layouter_domain.GridTrackFr,
										},
									}
									s.BorderTopColour = layouter_domain.NewRGBA(0.8901960784313725, 0.9019607843137255, 0.9176470588235294, 1)
									s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
									s.PaddingTop = 6
									s.PaddingBottom = 6
									s.BorderTopWidth = 0.75
									s.FontSize = 9.5
									s.LineHeight = 13.775
									s.LineHeightFactor = 1.45
									s.Display = layouter_domain.DisplayGrid
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.LineHeightAuto = false
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    6,
									Bottom: 6,
								},
								Border: layouter_domain.BoxEdges{
									Top: 0.75,
								},
								ContentY:      162.22500000000002,
								ContentWidth:  510.24062993,
								ContentHeight: 137.25,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxGridItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.PaddingRight = 18
											s.FontSize = 9.5
											s.LineHeight = 13.775
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
											s.AlignSelf = layouter_domain.AlignSelfFlexStart
											s.PageBreakInside = layouter_domain.PageBreakAvoid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Right: 18,
										},
										ContentY:      162.22500000000002,
										ContentWidth:  90,
										ContentHeight: 44.974999999999994,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.FontWeight = 700
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      162.22500000000002,
												ContentWidth:  90,
												ContentHeight: 12.325,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.FontWeight = 700
															s.LineHeightAuto = false
														}),
														Text:          "2022-Present",
														ContentY:      162.22500000000002,
														ContentWidth:  57.90234375,
														ContentHeight: 12.325,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      176.55,
												ContentWidth:  90,
												ContentHeight: 12.325,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Example Studio",
														ContentY:      176.55,
														ContentWidth:  65.5546875,
														ContentHeight: 12.325,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      190.87500000000003,
												ContentWidth:  90,
												ContentHeight: 12.325,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Remote",
														ContentY:      190.87500000000003,
														ContentWidth:  32.859375,
														ContentHeight: 12.325,
													},
												},
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxGridItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.8901960784313725, 0.9019607843137255, 0.9176470588235294, 1)
											s.PaddingLeft = 12
											s.BorderLeftWidth = 0.75
											s.FontSize = 9.5
											s.LineHeight = 13.775
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Left: 12,
										},
										Border: layouter_domain.BoxEdges{
											Left: 0.75,
										},
										ContentX:      120.75,
										ContentY:      162.22500000000002,
										ContentWidth:  389.49062993,
										ContentHeight: 137.25,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginBottom = layouter_domain.DimensionPt(4)
													s.FontSize = 11
													s.LineHeight = 15.95
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.PageBreakAfter = layouter_domain.PageBreakAvoid
													s.PageBreakInside = layouter_domain.PageBreakAvoid
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 4,
												},
												ContentX:      120.75,
												ContentY:      162.22500000000002,
												ContentWidth:  389.49062993,
												ContentHeight: 15.95,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 11
															s.LineHeight = 15.95
															s.LineHeightFactor = 1.45
															s.PageBreakAfter = layouter_domain.PageBreakAvoid
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Text:          "Founder and Principal Engineer",
														ContentX:      120.75,
														ContentY:      162.22500000000002,
														ContentWidth:  166.453125,
														ContentHeight: 15.95,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.PaddingLeft = 12
													s.FontSize = 9.5
													s.LineHeight = 13.775
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Padding: layouter_domain.BoxEdges{
													Left: 12,
												},
												Margin: layouter_domain.BoxEdges{
													Bottom: 3,
												},
												ContentX:      132.75,
												ContentY:      182.175,
												ContentWidth:  377.49062993,
												ContentHeight: 88.65,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      182.175,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      182.175,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Rebuilt the deployment pipeline so that every service shipped through the same",
																ContentX:      132.75,
																ContentY:      182.175,
																ContentWidth:  365.98828125,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "reviewed path, which cut release preparation from days to under an hour",
																ContentX:      132.75,
																ContentY:      195.95000000000002,
																ContentWidth:  334.546875,
																ContentHeight: 13.775,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      212.72500000000002,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      212.72500000000002,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Ran the container platform and its monitoring through a period of fast growth,",
																ContentX:      132.75,
																ContentY:      212.72500000000002,
																ContentWidth:  360.64453125,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "keeping the service within its availability target",
																ContentX:      132.75,
																ContentY:      226.50000000000003,
																ContentWidth:  214.18359375,
																ContentHeight: 13.775,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      243.275,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      243.275,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Moved a legacy reporting system onto managed storage, checking every migrated",
																ContentX:      132.75,
																ContentY:      243.275,
																ContentWidth:  376.46484375,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "record against its source before the old system was retired",
																ContentX:      132.75,
																ContentY:      257.05,
																ContentWidth:  268.04296875,
																ContentHeight: 13.775,
															},
														},
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginTop = layouter_domain.DimensionPt(4)
													s.FontSize = 9.5
													s.LineHeight = 13.775
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.PageBreakInside = layouter_domain.PageBreakAvoid
													s.LineHeightAuto = false
												}),
												ContentX:      120.75,
												ContentY:      274.82500000000005,
												ContentWidth:  389.49062993,
												ContentHeight: 24.65,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxBlock,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 7.5
															s.LineHeight = 10.875
															s.LineHeightFactor = 1.45
															s.LetterSpacing = 0.76
															s.Display = layouter_domain.DisplayBlock
															s.BoxSizing = border - box
															s.TextTransform = layouter_domain.TextTransformUppercase
															s.LineHeightAuto = false
														}),
														ContentX:      120.75,
														ContentY:      274.82500000000005,
														ContentWidth:  389.49062993,
														ContentHeight: 10.875,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
																	s.FontSize = 7.5
																	s.LineHeight = 10.875
																	s.LineHeightFactor = 1.45
																	s.LetterSpacing = 0.76
																	s.TextTransform = layouter_domain.TextTransformUppercase
																	s.LineHeightAuto = false
																}),
																Text:          "TECH",
																ContentX:      120.75,
																ContentY:      274.82500000000005,
																ContentWidth:  21.6728125,
																ContentHeight: 10.875,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxBlock,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayBlock
															s.BoxSizing = border - box
															s.LineHeightAuto = false
														}),
														ContentX:      120.75,
														ContentY:      285.70000000000005,
														ContentWidth:  389.49062993,
														ContentHeight: 13.775,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.LineHeightAuto = false
																}),
																Text:          "Go, Kubernetes, Terraform, PostgreSQL, Kafka, Prometheus, Grafana, AWS, GCP",
																ContentX:      120.75,
																ContentY:      285.70000000000005,
																ContentWidth:  364.01953125,
																ContentHeight: 13.775,
															},
														},
													},
												},
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxGrid,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.GridTemplateColumns = []layouter_domain.GridTrack{
										layouter_domain.GridTrack{
											Value: 108,
											Unit:  layouter_domain.GridTrackPoints,
										},
										layouter_domain.GridTrack{
											Value: 1,
											Unit:  layouter_domain.GridTrackFr,
										},
									}
									s.BorderTopColour = layouter_domain.NewRGBA(0.8901960784313725, 0.9019607843137255, 0.9176470588235294, 1)
									s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
									s.PaddingTop = 6
									s.PaddingBottom = 6
									s.BorderTopWidth = 0.75
									s.FontSize = 9.5
									s.LineHeight = 13.775
									s.LineHeightFactor = 1.45
									s.Display = layouter_domain.DisplayGrid
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.LineHeightAuto = false
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    6,
									Bottom: 6,
								},
								Border: layouter_domain.BoxEdges{
									Top: 0.75,
								},
								ContentY:      312.225,
								ContentWidth:  510.24062993,
								ContentHeight: 198.35,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxGridItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.PaddingRight = 18
											s.FontSize = 9.5
											s.LineHeight = 13.775
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
											s.AlignSelf = layouter_domain.AlignSelfFlexStart
											s.PageBreakInside = layouter_domain.PageBreakAvoid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Right: 18,
										},
										ContentY:      312.225,
										ContentWidth:  90,
										ContentHeight: 57.3,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.FontWeight = 700
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      312.225,
												ContentWidth:  90,
												ContentHeight: 12.325,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.FontWeight = 700
															s.LineHeightAuto = false
														}),
														Text:          "2020-2022",
														ContentY:      312.225,
														ContentWidth:  44.0390625,
														ContentHeight: 12.325,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      326.55,
												ContentWidth:  90,
												ContentHeight: 12.325,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Harbour Logistics",
														ContentY:      326.55,
														ContentWidth:  74.56640625,
														ContentHeight: 12.325,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      340.875,
												ContentWidth:  90,
												ContentHeight: 24.65,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Rotterdam, The",
														ContentY:      340.875,
														ContentWidth:  65.5546875,
														ContentHeight: 12.325,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Netherlands",
														ContentY:      353.2,
														ContentWidth:  52.30078125,
														ContentHeight: 12.325,
													},
												},
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxGridItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.8901960784313725, 0.9019607843137255, 0.9176470588235294, 1)
											s.PaddingLeft = 12
											s.BorderLeftWidth = 0.75
											s.FontSize = 9.5
											s.LineHeight = 13.775
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Left: 12,
										},
										Border: layouter_domain.BoxEdges{
											Left: 0.75,
										},
										ContentX:      120.75,
										ContentY:      312.225,
										ContentWidth:  389.49062993,
										ContentHeight: 198.35,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginBottom = layouter_domain.DimensionPt(4)
													s.FontSize = 11
													s.LineHeight = 15.95
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.PageBreakAfter = layouter_domain.PageBreakAvoid
													s.PageBreakInside = layouter_domain.PageBreakAvoid
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 4,
												},
												ContentX:      120.75,
												ContentY:      312.225,
												ContentWidth:  389.49062993,
												ContentHeight: 15.95,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 11
															s.LineHeight = 15.95
															s.LineHeightFactor = 1.45
															s.PageBreakAfter = layouter_domain.PageBreakAvoid
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Text:          "Lead Infrastructure Engineer",
														ContentX:      120.75,
														ContentY:      312.225,
														ContentWidth:  152.203125,
														ContentHeight: 15.95,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.PaddingLeft = 12
													s.FontSize = 9.5
													s.LineHeight = 13.775
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Padding: layouter_domain.BoxEdges{
													Left: 12,
												},
												Margin: layouter_domain.BoxEdges{
													Bottom: 3,
												},
												ContentX:      132.75,
												ContentY:      332.175,
												ContentWidth:  377.49062993,
												ContentHeight: 149.75,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      332.175,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      332.175,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Rebuilt the deployment pipeline so that every service shipped through the same",
																ContentX:      132.75,
																ContentY:      332.175,
																ContentWidth:  365.98828125,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "reviewed path, which cut release preparation from days to under an hour",
																ContentX:      132.75,
																ContentY:      345.95,
																ContentWidth:  334.546875,
																ContentHeight: 13.775,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      362.725,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      362.725,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Ran the container platform and its monitoring through a period of fast growth,",
																ContentX:      132.75,
																ContentY:      362.725,
																ContentWidth:  360.64453125,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "keeping the service within its availability target",
																ContentX:      132.75,
																ContentY:      376.5,
																ContentWidth:  214.18359375,
																ContentHeight: 13.775,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      393.27500000000003,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      393.27500000000003,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Moved a legacy reporting system onto managed storage, checking every migrated",
																ContentX:      132.75,
																ContentY:      393.27500000000003,
																ContentWidth:  376.46484375,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "record against its source before the old system was retired",
																ContentX:      132.75,
																ContentY:      407.05,
																ContentWidth:  268.04296875,
																ContentHeight: 13.775,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      423.82500000000005,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      423.82500000000005,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Introduced load testing to the release process, so capacity problems surfaced in",
																ContentX:      132.75,
																ContentY:      423.82500000000005,
																ContentWidth:  365.26171875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "staging instead of during traffic peaks",
																ContentX:      132.75,
																ContentY:      437.6,
																ContentWidth:  173.98828125,
																ContentHeight: 13.775,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      454.375,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      454.375,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Wrote the operating guides and trained the internal team to run the platform",
																ContentX:      132.75,
																ContentY:      454.375,
																ContentWidth:  353.6484375,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "after handover",
																ContentX:      132.75,
																ContentY:      468.15,
																ContentWidth:  68.109375,
																ContentHeight: 13.775,
															},
														},
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginTop = layouter_domain.DimensionPt(4)
													s.FontSize = 9.5
													s.LineHeight = 13.775
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.PageBreakInside = layouter_domain.PageBreakAvoid
													s.LineHeightAuto = false
												}),
												ContentX:      120.75,
												ContentY:      485.925,
												ContentWidth:  389.49062993,
												ContentHeight: 24.65,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxBlock,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 7.5
															s.LineHeight = 10.875
															s.LineHeightFactor = 1.45
															s.LetterSpacing = 0.76
															s.Display = layouter_domain.DisplayBlock
															s.BoxSizing = border - box
															s.TextTransform = layouter_domain.TextTransformUppercase
															s.LineHeightAuto = false
														}),
														ContentX:      120.75,
														ContentY:      485.925,
														ContentWidth:  389.49062993,
														ContentHeight: 10.875,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
																	s.FontSize = 7.5
																	s.LineHeight = 10.875
																	s.LineHeightFactor = 1.45
																	s.LetterSpacing = 0.76
																	s.TextTransform = layouter_domain.TextTransformUppercase
																	s.LineHeightAuto = false
																}),
																Text:          "TECH",
																ContentX:      120.75,
																ContentY:      485.925,
																ContentWidth:  21.6728125,
																ContentHeight: 10.875,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxBlock,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayBlock
															s.BoxSizing = border - box
															s.LineHeightAuto = false
														}),
														ContentX:      120.75,
														ContentY:      496.8,
														ContentWidth:  389.49062993,
														ContentHeight: 13.775,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.LineHeightAuto = false
																}),
																Text:          "Go, Kubernetes, Terraform, PostgreSQL, Kafka, Prometheus, Grafana, AWS, GCP",
																ContentX:      120.75,
																ContentY:      496.8,
																ContentWidth:  364.01953125,
																ContentHeight: 13.775,
															},
														},
													},
												},
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxGrid,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.GridTemplateColumns = []layouter_domain.GridTrack{
										layouter_domain.GridTrack{
											Value: 108,
											Unit:  layouter_domain.GridTrackPoints,
										},
										layouter_domain.GridTrack{
											Value: 1,
											Unit:  layouter_domain.GridTrackFr,
										},
									}
									s.BorderTopColour = layouter_domain.NewRGBA(0.8901960784313725, 0.9019607843137255, 0.9176470588235294, 1)
									s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
									s.PaddingTop = 6
									s.PaddingBottom = 6
									s.BorderTopWidth = 0.75
									s.FontSize = 9.5
									s.LineHeight = 13.775
									s.LineHeightFactor = 1.45
									s.Display = layouter_domain.DisplayGrid
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.LineHeightAuto = false
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    6,
									Bottom: 6,
								},
								Border: layouter_domain.BoxEdges{
									Top: 0.75,
								},
								ContentY:      523.325,
								ContentWidth:  510.24062993,
								ContentHeight: 167.8,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxGridItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.PaddingRight = 18
											s.FontSize = 9.5
											s.LineHeight = 13.775
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
											s.AlignSelf = layouter_domain.AlignSelfFlexStart
											s.PageBreakInside = layouter_domain.PageBreakAvoid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Right: 18,
										},
										ContentY:      523.325,
										ContentWidth:  90,
										ContentHeight: 57.3,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.FontWeight = 700
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      523.325,
												ContentWidth:  90,
												ContentHeight: 12.325,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.FontWeight = 700
															s.LineHeightAuto = false
														}),
														Text:          "2018-2020",
														ContentY:      523.325,
														ContentWidth:  44.0390625,
														ContentHeight: 12.325,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      537.6500000000001,
												ContentWidth:  90,
												ContentHeight: 12.325,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Northwind Finance",
														ContentY:      537.6500000000001,
														ContentWidth:  80.23828125,
														ContentHeight: 12.325,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      551.975,
												ContentWidth:  90,
												ContentHeight: 24.65,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Amsterdam, The",
														ContentY:      551.975,
														ContentWidth:  69.73828125,
														ContentHeight: 12.325,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Netherlands",
														ContentY:      564.3000000000001,
														ContentWidth:  52.30078125,
														ContentHeight: 12.325,
													},
												},
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxGridItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.8901960784313725, 0.9019607843137255, 0.9176470588235294, 1)
											s.PaddingLeft = 12
											s.BorderLeftWidth = 0.75
											s.FontSize = 9.5
											s.LineHeight = 13.775
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Left: 12,
										},
										Border: layouter_domain.BoxEdges{
											Left: 0.75,
										},
										ContentX:      120.75,
										ContentY:      523.325,
										ContentWidth:  389.49062993,
										ContentHeight: 167.8,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginBottom = layouter_domain.DimensionPt(4)
													s.FontSize = 11
													s.LineHeight = 15.95
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.PageBreakAfter = layouter_domain.PageBreakAvoid
													s.PageBreakInside = layouter_domain.PageBreakAvoid
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 4,
												},
												ContentX:      120.75,
												ContentY:      523.325,
												ContentWidth:  389.49062993,
												ContentHeight: 15.95,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 11
															s.LineHeight = 15.95
															s.LineHeightFactor = 1.45
															s.PageBreakAfter = layouter_domain.PageBreakAvoid
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Text:          "Senior Platform Engineer",
														ContentX:      120.75,
														ContentY:      523.325,
														ContentWidth:  132.9140625,
														ContentHeight: 15.95,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.PaddingLeft = 12
													s.FontSize = 9.5
													s.LineHeight = 13.775
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Padding: layouter_domain.BoxEdges{
													Left: 12,
												},
												Margin: layouter_domain.BoxEdges{
													Bottom: 3,
												},
												ContentX:      132.75,
												ContentY:      543.2750000000001,
												ContentWidth:  377.49062993,
												ContentHeight: 119.2,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      543.2750000000001,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      543.2750000000001,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Rebuilt the deployment pipeline so that every service shipped through the same",
																ContentX:      132.75,
																ContentY:      543.2750000000001,
																ContentWidth:  365.98828125,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "reviewed path, which cut release preparation from days to under an hour",
																ContentX:      132.75,
																ContentY:      557.0500000000001,
																ContentWidth:  334.546875,
																ContentHeight: 13.775,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      573.825,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      573.825,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Ran the container platform and its monitoring through a period of fast growth,",
																ContentX:      132.75,
																ContentY:      573.825,
																ContentWidth:  360.64453125,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "keeping the service within its availability target",
																ContentX:      132.75,
																ContentY:      587.6,
																ContentWidth:  214.18359375,
																ContentHeight: 13.775,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      604.3750000000001,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      604.3750000000001,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Moved a legacy reporting system onto managed storage, checking every migrated",
																ContentX:      132.75,
																ContentY:      604.3750000000001,
																ContentWidth:  376.46484375,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "record against its source before the old system was retired",
																ContentX:      132.75,
																ContentY:      618.1500000000001,
																ContentWidth:  268.04296875,
																ContentHeight: 13.775,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      634.9250000000001,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      634.9250000000001,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Introduced load testing to the release process, so capacity problems surfaced in",
																ContentX:      132.75,
																ContentY:      634.9250000000001,
																ContentWidth:  365.26171875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "staging instead of during traffic peaks",
																ContentX:      132.75,
																ContentY:      648.7,
																ContentWidth:  173.98828125,
																ContentHeight: 13.775,
															},
														},
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginTop = layouter_domain.DimensionPt(4)
													s.FontSize = 9.5
													s.LineHeight = 13.775
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.PageBreakInside = layouter_domain.PageBreakAvoid
													s.LineHeightAuto = false
												}),
												ContentX:      120.75,
												ContentY:      666.475,
												ContentWidth:  389.49062993,
												ContentHeight: 24.65,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxBlock,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 7.5
															s.LineHeight = 10.875
															s.LineHeightFactor = 1.45
															s.LetterSpacing = 0.76
															s.Display = layouter_domain.DisplayBlock
															s.BoxSizing = border - box
															s.TextTransform = layouter_domain.TextTransformUppercase
															s.LineHeightAuto = false
														}),
														ContentX:      120.75,
														ContentY:      666.475,
														ContentWidth:  389.49062993,
														ContentHeight: 10.875,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
																	s.FontSize = 7.5
																	s.LineHeight = 10.875
																	s.LineHeightFactor = 1.45
																	s.LetterSpacing = 0.76
																	s.TextTransform = layouter_domain.TextTransformUppercase
																	s.LineHeightAuto = false
																}),
																Text:          "TECH",
																ContentX:      120.75,
																ContentY:      666.475,
																ContentWidth:  21.6728125,
																ContentHeight: 10.875,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxBlock,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayBlock
															s.BoxSizing = border - box
															s.LineHeightAuto = false
														}),
														ContentX:      120.75,
														ContentY:      677.35,
														ContentWidth:  389.49062993,
														ContentHeight: 13.775,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.LineHeightAuto = false
																}),
																Text:          "Go, Kubernetes, Terraform, PostgreSQL, Kafka, Prometheus, Grafana, AWS, GCP",
																ContentX:      120.75,
																ContentY:      677.35,
																ContentWidth:  364.01953125,
																ContentHeight: 13.775,
															},
														},
													},
												},
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxGrid,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.GridTemplateColumns = []layouter_domain.GridTrack{
										layouter_domain.GridTrack{
											Value: 108,
											Unit:  layouter_domain.GridTrackPoints,
										},
										layouter_domain.GridTrack{
											Value: 1,
											Unit:  layouter_domain.GridTrackFr,
										},
									}
									s.BorderTopColour = layouter_domain.NewRGBA(0.8901960784313725, 0.9019607843137255, 0.9176470588235294, 1)
									s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
									s.PaddingTop = 6
									s.PaddingBottom = 6
									s.BorderTopWidth = 0.75
									s.FontSize = 9.5
									s.LineHeight = 13.775
									s.LineHeightFactor = 1.45
									s.Display = layouter_domain.DisplayGrid
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.LineHeightAuto = false
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    6,
									Bottom: 6,
								},
								Border: layouter_domain.BoxEdges{
									Top: 0.75,
								},
								ContentY:      703.875,
								ContentWidth:  510.24062993,
								ContentHeight: 198.35,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxGridItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.PaddingRight = 18
											s.FontSize = 9.5
											s.LineHeight = 13.775
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
											s.AlignSelf = layouter_domain.AlignSelfFlexStart
											s.PageBreakInside = layouter_domain.PageBreakAvoid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Right: 18,
										},
										ContentY:      703.875,
										ContentWidth:  90,
										ContentHeight: 57.3,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.FontWeight = 700
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      703.875,
												ContentWidth:  90,
												ContentHeight: 12.325,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.FontWeight = 700
															s.LineHeightAuto = false
														}),
														Text:          "2016-2018",
														ContentY:      703.875,
														ContentWidth:  44.0390625,
														ContentHeight: 12.325,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      718.2,
												ContentWidth:  90,
												ContentHeight: 12.325,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Contoso Telemetry",
														ContentY:      718.2,
														ContentWidth:  79.39453125,
														ContentHeight: 12.325,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      732.525,
												ContentWidth:  90,
												ContentHeight: 24.65,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Utrecht, The",
														ContentY:      732.525,
														ContentWidth:  51.97265625,
														ContentHeight: 12.325,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Netherlands",
														ContentY:      744.85,
														ContentWidth:  52.30078125,
														ContentHeight: 12.325,
													},
												},
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxGridItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.8901960784313725, 0.9019607843137255, 0.9176470588235294, 1)
											s.PaddingLeft = 12
											s.BorderLeftWidth = 0.75
											s.FontSize = 9.5
											s.LineHeight = 13.775
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Left: 12,
										},
										Border: layouter_domain.BoxEdges{
											Left: 0.75,
										},
										ContentX:      120.75,
										ContentY:      703.875,
										ContentWidth:  389.49062993,
										ContentHeight: 198.35,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginBottom = layouter_domain.DimensionPt(4)
													s.FontSize = 11
													s.LineHeight = 15.95
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.PageBreakAfter = layouter_domain.PageBreakAvoid
													s.PageBreakInside = layouter_domain.PageBreakAvoid
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 4,
												},
												ContentX:      120.75,
												ContentY:      703.875,
												ContentWidth:  389.49062993,
												ContentHeight: 15.95,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 11
															s.LineHeight = 15.95
															s.LineHeightFactor = 1.45
															s.PageBreakAfter = layouter_domain.PageBreakAvoid
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Text:          "Site Reliability Engineer",
														ContentX:      120.75,
														ContentY:      703.875,
														ContentWidth:  123.55078125,
														ContentHeight: 15.95,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.PaddingLeft = 12
													s.FontSize = 9.5
													s.LineHeight = 13.775
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Padding: layouter_domain.BoxEdges{
													Left: 12,
												},
												Margin: layouter_domain.BoxEdges{
													Bottom: 3,
												},
												ContentX:      132.75,
												ContentY:      723.825,
												ContentWidth:  377.49062993,
												ContentHeight: 149.75,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      723.825,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      723.825,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Rebuilt the deployment pipeline so that every service shipped through the same",
																ContentX:      132.75,
																ContentY:      723.825,
																ContentWidth:  365.98828125,
																ContentHeight: 13.775,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "reviewed path, which cut release preparation from days to under an hour",
																ContentX:      132.75,
																ContentY:      737.6,
																ContentWidth:  334.546875,
																ContentHeight: 13.775,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      754.375,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      754.375,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Ran the container platform and its monitoring through a period of fast growth,",
																ContentX:      132.75,
																ContentY:      754.375,
																ContentWidth:  360.64453125,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "keeping the service within its availability target",
																ContentX:      132.75,
																ContentY:      768.15,
																ContentWidth:  214.18359375,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      784.9250000000001,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      784.9250000000001,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Moved a legacy reporting system onto managed storage, checking every migrated",
																ContentX:      132.75,
																ContentY:      784.9250000000001,
																ContentWidth:  376.46484375,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "record against its source before the old system was retired",
																ContentX:      132.75,
																ContentY:      798.7,
																ContentWidth:  268.04296875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      815.475,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      815.475,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Introduced load testing to the release process, so capacity problems surfaced in",
																ContentX:      132.75,
																ContentY:      815.475,
																ContentWidth:  365.26171875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "staging instead of during traffic peaks",
																ContentX:      132.75,
																ContentY:      829.25,
																ContentWidth:  173.98828125,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      846.0250000000001,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      846.0250000000001,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Wrote the operating guides and trained the internal team to run the platform",
																ContentX:      132.75,
																ContentY:      846.0250000000001,
																ContentWidth:  353.6484375,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "after handover",
																ContentX:      132.75,
																ContentY:      859.8000000000001,
																ContentWidth:  68.109375,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginTop = layouter_domain.DimensionPt(4)
													s.FontSize = 9.5
													s.LineHeight = 13.775
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.PageBreakInside = layouter_domain.PageBreakAvoid
													s.LineHeightAuto = false
												}),
												ContentX:      120.75,
												ContentY:      877.575,
												ContentWidth:  389.49062993,
												ContentHeight: 24.65,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxBlock,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 7.5
															s.LineHeight = 10.875
															s.LineHeightFactor = 1.45
															s.LetterSpacing = 0.76
															s.Display = layouter_domain.DisplayBlock
															s.BoxSizing = border - box
															s.TextTransform = layouter_domain.TextTransformUppercase
															s.LineHeightAuto = false
														}),
														ContentX:      120.75,
														ContentY:      877.575,
														ContentWidth:  389.49062993,
														ContentHeight: 10.875,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
																	s.FontSize = 7.5
																	s.LineHeight = 10.875
																	s.LineHeightFactor = 1.45
																	s.LetterSpacing = 0.76
																	s.TextTransform = layouter_domain.TextTransformUppercase
																	s.LineHeightAuto = false
																}),
																Text:          "TECH",
																ContentX:      120.75,
																ContentY:      877.575,
																ContentWidth:  21.6728125,
																ContentHeight: 10.875,
																PageIndex:     1,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxBlock,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayBlock
															s.BoxSizing = border - box
															s.LineHeightAuto = false
														}),
														ContentX:      120.75,
														ContentY:      888.45,
														ContentWidth:  389.49062993,
														ContentHeight: 13.775,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.LineHeightAuto = false
																}),
																Text:          "Go, Kubernetes, Terraform, PostgreSQL, Kafka, Prometheus, Grafana, AWS, GCP",
																ContentX:      120.75,
																ContentY:      888.45,
																ContentWidth:  364.01953125,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
												},
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxGrid,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.GridTemplateColumns = []layouter_domain.GridTrack{
										layouter_domain.GridTrack{
											Value: 108,
											Unit:  layouter_domain.GridTrackPoints,
										},
										layouter_domain.GridTrack{
											Value: 1,
											Unit:  layouter_domain.GridTrackFr,
										},
									}
									s.BorderTopColour = layouter_domain.NewRGBA(0.8901960784313725, 0.9019607843137255, 0.9176470588235294, 1)
									s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
									s.PaddingTop = 6
									s.PaddingBottom = 6
									s.BorderTopWidth = 0.75
									s.FontSize = 9.5
									s.LineHeight = 13.775
									s.LineHeightFactor = 1.45
									s.Display = layouter_domain.DisplayGrid
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.LineHeightAuto = false
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    6,
									Bottom: 6,
								},
								Border: layouter_domain.BoxEdges{
									Top: 0.75,
								},
								ContentY:      914.9750000000001,
								ContentWidth:  510.24062993,
								ContentHeight: 137.25,
								PageIndex:     1,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxGridItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.PaddingRight = 18
											s.FontSize = 9.5
											s.LineHeight = 13.775
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
											s.AlignSelf = layouter_domain.AlignSelfFlexStart
											s.PageBreakInside = layouter_domain.PageBreakAvoid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Right: 18,
										},
										ContentY:      914.9750000000001,
										ContentWidth:  90,
										ContentHeight: 57.3,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.FontWeight = 700
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      914.9750000000001,
												ContentWidth:  90,
												ContentHeight: 12.325,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.FontWeight = 700
															s.LineHeightAuto = false
														}),
														Text:          "2014-2016",
														ContentY:      914.9750000000001,
														ContentWidth:  44.0390625,
														ContentHeight: 12.325,
														PageIndex:     1,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      929.3000000000002,
												ContentWidth:  90,
												ContentHeight: 12.325,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Fabrikam Games",
														ContentY:      929.3000000000002,
														ContentWidth:  71.30859375,
														ContentHeight: 12.325,
														PageIndex:     1,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      943.6250000000001,
												ContentWidth:  90,
												ContentHeight: 24.65,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "London, United",
														ContentY:      943.6250000000001,
														ContentWidth:  65.34375,
														ContentHeight: 12.325,
														PageIndex:     1,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Kingdom",
														ContentY:      955.9500000000002,
														ContentWidth:  38.37890625,
														ContentHeight: 12.325,
														PageIndex:     1,
													},
												},
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxGridItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.8901960784313725, 0.9019607843137255, 0.9176470588235294, 1)
											s.PaddingLeft = 12
											s.BorderLeftWidth = 0.75
											s.FontSize = 9.5
											s.LineHeight = 13.775
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Left: 12,
										},
										Border: layouter_domain.BoxEdges{
											Left: 0.75,
										},
										ContentX:      120.75,
										ContentY:      914.9750000000001,
										ContentWidth:  389.49062993,
										ContentHeight: 137.25,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginBottom = layouter_domain.DimensionPt(4)
													s.FontSize = 11
													s.LineHeight = 15.95
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.PageBreakAfter = layouter_domain.PageBreakAvoid
													s.PageBreakInside = layouter_domain.PageBreakAvoid
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 4,
												},
												ContentX:      120.75,
												ContentY:      914.9750000000001,
												ContentWidth:  389.49062993,
												ContentHeight: 15.95,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 11
															s.LineHeight = 15.95
															s.LineHeightFactor = 1.45
															s.PageBreakAfter = layouter_domain.PageBreakAvoid
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Text:          "Full-Stack Developer",
														ContentX:      120.75,
														ContentY:      914.9750000000001,
														ContentWidth:  107.12109375,
														ContentHeight: 15.95,
														PageIndex:     1,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.PaddingLeft = 12
													s.FontSize = 9.5
													s.LineHeight = 13.775
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Padding: layouter_domain.BoxEdges{
													Left: 12,
												},
												Margin: layouter_domain.BoxEdges{
													Bottom: 3,
												},
												ContentX:      132.75,
												ContentY:      934.9250000000002,
												ContentWidth:  377.49062993,
												ContentHeight: 88.65,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      934.9250000000002,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      934.9250000000002,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Rebuilt the deployment pipeline so that every service shipped through the same",
																ContentX:      132.75,
																ContentY:      934.9250000000002,
																ContentWidth:  365.98828125,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "reviewed path, which cut release preparation from days to under an hour",
																ContentX:      132.75,
																ContentY:      948.7000000000002,
																ContentWidth:  334.546875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      965.4750000000001,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      965.4750000000001,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Ran the container platform and its monitoring through a period of fast growth,",
																ContentX:      132.75,
																ContentY:      965.4750000000001,
																ContentWidth:  360.64453125,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "keeping the service within its availability target",
																ContentX:      132.75,
																ContentY:      979.2500000000001,
																ContentWidth:  214.18359375,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      996.0250000000002,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      996.0250000000002,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Moved a legacy reporting system onto managed storage, checking every migrated",
																ContentX:      132.75,
																ContentY:      996.0250000000002,
																ContentWidth:  376.46484375,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "record against its source before the old system was retired",
																ContentX:      132.75,
																ContentY:      1009.8000000000002,
																ContentWidth:  268.04296875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginTop = layouter_domain.DimensionPt(4)
													s.FontSize = 9.5
													s.LineHeight = 13.775
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.PageBreakInside = layouter_domain.PageBreakAvoid
													s.LineHeightAuto = false
												}),
												ContentX:      120.75,
												ContentY:      1027.575,
												ContentWidth:  389.49062993,
												ContentHeight: 24.65,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxBlock,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 7.5
															s.LineHeight = 10.875
															s.LineHeightFactor = 1.45
															s.LetterSpacing = 0.76
															s.Display = layouter_domain.DisplayBlock
															s.BoxSizing = border - box
															s.TextTransform = layouter_domain.TextTransformUppercase
															s.LineHeightAuto = false
														}),
														ContentX:      120.75,
														ContentY:      1027.575,
														ContentWidth:  389.49062993,
														ContentHeight: 10.875,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
																	s.FontSize = 7.5
																	s.LineHeight = 10.875
																	s.LineHeightFactor = 1.45
																	s.LetterSpacing = 0.76
																	s.TextTransform = layouter_domain.TextTransformUppercase
																	s.LineHeightAuto = false
																}),
																Text:          "TECH",
																ContentX:      120.75,
																ContentY:      1027.575,
																ContentWidth:  21.6728125,
																ContentHeight: 10.875,
																PageIndex:     1,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxBlock,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayBlock
															s.BoxSizing = border - box
															s.LineHeightAuto = false
														}),
														ContentX:      120.75,
														ContentY:      1038.45,
														ContentWidth:  389.49062993,
														ContentHeight: 13.775,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.LineHeightAuto = false
																}),
																Text:          "Go, Kubernetes, Terraform, PostgreSQL, Kafka, Prometheus, Grafana, AWS, GCP",
																ContentX:      120.75,
																ContentY:      1038.45,
																ContentWidth:  364.01953125,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
												},
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxGrid,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.GridTemplateColumns = []layouter_domain.GridTrack{
										layouter_domain.GridTrack{
											Value: 108,
											Unit:  layouter_domain.GridTrackPoints,
										},
										layouter_domain.GridTrack{
											Value: 1,
											Unit:  layouter_domain.GridTrackFr,
										},
									}
									s.BorderTopColour = layouter_domain.NewRGBA(0.8901960784313725, 0.9019607843137255, 0.9176470588235294, 1)
									s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
									s.PaddingTop = 6
									s.PaddingBottom = 6
									s.BorderTopWidth = 0.75
									s.FontSize = 9.5
									s.LineHeight = 13.775
									s.LineHeightFactor = 1.45
									s.Display = layouter_domain.DisplayGrid
									s.BoxSizing = border - box
									s.BorderTopStyle = layouter_domain.BorderStyleSolid
									s.LineHeightAuto = false
								}),
								Padding: layouter_domain.BoxEdges{
									Top:    6,
									Bottom: 6,
								},
								Border: layouter_domain.BoxEdges{
									Top: 0.75,
								},
								ContentY:      1064.9750000000001,
								ContentWidth:  510.24062993,
								ContentHeight: 167.8,
								PageIndex:     1,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxGridItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.PaddingRight = 18
											s.FontSize = 9.5
											s.LineHeight = 13.775
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
											s.AlignSelf = layouter_domain.AlignSelfFlexStart
											s.PageBreakInside = layouter_domain.PageBreakAvoid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Right: 18,
										},
										ContentY:      1064.9750000000001,
										ContentWidth:  90,
										ContentHeight: 57.3,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.FontWeight = 700
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      1064.9750000000001,
												ContentWidth:  90,
												ContentHeight: 12.325,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.FontWeight = 700
															s.LineHeightAuto = false
														}),
														Text:          "2012-2014",
														ContentY:      1064.9750000000001,
														ContentWidth:  44.0390625,
														ContentHeight: 12.325,
														PageIndex:     1,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      1079.3000000000002,
												ContentWidth:  90,
												ContentHeight: 12.325,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Tailspin Services",
														ContentY:      1079.3000000000002,
														ContentWidth:  68.390625,
														ContentHeight: 12.325,
														PageIndex:     1,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2)
													s.FontSize = 8.5
													s.LineHeight = 12.325
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2,
												},
												ContentY:      1093.6250000000002,
												ContentWidth:  90,
												ContentHeight: 24.65,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Leeds, United",
														ContentY:      1093.6250000000002,
														ContentWidth:  57.7734375,
														ContentHeight: 12.325,
														PageIndex:     1,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 8.5
															s.LineHeight = 12.325
															s.LineHeightFactor = 1.45
															s.LineHeightAuto = false
														}),
														Text:          "Kingdom",
														ContentY:      1105.9500000000003,
														ContentWidth:  38.37890625,
														ContentHeight: 12.325,
														PageIndex:     1,
													},
												},
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxGridItem,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.8901960784313725, 0.9019607843137255, 0.9176470588235294, 1)
											s.PaddingLeft = 12
											s.BorderLeftWidth = 0.75
											s.FontSize = 9.5
											s.LineHeight = 13.775
											s.LineHeightFactor = 1.45
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
											s.BorderLeftStyle = layouter_domain.BorderStyleSolid
											s.LineHeightAuto = false
										}),
										Padding: layouter_domain.BoxEdges{
											Left: 12,
										},
										Border: layouter_domain.BoxEdges{
											Left: 0.75,
										},
										ContentX:      120.75,
										ContentY:      1064.9750000000001,
										ContentWidth:  389.49062993,
										ContentHeight: 167.8,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginBottom = layouter_domain.DimensionPt(4)
													s.FontSize = 11
													s.LineHeight = 15.95
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.PageBreakAfter = layouter_domain.PageBreakAvoid
													s.PageBreakInside = layouter_domain.PageBreakAvoid
													s.LineHeightAuto = false
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 4,
												},
												ContentX:      120.75,
												ContentY:      1064.9750000000001,
												ContentWidth:  389.49062993,
												ContentHeight: 15.95,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 11
															s.LineHeight = 15.95
															s.LineHeightFactor = 1.45
															s.PageBreakAfter = layouter_domain.PageBreakAvoid
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Text:          "Systems Administrator",
														ContentX:      120.75,
														ContentY:      1064.9750000000001,
														ContentWidth:  119.6484375,
														ContentHeight: 15.95,
														PageIndex:     1,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.PaddingLeft = 12
													s.FontSize = 9.5
													s.LineHeight = 13.775
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.LineHeightAuto = false
												}),
												Padding: layouter_domain.BoxEdges{
													Left: 12,
												},
												Margin: layouter_domain.BoxEdges{
													Bottom: 3,
												},
												ContentX:      132.75,
												ContentY:      1084.9250000000002,
												ContentWidth:  377.49062993,
												ContentHeight: 119.2,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      1084.9250000000002,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      1084.9250000000002,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Rebuilt the deployment pipeline so that every service shipped through the same",
																ContentX:      132.75,
																ContentY:      1084.9250000000002,
																ContentWidth:  365.98828125,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "reviewed path, which cut release preparation from days to under an hour",
																ContentX:      132.75,
																ContentY:      1098.7000000000003,
																ContentWidth:  334.546875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      1115.4750000000001,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      1115.4750000000001,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Ran the container platform and its monitoring through a period of fast growth,",
																ContentX:      132.75,
																ContentY:      1115.4750000000001,
																ContentWidth:  360.64453125,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "keeping the service within its availability target",
																ContentX:      132.75,
																ContentY:      1129.2500000000002,
																ContentWidth:  214.18359375,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      1146.025,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      1146.025,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Moved a legacy reporting system onto managed storage, checking every migrated",
																ContentX:      132.75,
																ContentY:      1146.025,
																ContentWidth:  376.46484375,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "record against its source before the old system was retired",
																ContentX:      132.75,
																ContentY:      1159.8000000000002,
																ContentWidth:  268.04296875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListItem,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.MarginBottom = layouter_domain.DimensionPt(3)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayListItem
															s.BoxSizing = border - box
															s.PageBreakInside = layouter_domain.PageBreakAvoid
															s.LineHeightAuto = false
														}),
														Margin: layouter_domain.BoxEdges{
															Bottom: 3,
														},
														ContentX:      132.75,
														ContentY:      1176.5750000000003,
														ContentWidth:  377.49062993,
														ContentHeight: 27.55,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListMarker,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(3)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.BoxSizing = border - box
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "• ",
																ContentX:      126.55078125,
																ContentY:      1176.5750000000003,
																ContentWidth:  6.19921875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "Introduced load testing to the release process, so capacity problems surfaced in",
																ContentX:      132.75,
																ContentY:      1176.5750000000003,
																ContentWidth:  365.26171875,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.PageBreakInside = layouter_domain.PageBreakAvoid
																	s.LineHeightAuto = false
																}),
																Text:          "staging instead of during traffic peaks",
																ContentX:      132.75,
																ContentY:      1190.3500000000004,
																ContentWidth:  173.98828125,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxBlock,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
													s.MarginTop = layouter_domain.DimensionPt(4)
													s.FontSize = 9.5
													s.LineHeight = 13.775
													s.LineHeightFactor = 1.45
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.PageBreakInside = layouter_domain.PageBreakAvoid
													s.LineHeightAuto = false
												}),
												ContentX:      120.75,
												ContentY:      1208.1250000000002,
												ContentWidth:  389.49062993,
												ContentHeight: 24.65,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxBlock,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
															s.FontSize = 7.5
															s.LineHeight = 10.875
															s.LineHeightFactor = 1.45
															s.LetterSpacing = 0.76
															s.Display = layouter_domain.DisplayBlock
															s.BoxSizing = border - box
															s.TextTransform = layouter_domain.TextTransformUppercase
															s.LineHeightAuto = false
														}),
														ContentX:      120.75,
														ContentY:      1208.1250000000002,
														ContentWidth:  389.49062993,
														ContentHeight: 10.875,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.3568627450980392, 0.39215686274509803, 0.4392156862745098, 1)
																	s.FontSize = 7.5
																	s.LineHeight = 10.875
																	s.LineHeightFactor = 1.45
																	s.LetterSpacing = 0.76
																	s.TextTransform = layouter_domain.TextTransformUppercase
																	s.LineHeightAuto = false
																}),
																Text:          "TECH",
																ContentX:      120.75,
																ContentY:      1208.1250000000002,
																ContentWidth:  21.6728125,
																ContentHeight: 10.875,
																PageIndex:     1,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxBlock,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
															s.FontSize = 9.5
															s.LineHeight = 13.775
															s.LineHeightFactor = 1.45
															s.Display = layouter_domain.DisplayBlock
															s.BoxSizing = border - box
															s.LineHeightAuto = false
														}),
														ContentX:      120.75,
														ContentY:      1219.0000000000002,
														ContentWidth:  389.49062993,
														ContentHeight: 13.775,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.Colour = layouter_domain.NewRGBA(0.13725490196078433, 0.16470588235294117, 0.19215686274509805, 1)
																	s.FontSize = 9.5
																	s.LineHeight = 13.775
																	s.LineHeightFactor = 1.45
																	s.LineHeightAuto = false
																}),
																Text:          "Go, Kubernetes, Terraform, PostgreSQL, Kafka, Prometheus, Grafana, AWS, GCP",
																ContentX:      120.75,
																ContentY:      1219.0000000000002,
																ContentWidth:  364.01953125,
																ContentHeight: 13.775,
																PageIndex:     1,
															},
														},
													},
												},
											},
										},
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
