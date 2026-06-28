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
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				ContentWidth:  595.28,
				ContentHeight: 1342.85,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontFamily = "Helvetica"
							s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
							s.Width = layouter_domain.DimensionPt(416.25)
							s.PaddingTop = 15
							s.PaddingRight = 15
							s.PaddingLeft = 15
							s.FontSize = 9
							s.LineHeight = 12.6
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
						}),
						Padding: layouter_domain.BoxEdges{
							Top:   15,
							Right: 15,
							Left:  15,
						},
						ContentX:      15,
						ContentY:      15,
						ContentWidth:  386.25,
						ContentHeight: 30.75,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.7254901960784313, 0.10980392156862745, 0.10980392156862745, 1)
									s.MarginBottom = layouter_domain.DimensionPt(4.5)
									s.FontSize = 9.75
									s.LineHeight = 13.649999999999999
									s.FontWeight = 600
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 4.5,
								},
								ContentX:      15,
								ContentY:      15,
								ContentWidth:  386.25,
								ContentHeight: 13.649999999999999,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.7254901960784313, 0.10980392156862745, 0.10980392156862745, 1)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
										}),
										Text:          "F · direct p-html in the PDF template (no partial)",
										ContentX:      15,
										ContentY:      15,
										ContentWidth:  216.9140625,
										ContentHeight: 13.649999999999999,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								ContentX:      15,
								ContentY:      33.15,
								ContentWidth:  386.25,
								ContentHeight: 12.6,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
										}),
										Text:          "Direct ",
										ContentX:      15,
										ContentY:      33.15,
										ContentWidth:  27.421875,
										ContentHeight: 12.6,
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxInline,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.FontWeight = 700
											s.BoxSizing = border - box
										}),
										ContentX:      42.421875,
										ContentY:      33.15,
										ContentWidth:  19.67578125,
										ContentHeight: 12.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.FontWeight = 700
												}),
												Text:          "bold",
												ContentX:      42.421875,
												ContentY:      33.15,
												ContentWidth:  19.67578125,
												ContentHeight: 12.6,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
										}),
										Text:          " p-html in the PDF template itself — no partial involved.",
										ContentX:      62.09765625,
										ContentY:      33.15,
										ContentWidth:  233.2734375,
										ContentHeight: 12.6,
									},
								},
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.FontFamily = "Helvetica"
							s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
							s.Width = layouter_domain.DimensionPt(416.25)
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
						ContentY:      60.75,
						ContentWidth:  386.25,
						ContentHeight: 1267.1,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.06666666666666667, 0.0784313725490196, 0.09411764705882353, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.FontSize = 15
									s.LineHeight = 21
									s.FontWeight = 700
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      60.75,
								ContentWidth:  386.25,
								ContentHeight: 21,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.06666666666666667, 0.0784313725490196, 0.09411764705882353, 1)
											s.FontSize = 15
											s.LineHeight = 21
											s.FontWeight = 700
										}),
										Text:          "Template directive matrix",
										ContentX:      15,
										ContentY:      60.75,
										ContentWidth:  192.984375,
										ContentHeight: 21,
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      93.75,
								ContentWidth:  386.25,
								ContentHeight: 64.94999999999999,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      93.75,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "A · direct p-for + interpolation",
												ContentX:      15,
												ContentY:      93.75,
												ContentWidth:  134.9765625,
												ContentHeight: 13.649999999999999,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.PaddingLeft = 15
											s.FontSize = 9
											s.LineHeight = 12.6
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Padding: layouter_domain.BoxEdges{
											Left: 15,
										},
										Margin: layouter_domain.BoxEdges{
											Bottom: 2.25,
										},
										ContentX:      30,
										ContentY:      111.9,
										ContentWidth:  371.25,
										ContentHeight: 42.3,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxListItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2.25)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.Display = layouter_domain.DisplayListItem
													s.BoxSizing = border - box
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2.25,
												},
												ContentX:      30,
												ContentY:      111.9,
												ContentWidth:  371.25,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListMarker,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.MarginBottom = layouter_domain.DimensionPt(2.25)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.BoxSizing = border - box
														}),
														Text:          "• ",
														ContentX:      24.26953125,
														ContentY:      111.9,
														ContentWidth:  5.73046875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Alpha",
														ContentX:      30,
														ContentY:      111.9,
														ContentWidth:  24.22265625,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxListItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2.25)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.Display = layouter_domain.DisplayListItem
													s.BoxSizing = border - box
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2.25,
												},
												ContentX:      30,
												ContentY:      126.75,
												ContentWidth:  371.25,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListMarker,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.MarginBottom = layouter_domain.DimensionPt(2.25)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.BoxSizing = border - box
														}),
														Text:          "• ",
														ContentX:      24.26953125,
														ContentY:      126.75,
														ContentWidth:  5.73046875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Beta",
														ContentX:      30,
														ContentY:      126.75,
														ContentWidth:  19.21875,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxListItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2.25)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.Display = layouter_domain.DisplayListItem
													s.BoxSizing = border - box
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2.25,
												},
												ContentX:      30,
												ContentY:      141.6,
												ContentWidth:  371.25,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListMarker,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.MarginBottom = layouter_domain.DimensionPt(2.25)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.BoxSizing = border - box
														}),
														Text:          "• ",
														ContentX:      24.26953125,
														ContentY:      141.6,
														ContentWidth:  5.73046875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Gamma",
														ContentX:      30,
														ContentY:      141.6,
														ContentWidth:  33.48046875,
														ContentHeight: 12.6,
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
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      180.45,
								ContentWidth:  386.25,
								ContentHeight: 64.94999999999999,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      180.45,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "B · template p-for + interpolation",
												ContentX:      15,
												ContentY:      180.45,
												ContentWidth:  150.1640625,
												ContentHeight: 13.649999999999999,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.PaddingLeft = 15
											s.FontSize = 9
											s.LineHeight = 12.6
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Padding: layouter_domain.BoxEdges{
											Left: 15,
										},
										Margin: layouter_domain.BoxEdges{
											Bottom: 2.25,
										},
										ContentX:      30,
										ContentY:      198.6,
										ContentWidth:  371.25,
										ContentHeight: 42.3,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxListItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2.25)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.Display = layouter_domain.DisplayListItem
													s.BoxSizing = border - box
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2.25,
												},
												ContentX:      30,
												ContentY:      198.6,
												ContentWidth:  371.25,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListMarker,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.MarginBottom = layouter_domain.DimensionPt(2.25)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.BoxSizing = border - box
														}),
														Text:          "• ",
														ContentX:      24.26953125,
														ContentY:      198.6,
														ContentWidth:  5.73046875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Alpha",
														ContentX:      30,
														ContentY:      198.6,
														ContentWidth:  24.22265625,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxListItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2.25)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.Display = layouter_domain.DisplayListItem
													s.BoxSizing = border - box
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2.25,
												},
												ContentX:      30,
												ContentY:      213.45,
												ContentWidth:  371.25,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListMarker,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.MarginBottom = layouter_domain.DimensionPt(2.25)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.BoxSizing = border - box
														}),
														Text:          "• ",
														ContentX:      24.26953125,
														ContentY:      213.45,
														ContentWidth:  5.73046875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Beta",
														ContentX:      30,
														ContentY:      213.45,
														ContentWidth:  19.21875,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxListItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2.25)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.Display = layouter_domain.DisplayListItem
													s.BoxSizing = border - box
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2.25,
												},
												ContentX:      30,
												ContentY:      228.29999999999998,
												ContentWidth:  371.25,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListMarker,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.MarginBottom = layouter_domain.DimensionPt(2.25)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.BoxSizing = border - box
														}),
														Text:          "• ",
														ContentX:      24.26953125,
														ContentY:      228.29999999999998,
														ContentWidth:  5.73046875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Gamma",
														ContentX:      30,
														ContentY:      228.29999999999998,
														ContentWidth:  33.48046875,
														ContentHeight: 12.6,
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
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      267.15,
								ContentWidth:  386.25,
								ContentHeight: 30.75,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      267.15,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "C · single element p-html (no loop)",
												ContentX:      15,
												ContentY:      267.15,
												ContentWidth:  157.265625,
												ContentHeight: 13.649999999999999,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										ContentX:      15,
										ContentY:      285.29999999999995,
										ContentWidth:  386.25,
										ContentHeight: 12.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          "One line with ",
												ContentX:      15,
												ContentY:      285.29999999999995,
												ContentWidth:  58.1953125,
												ContentHeight: 12.6,
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxInline,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.FontWeight = 700
													s.BoxSizing = border - box
												}),
												ContentX:      73.1953125,
												ContentY:      285.29999999999995,
												ContentWidth:  19.67578125,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.FontWeight = 700
														}),
														Text:          "bold",
														ContentX:      73.1953125,
														ContentY:      285.29999999999995,
														ContentWidth:  19.67578125,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          " and ",
												ContentX:      92.87109375,
												ContentY:      285.29999999999995,
												ContentWidth:  20.8359375,
												ContentHeight: 12.6,
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxInline,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.BoxSizing = border - box
													s.FontStyle = layouter_domain.FontStyleItalic
												}),
												ContentX:      113.70703125,
												ContentY:      285.29999999999995,
												ContentWidth:  19.58203125,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.FontStyle = layouter_domain.FontStyleItalic
														}),
														Text:          "italic",
														ContentX:      113.70703125,
														ContentY:      285.29999999999995,
														ContentWidth:  19.58203125,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          " spans.",
												ContentX:      133.2890625,
												ContentY:      285.29999999999995,
												ContentWidth:  29.53125,
												ContentHeight: 12.6,
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      319.65,
								ContentWidth:  386.25,
								ContentHeight: 64.94999999999999,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      319.65,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "D · template p-for + p-html",
												ContentX:      15,
												ContentY:      319.65,
												ContentWidth:  121.828125,
												ContentHeight: 13.649999999999999,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.PaddingLeft = 15
											s.FontSize = 9
											s.LineHeight = 12.6
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Padding: layouter_domain.BoxEdges{
											Left: 15,
										},
										Margin: layouter_domain.BoxEdges{
											Bottom: 2.25,
										},
										ContentX:      30,
										ContentY:      337.79999999999995,
										ContentWidth:  371.25,
										ContentHeight: 42.3,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxListItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2.25)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.Display = layouter_domain.DisplayListItem
													s.BoxSizing = border - box
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2.25,
												},
												ContentX:      30,
												ContentY:      337.79999999999995,
												ContentWidth:  371.25,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListMarker,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.MarginBottom = layouter_domain.DimensionPt(2.25)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.BoxSizing = border - box
														}),
														Text:          "• ",
														ContentX:      24.26953125,
														ContentY:      337.79999999999995,
														ContentWidth:  5.73046875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Alpha — ",
														ContentX:      30,
														ContentY:      337.79999999999995,
														ContentWidth:  37.91015625,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxInline,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.FontWeight = 700
															s.BoxSizing = border - box
														}),
														ContentX:      67.91015625,
														ContentY:      337.79999999999995,
														ContentWidth:  19.67578125,
														ContentHeight: 12.6,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.FontFamily = "Helvetica"
																	s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																	s.FontSize = 9
																	s.LineHeight = 12.6
																	s.FontWeight = 700
																}),
																Text:          "bold",
																ContentX:      67.91015625,
																ContentY:      337.79999999999995,
																ContentWidth:  19.67578125,
																ContentHeight: 12.6,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          " emphasis",
														ContentX:      87.5859375,
														ContentY:      337.79999999999995,
														ContentWidth:  42.92578125,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxListItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2.25)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.Display = layouter_domain.DisplayListItem
													s.BoxSizing = border - box
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2.25,
												},
												ContentX:      30,
												ContentY:      352.65,
												ContentWidth:  371.25,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListMarker,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.MarginBottom = layouter_domain.DimensionPt(2.25)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.BoxSizing = border - box
														}),
														Text:          "• ",
														ContentX:      24.26953125,
														ContentY:      352.65,
														ContentWidth:  5.73046875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Beta — ",
														ContentX:      30,
														ContentY:      352.65,
														ContentWidth:  32.90625,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxInline,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.FontWeight = 700
															s.BoxSizing = border - box
														}),
														ContentX:      62.90625,
														ContentY:      352.65,
														ContentWidth:  19.67578125,
														ContentHeight: 12.6,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.FontFamily = "Helvetica"
																	s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																	s.FontSize = 9
																	s.LineHeight = 12.6
																	s.FontWeight = 700
																}),
																Text:          "bold",
																ContentX:      62.90625,
																ContentY:      352.65,
																ContentWidth:  19.67578125,
																ContentHeight: 12.6,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          " emphasis",
														ContentX:      82.58203125,
														ContentY:      352.65,
														ContentWidth:  42.92578125,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxListItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2.25)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.Display = layouter_domain.DisplayListItem
													s.BoxSizing = border - box
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2.25,
												},
												ContentX:      30,
												ContentY:      367.49999999999994,
												ContentWidth:  371.25,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListMarker,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.MarginBottom = layouter_domain.DimensionPt(2.25)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.BoxSizing = border - box
														}),
														Text:          "• ",
														ContentX:      24.26953125,
														ContentY:      367.49999999999994,
														ContentWidth:  5.73046875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Gamma — ",
														ContentX:      30,
														ContentY:      367.49999999999994,
														ContentWidth:  47.16796875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxInline,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.FontWeight = 700
															s.BoxSizing = border - box
														}),
														ContentX:      77.16796875,
														ContentY:      367.49999999999994,
														ContentWidth:  19.67578125,
														ContentHeight: 12.6,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.FontFamily = "Helvetica"
																	s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																	s.FontSize = 9
																	s.LineHeight = 12.6
																	s.FontWeight = 700
																}),
																Text:          "bold",
																ContentX:      77.16796875,
																ContentY:      367.49999999999994,
																ContentWidth:  19.67578125,
																ContentHeight: 12.6,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          " emphasis",
														ContentX:      96.84375,
														ContentY:      367.49999999999994,
														ContentWidth:  42.92578125,
														ContentHeight: 12.6,
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
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      406.34999999999997,
								ContentWidth:  386.25,
								ContentHeight: 64.94999999999999,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      406.34999999999997,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "E · direct p-for + p-html (co-located)",
												ContentX:      15,
												ContentY:      406.34999999999997,
												ContentWidth:  160.734375,
												ContentHeight: 13.649999999999999,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.PaddingLeft = 15
											s.FontSize = 9
											s.LineHeight = 12.6
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Padding: layouter_domain.BoxEdges{
											Left: 15,
										},
										Margin: layouter_domain.BoxEdges{
											Bottom: 2.25,
										},
										ContentX:      30,
										ContentY:      424.49999999999994,
										ContentWidth:  371.25,
										ContentHeight: 42.3,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxListItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2.25)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.Display = layouter_domain.DisplayListItem
													s.BoxSizing = border - box
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2.25,
												},
												ContentX:      30,
												ContentY:      424.49999999999994,
												ContentWidth:  371.25,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListMarker,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.MarginBottom = layouter_domain.DimensionPt(2.25)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.BoxSizing = border - box
														}),
														Text:          "• ",
														ContentX:      24.26953125,
														ContentY:      424.49999999999994,
														ContentWidth:  5.73046875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Alpha — ",
														ContentX:      30,
														ContentY:      424.49999999999994,
														ContentWidth:  37.91015625,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxInline,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.FontWeight = 700
															s.BoxSizing = border - box
														}),
														ContentX:      67.91015625,
														ContentY:      424.49999999999994,
														ContentWidth:  19.67578125,
														ContentHeight: 12.6,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.FontFamily = "Helvetica"
																	s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																	s.FontSize = 9
																	s.LineHeight = 12.6
																	s.FontWeight = 700
																}),
																Text:          "bold",
																ContentX:      67.91015625,
																ContentY:      424.49999999999994,
																ContentWidth:  19.67578125,
																ContentHeight: 12.6,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          " emphasis",
														ContentX:      87.5859375,
														ContentY:      424.49999999999994,
														ContentWidth:  42.92578125,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxListItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2.25)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.Display = layouter_domain.DisplayListItem
													s.BoxSizing = border - box
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2.25,
												},
												ContentX:      30,
												ContentY:      439.34999999999997,
												ContentWidth:  371.25,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListMarker,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.MarginBottom = layouter_domain.DimensionPt(2.25)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.BoxSizing = border - box
														}),
														Text:          "• ",
														ContentX:      24.26953125,
														ContentY:      439.34999999999997,
														ContentWidth:  5.73046875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Beta — ",
														ContentX:      30,
														ContentY:      439.34999999999997,
														ContentWidth:  32.90625,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxInline,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.FontWeight = 700
															s.BoxSizing = border - box
														}),
														ContentX:      62.90625,
														ContentY:      439.34999999999997,
														ContentWidth:  19.67578125,
														ContentHeight: 12.6,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.FontFamily = "Helvetica"
																	s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																	s.FontSize = 9
																	s.LineHeight = 12.6
																	s.FontWeight = 700
																}),
																Text:          "bold",
																ContentX:      62.90625,
																ContentY:      439.34999999999997,
																ContentWidth:  19.67578125,
																ContentHeight: 12.6,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          " emphasis",
														ContentX:      82.58203125,
														ContentY:      439.34999999999997,
														ContentWidth:  42.92578125,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxListItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.MarginBottom = layouter_domain.DimensionPt(2.25)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.Display = layouter_domain.DisplayListItem
													s.BoxSizing = border - box
												}),
												Margin: layouter_domain.BoxEdges{
													Bottom: 2.25,
												},
												ContentX:      30,
												ContentY:      454.19999999999993,
												ContentWidth:  371.25,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxListMarker,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.MarginBottom = layouter_domain.DimensionPt(2.25)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.BoxSizing = border - box
														}),
														Text:          "• ",
														ContentX:      24.26953125,
														ContentY:      454.19999999999993,
														ContentWidth:  5.73046875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Gamma — ",
														ContentX:      30,
														ContentY:      454.19999999999993,
														ContentWidth:  47.16796875,
														ContentHeight: 12.6,
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxInline,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.FontWeight = 700
															s.BoxSizing = border - box
														}),
														ContentX:      77.16796875,
														ContentY:      454.19999999999993,
														ContentWidth:  19.67578125,
														ContentHeight: 12.6,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.FontFamily = "Helvetica"
																	s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																	s.FontSize = 9
																	s.LineHeight = 12.6
																	s.FontWeight = 700
																}),
																Text:          "bold",
																ContentX:      77.16796875,
																ContentY:      454.19999999999993,
																ContentWidth:  19.67578125,
																ContentHeight: 12.6,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          " emphasis",
														ContentX:      96.84375,
														ContentY:      454.19999999999993,
														ContentWidth:  42.92578125,
														ContentHeight: 12.6,
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
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      493.04999999999995,
								ContentWidth:  386.25,
								ContentHeight: 30.75,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      493.04999999999995,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "G · inline span inside a block (control)",
												ContentX:      15,
												ContentY:      493.04999999999995,
												ContentWidth:  169.93359375,
												ContentHeight: 13.649999999999999,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										ContentX:      15,
										ContentY:      511.19999999999993,
										ContentWidth:  386.25,
										ContentHeight: 12.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          "Prefix text ",
												ContentX:      15,
												ContentY:      511.19999999999993,
												ContentWidth:  45.0703125,
												ContentHeight: 12.6,
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxInline,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.FontWeight = 700
													s.BoxSizing = border - box
												}),
												ContentX:      60.0703125,
												ContentY:      511.19999999999993,
												ContentWidth:  48.92578125,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.FontWeight = 700
														}),
														Text:          "inline span",
														ContentX:      60.0703125,
														ContentY:      511.19999999999993,
														ContentWidth:  48.92578125,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          " then suffix — should stay on one line.",
												ContentX:      108.99609375,
												ContentY:      511.19999999999993,
												ContentWidth:  161.109375,
												ContentHeight: 12.6,
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      545.55,
								ContentWidth:  386.25,
								ContentHeight: 30.75,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      545.55,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "H · flex: plain text item + flex-grow rule",
												ContentX:      15,
												ContentY:      545.55,
												ContentWidth:  176.0625,
												ContentHeight: 13.649999999999999,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxFlex,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.Display = layouter_domain.DisplayFlex
											s.BoxSizing = border - box
											s.AlignItems = layouter_domain.AlignItemsCentre
										}),
										ContentX:      15,
										ContentY:      563.6999999999999,
										ContentWidth:  386.25,
										ContentHeight: 12.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxFlexItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.BoxSizing = border - box
												}),
												ContentX:      15,
												ContentY:      563.6999999999999,
												ContentWidth:  53.56640625,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Section label",
														ContentX:      15,
														ContentY:      563.6999999999999,
														ContentWidth:  53.56640625,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxFlexItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.BorderBottomColour = layouter_domain.NewRGBA(0.611764705882353, 0.6392156862745098, 0.6862745098039216, 1)
													s.Height = layouter_domain.DimensionPt(0)
													s.MarginLeft = layouter_domain.DimensionPt(7.5)
													s.FlexBasis = layouter_domain.DimensionPt(0)
													s.BorderBottomWidth = 0.75
													s.FontSize = 9
													s.LineHeight = 12.6
													s.FlexGrow = 1
													s.BoxSizing = border - box
													s.BorderBottomStyle = layouter_domain.BorderStyleSolid
												}),
												Border: layouter_domain.BoxEdges{
													Bottom: 0.75,
												},
												Margin: layouter_domain.BoxEdges{
													Left: 7.5,
												},
												ContentX:     76.06640625,
												ContentY:     569.6249999999999,
												ContentWidth: 325.18359375,
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      598.05,
								ContentWidth:  386.25,
								ContentHeight: 30.75,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      598.05,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "I · flex: nested inline span + text + rule (the CV heading pattern)",
												ContentX:      15,
												ContentY:      598.05,
												ContentWidth:  287.47265625,
												ContentHeight: 13.649999999999999,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxFlex,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.Display = layouter_domain.DisplayFlex
											s.BoxSizing = border - box
											s.AlignItems = layouter_domain.AlignItemsCentre
										}),
										ContentX:      15,
										ContentY:      616.1999999999999,
										ContentWidth:  386.25,
										ContentHeight: 12.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxFlexItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.BoxSizing = border - box
												}),
												ContentX:      15,
												ContentY:      616.1999999999999,
												ContentWidth:  46.44140625,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxInline,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.BoxSizing = border - box
														}),
														ContentX:      15,
														ContentY:      616.1999999999999,
														ContentWidth:  15.29296875,
														ContentHeight: 12.6,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxTextRun,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.FontFamily = "Helvetica"
																	s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
																	s.FontSize = 9
																	s.LineHeight = 12.6
																}),
																Text:          "Exp",
																ContentX:      15,
																ContentY:      616.1999999999999,
																ContentWidth:  15.29296875,
																ContentHeight: 12.6,
															},
														},
													},
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "erience",
														ContentX:      30.29296875,
														ContentY:      616.1999999999999,
														ContentWidth:  31.1484375,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxFlexItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.BorderBottomColour = layouter_domain.NewRGBA(0.611764705882353, 0.6392156862745098, 0.6862745098039216, 1)
													s.Height = layouter_domain.DimensionPt(0)
													s.MarginLeft = layouter_domain.DimensionPt(7.5)
													s.FlexBasis = layouter_domain.DimensionPt(0)
													s.BorderBottomWidth = 0.75
													s.FontSize = 9
													s.LineHeight = 12.6
													s.FlexGrow = 1
													s.BoxSizing = border - box
													s.BorderBottomStyle = layouter_domain.BorderStyleSolid
												}),
												Border: layouter_domain.BoxEdges{
													Bottom: 0.75,
												},
												Margin: layouter_domain.BoxEdges{
													Left: 7.5,
												},
												ContentX:     68.94140625,
												ContentY:     622.1249999999999,
												ContentWidth: 332.30859375,
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      650.55,
								ContentWidth:  386.25,
								ContentHeight: 30.75,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      650.55,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "J · flex row: label + right-aligned value (the CV aside pattern)",
												ContentX:      15,
												ContentY:      650.55,
												ContentWidth:  271.72265625,
												ContentHeight: 13.649999999999999,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxFlex,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.Display = layouter_domain.DisplayFlex
											s.BoxSizing = border - box
										}),
										ContentX:      15,
										ContentY:      668.6999999999999,
										ContentWidth:  386.25,
										ContentHeight: 12.6,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxFlexItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.BoxSizing = border - box
												}),
												ContentX:      15,
												ContentY:      668.6999999999999,
												ContentWidth:  81.52734375,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Organisation name",
														ContentX:      15,
														ContentY:      668.6999999999999,
														ContentWidth:  81.52734375,
														ContentHeight: 12.6,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxFlexItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.MarginLeft = layouter_domain.DimensionAuto()
													s.FontSize = 9
													s.LineHeight = 12.6
													s.BoxSizing = border - box
													s.TextAlign = layouter_domain.TextAlignRight
												}),
												ContentX:      323.44921875,
												ContentY:      668.6999999999999,
												ContentWidth:  77.80078125,
												ContentHeight: 12.6,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.TextAlign = layouter_domain.TextAlignRight
														}),
														Text:          "Right-aligned date",
														ContentX:      323.44921875,
														ContentY:      668.6999999999999,
														ContentWidth:  77.80078125,
														ContentHeight: 12.6,
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
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      703.05,
								ContentWidth:  386.25,
								ContentHeight: 34.15,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      703.05,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "K · raw inline <svg>",
												ContentX:      15,
												ContentY:      703.05,
												ContentWidth:  87.64453125,
												ContentHeight: 13.649999999999999,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										ContentX:      15,
										ContentY:      721.1999999999999,
										ContentWidth:  386.25,
										ContentHeight: 16,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxReplaced,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.BoxSizing = border - box
												}),
												ContentX:        15,
												ContentY:        721.1999999999999,
												ContentWidth:    16,
												ContentHeight:   16,
												IntrinsicWidth:  16,
												IntrinsicHeight: 16,
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          " green dot via raw inline svg",
												ContentX:      31,
												ContentY:      721.1999999999999,
												ContentWidth:  118.265625,
												ContentHeight: 12.6,
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      758.9499999999999,
								ContentWidth:  386.25,
								ContentHeight: 118.15,
								PageIndex:     1,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      758.9499999999999,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "L · piko:svg inlined from an asset",
												ContentX:      15,
												ContentY:      758.9499999999999,
												ContentWidth:  149.4140625,
												ContentHeight: 13.649999999999999,
												PageIndex:     1,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										ContentX:      15,
										ContentY:      777.0999999999999,
										ContentWidth:  386.25,
										ContentHeight: 100,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxReplaced,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.BoxSizing = border - box
												}),
												ContentX:        15,
												ContentY:        777.0999999999999,
												ContentWidth:    100,
												ContentHeight:   100,
												IntrinsicWidth:  100,
												IntrinsicHeight: 100,
												PageIndex:       1,
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          " green dot via piko:svg",
												ContentX:      115,
												ContentY:      777.0999999999999,
												ContentWidth:  95.26171875,
												ContentHeight: 12.6,
												PageIndex:     1,
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      898.8499999999999,
								ContentWidth:  386.25,
								ContentHeight: 60.449999999999996,
								PageIndex:     1,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      898.8499999999999,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "S · p-html wrapping multi-word bold in a flex column",
												ContentX:      15,
												ContentY:      898.8499999999999,
												ContentWidth:  239.84765625,
												ContentHeight: 13.649999999999999,
												PageIndex:     1,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxFlex,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.RowGap = 18
											s.ColumnGap = 18
											s.Display = layouter_domain.DisplayFlex
											s.BoxSizing = border - box
										}),
										ContentX:      15,
										ContentY:      916.9999999999999,
										ContentWidth:  386.25,
										ContentHeight: 42.3,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxFlexItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FlexBasis = layouter_domain.DimensionPt(0)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.FlexGrow = 1
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
												}),
												ContentX:      15,
												ContentY:      916.9999999999999,
												ContentWidth:  285.75,
												ContentHeight: 42.3,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxBlock,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.PaddingLeft = 15
															s.FontSize = 9
															s.LineHeight = 12.6
															s.Display = layouter_domain.DisplayBlock
															s.BoxSizing = border - box
														}),
														Padding: layouter_domain.BoxEdges{
															Left: 15,
														},
														Margin: layouter_domain.BoxEdges{
															Bottom: 2.25,
														},
														ContentX:      30,
														ContentY:      916.9999999999999,
														ContentWidth:  270.75,
														ContentHeight: 37.8,
														PageIndex:     1,
														Children: []*layouter_domain.LayoutBox{
															&layouter_domain.LayoutBox{
																Type: layouter_domain.BoxListItem,
																Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																	s.FontFamily = "Helvetica"
																	s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																	s.MarginBottom = layouter_domain.DimensionPt(2.25)
																	s.FontSize = 9
																	s.LineHeight = 12.6
																	s.Display = layouter_domain.DisplayListItem
																	s.BoxSizing = border - box
																}),
																Margin: layouter_domain.BoxEdges{
																	Bottom: 2.25,
																},
																ContentX:      30,
																ContentY:      916.9999999999999,
																ContentWidth:  270.75,
																ContentHeight: 37.8,
																PageIndex:     1,
																Children: []*layouter_domain.LayoutBox{
																	&layouter_domain.LayoutBox{
																		Type: layouter_domain.BoxListMarker,
																		Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																			s.FontFamily = "Helvetica"
																			s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																			s.MarginBottom = layouter_domain.DimensionPt(2.25)
																			s.FontSize = 9
																			s.LineHeight = 12.6
																			s.BoxSizing = border - box
																		}),
																		Text:          "• ",
																		ContentX:      24.26953125,
																		ContentY:      916.9999999999999,
																		ContentWidth:  5.73046875,
																		ContentHeight: 12.6,
																		PageIndex:     1,
																	},
																	&layouter_domain.LayoutBox{
																		Type: layouter_domain.BoxTextRun,
																		Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																			s.FontFamily = "Helvetica"
																			s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																			s.FontSize = 9
																			s.LineHeight = 12.6
																		}),
																		Text:          "Created a golden path for developers by standardising Helm",
																		ContentX:      30,
																		ContentY:      916.9999999999999,
																		ContentWidth:  254.90625,
																		ContentHeight: 12.6,
																		PageIndex:     1,
																	},
																	&layouter_domain.LayoutBox{
																		Type: layouter_domain.BoxTextRun,
																		Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																			s.FontFamily = "Helvetica"
																			s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																			s.FontSize = 9
																			s.LineHeight = 12.6
																		}),
																		Text:          "charts and CI/CD pipelines, which enabled ",
																		ContentX:      30,
																		ContentY:      929.5999999999999,
																		ContentWidth:  180.234375,
																		ContentHeight: 12.6,
																		PageIndex:     1,
																	},
																	&layouter_domain.LayoutBox{
																		Type: layouter_domain.BoxInline,
																		Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																			s.FontFamily = "Helvetica"
																			s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																			s.FontSize = 9
																			s.LineHeight = 12.6
																			s.FontWeight = 700
																			s.BoxSizing = border - box
																		}),
																		ContentX:      210.234375,
																		ContentY:      929.5999999999999,
																		ContentWidth:  -129.5390625,
																		ContentHeight: 12.6,
																		PageIndex:     1,
																		Children: []*layouter_domain.LayoutBox{
																			&layouter_domain.LayoutBox{
																				Type: layouter_domain.BoxTextRun,
																				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																					s.FontFamily = "Helvetica"
																					s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																					s.FontSize = 9
																					s.LineHeight = 12.6
																					s.FontWeight = 700
																				}),
																				Text:          "developer",
																				ContentX:      210.234375,
																				ContentY:      929.5999999999999,
																				ContentWidth:  44.765625,
																				ContentHeight: 12.6,
																				PageIndex:     1,
																			},
																			&layouter_domain.LayoutBox{
																				Type: layouter_domain.BoxTextRun,
																				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																					s.FontFamily = "Helvetica"
																					s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																					s.FontSize = 9
																					s.LineHeight = 12.6
																					s.FontWeight = 700
																				}),
																				Text:          "self-service",
																				ContentX:      30,
																				ContentY:      942.1999999999999,
																				ContentWidth:  50.6953125,
																				ContentHeight: 12.6,
																				PageIndex:     1,
																			},
																		},
																	},
																	&layouter_domain.LayoutBox{
																		Type: layouter_domain.BoxTextRun,
																		Style: withStyle(func(s *layouter_domain.ComputedStyle) {
																			s.FontFamily = "Helvetica"
																			s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
																			s.FontSize = 9
																			s.LineHeight = 12.6
																		}),
																		Text:          " and reduced deployment friction across the team.",
																		ContentX:      80.6953125,
																		ContentY:      942.1999999999999,
																		ContentWidth:  212.9296875,
																		ContentHeight: 12.6,
																		PageIndex:     1,
																	},
																},
															},
														},
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxFlexItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FlexBasis = layouter_domain.DimensionPt(82.5)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.FlexShrink = 0
													s.Display = layouter_domain.DisplayBlock
													s.BoxSizing = border - box
													s.TextAlign = layouter_domain.TextAlignRight
												}),
												ContentX:      318.75,
												ContentY:      916.9999999999999,
												ContentWidth:  82.5,
												ContentHeight: 42.3,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FlexBasis = layouter_domain.DimensionPt(82.5)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.FlexShrink = 0
															s.TextAlign = layouter_domain.TextAlignRight
														}),
														Text:          "aside",
														ContentX:      378.9609375,
														ContentY:      916.9999999999999,
														ContentWidth:  22.2890625,
														ContentHeight: 12.6,
														PageIndex:     1,
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
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      981.05,
								ContentWidth:  386.25,
								ContentHeight: 30.75,
								PageIndex:     1,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      981.05,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "M · p-if (show / hide)",
												ContentX:      15,
												ContentY:      981.05,
												ContentWidth:  92.73046875,
												ContentHeight: 13.649999999999999,
												PageIndex:     1,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										ContentX:      15,
										ContentY:      999.1999999999999,
										ContentWidth:  386.25,
										ContentHeight: 12.6,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          "VISIBLE — p-if true rendered this line.",
												ContentX:      15,
												ContentY:      999.1999999999999,
												ContentWidth:  158.578125,
												ContentHeight: 12.6,
												PageIndex:     1,
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      1033.55,
								ContentWidth:  386.25,
								ContentHeight: 30.75,
								PageIndex:     1,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      1033.55,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "N · :class bound from state",
												ContentX:      15,
												ContentY:      1033.55,
												ContentWidth:  122.1328125,
												ContentHeight: 13.649999999999999,
												PageIndex:     1,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.FontWeight = 700
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										ContentX:      15,
										ContentY:      1051.7,
										ContentWidth:  386.25,
										ContentHeight: 12.6,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.FontWeight = 700
												}),
												Text:          "Coloured via a state-bound class binding.",
												ContentX:      15,
												ContentY:      1051.7,
												ContentWidth:  183.87890625,
												ContentHeight: 12.6,
												PageIndex:     1,
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      1086.05,
								ContentWidth:  386.25,
								ContentHeight: 30.75,
								PageIndex:     1,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      1086.05,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "O · p-class object syntax",
												ContentX:      15,
												ContentY:      1086.05,
												ContentWidth:  109.4296875,
												ContentHeight: 13.649999999999999,
												PageIndex:     1,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.FontWeight = 700
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										ContentX:      15,
										ContentY:      1104.2,
										ContentWidth:  386.25,
										ContentHeight: 12.6,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.FontWeight = 700
												}),
												Text:          "Coloured via a toggled p-class.",
												ContentX:      15,
												ContentY:      1104.2,
												ContentWidth:  135.9375,
												ContentHeight: 12.6,
												PageIndex:     1,
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      1138.55,
								ContentWidth:  386.25,
								ContentHeight: 35.25,
								PageIndex:     1,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      1138.55,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "P · flex gap",
												ContentX:      15,
												ContentY:      1138.55,
												ContentWidth:  49.91015625,
												ContentHeight: 13.649999999999999,
												PageIndex:     1,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxFlex,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.RowGap = 22.5
											s.ColumnGap = 22.5
											s.Display = layouter_domain.DisplayFlex
											s.BoxSizing = border - box
										}),
										ContentX:      15,
										ContentY:      1156.7,
										ContentWidth:  386.25,
										ContentHeight: 17.1,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxFlexItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.BackgroundColour = layouter_domain.NewRGBA(0.8588235294117647, 0.9176470588235294, 0.996078431372549, 1)
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.PaddingTop = 2.25
													s.PaddingRight = 4.5
													s.PaddingBottom = 2.25
													s.PaddingLeft = 4.5
													s.FontSize = 9
													s.LineHeight = 12.6
													s.BoxSizing = border - box
												}),
												Padding: layouter_domain.BoxEdges{
													Top:    2.25,
													Right:  4.5,
													Bottom: 2.25,
													Left:   4.5,
												},
												ContentX:      19.5,
												ContentY:      1158.95,
												ContentWidth:  17.671875,
												ContentHeight: 12.600000000000001,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "One",
														ContentX:      19.5,
														ContentY:      1158.95,
														ContentWidth:  17.671875,
														ContentHeight: 12.6,
														PageIndex:     1,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxFlexItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.BackgroundColour = layouter_domain.NewRGBA(0.8588235294117647, 0.9176470588235294, 0.996078431372549, 1)
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.PaddingTop = 2.25
													s.PaddingRight = 4.5
													s.PaddingBottom = 2.25
													s.PaddingLeft = 4.5
													s.FontSize = 9
													s.LineHeight = 12.6
													s.BoxSizing = border - box
												}),
												Padding: layouter_domain.BoxEdges{
													Top:    2.25,
													Right:  4.5,
													Bottom: 2.25,
													Left:   4.5,
												},
												ContentX:      68.671875,
												ContentY:      1158.95,
												ContentWidth:  17.35546875,
												ContentHeight: 12.600000000000001,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Two",
														ContentX:      68.671875,
														ContentY:      1158.95,
														ContentWidth:  17.35546875,
														ContentHeight: 12.6,
														PageIndex:     1,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxFlexItem,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.BackgroundColour = layouter_domain.NewRGBA(0.8588235294117647, 0.9176470588235294, 0.996078431372549, 1)
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.PaddingTop = 2.25
													s.PaddingRight = 4.5
													s.PaddingBottom = 2.25
													s.PaddingLeft = 4.5
													s.FontSize = 9
													s.LineHeight = 12.6
													s.BoxSizing = border - box
												}),
												Padding: layouter_domain.BoxEdges{
													Top:    2.25,
													Right:  4.5,
													Bottom: 2.25,
													Left:   4.5,
												},
												ContentX:      117.52734375,
												ContentY:      1158.95,
												ContentWidth:  24.2578125,
												ContentHeight: 12.600000000000001,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
														}),
														Text:          "Three",
														ContentX:      117.52734375,
														ContentY:      1158.95,
														ContentWidth:  24.2578125,
														ContentHeight: 12.6,
														PageIndex:     1,
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
									s.CustomProperties = map[string]string{
										"--probe-size":   "18px",
										"--probe-colour": "#2563eb",
									}
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      1195.55,
								ContentWidth:  386.25,
								ContentHeight: 46.05,
								PageIndex:     1,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.CustomProperties = map[string]string{
												"--probe-size":   "18px",
												"--probe-colour": "#2563eb",
											}
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      1195.55,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.CustomProperties = map[string]string{
														"--probe-size":   "18px",
														"--probe-colour": "#2563eb",
													}
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "Q · CSS variable for size + border colour",
												ContentX:      15,
												ContentY:      1195.55,
												ContentWidth:  180.2578125,
												ContentHeight: 13.649999999999999,
												PageIndex:     1,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.CustomProperties = map[string]string{
												"--probe-size":   "18px",
												"--probe-colour": "#2563eb",
											}
											s.FontFamily = "Helvetica"
											s.BorderTopColour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.BorderRightColour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.BorderBottomColour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.BorderLeftColour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.PaddingTop = 3
											s.PaddingRight = 3
											s.PaddingBottom = 3
											s.PaddingLeft = 3
											s.BorderTopWidth = 1.5
											s.BorderRightWidth = 1.5
											s.BorderBottomWidth = 1.5
											s.BorderLeftWidth = 1.5
											s.FontSize = 13.5
											s.LineHeight = 18.9
											s.Display = layouter_domain.DisplayBlock
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
											Top:    1.5,
											Right:  1.5,
											Bottom: 1.5,
											Left:   1.5,
										},
										ContentX:      19.5,
										ContentY:      1218.2,
										ContentWidth:  377.25,
										ContentHeight: 18.9,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.CustomProperties = map[string]string{
														"--probe-size":   "18px",
														"--probe-colour": "#2563eb",
													}
													s.FontFamily = "Helvetica"
													s.BorderTopColour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.BorderRightColour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.BorderBottomColour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.BorderLeftColour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 13.5
													s.LineHeight = 18.9
													s.BorderTopStyle = layouter_domain.BorderStyleSolid
													s.BorderRightStyle = layouter_domain.BorderStyleSolid
													s.BorderBottomStyle = layouter_domain.BorderStyleSolid
													s.BorderLeftStyle = layouter_domain.BorderStyleSolid
												}),
												Text:          "Sized 18px and blue-bordered via CSS variables.",
												ContentX:      19.5,
												ContentY:      1218.2,
												ContentWidth:  301.83984375,
												ContentHeight: 18.9,
												PageIndex:     1,
											},
										},
									},
								},
							},
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.FontFamily = "Helvetica"
									s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.8980392156862745, 0.9058823529411765, 0.9215686274509803, 1)
									s.MarginBottom = layouter_domain.DimensionPt(12)
									s.PaddingBottom = 9
									s.BorderBottomWidth = 0.75
									s.FontSize = 9
									s.LineHeight = 12.6
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
									s.BorderBottomStyle = layouter_domain.BorderStyleSolid
								}),
								Padding: layouter_domain.BoxEdges{
									Bottom: 9,
								},
								Border: layouter_domain.BoxEdges{
									Bottom: 0.75,
								},
								Margin: layouter_domain.BoxEdges{
									Bottom: 12,
								},
								ContentX:      15,
								ContentY:      1263.35,
								ContentWidth:  386.25,
								ContentHeight: 30.75,
								PageIndex:     1,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
											s.MarginBottom = layouter_domain.DimensionPt(4.5)
											s.FontSize = 9.75
											s.LineHeight = 13.649999999999999
											s.FontWeight = 600
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										Margin: layouter_domain.BoxEdges{
											Bottom: 4.5,
										},
										ContentX:      15,
										ContentY:      1263.35,
										ContentWidth:  386.25,
										ContentHeight: 13.649999999999999,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.1450980392156863, 0.38823529411764707, 0.9215686274509803, 1)
													s.FontSize = 9.75
													s.LineHeight = 13.649999999999999
													s.FontWeight = 600
												}),
												Text:          "R · literal inline <b> / <em> (written in template, not p-html)",
												ContentX:      15,
												ContentY:      1263.35,
												ContentWidth:  270.73828125,
												ContentHeight: 13.649999999999999,
												PageIndex:     1,
											},
										},
									},
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxBlock,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.FontFamily = "Helvetica"
											s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
											s.FontSize = 9
											s.LineHeight = 12.6
											s.Display = layouter_domain.DisplayBlock
											s.BoxSizing = border - box
										}),
										ContentX:      15,
										ContentY:      1281.5,
										ContentWidth:  386.25,
										ContentHeight: 12.6,
										PageIndex:     1,
										Children: []*layouter_domain.LayoutBox{
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          "Plain text with ",
												ContentX:      15,
												ContentY:      1281.5,
												ContentWidth:  62.09765625,
												ContentHeight: 12.6,
												PageIndex:     1,
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxInline,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.FontWeight = 700
													s.BoxSizing = border - box
												}),
												ContentX:      77.09765625,
												ContentY:      1281.5,
												ContentWidth:  48.5859375,
												ContentHeight: 12.6,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.FontWeight = 700
														}),
														Text:          "literal bold",
														ContentX:      77.09765625,
														ContentY:      1281.5,
														ContentWidth:  48.5859375,
														ContentHeight: 12.6,
														PageIndex:     1,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          " and ",
												ContentX:      125.68359375,
												ContentY:      1281.5,
												ContentWidth:  20.8359375,
												ContentHeight: 12.6,
												PageIndex:     1,
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxInline,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
													s.BoxSizing = border - box
													s.FontStyle = layouter_domain.FontStyleItalic
												}),
												ContentX:      146.51953125,
												ContentY:      1281.5,
												ContentWidth:  45.796875,
												ContentHeight: 12.6,
												PageIndex:     1,
												Children: []*layouter_domain.LayoutBox{
													&layouter_domain.LayoutBox{
														Type: layouter_domain.BoxTextRun,
														Style: withStyle(func(s *layouter_domain.ComputedStyle) {
															s.FontFamily = "Helvetica"
															s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
															s.FontSize = 9
															s.LineHeight = 12.6
															s.FontStyle = layouter_domain.FontStyleItalic
														}),
														Text:          "literal italic",
														ContentX:      146.51953125,
														ContentY:      1281.5,
														ContentWidth:  45.796875,
														ContentHeight: 12.6,
														PageIndex:     1,
													},
												},
											},
											&layouter_domain.LayoutBox{
												Type: layouter_domain.BoxTextRun,
												Style: withStyle(func(s *layouter_domain.ComputedStyle) {
													s.FontFamily = "Helvetica"
													s.Colour = layouter_domain.NewRGBA(0.12156862745098039, 0.1607843137254902, 0.21568627450980393, 1)
													s.FontSize = 9
													s.LineHeight = 12.6
												}),
												Text:          " in the template.",
												ContentX:      192.31640625,
												ContentY:      1281.5,
												ContentWidth:  69.17578125,
												ContentHeight: 12.6,
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
	}
}()
