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
				ContentHeight: 238.95,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BorderTopColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.BorderRightColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.BackgroundColour = layouter_domain.NewRGBA(0.9254901960784314, 0.9411764705882353, 0.9450980392156862, 1)
							s.BorderBottomColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.BorderLeftColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.MarginBottom = layouter_domain.DimensionPt(11.25)
							s.PaddingTop = 11.25
							s.PaddingRight = 11.25
							s.PaddingBottom = 11.25
							s.PaddingLeft = 11.25
							s.BorderTopWidth = 6
							s.BorderRightWidth = 6
							s.BorderBottomWidth = 6
							s.BorderLeftWidth = 6
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.BorderTopStyle = layouter_domain.BorderStyleGroove
							s.BorderRightStyle = layouter_domain.BorderStyleGroove
							s.BorderBottomStyle = layouter_domain.BorderStyleGroove
							s.BorderLeftStyle = layouter_domain.BorderStyleGroove
						}),
						Padding: layouter_domain.BoxEdges{
							Top:    11.25,
							Right:  11.25,
							Bottom: 11.25,
							Left:   11.25,
						},
						Border: layouter_domain.BoxEdges{
							Top:    6,
							Right:  6,
							Bottom: 6,
							Left:   6,
						},
						Margin: layouter_domain.BoxEdges{
							Bottom: 11.25,
						},
						ContentX:      32.25,
						ContentY:      32.25,
						ContentWidth:  235.5,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.LineHeight = 16.799999999999997
									s.BorderTopStyle = layouter_domain.BorderStyleGroove
									s.BorderRightStyle = layouter_domain.BorderStyleGroove
									s.BorderBottomStyle = layouter_domain.BorderStyleGroove
									s.BorderLeftStyle = layouter_domain.BorderStyleGroove
								}),
								Text:          "Groove border",
								ContentX:      32.25,
								ContentY:      32.25,
								ContentWidth:  82.21875,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BorderTopColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.BorderRightColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.BackgroundColour = layouter_domain.NewRGBA(0.9254901960784314, 0.9411764705882353, 0.9450980392156862, 1)
							s.BorderBottomColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.BorderLeftColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.MarginBottom = layouter_domain.DimensionPt(11.25)
							s.PaddingTop = 11.25
							s.PaddingRight = 11.25
							s.PaddingBottom = 11.25
							s.PaddingLeft = 11.25
							s.BorderTopWidth = 6
							s.BorderRightWidth = 6
							s.BorderBottomWidth = 6
							s.BorderLeftWidth = 6
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.BorderTopStyle = layouter_domain.BorderStyleRidge
							s.BorderRightStyle = layouter_domain.BorderStyleRidge
							s.BorderBottomStyle = layouter_domain.BorderStyleRidge
							s.BorderLeftStyle = layouter_domain.BorderStyleRidge
						}),
						Padding: layouter_domain.BoxEdges{
							Top:    11.25,
							Right:  11.25,
							Bottom: 11.25,
							Left:   11.25,
						},
						Border: layouter_domain.BoxEdges{
							Top:    6,
							Right:  6,
							Bottom: 6,
							Left:   6,
						},
						Margin: layouter_domain.BoxEdges{
							Bottom: 11.25,
						},
						ContentX:      32.25,
						ContentY:      94.8,
						ContentWidth:  235.5,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.LineHeight = 16.799999999999997
									s.BorderTopStyle = layouter_domain.BorderStyleRidge
									s.BorderRightStyle = layouter_domain.BorderStyleRidge
									s.BorderBottomStyle = layouter_domain.BorderStyleRidge
									s.BorderLeftStyle = layouter_domain.BorderStyleRidge
								}),
								Text:          "Ridge border",
								ContentX:      32.25,
								ContentY:      94.8,
								ContentWidth:  73.69921875,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BorderTopColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.BorderRightColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.BackgroundColour = layouter_domain.NewRGBA(0.9254901960784314, 0.9411764705882353, 0.9450980392156862, 1)
							s.BorderBottomColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.BorderLeftColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.MarginBottom = layouter_domain.DimensionPt(11.25)
							s.PaddingTop = 11.25
							s.PaddingRight = 11.25
							s.PaddingBottom = 11.25
							s.PaddingLeft = 11.25
							s.BorderTopWidth = 6
							s.BorderRightWidth = 6
							s.BorderBottomWidth = 6
							s.BorderLeftWidth = 6
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.BorderTopStyle = layouter_domain.BorderStyleInset
							s.BorderRightStyle = layouter_domain.BorderStyleInset
							s.BorderBottomStyle = layouter_domain.BorderStyleInset
							s.BorderLeftStyle = layouter_domain.BorderStyleInset
						}),
						Padding: layouter_domain.BoxEdges{
							Top:    11.25,
							Right:  11.25,
							Bottom: 11.25,
							Left:   11.25,
						},
						Border: layouter_domain.BoxEdges{
							Top:    6,
							Right:  6,
							Bottom: 6,
							Left:   6,
						},
						Margin: layouter_domain.BoxEdges{
							Bottom: 11.25,
						},
						ContentX:      32.25,
						ContentY:      157.35,
						ContentWidth:  235.5,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.LineHeight = 16.799999999999997
									s.BorderTopStyle = layouter_domain.BorderStyleInset
									s.BorderRightStyle = layouter_domain.BorderStyleInset
									s.BorderBottomStyle = layouter_domain.BorderStyleInset
									s.BorderLeftStyle = layouter_domain.BorderStyleInset
								}),
								Text:          "Inset border",
								ContentX:      32.25,
								ContentY:      157.35,
								ContentWidth:  69.9375,
								ContentHeight: 16.799999999999997,
							},
						},
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.BorderTopColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.BorderRightColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.BackgroundColour = layouter_domain.NewRGBA(0.9254901960784314, 0.9411764705882353, 0.9450980392156862, 1)
							s.BorderBottomColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.BorderLeftColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
							s.PaddingTop = 11.25
							s.PaddingRight = 11.25
							s.PaddingBottom = 11.25
							s.PaddingLeft = 11.25
							s.BorderTopWidth = 6
							s.BorderRightWidth = 6
							s.BorderBottomWidth = 6
							s.BorderLeftWidth = 6
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
							s.BoxSizing = border - box
							s.BorderTopStyle = layouter_domain.BorderStyleOutset
							s.BorderRightStyle = layouter_domain.BorderStyleOutset
							s.BorderBottomStyle = layouter_domain.BorderStyleOutset
							s.BorderLeftStyle = layouter_domain.BorderStyleOutset
						}),
						Padding: layouter_domain.BoxEdges{
							Top:    11.25,
							Right:  11.25,
							Bottom: 11.25,
							Left:   11.25,
						},
						Border: layouter_domain.BoxEdges{
							Top:    6,
							Right:  6,
							Bottom: 6,
							Left:   6,
						},
						ContentX:      32.25,
						ContentY:      219.89999999999998,
						ContentWidth:  235.5,
						ContentHeight: 16.799999999999997,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxTextRun,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.BorderTopColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.BorderRightColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.BorderBottomColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.BorderLeftColour = layouter_domain.NewRGBA(0.20392156862745098, 0.596078431372549, 0.8588235294117647, 1)
									s.LineHeight = 16.799999999999997
									s.BorderTopStyle = layouter_domain.BorderStyleOutset
									s.BorderRightStyle = layouter_domain.BorderStyleOutset
									s.BorderBottomStyle = layouter_domain.BorderStyleOutset
									s.BorderLeftStyle = layouter_domain.BorderStyleOutset
								}),
								Text:          "Outset border",
								ContentX:      32.25,
								ContentY:      219.89999999999998,
								ContentWidth:  79.58203125,
								ContentHeight: 16.799999999999997,
							},
						},
					},
				},
			},
		},
	}
}()
