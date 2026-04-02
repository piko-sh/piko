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
				Type: layouter_domain.BoxGrid,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.GridAutoRepeatColumns = &layouter_domain.GridAutoRepeat{
						Pattern: []layouter_domain.GridTrack{
							layouter_domain.GridTrack{
								Value: 150,
								Unit:  layouter_domain.GridTrackPoints,
							},
						},
					}
					s.Width = layouter_domain.DimensionPt(450)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayGrid
				}),
				ContentWidth:  450,
				ContentHeight: 30,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxGridItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(30)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentWidth:  150,
						ContentHeight: 30,
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxGridItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(30)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentX:      150,
						ContentWidth:  150,
						ContentHeight: 30,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxGrid,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.GridAutoRepeatColumns = &layouter_domain.GridAutoRepeat{
						Pattern: []layouter_domain.GridTrack{
							layouter_domain.GridTrack{
								Value: 150,
								Unit:  layouter_domain.GridTrackPoints,
							},
						},
						Type: layouter_domain.GridAutoRepeatFit,
					}
					s.Width = layouter_domain.DimensionPt(450)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayGrid
				}),
				ContentY:      30,
				ContentWidth:  450,
				ContentHeight: 30,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxGridItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(30)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentY:      30,
						ContentWidth:  150,
						ContentHeight: 30,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxGrid,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.GridAutoRepeatColumns = &layouter_domain.GridAutoRepeat{
						Pattern: []layouter_domain.GridTrack{
							layouter_domain.GridTrack{
								Value: 112.5,
								Unit:  layouter_domain.GridTrackPoints,
							},
						},
						InsertIndex: 1,
						AfterCount:  1,
					}
					s.GridTemplateColumns = []layouter_domain.GridTrack{
						layouter_domain.GridTrack{
							Value: 75,
							Unit:  layouter_domain.GridTrackPoints,
						},
						layouter_domain.GridTrack{
							Value: 75,
							Unit:  layouter_domain.GridTrackPoints,
						},
					}
					s.Width = layouter_domain.DimensionPt(450)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayGrid
				}),
				ContentY:      60,
				ContentWidth:  450,
				ContentHeight: 30,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxGridItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(30)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentY:      60,
						ContentWidth:  75,
						ContentHeight: 30,
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxGridItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(30)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentX:      75,
						ContentY:      60,
						ContentWidth:  112.5,
						ContentHeight: 30,
					},
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxGridItem,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(30)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayBlock
						}),
						ContentX:      187.5,
						ContentY:      60,
						ContentWidth:  112.5,
						ContentHeight: 30,
					},
				},
			},
		},
	}
}()
