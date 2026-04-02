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
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
				}),
				ContentWidth:  300,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
						}),
						Text:          "LTR text content",
						ContentWidth:  96,
						ContentHeight: 16.799999999999997,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.Width = layouter_domain.DimensionPt(300)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.Direction = layouter_domain.DirectionRtl
				}),
				ContentY:      16.799999999999997,
				ContentWidth:  300,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.Direction = layouter_domain.DirectionRtl
						}),
						Text:          "RTL text content",
						ContentX:      204,
						ContentY:      16.799999999999997,
						ContentWidth:  96,
						ContentHeight: 16.799999999999997,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.Width = layouter_domain.DimensionPt(300)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.Direction = layouter_domain.DirectionRtl
					s.UnicodeBidi = layouter_domain.UnicodeBidiBidiOverride
				}),
				ContentY:      33.599999999999994,
				ContentWidth:  300,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.Direction = layouter_domain.DirectionRtl
							s.UnicodeBidi = layouter_domain.UnicodeBidiBidiOverride
						}),
						Text:          "Override text",
						ContentX:      222,
						ContentY:      33.599999999999994,
						ContentWidth:  78,
						ContentHeight: 16.799999999999997,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.Width = layouter_domain.DimensionPt(300)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.Direction = layouter_domain.DirectionRtl
				}),
				ContentY:      50.39999999999999,
				ContentWidth:  300,
				ContentHeight: 15,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxInlineBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(15)
							s.Width = layouter_domain.DimensionPt(75)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayInlineBlock
						}),
						ContentX:      225,
						ContentY:      50.39999999999999,
						ContentWidth:  75,
						ContentHeight: 15,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.Width = layouter_domain.DimensionPt(300)
					s.LineHeight = 16.799999999999997
					s.Display = layouter_domain.DisplayBlock
					s.TextAlign = layouter_domain.TextAlignEnd
				}),
				ContentY:      65.39999999999999,
				ContentWidth:  300,
				ContentHeight: 15,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxInlineBlock,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.Height = layouter_domain.DimensionPt(15)
							s.Width = layouter_domain.DimensionPt(75)
							s.LineHeight = 16.799999999999997
							s.Display = layouter_domain.DisplayInlineBlock
							s.TextAlign = layouter_domain.TextAlignEnd
						}),
						ContentX:      225,
						ContentY:      65.39999999999999,
						ContentWidth:  75,
						ContentHeight: 15,
					},
				},
			},
		},
	}
}()
