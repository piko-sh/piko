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
 ContentHeight: 202.5,
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
rgb(0.91,
 0.30,
 0.24) -1,
}] 90 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(60)
		s.Width = layouter_domain.DimensionPt(225)
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
 ContentWidth: 225,
 ContentHeight: 60,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgb(0.20,
 0.60,
 0.86) -1,
} {
rgb(0.18,
 0.80,
 0.44) -1,
} {
rgb(0.91,
 0.30,
 0.24) -1,
}] 135 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(60)
		s.Width = layouter_domain.DimensionPt(225)
		s.MarginBottom = layouter_domain.DimensionPt(11.25)
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 11.25,
},
 ContentX: 15,
 ContentY: 86.25,
 ContentWidth: 225,
 ContentHeight: 60,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgb(0.95,
 0.61,
 0.07) -1,
} {
rgb(0.61,
 0.35,
 0.71) -1,
}] 180 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(60)
		s.Width = layouter_domain.DimensionPt(225)
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 157.5,
 ContentWidth: 225,
 ContentHeight: 60,
},
},
},
},
}
}()
