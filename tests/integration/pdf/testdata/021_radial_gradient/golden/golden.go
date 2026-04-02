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
 ContentWidth: 595.28,
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
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 15,
 Right: 15,
 Bottom: 15,
 Left: 15,
},
 ContentX: 15,
 ContentY: 15,
 ContentWidth: 270,
 ContentHeight: 273.75,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgb(0.20,
 0.60,
 0.86) -1,
} {
rgb(0.17,
 0.24,
 0.31) -1,
}] 0 radial-gradient circle,
}]
		s.Height = layouter_domain.DimensionPt(150)
		s.Width = layouter_domain.DimensionPt(150)
		s.MarginBottom = layouter_domain.DimensionPt(11.25)
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 11.25,
},
 ContentX: 15,
 ContentY: 15,
 ContentWidth: 150,
 ContentHeight: 150,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgb(0.91,
 0.30,
 0.24) -1,
} {
rgb(0.95,
 0.61,
 0.07) -1,
} {
rgb(0.18,
 0.80,
 0.44) -1,
}] 0 radial-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(112.5)
		s.Width = layouter_domain.DimensionPt(225)
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 176.25,
 ContentWidth: 225,
 ContentHeight: 112.5,
},
},
},
},
}
}()
