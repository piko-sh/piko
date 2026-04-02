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
					s.BackgroundColour = layouter_domain.NewRGBA(0.9254901960784314, 0.9411764705882353, 0.9450980392156862, 1)
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.PaddingTop = 7.5
					s.PaddingRight = 7.5
					s.PaddingBottom = 7.5
					s.PaddingLeft = 7.5
					s.LineHeight = 16.799999999999997
					s.LetterSpacing = 3.75
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    7.5,
					Right:  7.5,
					Bottom: 7.5,
					Left:   7.5,
				},
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      22.5,
				ContentY:      22.5,
				ContentWidth:  550.28,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.LetterSpacing = 3.75
						}),
						Text:          "Spaced out letters ",
						ContentX:      22.5,
						ContentY:      22.5,
						ContentWidth:  176.25,
						ContentHeight: 16.799999999999997,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.BackgroundColour = layouter_domain.NewRGBA(0.9254901960784314, 0.9411764705882353, 0.9450980392156862, 1)
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.PaddingTop = 7.5
					s.PaddingRight = 7.5
					s.PaddingBottom = 7.5
					s.PaddingLeft = 7.5
					s.LineHeight = 16.799999999999997
					s.WordSpacing = 15
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    7.5,
					Right:  7.5,
					Bottom: 7.5,
					Left:   7.5,
				},
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      22.5,
				ContentY:      69.3,
				ContentWidth:  550.28,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.WordSpacing = 15
						}),
						Text:          "Words with extra spacing between them ",
						ContentX:      22.5,
						ContentY:      69.3,
						ContentWidth:  229.83984375,
						ContentHeight: 16.799999999999997,
					},
				},
			},
			&layouter_domain.LayoutBox{
				Type: layouter_domain.BoxBlock,
				Style: withStyle(func(s *layouter_domain.ComputedStyle) {
					s.BackgroundColour = layouter_domain.NewRGBA(0.9254901960784314, 0.9411764705882353, 0.9450980392156862, 1)
					s.MarginTop = layouter_domain.DimensionPt(15)
					s.MarginRight = layouter_domain.DimensionPt(15)
					s.MarginBottom = layouter_domain.DimensionPt(15)
					s.MarginLeft = layouter_domain.DimensionPt(15)
					s.PaddingTop = 7.5
					s.PaddingRight = 7.5
					s.PaddingBottom = 7.5
					s.PaddingLeft = 7.5
					s.LineHeight = 16.799999999999997
					s.LetterSpacing = 2.25
					s.WordSpacing = 7.5
					s.Display = layouter_domain.DisplayBlock
					s.BoxSizing = border - box
				}),
				Padding: layouter_domain.BoxEdges{
					Top:    7.5,
					Right:  7.5,
					Bottom: 7.5,
					Left:   7.5,
				},
				Margin: layouter_domain.BoxEdges{
					Right:  15,
					Bottom: 15,
					Left:   15,
				},
				ContentX:      22.5,
				ContentY:      116.1,
				ContentWidth:  550.28,
				ContentHeight: 16.799999999999997,
				Children: []*layouter_domain.LayoutBox{
					&layouter_domain.LayoutBox{
						Type: layouter_domain.BoxTextRun,
						Style: withStyle(func(s *layouter_domain.ComputedStyle) {
							s.LineHeight = 16.799999999999997
							s.LetterSpacing = 2.25
							s.WordSpacing = 7.5
						}),
						Text:          "Both letter and word spacing combined ",
						ContentX:      22.5,
						ContentY:      116.1,
						ContentWidth:  311.4375,
						ContentHeight: 16.799999999999997,
					},
				},
			},
		},
	}
}()
