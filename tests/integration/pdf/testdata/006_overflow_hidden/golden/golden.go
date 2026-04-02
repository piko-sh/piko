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
				ContentHeight: 165,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BorderTopColour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
							s.BorderRightColour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
							s.BorderBottomColour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
							s.BorderLeftColour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
							s.Height = layouter_domain.DimensionPt(75)
							s.Width = layouter_domain.DimensionPt(150)
							s.MarginBottom = layouter_domain.DimensionPt(15)
							s.BorderTopWidth = 1.5
							s.BorderRightWidth = 1.5
							s.BorderBottomWidth = 1.5
							s.BorderLeftWidth = 1.5
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.OverflowX = layouter_domain.OverflowHidden
							s.OverflowY = layouter_domain.OverflowHidden
							s.BorderTopStyle = layouter_domain.BorderStyleSolid
							s.BorderRightStyle = layouter_domain.BorderStyleSolid
							s.BorderBottomStyle = layouter_domain.BorderStyleSolid
							s.BorderLeftStyle = layouter_domain.BorderStyleSolid
						}),
						Border: layouter_domain.BoxEdges{
							Top:    1.5,
							Right:  1.5,
							Bottom: 1.5,
							Left:   1.5,
						},
						Margin: layouter_domain.BoxEdges{
							Bottom: 15,
						},
						ContentX:      16.5,
						ContentY:      16.5,
						ContentWidth:  147,
						ContentHeight: 72,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BackgroundColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.Height = layouter_domain.DimensionPt(150)
									s.Width = layouter_domain.DimensionPt(225)
									s.LineHeight = 16.799999999999997
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								ContentX:      16.5,
								ContentY:      16.5,
								ContentWidth:  225,
								ContentHeight: 150,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.LineHeight = 16.799999999999997
										}),
										Text:          "Overflowing content",
										ContentX:      16.5,
										ContentY:      16.5,
										ContentWidth:  115.20703125,
										ContentHeight: 16.799999999999997,
									},
								},
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BorderTopColour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
							s.BorderRightColour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
							s.BorderBottomColour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
							s.BorderLeftColour = layouter_domain.NewRGBA(0.9058823529411765, 0.2980392156862745, 0.23529411764705882, 1)
							s.Height = layouter_domain.DimensionPt(75)
							s.Width = layouter_domain.DimensionPt(150)
							s.BorderTopWidth = 1.5
							s.BorderRightWidth = 1.5
							s.BorderBottomWidth = 1.5
							s.BorderLeftWidth = 1.5
							s.BorderTopLeftRadius = 15
							s.BorderTopRightRadius = 15
							s.BorderBottomRightRadius = 15
							s.BorderBottomLeftRadius = 15
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.OverflowX = layouter_domain.OverflowHidden
							s.OverflowY = layouter_domain.OverflowHidden
							s.BorderTopStyle = layouter_domain.BorderStyleSolid
							s.BorderRightStyle = layouter_domain.BorderStyleSolid
							s.BorderBottomStyle = layouter_domain.BorderStyleSolid
							s.BorderLeftStyle = layouter_domain.BorderStyleSolid
						}),
						Border: layouter_domain.BoxEdges{
							Top:    1.5,
							Right:  1.5,
							Bottom: 1.5,
							Left:   1.5,
						},
						ContentX:      16.5,
						ContentY:      106.5,
						ContentWidth:  147,
						ContentHeight: 72,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BackgroundColour = layouter_domain.NewRGBA(0.1803921568627451, 0.8, 0.44313725490196076, 1)
									s.Height = layouter_domain.DimensionPt(150)
									s.Width = layouter_domain.DimensionPt(225)
									s.LineHeight = 16.799999999999997
									s.Display = layouter_domain.DisplayBlock
									s.BoxSizing = border - box
								}),
								ContentX:      16.5,
								ContentY:      106.5,
								ContentWidth:  225,
								ContentHeight: 150,
								Children: []*layouter_domain.LayoutBox{
									&layouter_domain.LayoutBox{
										Type: layouter_domain.BoxTextRun,
										Style: withStyle(func(s *layouter_domain.ComputedStyle) {
											s.LineHeight = 16.799999999999997
										}),
										Text:          "Rounded overflow clip",
										ContentX:      16.5,
										ContentY:      106.5,
										ContentWidth:  125.21484375,
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
