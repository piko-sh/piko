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
					s.Height = layouter_domain.DimensionFitContentStretch()
					s.Width = layouter_domain.DimensionPt(225)
					s.PaddingTop = 7.5
					s.PaddingRight = 7.5
					s.PaddingBottom = 7.5
					s.PaddingLeft = 7.5
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    7.5,
					Right:  7.5,
					Bottom: 7.5,
					Left:   7.5,
				},
				ContentX:      7.5,
				ContentY:      7.5,
				ContentWidth:  225,
				ContentHeight: 37.5,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(37.5)
							s.Width = layouter_domain.DimensionPt(75)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentX:      7.5,
						ContentY:      7.5,
						ContentWidth:  75,
						ContentHeight: 37.5,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.Width = layouter_domain.DimensionPt(225)
					s.PaddingTop = 7.5
					s.PaddingRight = 7.5
					s.PaddingBottom = 7.5
					s.PaddingLeft = 7.5
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    7.5,
					Right:  7.5,
					Bottom: 7.5,
					Left:   7.5,
				},
				ContentX:      7.5,
				ContentY:      60,
				ContentWidth:  225,
				ContentHeight: 37.5,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(37.5)
							s.Width = layouter_domain.DimensionPt(75)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentX:      7.5,
						ContentY:      60,
						ContentWidth:  75,
						ContentHeight: 37.5,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.Width = layouter_domain.DimensionPt(37.5)
					s.MinWidth = layouter_domain.DimensionFitContentStretch()
					s.PaddingTop = 7.5
					s.PaddingRight = 7.5
					s.PaddingBottom = 7.5
					s.PaddingLeft = 7.5
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    7.5,
					Right:  7.5,
					Bottom: 7.5,
					Left:   7.5,
				},
				ContentX:      7.5,
				ContentY:      112.5,
				ContentWidth:  150,
				ContentHeight: 22.5,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(22.5)
							s.Width = layouter_domain.DimensionPt(150)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentX:      7.5,
						ContentY:      112.5,
						ContentWidth:  150,
						ContentHeight: 22.5,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.Width = layouter_domain.DimensionPt(600)
					s.MaxWidth = layouter_domain.DimensionFitContentStretch()
					s.PaddingTop = 7.5
					s.PaddingRight = 7.5
					s.PaddingBottom = 7.5
					s.PaddingLeft = 7.5
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    7.5,
					Right:  7.5,
					Bottom: 7.5,
					Left:   7.5,
				},
				ContentX:      7.5,
				ContentY:      150,
				ContentWidth:  150,
				ContentHeight: 22.5,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(22.5)
							s.Width = layouter_domain.DimensionPt(150)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentX:      7.5,
						ContentY:      150,
						ContentWidth:  150,
						ContentHeight: 22.5,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxFlex,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.Width = layouter_domain.DimensionPt(300)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayFlex
				}),
				ContentY:      180,
				ContentWidth:  300,
				ContentHeight: 37.5,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxFlexItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionFitContentStretch()
							s.Width = layouter_domain.DimensionPt(75)
							s.PaddingTop = 3.75
							s.PaddingRight = 3.75
							s.PaddingBottom = 3.75
							s.PaddingLeft = 3.75
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						Padding: layouter_domain.BoxEdges{
							Top:    3.75,
							Right:  3.75,
							Bottom: 3.75,
							Left:   3.75,
						},
						ContentX:      3.75,
						ContentY:      183.75,
						ContentWidth:  75,
						ContentHeight: 30,
						Children: []*layouter_domain.LayoutBox{
							&layouter_domain.LayoutBox{
								Type: layouter_domain.BoxBlock,
								Style: withStyle(func(s *layouter_domain.ComputedStyle) {
									s.Height = layouter_domain.DimensionPt(30)
									s.Width = layouter_domain.DimensionPt(60)
									s.LineHeight = 16.799999999999997
									s.Display = layouter_domain.DisplayBlock
								}),
								ContentX:      3.75,
								ContentY:      183.75,
								ContentWidth:  60,
								ContentHeight: 30,
							},
						},
					},
				},
			},
		},
	}
}()
