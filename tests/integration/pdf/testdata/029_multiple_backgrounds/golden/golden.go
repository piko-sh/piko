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
		s.BgImages = [{
 [{
rgba(0.20,
 0.60,
 0.86,
 0.50) -1,
} {
rgba(0.18,
 0.80,
 0.44,
 0.50) -1,
}] 90 linear-gradient ellipse,
} {
 [{
rgba(0.91,
 0.30,
 0.24,
 0.50) -1,
} {
rgba(0.95,
 0.77,
 0.06,
 0.50) -1,
}] 180 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(150)
		s.Width = layouter_domain.DimensionPt(225)
		s.MarginTop = layouter_domain.DimensionPt(15)
		s.MarginRight = layouter_domain.DimensionPt(15)
		s.MarginBottom = layouter_domain.DimensionPt(15)
		s.MarginLeft = layouter_domain.DimensionPt(15)
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Right: 15,
 Bottom: 15,
 Left: 15,
},
 ContentX: 15,
 ContentY: 15,
 ContentWidth: 225,
 ContentHeight: 150,
},
},
}
}()
