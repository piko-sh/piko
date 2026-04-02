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
		s.PaddingTop = 15
		s.PaddingRight = 15
		s.PaddingBottom = 15
		s.PaddingLeft = 15
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionRelative
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
 ContentWidth: 386.25,
 ContentHeight: 571.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.ClipPath = "polygon(0% 0%, 100% 0%, 0% 100%)"
		s.BgImages = [{
 [{
rgb(0.72,
 0.47,
 0.12) -1,
} {
rgb(0.84,
 0.62,
 0.18) -1,
}] 135 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(37.5)
		s.Width = layouter_domain.DimensionPt(37.5)
		s.Top = layouter_domain.DimensionPt(10.5)
		s.Left = layouter_domain.DimensionPt(10.5)
		s.Opacity = 0.3
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionAbsolute
		s.BoxSizing = border-box
	}),
 ContentX: 10.5,
 ContentY: 10.5,
 ContentWidth: 37.5,
 ContentHeight: 37.5,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.ClipPath = "polygon(0% 0%, 100% 0%, 100% 100%)"
		s.BgImages = [{
 [{
rgb(0.72,
 0.47,
 0.12) -1,
} {
rgb(0.84,
 0.62,
 0.18) -1,
}] 225 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(37.5)
		s.Width = layouter_domain.DimensionPt(37.5)
		s.Right = layouter_domain.DimensionPt(10.5)
		s.Top = layouter_domain.DimensionPt(10.5)
		s.Opacity = 0.3
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionAbsolute
		s.BoxSizing = border-box
	}),
 ContentX: 368.25,
 ContentY: 10.5,
 ContentWidth: 37.5,
 ContentHeight: 37.5,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.ClipPath = "polygon(0% 0%, 0% 100%, 100% 100%)"
		s.BgImages = [{
 [{
rgb(0.72,
 0.47,
 0.12) -1,
} {
rgb(0.84,
 0.62,
 0.18) -1,
}] 45 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(37.5)
		s.Width = layouter_domain.DimensionPt(37.5)
		s.Bottom = layouter_domain.DimensionPt(10.5)
		s.Left = layouter_domain.DimensionPt(10.5)
		s.Opacity = 0.3
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionAbsolute
		s.BoxSizing = border-box
	}),
 ContentX: 10.5,
 ContentY: 553.5,
 ContentWidth: 37.5,
 ContentHeight: 37.5,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.ClipPath = "polygon(100% 0%, 0% 100%, 100% 100%)"
		s.BgImages = [{
 [{
rgb(0.72,
 0.47,
 0.12) -1,
} {
rgb(0.84,
 0.62,
 0.18) -1,
}] 315 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(37.5)
		s.Width = layouter_domain.DimensionPt(37.5)
		s.Right = layouter_domain.DimensionPt(10.5)
		s.Bottom = layouter_domain.DimensionPt(10.5)
		s.Opacity = 0.3
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionAbsolute
		s.BoxSizing = border-box
	}),
 ContentX: 368.25,
 ContentY: 553.5,
 ContentWidth: 37.5,
 ContentHeight: 37.5,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlex,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BoxShadow = []layouter_domain.BoxShadowValue{
layouter_domain.BoxShadowValue{
SpreadRadius: 1.5,
 Colour: layouter_domain.NewRGBA(0.8392156862745098,
 0.6196078431372549,
 0.1803921568627451,
 1),
},
 layouter_domain.BoxShadowValue{
SpreadRadius: 1.5,
 Colour: layouter_domain.NewRGBA(0.8392156862745098,
 0.6196078431372549,
 0.1803921568627451,
 1),
 Inset: true,
},
 layouter_domain.BoxShadowValue{
OffsetY: 3,
 BlurRadius: 9,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.15),
},
}
		s.BorderTopColour = layouter_domain.NewRGBA(0.7176470588235294, 0.4745098039215686, 0.12156862745098039, 1)
		s.BorderRightColour = layouter_domain.NewRGBA(0.7176470588235294, 0.4745098039215686, 0.12156862745098039, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.7176470588235294, 0.4745098039215686, 0.12156862745098039, 1)
		s.BorderLeftColour = layouter_domain.NewRGBA(0.7176470588235294, 0.4745098039215686, 0.12156862745098039, 1)
		s.Height = layouter_domain.DimensionPt(571.5)
		s.Width = layouter_domain.DimensionPt(386.25)
		s.PaddingTop = 30
		s.PaddingRight = 30
		s.PaddingBottom = 30
		s.PaddingLeft = 30
		s.BorderTopWidth = 4.5
		s.BorderRightWidth = 4.5
		s.BorderBottomWidth = 4.5
		s.BorderLeftWidth = 4.5
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayFlex
		s.Position = layouter_domain.PositionRelative
		s.BoxSizing = border-box
		s.OverflowX = layouter_domain.OverflowHidden
		s.OverflowY = layouter_domain.OverflowHidden
		s.TextAlign = layouter_domain.TextAlignCentre
		s.BorderTopStyle = layouter_domain.BorderStyleDouble
		s.BorderRightStyle = layouter_domain.BorderStyleDouble
		s.BorderBottomStyle = layouter_domain.BorderStyleDouble
		s.BorderLeftStyle = layouter_domain.BorderStyleDouble
		s.FlexDirection = layouter_domain.FlexDirectionColumn
	}),
 Padding: layouter_domain.BoxEdges{
Top: 30,
 Right: 30,
 Bottom: 30,
 Left: 30,
},
 Border: layouter_domain.BoxEdges{
Top: 4.5,
 Right: 4.5,
 Bottom: 4.5,
 Left: 4.5,
},
 ContentX: 49.5,
 ContentY: 49.5,
 ContentWidth: 317.25,
 ContentHeight: 502.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgba(0.84,
 0.62,
 0.18,
 0.06) -1,
} {
rgba(0.84,
 0.62,
 0.18,
 0.02) -1,
} {
rgba(1.00,
 1.00,
 1.00,
 0.00) -1,
}] 0 radial-gradient circle,
}]
		s.Height = layouter_domain.DimensionPt(300)
		s.Width = layouter_domain.DimensionPt(300)
		s.Top = layouter_domain.DimensionPct(50)
		s.MarginTop = layouter_domain.DimensionPt(-150)
		s.MarginLeft = layouter_domain.DimensionPt(-150)
		s.Left = layouter_domain.DimensionPct(50)
		s.BorderTopLeftRadius = 193.125
		s.BorderTopRightRadius = 193.125
		s.BorderBottomRightRadius = 193.125
		s.BorderBottomLeftRadius = 193.125
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.Position = layouter_domain.PositionAbsolute
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Margin: layouter_domain.BoxEdges{
Top: -150,
 Left: -150,
},
 ContentX: 58.125,
 ContentY: 150.75,
 ContentWidth: 300,
 ContentHeight: 300,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7176470588235294, 0.4745098039215686, 0.12156862745098039, 1)
		s.MarginBottom = layouter_domain.DimensionPt(6)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.LetterSpacing = 4.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 6,
},
 ContentX: 49.5,
 ContentY: 49.5,
 ContentWidth: 317.25,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7176470588235294, 0.4745098039215686, 0.12156862745098039, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.LetterSpacing = 4.5
		s.TextAlign = layouter_domain.TextAlignCentre
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "THE BOARD OF DIRECTORS",
 ContentX: 115.763671875,
 ContentY: 49.5,
 ContentWidth: 184.72265625,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7176470588235294, 0.4745098039215686, 0.12156862745098039, 1)
		s.MarginBottom = layouter_domain.DimensionPt(12)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.LetterSpacing = 2.25
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 12,
},
 ContentX: 49.5,
 ContentY: 58.95,
 ContentWidth: 317.25,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7176470588235294, 0.4745098039215686, 0.12156862745098039, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.LetterSpacing = 2.25
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "of Ashworth Holdings plc",
 ContentX: 141.333984375,
 ContentY: 58.95,
 ContentWidth: 133.58203125,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgba(0.72,
 0.47,
 0.12,
 0.00) -1,
} {
rgb(0.72,
 0.47,
 0.12) -1,
} {
rgba(0.72,
 0.47,
 0.12,
 0.00) -1,
}] 90 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(1.5)
		s.Width = layouter_domain.DimensionPt(150)
		s.MarginRight = layouter_domain.DimensionAuto()
		s.MarginBottom = layouter_domain.DimensionPt(18)
		s.MarginLeft = layouter_domain.DimensionAuto()
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 18,
},
 ContentX: 133.125,
 ContentY: 68.4,
 ContentWidth: 150,
 ContentHeight: 1.5,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7176470588235294, 0.4745098039215686, 0.12156862745098039, 1)
		s.MarginBottom = layouter_domain.DimensionPt(4.5)
		s.FontSize = 9
		s.LineHeight = 12.6
		s.LetterSpacing = 3.75
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 4.5,
},
 ContentX: 49.5,
 ContentY: 69.9,
 ContentWidth: 317.25,
 ContentHeight: 12.6,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7176470588235294, 0.4745098039215686, 0.12156862745098039, 1)
		s.FontSize = 9
		s.LineHeight = 12.6
		s.LetterSpacing = 3.75
		s.TextAlign = layouter_domain.TextAlignCentre
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "CERTIFICATE OF",
 ContentX: 148.40625,
 ContentY: 69.9,
 ContentWidth: 119.4375,
 ContentHeight: 12.6,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.TextShadow = []layouter_domain.TextShadowValue{
layouter_domain.TextShadowValue{
OffsetX: 0.75,
 OffsetY: 0.75,
 BlurRadius: 1.5,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.15),
},
}
		s.Colour = layouter_domain.NewRGBA(0.4549019607843137, 0.25882352941176473, 0.06274509803921569, 1)
		s.MarginBottom = layouter_domain.DimensionPt(15)
		s.FontSize = 25.5
		s.LineHeight = 35.699999999999996
		s.LetterSpacing = 2.25
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 15,
},
 ContentX: 49.5,
 ContentY: 82.5,
 ContentWidth: 317.25,
 ContentHeight: 35.699999999999996,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.TextShadow = []layouter_domain.TextShadowValue{
layouter_domain.TextShadowValue{
OffsetX: 0.75,
 OffsetY: 0.75,
 BlurRadius: 1.5,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.15),
},
}
		s.Colour = layouter_domain.NewRGBA(0.4549019607843137, 0.25882352941176473, 0.06274509803921569, 1)
		s.FontSize = 25.5
		s.LineHeight = 35.699999999999996
		s.LetterSpacing = 2.25
		s.FontWeight = 700
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "ACHIEVEMENT",
 ContentX: 104.818359375,
 ContentY: 82.5,
 ContentWidth: 206.61328125,
 ContentHeight: 35.699999999999996,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgba(0.72,
 0.47,
 0.12,
 0.00) -1,
} {
rgb(0.72,
 0.47,
 0.12) -1,
} {
rgba(0.72,
 0.47,
 0.12,
 0.00) -1,
}] 90 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(0.75)
		s.Width = layouter_domain.DimensionPt(225)
		s.MarginRight = layouter_domain.DimensionAuto()
		s.MarginBottom = layouter_domain.DimensionPt(18)
		s.MarginLeft = layouter_domain.DimensionAuto()
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 18,
},
 ContentX: 95.625,
 ContentY: 118.19999999999999,
 ContentWidth: 225,
 ContentHeight: 0.75,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.MarginBottom = layouter_domain.DimensionPt(9)
		s.FontSize = 8.25
		s.LineHeight = 11.549999999999999
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 9,
},
 ContentX: 49.5,
 ContentY: 118.94999999999999,
 ContentWidth: 317.25,
 ContentHeight: 11.549999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FontSize = 8.25
		s.LineHeight = 11.549999999999999
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "This certificate is presented to",
 ContentX: 150.08203125,
 ContentY: 118.94999999999999,
 ContentWidth: 116.0859375,
 ContentHeight: 11.549999999999999,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.TextShadow = []layouter_domain.TextShadowValue{
layouter_domain.TextShadowValue{
OffsetY: 0.75,
 BlurRadius: 0.75,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.1),
},
}
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.MarginBottom = layouter_domain.DimensionPt(4.5)
		s.FontSize = 21
		s.LineHeight = 29.4
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 4.5,
},
 ContentX: 49.5,
 ContentY: 130.5,
 ContentWidth: 317.25,
 ContentHeight: 29.4,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.TextShadow = []layouter_domain.TextShadowValue{
layouter_domain.TextShadowValue{
OffsetY: 0.75,
 BlurRadius: 0.75,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.1),
},
}
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FontSize = 21
		s.LineHeight = 29.4
		s.FontWeight = 700
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "Eleanor Fitzwilliam",
 ContentX: 109.0546875,
 ContentY: 130.5,
 ContentWidth: 198.140625,
 ContentHeight: 29.4,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgba(0.72,
 0.47,
 0.12,
 0.00) -1,
} {
rgb(0.84,
 0.62,
 0.18) -1,
} {
rgba(0.72,
 0.47,
 0.12,
 0.00) -1,
}] 90 linear-gradient ellipse,
}]
		s.Height = layouter_domain.DimensionPt(0.75)
		s.Width = layouter_domain.DimensionPt(187.5)
		s.MarginRight = layouter_domain.DimensionAuto()
		s.MarginBottom = layouter_domain.DimensionPt(15)
		s.MarginLeft = layouter_domain.DimensionAuto()
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 15,
},
 ContentX: 114.375,
 ContentY: 159.89999999999998,
 ContentWidth: 187.5,
 ContentHeight: 0.75,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.MarginBottom = layouter_domain.DimensionPt(4.5)
		s.PaddingRight = 22.5
		s.PaddingLeft = 22.5
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Padding: layouter_domain.BoxEdges{
Right: 22.5,
 Left: 22.5,
},
 Margin: layouter_domain.BoxEdges{
Bottom: 4.5,
},
 ContentX: 72,
 ContentY: 160.64999999999998,
 ContentWidth: 272.25,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "In recognition of outstanding contributions to the Northern Infrastructure",
 ContentX: 77.748046875,
 ContentY: 160.64999999999998,
 ContentWidth: 260.75390625,
 ContentHeight: 10.5,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "Modernisation Programme,",
 ContentX: 159.38671875,
 ContentY: 171.14999999999998,
 ContentWidth: 97.4765625,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.MarginBottom = layouter_domain.DimensionPt(4.5)
		s.PaddingRight = 22.5
		s.PaddingLeft = 22.5
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Padding: layouter_domain.BoxEdges{
Right: 22.5,
 Left: 22.5,
},
 Margin: layouter_domain.BoxEdges{
Bottom: 4.5,
},
 ContentX: 72,
 ContentY: 171.14999999999998,
 ContentWidth: 272.25,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "demonstrating exceptional leadership and technical expertise in the delivery",
 ContentX: 73.810546875,
 ContentY: 171.14999999999998,
 ContentWidth: 268.62890625,
 ContentHeight: 10.5,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "of",
 ContentX: 204.568359375,
 ContentY: 181.64999999999998,
 ContentWidth: 7.11328125,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.MarginBottom = layouter_domain.DimensionPt(18)
		s.PaddingRight = 22.5
		s.PaddingLeft = 22.5
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Padding: layouter_domain.BoxEdges{
Right: 22.5,
 Left: 22.5,
},
 Margin: layouter_domain.BoxEdges{
Bottom: 18,
},
 ContentX: 72,
 ContentY: 181.64999999999998,
 ContentWidth: 272.25,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "critical network upgrades across the Yorkshire and Humber region.",
 ContentX: 90.31640625,
 ContentY: 181.64999999999998,
 ContentWidth: 235.6171875,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.MarginBottom = layouter_domain.DimensionPt(22.5)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 22.5,
},
 ContentX: 49.5,
 ContentY: 192.14999999999998,
 ContentWidth: 317.25,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "Awarded on the Twenty-First of March, Two Thousand and Twenty-Six",
 ContentX: 85.828125,
 ContentY: 192.14999999999998,
 ContentWidth: 244.59375,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.LineHeight = 16.799999999999997
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 ContentX: 49.5,
 ContentY: 202.64999999999998,
 ContentWidth: 317.25,
 ContentHeight: 297.15000000000003,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.MarginBottom = layouter_domain.DimensionPt(15)
		s.PaddingRight = 30
		s.PaddingLeft = 30
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayFlex
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Padding: layouter_domain.BoxEdges{
Right: 30,
 Left: 30,
},
 Margin: layouter_domain.BoxEdges{
Bottom: 15,
},
 ContentX: 79.5,
 ContentY: 499.8,
 ContentWidth: 257.25,
 ContentHeight: 44.85,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.MarginRight = layouter_domain.DimensionPt(22.5)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.LineHeight = 16.799999999999997
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Margin: layouter_domain.BoxEdges{
Right: 22.5,
},
 ContentX: 79.5,
 ContentY: 499.8,
 ContentWidth: 106.125,
 ContentHeight: 44.85,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BorderBottomColour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.Height = layouter_domain.DimensionPt(22.5)
		s.MarginBottom = layouter_domain.DimensionPt(4.5)
		s.BorderBottomWidth = 0.75
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Border: layouter_domain.BoxEdges{
Bottom: 0.75,
},
 Margin: layouter_domain.BoxEdges{
Bottom: 4.5,
},
 ContentX: 79.5,
 ContentY: 499.8,
 ContentWidth: 106.125,
 ContentHeight: 21.75,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 ContentX: 79.5,
 ContentY: 526.8,
 ContentWidth: 106.125,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "Margaret Ashworth",
 ContentX: 99.544921875,
 ContentY: 526.8,
 ContentWidth: 66.03515625,
 ContentHeight: 9.45,
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
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 ContentX: 79.5,
 ContentY: 536.25,
 ContentWidth: 106.125,
 ContentHeight: 8.399999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "Chair of the Board",
 ContentX: 106.833984375,
 ContentY: 536.25,
 ContentWidth: 51.45703125,
 ContentHeight: 8.399999999999999,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.LineHeight = 16.799999999999997
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 ContentX: 208.125,
 ContentY: 499.8,
 ContentWidth: 128.625,
 ContentHeight: 44.85,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BorderBottomColour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.Height = layouter_domain.DimensionPt(22.5)
		s.MarginBottom = layouter_domain.DimensionPt(4.5)
		s.BorderBottomWidth = 0.75
		s.LineHeight = 16.799999999999997
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Border: layouter_domain.BoxEdges{
Bottom: 0.75,
},
 Margin: layouter_domain.BoxEdges{
Bottom: 4.5,
},
 ContentX: 208.125,
 ContentY: 499.8,
 ContentWidth: 128.625,
 ContentHeight: 21.75,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 ContentX: 208.125,
 ContentY: 526.8,
 ContentWidth: 128.625,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "David Chen-Ramirez",
 ContentX: 238.72265625,
 ContentY: 526.8,
 ContentWidth: 67.4296875,
 ContentHeight: 9.45,
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
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 ContentX: 208.125,
 ContentY: 536.25,
 ContentWidth: 128.625,
 ContentHeight: 8.399999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "Programme Director",
 ContentX: 243.421875,
 ContentY: 536.25,
 ContentWidth: 58.03125,
 ContentHeight: 8.399999999999999,
},
},
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.796078431372549, 0.8352941176470589, 0.8784313725490196, 1)
		s.FontSize = 5.25
		s.LineHeight = 7.35
		s.LetterSpacing = 0.75
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 ContentX: 49.5,
 ContentY: 544.6500000000001,
 ContentWidth: 317.25,
 ContentHeight: 7.35,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.796078431372549, 0.8352941176470589, 0.8784313725490196, 1)
		s.FontSize = 5.25
		s.LineHeight = 7.35
		s.LetterSpacing = 0.75
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "Certificate Ref: AH-NIMP-2026-0042",
 ContentX: 151.693359375,
 ContentY: 544.6500000000001,
 ContentWidth: 112.86328125,
 ContentHeight: 7.35,
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
