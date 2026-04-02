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
		s.Height = layouter_domain.DimensionPt(601.5)
		s.Width = layouter_domain.DimensionPt(416.25)
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionRelative
		s.BoxSizing = border-box
		s.OverflowX = layouter_domain.OverflowHidden
		s.OverflowY = layouter_domain.OverflowHidden
	}),
 ContentWidth: 416.25,
 ContentHeight: 601.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgb(0.06,
 0.13,
 0.15) -1,
} {
rgb(0.13,
 0.23,
 0.26) -1,
} {
rgb(0.17,
 0.33,
 0.39) -1,
}] 160 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(255)
		s.Width = layouter_domain.DimensionPt(416.25)
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionRelative
		s.BoxSizing = border-box
		s.OverflowX = layouter_domain.OverflowHidden
		s.OverflowY = layouter_domain.OverflowHidden
	}),
 ContentWidth: 416.25,
 ContentHeight: 255,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.TransformValue = "rotate(35deg)"
		s.BackgroundColour = layouter_domain.ColourWhite
		s.Height = layouter_domain.DimensionPt(90)
		s.Width = layouter_domain.DimensionPt(90)
		s.Right = layouter_domain.DimensionPt(-15)
		s.Top = layouter_domain.DimensionPt(-22.5)
		s.Opacity = 0.06
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionAbsolute
		s.BoxSizing = border-box
		s.HasTransform = true
	}),
 ContentX: 341.25,
 ContentY: -22.5,
 ContentWidth: 90,
 ContentHeight: 90,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.TransformValue = "rotate(20deg)"
		s.BackgroundColour = layouter_domain.ColourWhite
		s.Height = layouter_domain.DimensionPt(60)
		s.Width = layouter_domain.DimensionPt(60)
		s.Right = layouter_domain.DimensionPt(37.5)
		s.Top = layouter_domain.DimensionPt(45)
		s.Opacity = 0.04
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionAbsolute
		s.BoxSizing = border-box
		s.HasTransform = true
	}),
 ContentX: 318.75,
 ContentY: 45,
 ContentWidth: 60,
 ContentHeight: 60,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.TransformValue = "rotate(55deg)"
		s.BackgroundColour = layouter_domain.ColourWhite
		s.Height = layouter_domain.DimensionPt(75)
		s.Width = layouter_domain.DimensionPt(75)
		s.Top = layouter_domain.DimensionPt(150)
		s.Left = layouter_domain.DimensionPt(-30)
		s.Opacity = 0.05
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionAbsolute
		s.BoxSizing = border-box
		s.HasTransform = true
	}),
 ContentX: -30,
 ContentY: 150,
 ContentWidth: 75,
 ContentHeight: 75,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.ClipPath = "polygon(100% 0%, 100% 100%, 0% 100%)"
		s.BgImages = [{
 [{
rgba(1.00,
 1.00,
 1.00,
 0.08) -1,
} {
rgba(1.00,
 1.00,
 1.00,
 0.02) -1,
}] 135 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(150)
		s.Width = layouter_domain.DimensionPt(150)
		s.Right = layouter_domain.DimensionPt(0)
		s.Bottom = layouter_domain.DimensionPt(0)
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionAbsolute
		s.BoxSizing = border-box
	}),
 ContentX: 266.25,
 ContentY: 105,
 ContentWidth: 150,
 ContentHeight: 150,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Right = layouter_domain.DimensionPt(30)
		s.Bottom = layouter_domain.DimensionPt(37.5)
		s.Left = layouter_domain.DimensionPt(30)
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionAbsolute
		s.BoxSizing = border-box
	}),
 ContentX: 30,
 ContentY: 144.75,
 ContentWidth: 356.25,
 ContentHeight: 72.75,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.5058823529411764, 0.9019607843137255, 0.8509803921568627, 1)
		s.MarginBottom = layouter_domain.DimensionPt(9)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.LetterSpacing = 3
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 9,
},
 ContentX: 30,
 ContentY: 144.75,
 ContentWidth: 356.25,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.5058823529411764, 0.9019607843137255, 0.8509803921568627, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.LetterSpacing = 3
		s.FontWeight = 700
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "ANNUAL PERFORMANCE REPORT",
 ContentX: 30,
 ContentY: 144.75,
 ContentWidth: 184.25390625,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.TextShadow = []layouter_domain.TextShadowValue{
layouter_domain.TextShadowValue{
OffsetX: 1.5,
 OffsetY: 1.5,
 BlurRadius: 6,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.5),
},
}
		s.Colour = layouter_domain.ColourWhite
		s.MarginBottom = layouter_domain.DimensionPt(6)
		s.FontSize = 24
		s.LineHeight = 33.599999999999994
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 6,
},
 ContentX: 30,
 ContentY: 163.2,
 ContentWidth: 356.25,
 ContentHeight: 33.599999999999994,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.TextShadow = []layouter_domain.TextShadowValue{
layouter_domain.TextShadowValue{
OffsetX: 1.5,
 OffsetY: 1.5,
 BlurRadius: 6,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.5),
},
}
		s.Colour = layouter_domain.ColourWhite
		s.FontSize = 24
		s.LineHeight = 33.599999999999994
		s.FontWeight = 700
	}),
 Text: "Infrastructure Division",
 ContentX: 30,
 ContentY: 163.2,
 ContentWidth: 272.296875,
 ContentHeight: 33.599999999999994,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 10.5
		s.LineHeight = 14.7
		s.WordSpacing = 1.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 30,
 ContentY: 202.8,
 ContentWidth: 356.25,
 ContentHeight: 14.7,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 10.5
		s.LineHeight = 14.7
		s.WordSpacing = 1.5
	}),
 Text: "Operational Review and Strategic Outlook",
 ContentX: 30,
 ContentY: 202.8,
 ContentWidth: 205.546875,
 ContentHeight: 14.7,
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
		s.BgImages = [{
 [{
rgb(0.51,
 0.90,
 0.85) -1,
} {
rgb(0.17,
 0.33,
 0.39) -1,
} {
rgb(0.06,
 0.13,
 0.15) -1,
}] 90 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(3)
		s.Width = layouter_domain.DimensionPt(416.25)
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentY: 255,
 ContentWidth: 416.25,
 ContentHeight: 3,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.PaddingTop = 30
		s.PaddingRight = 30
		s.PaddingBottom = 30
		s.PaddingLeft = 30
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 30,
 Right: 30,
 Bottom: 30,
 Left: 30,
},
 ContentX: 30,
 ContentY: 288,
 ContentWidth: 356.25,
 ContentHeight: 291.15,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BackgroundColour = layouter_domain.NewRGBA(0.17254901960784313, 0.3254901960784314, 0.39215686274509803, 1)
		s.Width = layouter_domain.DimensionPt(67.5)
		s.MarginBottom = layouter_domain.DimensionPt(22.5)
		s.PaddingTop = 6
		s.PaddingBottom = 6
		s.BorderTopLeftRadius = 3
		s.BorderTopRightRadius = 3
		s.BorderBottomRightRadius = 3
		s.BorderBottomLeftRadius = 3
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Bottom: 6,
},
 Margin: layouter_domain.BoxEdges{
Bottom: 22.5,
},
 ContentX: 30,
 ContentY: 294,
 ContentWidth: 67.5,
 ContentHeight: 23.099999999999998,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.FontSize = 16.5
		s.LineHeight = 23.099999999999998
		s.LetterSpacing = 2.25
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 ContentX: 30,
 ContentY: 294,
 ContentWidth: 67.5,
 ContentHeight: 23.099999999999998,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.FontSize = 16.5
		s.LineHeight = 23.099999999999998
		s.LetterSpacing = 2.25
		s.FontWeight = 700
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "2026",
 ContentX: 40.3828125,
 ContentY: 294,
 ContentWidth: 46.734375,
 ContentHeight: 23.099999999999998,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgb(0.17,
 0.33,
 0.39) -1,
} {
rgb(0.51,
 0.90,
 0.85) -1,
}] 90 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(2.25)
		s.Width = layouter_domain.DimensionPt(45)
		s.MarginBottom = layouter_domain.DimensionPt(18)
		s.BorderTopLeftRadius = 1.5
		s.BorderTopRightRadius = 1.5
		s.BorderBottomRightRadius = 1.5
		s.BorderBottomLeftRadius = 1.5
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 18,
},
 ContentX: 30,
 ContentY: 345.6,
 ContentWidth: 45,
 ContentHeight: 2.25,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.MarginBottom = layouter_domain.DimensionPt(15)
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 15,
},
 ContentX: 30,
 ContentY: 365.85,
 ContentWidth: 356.25,
 ContentHeight: 43.2,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.MarginBottom = layouter_domain.DimensionPt(6)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.LetterSpacing = 1.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 6,
},
 ContentX: 30,
 ContentY: 365.85,
 ContentWidth: 356.25,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.LetterSpacing = 1.5
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "PREPARED BY",
 ContentX: 30,
 ContentY: 365.85,
 ContentWidth: 64.1953125,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FontSize = 10.5
		s.LineHeight = 14.7
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 30,
 ContentY: 382.35,
 ContentWidth: 356.25,
 ContentHeight: 14.7,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FontSize = 10.5
		s.LineHeight = 14.7
		s.FontWeight = 700
	}),
 Text: "Dr. Catherine Hargreaves",
 ContentX: 30,
 ContentY: 382.35,
 ContentWidth: 130.8984375,
 ContentHeight: 14.7,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.MarginTop = layouter_domain.DimensionPt(1.5)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 30,
 ContentY: 398.55,
 ContentWidth: 356.25,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
	}),
 Text: "Director of Operations, Infrastructure Division",
 ContentX: 30,
 ContentY: 398.55,
 ContentWidth: 161.66015625,
 ContentHeight: 10.5,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.MarginBottom = layouter_domain.DimensionPt(15)
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 15,
},
 ContentX: 30,
 ContentY: 424.05,
 ContentWidth: 356.25,
 ContentHeight: 43.2,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.MarginBottom = layouter_domain.DimensionPt(6)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.LetterSpacing = 1.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 6,
},
 ContentX: 30,
 ContentY: 424.05,
 ContentWidth: 356.25,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.LetterSpacing = 1.5
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "REVIEWED BY",
 ContentX: 30,
 ContentY: 424.05,
 ContentWidth: 64.23046875,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FontSize = 10.5
		s.LineHeight = 14.7
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 30,
 ContentY: 440.55,
 ContentWidth: 356.25,
 ContentHeight: 14.7,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FontSize = 10.5
		s.LineHeight = 14.7
		s.FontWeight = 700
	}),
 Text: "James Ellsworth-Kent",
 ContentX: 30,
 ContentY: 440.55,
 ContentWidth: 111.31640625,
 ContentHeight: 14.7,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.MarginTop = layouter_domain.DimensionPt(1.5)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 30,
 ContentY: 456.75,
 ContentWidth: 356.25,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
	}),
 Text: "Chief Executive Officer",
 ContentX: 30,
 ContentY: 456.75,
 ContentWidth: 79.03125,
 ContentHeight: 10.5,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 30,
 ContentY: 482.25,
 ContentWidth: 356.25,
 ContentHeight: 31.2,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.MarginBottom = layouter_domain.DimensionPt(6)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.LetterSpacing = 1.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 6,
},
 ContentX: 30,
 ContentY: 482.25,
 ContentWidth: 356.25,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.LetterSpacing = 1.5
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "DATE OF PUBLICATION",
 ContentX: 30,
 ContentY: 482.25,
 ContentWidth: 108.73828125,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FontSize = 10.5
		s.LineHeight = 14.7
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 30,
 ContentY: 498.75,
 ContentWidth: 356.25,
 ContentHeight: 14.7,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FontSize = 10.5
		s.LineHeight = 14.7
		s.FontWeight = 700
	}),
 Text: "21 March 2026",
 ContentX: 30,
 ContentY: 498.75,
 ContentWidth: 74.4609375,
 ContentHeight: 14.7,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BorderTopColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderRightColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderLeftColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.MarginTop = layouter_domain.DimensionPt(22.5)
		s.PaddingTop = 7.5
		s.PaddingRight = 10.5
		s.PaddingBottom = 7.5
		s.PaddingLeft = 10.5
		s.BorderTopWidth = 0.75
		s.BorderRightWidth = 0.75
		s.BorderBottomWidth = 0.75
		s.BorderLeftWidth = 0.75
		s.BorderTopLeftRadius = 3
		s.BorderTopRightRadius = 3
		s.BorderBottomRightRadius = 3
		s.BorderBottomLeftRadius = 3
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.BorderTopStyle = layouter_domain.BorderStyleSolid
		s.BorderRightStyle = layouter_domain.BorderStyleSolid
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
		s.BorderLeftStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 7.5,
 Right: 10.5,
 Bottom: 7.5,
 Left: 10.5,
},
 Border: layouter_domain.BoxEdges{
Top: 0.75,
 Right: 0.75,
 Bottom: 0.75,
 Left: 0.75,
},
 ContentX: 41.25,
 ContentY: 544.2,
 ContentWidth: 333.75,
 ContentHeight: 26.699999999999996,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.MarginBottom = layouter_domain.DimensionPt(1.5)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.LetterSpacing = 0.75
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 1.5,
},
 ContentX: 41.25,
 ContentY: 544.2,
 ContentWidth: 333.75,
 ContentHeight: 8.399999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.LetterSpacing = 0.75
		s.FontWeight = 700
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "CLASSIFICATION: INTERNAL",
 ContentX: 41.25,
 ContentY: 544.2,
 ContentWidth: 100.51171875,
 ContentHeight: 8.399999999999999,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 41.25,
 ContentY: 554.1,
 ContentWidth: 333.75,
 ContentHeight: 16.799999999999997,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
	}),
 Text: "This document is intended for internal distribution only. Reproduction or distribution to third parties requires written",
 ContentX: 41.25,
 ContentY: 554.1,
 ContentWidth: 328.6171875,
 ContentHeight: 8.399999999999999,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
	}),
 Text: "authorisation from the Executive Office.",
 ContentX: 41.25,
 ContentY: 562.5,
 ContentWidth: 111.78515625,
 ContentHeight: 8.399999999999999,
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
		s.BgImages = [{
 [{
rgb(0.17,
 0.33,
 0.39) -1,
} {
rgb(0.17,
 0.33,
 0.39) -1,
} {
rgb(0.13,
 0.23,
 0.26) -1,
} {
rgb(0.13,
 0.23,
 0.26) -1,
}] 90 repeating-linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(12)
		s.Width = layouter_domain.DimensionPt(416.25)
		s.Bottom = layouter_domain.DimensionPt(0)
		s.Left = layouter_domain.DimensionPt(0)
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionAbsolute
		s.BoxSizing = border-box
	}),
 ContentY: 589.5,
 ContentWidth: 416.25,
 ContentHeight: 12,
},
},
},
},
}
}()
