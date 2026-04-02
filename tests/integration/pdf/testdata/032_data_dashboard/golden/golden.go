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
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.Width = layouter_domain.DimensionPt(416.25)
		s.PaddingTop = 15
		s.PaddingRight = 15
		s.PaddingBottom = 15
		s.PaddingLeft = 15
		s.FontSize = 7.5
		s.LineHeight = 10.5
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
 ContentWidth: 386.25,
 ContentHeight: 407.25,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlex,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgb(0.33,
 0.24,
 0.60) -1,
} {
rgb(0.42,
 0.27,
 0.76) -1,
} {
rgb(0.50,
 0.35,
 0.84) -1,
}] 90 linear-gradient ellipse,
}]
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.PaddingTop = 10.5
		s.PaddingRight = 15
		s.PaddingBottom = 10.5
		s.PaddingLeft = 15
		s.BorderTopLeftRadius = 4.5
		s.BorderTopRightRadius = 4.5
		s.BorderBottomRightRadius = 4.5
		s.BorderBottomLeftRadius = 4.5
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayFlex
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 10.5,
 Right: 15,
 Bottom: 10.5,
 Left: 15,
},
 ContentX: 30,
 ContentY: 25.5,
 ContentWidth: 356.25,
 ContentHeight: 27.749999999999996,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 30,
 ContentY: 25.5,
 ContentWidth: 272.53125,
 ContentHeight: 27.749999999999996,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.LineHeight = 16.799999999999997
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 30,
 ContentY: 25.5,
 ContentWidth: 272.53125,
 ContentHeight: 16.799999999999997,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.LineHeight = 16.799999999999997
		s.FontWeight = 700
	}),
 Text: "Operations Dashboard",
 ContentX: 30,
 ContentY: 25.5,
 ContentWidth: 134.56640625,
 ContentHeight: 16.799999999999997,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.8392156862745098, 0.7372549019607844, 0.9803921568627451, 1)
		s.MarginTop = layouter_domain.DimensionPt(1.5)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 30,
 ContentY: 43.8,
 ContentWidth: 272.53125,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.8392156862745098, 0.7372549019607844, 0.9803921568627451, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "Reporting Period: 1 January - 14 March 2026",
 ContentX: 30,
 ContentY: 43.8,
 ContentWidth: 139.39453125,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.9137254901960784, 0.8470588235294118, 0.9921568627450981, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 ContentX: 302.53125,
 ContentY: 25.5,
 ContentWidth: 83.71875,
 ContentHeight: 27.749999999999996,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.9137254901960784, 0.8470588235294118, 0.9921568627450981, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 ContentX: 302.53125,
 ContentY: 25.5,
 ContentWidth: 83.71875,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.9137254901960784, 0.8470588235294118, 0.9921568627450981, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "Generated: 14 March 2026",
 ContentX: 302.53125,
 ContentY: 25.5,
 ContentWidth: 83.71875,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.9137254901960784, 0.8470588235294118, 0.9921568627450981, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 ContentX: 302.53125,
 ContentY: 34.95,
 ContentWidth: 83.71875,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.9137254901960784, 0.8470588235294118, 0.9921568627450981, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "Region: Northern Europe",
 ContentX: 306.78515625,
 ContentY: 34.95,
 ContentWidth: 79.46484375,
 ContentHeight: 9.45,
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
		s.Colour = layouter_domain.NewRGBA(0.3333333333333333, 0.23529411764705882, 0.6039215686274509, 1)
		s.MarginTop = layouter_domain.DimensionPt(12)
		s.MarginBottom = layouter_domain.DimensionPt(6)
		s.FontSize = 8.25
		s.LineHeight = 11.549999999999999
		s.LetterSpacing = 0.75
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 6,
},
 ContentX: 15,
 ContentY: 75.75,
 ContentWidth: 386.25,
 ContentHeight: 11.549999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.3333333333333333, 0.23529411764705882, 0.6039215686274509, 1)
		s.FontSize = 8.25
		s.LineHeight = 11.549999999999999
		s.LetterSpacing = 0.75
		s.FontWeight = 700
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "KEY PERFORMANCE INDICATORS",
 ContentX: 15,
 ContentY: 75.75,
 ContentWidth: 151.44140625,
 ContentHeight: 11.549999999999999,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlex,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.MarginBottom = layouter_domain.DimensionPt(7.5)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayFlex
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 7.5,
},
 ContentX: 15,
 ContentY: 93.3,
 ContentWidth: 386.25,
 ContentHeight: 70.05,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BoxShadow = []layouter_domain.BoxShadowValue{
layouter_domain.BoxShadowValue{
OffsetY: 0.75,
 BlurRadius: 2.25,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.1),
},
 layouter_domain.BoxShadowValue{
OffsetY: 0.75,
 BlurRadius: 1.5,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.06),
},
}
		s.BorderTopColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderRightColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BackgroundColour = layouter_domain.ColourWhite
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderLeftColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.MarginRight = layouter_domain.DimensionPt(6)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.PaddingTop = 10.5
		s.PaddingRight = 10.5
		s.PaddingBottom = 10.5
		s.PaddingLeft = 10.5
		s.BorderTopWidth = 0.75
		s.BorderRightWidth = 0.75
		s.BorderBottomWidth = 0.75
		s.BorderLeftWidth = 0.75
		s.BorderTopLeftRadius = 4.5
		s.BorderTopRightRadius = 4.5
		s.BorderBottomRightRadius = 4.5
		s.BorderBottomLeftRadius = 4.5
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.BorderTopStyle = layouter_domain.BorderStyleSolid
		s.BorderRightStyle = layouter_domain.BorderStyleSolid
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
		s.BorderLeftStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 10.5,
 Right: 10.5,
 Bottom: 10.5,
 Left: 10.5,
},
 Border: layouter_domain.BoxEdges{
Top: 0.75,
 Right: 0.75,
 Bottom: 0.75,
 Left: 0.75,
},
 Margin: layouter_domain.BoxEdges{
Right: 6,
},
 ContentX: 26.25,
 ContentY: 104.55,
 ContentWidth: 68.0625,
 ContentHeight: 47.55,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.LetterSpacing = 0.75
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 ContentX: 26.25,
 ContentY: 104.55,
 ContentWidth: 68.0625,
 ContentHeight: 8.399999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.LetterSpacing = 0.75
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "REVENUE",
 ContentX: 26.25,
 ContentY: 104.55,
 ContentWidth: 31.53515625,
 ContentHeight: 8.399999999999999,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.15294117647058825, 0.403921568627451, 0.28627450980392155, 1)
		s.MarginTop = layouter_domain.DimensionPt(3)
		s.FontSize = 18
		s.LineHeight = 25.2
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 26.25,
 ContentY: 115.94999999999999,
 ContentWidth: 68.0625,
 ContentHeight: 25.2,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.15294117647058825, 0.403921568627451, 0.28627450980392155, 1)
		s.FontSize = 18
		s.LineHeight = 25.2
		s.FontWeight = 700
	}),
 Text: "2.34M",
 ContentX: 26.25,
 ContentY: 115.94999999999999,
 ContentWidth: 52.93359375,
 ContentHeight: 25.2,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.15294117647058825, 0.403921568627451, 0.28627450980392155, 1)
		s.MarginTop = layouter_domain.DimensionPt(1.5)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 26.25,
 ContentY: 142.64999999999998,
 ContentWidth: 68.0625,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.15294117647058825, 0.403921568627451, 0.28627450980392155, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "+12.4% vs prior year",
 ContentX: 26.25,
 ContentY: 142.64999999999998,
 ContentWidth: 64.16015625,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BoxShadow = []layouter_domain.BoxShadowValue{
layouter_domain.BoxShadowValue{
OffsetY: 0.75,
 BlurRadius: 2.25,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.1),
},
 layouter_domain.BoxShadowValue{
OffsetY: 0.75,
 BlurRadius: 1.5,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.06),
},
}
		s.BorderTopColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderRightColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BackgroundColour = layouter_domain.ColourWhite
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderLeftColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.MarginRight = layouter_domain.DimensionPt(6)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.PaddingTop = 10.5
		s.PaddingRight = 10.5
		s.PaddingBottom = 10.5
		s.PaddingLeft = 10.5
		s.BorderTopWidth = 0.75
		s.BorderRightWidth = 0.75
		s.BorderBottomWidth = 0.75
		s.BorderLeftWidth = 0.75
		s.BorderTopLeftRadius = 4.5
		s.BorderTopRightRadius = 4.5
		s.BorderBottomRightRadius = 4.5
		s.BorderBottomLeftRadius = 4.5
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.BorderTopStyle = layouter_domain.BorderStyleSolid
		s.BorderRightStyle = layouter_domain.BorderStyleSolid
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
		s.BorderLeftStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 10.5,
 Right: 10.5,
 Bottom: 10.5,
 Left: 10.5,
},
 Border: layouter_domain.BoxEdges{
Top: 0.75,
 Right: 0.75,
 Bottom: 0.75,
 Left: 0.75,
},
 Margin: layouter_domain.BoxEdges{
Right: 6,
},
 ContentX: 122.8125,
 ContentY: 104.55,
 ContentWidth: 68.0625,
 ContentHeight: 47.55,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.LetterSpacing = 0.75
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 ContentX: 122.8125,
 ContentY: 104.55,
 ContentWidth: 68.0625,
 ContentHeight: 8.399999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.LetterSpacing = 0.75
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "ORDERS",
 ContentX: 122.8125,
 ContentY: 104.55,
 ContentWidth: 27.65625,
 ContentHeight: 8.399999999999999,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.16862745098039217, 0.4235294117647059, 0.6901960784313725, 1)
		s.MarginTop = layouter_domain.DimensionPt(3)
		s.FontSize = 18
		s.LineHeight = 25.2
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 122.8125,
 ContentY: 115.94999999999999,
 ContentWidth: 68.0625,
 ContentHeight: 25.2,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.16862745098039217, 0.4235294117647059, 0.6901960784313725, 1)
		s.FontSize = 18
		s.LineHeight = 25.2
		s.FontWeight = 700
	}),
 Text: "1,847",
 ContentX: 122.8125,
 ContentY: 115.94999999999999,
 ContentWidth: 46.3359375,
 ContentHeight: 25.2,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.16862745098039217, 0.4235294117647059, 0.6901960784313725, 1)
		s.MarginTop = layouter_domain.DimensionPt(1.5)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 122.8125,
 ContentY: 142.64999999999998,
 ContentWidth: 68.0625,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.16862745098039217, 0.4235294117647059, 0.6901960784313725, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "+8.1% vs prior year",
 ContentX: 122.8125,
 ContentY: 142.64999999999998,
 ContentWidth: 60.3046875,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BoxShadow = []layouter_domain.BoxShadowValue{
layouter_domain.BoxShadowValue{
OffsetY: 0.75,
 BlurRadius: 2.25,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.1),
},
 layouter_domain.BoxShadowValue{
OffsetY: 0.75,
 BlurRadius: 1.5,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.06),
},
}
		s.BorderTopColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderRightColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BackgroundColour = layouter_domain.ColourWhite
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderLeftColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.MarginRight = layouter_domain.DimensionPt(6)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.PaddingTop = 10.5
		s.PaddingRight = 10.5
		s.PaddingBottom = 10.5
		s.PaddingLeft = 10.5
		s.BorderTopWidth = 0.75
		s.BorderRightWidth = 0.75
		s.BorderBottomWidth = 0.75
		s.BorderLeftWidth = 0.75
		s.BorderTopLeftRadius = 4.5
		s.BorderTopRightRadius = 4.5
		s.BorderBottomRightRadius = 4.5
		s.BorderBottomLeftRadius = 4.5
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.BorderTopStyle = layouter_domain.BorderStyleSolid
		s.BorderRightStyle = layouter_domain.BorderStyleSolid
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
		s.BorderLeftStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 10.5,
 Right: 10.5,
 Bottom: 10.5,
 Left: 10.5,
},
 Border: layouter_domain.BoxEdges{
Top: 0.75,
 Right: 0.75,
 Bottom: 0.75,
 Left: 0.75,
},
 Margin: layouter_domain.BoxEdges{
Right: 6,
},
 ContentX: 219.375,
 ContentY: 104.55,
 ContentWidth: 68.0625,
 ContentHeight: 47.55,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.LetterSpacing = 0.75
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 ContentX: 219.375,
 ContentY: 104.55,
 ContentWidth: 68.0625,
 ContentHeight: 8.399999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.LetterSpacing = 0.75
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "FULFILMENT RATE",
 ContentX: 219.375,
 ContentY: 104.55,
 ContentWidth: 62.25,
 ContentHeight: 8.399999999999999,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.3333333333333333, 0.23529411764705882, 0.6039215686274509, 1)
		s.MarginTop = layouter_domain.DimensionPt(3)
		s.FontSize = 18
		s.LineHeight = 25.2
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 219.375,
 ContentY: 115.94999999999999,
 ContentWidth: 68.0625,
 ContentHeight: 25.2,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.3333333333333333, 0.23529411764705882, 0.6039215686274509, 1)
		s.FontSize = 18
		s.LineHeight = 25.2
		s.FontWeight = 700
	}),
 Text: "96.2%",
 ContentX: 219.375,
 ContentY: 115.94999999999999,
 ContentWidth: 52.1953125,
 ContentHeight: 25.2,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.3333333333333333, 0.23529411764705882, 0.6039215686274509, 1)
		s.MarginTop = layouter_domain.DimensionPt(1.5)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 219.375,
 ContentY: 142.64999999999998,
 ContentWidth: 68.0625,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.3333333333333333, 0.23529411764705882, 0.6039215686274509, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "+1.8pp vs target",
 ContentX: 219.375,
 ContentY: 142.64999999999998,
 ContentWidth: 51.19921875,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BoxShadow = []layouter_domain.BoxShadowValue{
layouter_domain.BoxShadowValue{
OffsetY: 0.75,
 BlurRadius: 2.25,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.1),
},
 layouter_domain.BoxShadowValue{
OffsetY: 0.75,
 BlurRadius: 1.5,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.06),
},
}
		s.BorderTopColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderRightColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BackgroundColour = layouter_domain.ColourWhite
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderLeftColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.PaddingTop = 10.5
		s.PaddingRight = 10.5
		s.PaddingBottom = 10.5
		s.PaddingLeft = 10.5
		s.BorderTopWidth = 0.75
		s.BorderRightWidth = 0.75
		s.BorderBottomWidth = 0.75
		s.BorderLeftWidth = 0.75
		s.BorderTopLeftRadius = 4.5
		s.BorderTopRightRadius = 4.5
		s.BorderBottomRightRadius = 4.5
		s.BorderBottomLeftRadius = 4.5
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.BorderTopStyle = layouter_domain.BorderStyleSolid
		s.BorderRightStyle = layouter_domain.BorderStyleSolid
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
		s.BorderLeftStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 10.5,
 Right: 10.5,
 Bottom: 10.5,
 Left: 10.5,
},
 Border: layouter_domain.BoxEdges{
Top: 0.75,
 Right: 0.75,
 Bottom: 0.75,
 Left: 0.75,
},
 ContentX: 315.9375,
 ContentY: 104.55,
 ContentWidth: 74.0625,
 ContentHeight: 47.55,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.LetterSpacing = 0.75
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 ContentX: 315.9375,
 ContentY: 104.55,
 ContentWidth: 74.0625,
 ContentHeight: 8.399999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.LetterSpacing = 0.75
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "AVG RESPONSE TIME",
 ContentX: 315.9375,
 ContentY: 104.55,
 ContentWidth: 71.40234375,
 ContentHeight: 8.399999999999999,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7529411764705882, 0.33725490196078434, 0.12941176470588237, 1)
		s.MarginTop = layouter_domain.DimensionPt(3)
		s.FontSize = 18
		s.LineHeight = 25.2
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 315.9375,
 ContentY: 115.94999999999999,
 ContentWidth: 74.0625,
 ContentHeight: 25.2,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7529411764705882, 0.33725490196078434, 0.12941176470588237, 1)
		s.FontSize = 18
		s.LineHeight = 25.2
		s.FontWeight = 700
	}),
 Text: "2.4h",
 ContentX: 315.9375,
 ContentY: 115.94999999999999,
 ContentWidth: 37.359375,
 ContentHeight: 25.2,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7529411764705882, 0.33725490196078434, 0.12941176470588237, 1)
		s.MarginTop = layouter_domain.DimensionPt(1.5)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 315.9375,
 ContentY: 142.64999999999998,
 ContentWidth: 74.0625,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7529411764705882, 0.33725490196078434, 0.12941176470588237, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "-18 min vs prior year",
 ContentX: 315.9375,
 ContentY: 142.64999999999998,
 ContentWidth: 65.19140625,
 ContentHeight: 9.45,
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
		s.Colour = layouter_domain.NewRGBA(0.3333333333333333, 0.23529411764705882, 0.6039215686274509, 1)
		s.MarginTop = layouter_domain.DimensionPt(10.5)
		s.MarginBottom = layouter_domain.DimensionPt(6)
		s.FontSize = 8.25
		s.LineHeight = 11.549999999999999
		s.LetterSpacing = 0.75
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 6,
},
 ContentX: 15,
 ContentY: 173.85,
 ContentWidth: 386.25,
 ContentHeight: 11.549999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.3333333333333333, 0.23529411764705882, 0.6039215686274509, 1)
		s.FontSize = 8.25
		s.LineHeight = 11.549999999999999
		s.LetterSpacing = 0.75
		s.FontWeight = 700
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "MONTHLY REVENUE",
 ContentX: 15,
 ContentY: 173.85,
 ContentWidth: 92.58984375,
 ContentHeight: 11.549999999999999,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BoxShadow = []layouter_domain.BoxShadowValue{
layouter_domain.BoxShadowValue{
OffsetY: 0.75,
 BlurRadius: 2.25,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.1),
},
}
		s.BorderTopColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderRightColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BackgroundColour = layouter_domain.ColourWhite
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderLeftColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 10.5
		s.PaddingRight = 10.5
		s.PaddingBottom = 10.5
		s.PaddingLeft = 10.5
		s.BorderTopWidth = 0.75
		s.BorderRightWidth = 0.75
		s.BorderBottomWidth = 0.75
		s.BorderLeftWidth = 0.75
		s.BorderTopLeftRadius = 4.5
		s.BorderTopRightRadius = 4.5
		s.BorderBottomRightRadius = 4.5
		s.BorderBottomLeftRadius = 4.5
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.BorderTopStyle = layouter_domain.BorderStyleSolid
		s.BorderRightStyle = layouter_domain.BorderStyleSolid
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
		s.BorderLeftStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 10.5,
 Right: 10.5,
 Bottom: 10.5,
 Left: 10.5,
},
 Border: layouter_domain.BoxEdges{
Top: 0.75,
 Right: 0.75,
 Bottom: 0.75,
 Left: 0.75,
},
 ContentX: 26.25,
 ContentY: 202.64999999999998,
 ContentWidth: 363.75,
 ContentHeight: 56.4,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlex,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.MarginBottom = layouter_domain.DimensionPt(4.5)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayFlex
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 4.5,
},
 ContentX: 26.25,
 ContentY: 202.64999999999998,
 ContentWidth: 363.75,
 ContentHeight: 12,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.Width = layouter_domain.DimensionPt(37.5)
		s.PaddingTop = 1.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 1.5,
},
 ContentX: 26.25,
 ContentY: 204.14999999999998,
 ContentWidth: 37.5,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "Jan",
 ContentX: 26.25,
 ContentY: 204.14999999999998,
 ContentWidth: 9.796875,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.PaddingTop = 1.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 1.5,
},
 ContentX: 63.75,
 ContentY: 204.14999999999998,
 ContentWidth: 288.75,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgb(0.42,
 0.27,
 0.76) -1,
} {
rgb(0.50,
 0.35,
 0.84) -1,
}] 90 linear-gradient ellipse,
}]
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.Height = layouter_domain.DimensionPt(10.5)
		s.Width = layouter_domain.DimensionPct(78)
		s.BorderTopLeftRadius = 1.5
		s.BorderTopRightRadius = 1.5
		s.BorderBottomRightRadius = 1.5
		s.BorderBottomLeftRadius = 1.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 63.75,
 ContentY: 204.14999999999998,
 ContentWidth: 225.225,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.Width = layouter_domain.DimensionPt(37.5)
		s.PaddingTop = 1.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Padding: layouter_domain.BoxEdges{
Top: 1.5,
},
 ContentX: 352.5,
 ContentY: 204.14999999999998,
 ContentWidth: 37.5,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "780K",
 ContentX: 374.25,
 ContentY: 204.14999999999998,
 ContentWidth: 15.75,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlex,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.MarginBottom = layouter_domain.DimensionPt(4.5)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayFlex
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 4.5,
},
 ContentX: 26.25,
 ContentY: 219.14999999999998,
 ContentWidth: 363.75,
 ContentHeight: 12,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.Width = layouter_domain.DimensionPt(37.5)
		s.PaddingTop = 1.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 1.5,
},
 ContentX: 26.25,
 ContentY: 220.64999999999998,
 ContentWidth: 37.5,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "Feb",
 ContentX: 26.25,
 ContentY: 220.64999999999998,
 ContentWidth: 11.4609375,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.PaddingTop = 1.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 1.5,
},
 ContentX: 63.75,
 ContentY: 220.64999999999998,
 ContentWidth: 288.75,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgb(0.42,
 0.27,
 0.76) -1,
} {
rgb(0.50,
 0.35,
 0.84) -1,
}] 90 linear-gradient ellipse,
}]
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.Height = layouter_domain.DimensionPt(10.5)
		s.Width = layouter_domain.DimensionPct(82)
		s.BorderTopLeftRadius = 1.5
		s.BorderTopRightRadius = 1.5
		s.BorderBottomRightRadius = 1.5
		s.BorderBottomLeftRadius = 1.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 63.75,
 ContentY: 220.64999999999998,
 ContentWidth: 236.77499999999998,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.Width = layouter_domain.DimensionPt(37.5)
		s.PaddingTop = 1.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Padding: layouter_domain.BoxEdges{
Top: 1.5,
},
 ContentX: 352.5,
 ContentY: 220.64999999999998,
 ContentWidth: 37.5,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "820K",
 ContentX: 374.25,
 ContentY: 220.64999999999998,
 ContentWidth: 15.75,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlex,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayFlex
		s.BoxSizing = border-box
	}),
 ContentX: 26.25,
 ContentY: 235.64999999999998,
 ContentWidth: 363.75,
 ContentHeight: 12,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.Width = layouter_domain.DimensionPt(37.5)
		s.PaddingTop = 1.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 1.5,
},
 ContentX: 26.25,
 ContentY: 237.14999999999998,
 ContentWidth: 37.5,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "Mar",
 ContentX: 26.25,
 ContentY: 237.14999999999998,
 ContentWidth: 12.69140625,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.PaddingTop = 1.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 1.5,
},
 ContentX: 63.75,
 ContentY: 237.14999999999998,
 ContentWidth: 288.75,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgb(0.62,
 0.48,
 0.92) -1,
} {
rgb(0.72,
 0.58,
 0.96) -1,
}] 90 linear-gradient ellipse,
}]
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.Height = layouter_domain.DimensionPt(10.5)
		s.Width = layouter_domain.DimensionPct(74)
		s.Opacity = 0.7
		s.BorderTopLeftRadius = 1.5
		s.BorderTopRightRadius = 1.5
		s.BorderBottomRightRadius = 1.5
		s.BorderBottomLeftRadius = 1.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 63.75,
 ContentY: 237.14999999999998,
 ContentWidth: 213.675,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.Width = layouter_domain.DimensionPt(37.5)
		s.PaddingTop = 1.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Padding: layouter_domain.BoxEdges{
Top: 1.5,
},
 ContentX: 352.5,
 ContentY: 237.14999999999998,
 ContentWidth: 37.5,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "740K*",
 ContentX: 370.53515625,
 ContentY: 237.14999999999998,
 ContentWidth: 19.46484375,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.MarginTop = layouter_domain.DimensionPt(3)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 26.25,
 ContentY: 250.64999999999998,
 ContentWidth: 363.75,
 ContentHeight: 8.399999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
	}),
 Text: "* Projected based on data through 14 March",
 ContentX: 26.25,
 ContentY: 250.64999999999998,
 ContentWidth: 125.390625,
 ContentHeight: 8.399999999999999,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlex,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.MarginTop = layouter_domain.DimensionPt(9)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayFlex
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 279.29999999999995,
 ContentWidth: 386.25,
 ContentHeight: 126.6,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BoxShadow = []layouter_domain.BoxShadowValue{
layouter_domain.BoxShadowValue{
OffsetY: 0.75,
 BlurRadius: 2.25,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.1),
},
}
		s.BorderTopColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderRightColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BackgroundColour = layouter_domain.ColourWhite
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderLeftColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.MarginRight = layouter_domain.DimensionPt(6)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.PaddingTop = 10.5
		s.PaddingRight = 10.5
		s.PaddingBottom = 10.5
		s.PaddingLeft = 10.5
		s.BorderTopWidth = 0.75
		s.BorderRightWidth = 0.75
		s.BorderBottomWidth = 0.75
		s.BorderLeftWidth = 0.75
		s.BorderTopLeftRadius = 4.5
		s.BorderTopRightRadius = 4.5
		s.BorderBottomRightRadius = 4.5
		s.BorderBottomLeftRadius = 4.5
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.BorderTopStyle = layouter_domain.BorderStyleSolid
		s.BorderRightStyle = layouter_domain.BorderStyleSolid
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
		s.BorderLeftStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 10.5,
 Right: 10.5,
 Bottom: 10.5,
 Left: 10.5,
},
 Border: layouter_domain.BoxEdges{
Top: 0.75,
 Right: 0.75,
 Bottom: 0.75,
 Left: 0.75,
},
 Margin: layouter_domain.BoxEdges{
Right: 6,
},
 ContentX: 26.25,
 ContentY: 290.54999999999995,
 ContentWidth: 164.625,
 ContentHeight: 104.1,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.3333333333333333, 0.23529411764705882, 0.6039215686274509, 1)
		s.MarginBottom = layouter_domain.DimensionPt(6)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.LetterSpacing = 0.75
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 6,
},
 ContentX: 26.25,
 ContentY: 290.54999999999995,
 ContentWidth: 164.625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.3333333333333333, 0.23529411764705882, 0.6039215686274509, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.LetterSpacing = 0.75
		s.FontWeight = 700
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "RECENT ACTIVITY",
 ContentX: 26.25,
 ContentY: 290.54999999999995,
 ContentWidth: 75.66796875,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.PaddingLeft = 12
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Left: 12,
},
 ContentX: 38.25,
 ContentY: 307.04999999999995,
 ContentWidth: 152.625,
 ContentHeight: 87.6,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxListItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.MarginBottom = layouter_domain.DimensionPt(3)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayListItem
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 3,
},
 ContentX: 38.25,
 ContentY: 307.04999999999995,
 ContentWidth: 152.625,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxListMarker,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.MarginBottom = layouter_domain.DimensionPt(3)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.BoxSizing = border-box
	}),
 Text: "• ",
 ContentX: 33.94921875,
 ContentY: 307.04999999999995,
 ContentWidth: 4.30078125,
 ContentHeight: 9.45,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "Warehouse 3 expansion approved (12 Mar)",
 ContentX: 38.25,
 ContentY: 307.04999999999995,
 ContentWidth: 135.64453125,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxListItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.MarginBottom = layouter_domain.DimensionPt(3)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayListItem
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 3,
},
 ContentX: 38.25,
 ContentY: 319.49999999999994,
 ContentWidth: 152.625,
 ContentHeight: 18.9,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxListMarker,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.MarginBottom = layouter_domain.DimensionPt(3)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.BoxSizing = border-box
	}),
 Text: "• ",
 ContentX: 33.94921875,
 ContentY: 319.49999999999994,
 ContentWidth: 4.30078125,
 ContentHeight: 9.45,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "New supplier contract signed with Haldane Ltd",
 ContentX: 38.25,
 ContentY: 319.49999999999994,
 ContentWidth: 147.99609375,
 ContentHeight: 9.45,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "(10 Mar)",
 ContentX: 38.25,
 ContentY: 328.94999999999993,
 ContentWidth: 26.21484375,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxListItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.MarginBottom = layouter_domain.DimensionPt(3)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayListItem
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 3,
},
 ContentX: 38.25,
 ContentY: 341.4,
 ContentWidth: 152.625,
 ContentHeight: 18.9,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxListMarker,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.MarginBottom = layouter_domain.DimensionPt(3)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.BoxSizing = border-box
	}),
 Text: "• ",
 ContentX: 33.94921875,
 ContentY: 341.4,
 ContentWidth: 4.30078125,
 ContentHeight: 9.45,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "Q1 safety audit completed, zero incidents (8",
 ContentX: 38.25,
 ContentY: 341.4,
 ContentWidth: 138.83203125,
 ContentHeight: 9.45,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "Mar)",
 ContentX: 38.25,
 ContentY: 350.84999999999997,
 ContentWidth: 14.71875,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxListItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.MarginBottom = layouter_domain.DimensionPt(3)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayListItem
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 3,
},
 ContentX: 38.25,
 ContentY: 363.29999999999995,
 ContentWidth: 152.625,
 ContentHeight: 18.9,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxListMarker,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.MarginBottom = layouter_domain.DimensionPt(3)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.BoxSizing = border-box
	}),
 Text: "• ",
 ContentX: 33.94921875,
 ContentY: 363.29999999999995,
 ContentWidth: 4.30078125,
 ContentHeight: 9.45,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "Fleet vehicles serviced, 12 of 15 returned (6",
 ContentX: 38.25,
 ContentY: 363.29999999999995,
 ContentWidth: 136.5,
 ContentHeight: 9.45,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "Mar)",
 ContentX: 38.25,
 ContentY: 372.74999999999994,
 ContentWidth: 14.71875,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxListItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayListItem
		s.BoxSizing = border-box
	}),
 ContentX: 38.25,
 ContentY: 385.19999999999993,
 ContentWidth: 152.625,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxListMarker,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.BoxSizing = border-box
	}),
 Text: "• ",
 ContentX: 33.94921875,
 ContentY: 385.19999999999993,
 ContentWidth: 4.30078125,
 ContentHeight: 9.45,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.2901960784313726, 0.3333333333333333, 0.40784313725490196, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "Staff onboarding: 4 new hires in logistics (3 Mar)",
 ContentX: 38.25,
 ContentY: 385.19999999999993,
 ContentWidth: 152.33203125,
 ContentHeight: 9.45,
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
		s.BoxShadow = []layouter_domain.BoxShadowValue{
layouter_domain.BoxShadowValue{
OffsetY: 0.75,
 BlurRadius: 2.25,
 Colour: layouter_domain.NewRGBA(0,
 0,
 0,
 0.1),
},
}
		s.BorderTopColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderRightColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BackgroundColour = layouter_domain.ColourWhite
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderLeftColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.PaddingTop = 10.5
		s.PaddingRight = 10.5
		s.PaddingBottom = 10.5
		s.PaddingLeft = 10.5
		s.BorderTopWidth = 0.75
		s.BorderRightWidth = 0.75
		s.BorderBottomWidth = 0.75
		s.BorderLeftWidth = 0.75
		s.BorderTopLeftRadius = 4.5
		s.BorderTopRightRadius = 4.5
		s.BorderBottomRightRadius = 4.5
		s.BorderBottomLeftRadius = 4.5
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.BorderTopStyle = layouter_domain.BorderStyleSolid
		s.BorderRightStyle = layouter_domain.BorderStyleSolid
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
		s.BorderLeftStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 10.5,
 Right: 10.5,
 Bottom: 10.5,
 Left: 10.5,
},
 Border: layouter_domain.BoxEdges{
Top: 0.75,
 Right: 0.75,
 Bottom: 0.75,
 Left: 0.75,
},
 ContentX: 219.375,
 ContentY: 290.54999999999995,
 ContentWidth: 170.625,
 ContentHeight: 104.1,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.3333333333333333, 0.23529411764705882, 0.6039215686274509, 1)
		s.MarginBottom = layouter_domain.DimensionPt(6)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.LetterSpacing = 0.75
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 6,
},
 ContentX: 219.375,
 ContentY: 290.54999999999995,
 ContentWidth: 170.625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.3333333333333333, 0.23529411764705882, 0.6039215686274509, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.LetterSpacing = 0.75
		s.FontWeight = 700
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "TOP REGIONS",
 ContentX: 219.375,
 ContentY: 290.54999999999995,
 ContentWidth: 58.58203125,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTable,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.Width = layouter_domain.DimensionPct(100)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.BorderSpacing = 1.5
		s.Display = layouter_domain.DisplayTable
		s.BoxSizing = border-box
		s.BorderCollapse = layouter_domain.BorderCollapseCollapse
	}),
 ContentX: 219.375,
 ContentY: 307.04999999999995,
 ContentWidth: 170.625,
 ContentHeight: 61.8,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableRow,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderBottomWidth = 0.75
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableRow
		s.BoxSizing = border-box
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 ContentX: 219.375,
 ContentY: 307.04999999999995,
 ContentWidth: 170.625,
 ContentHeight: 15.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.PaddingTop = 3
		s.PaddingBottom = 3
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 3,
 Bottom: 3,
},
 ContentX: 219.375,
 ContentY: 310.04999999999995,
 ContentWidth: 121.1172279792746,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
	}),
 Text: "South East",
 ContentX: 219.375,
 ContentY: 310.04999999999995,
 ContentWidth: 35.53125,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.15294117647058825, 0.403921568627451, 0.28627450980392155, 1)
		s.PaddingTop = 3
		s.PaddingBottom = 3
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Padding: layouter_domain.BoxEdges{
Top: 3,
 Bottom: 3,
},
 ContentX: 340.4922279792746,
 ContentY: 310.04999999999995,
 ContentWidth: 49.50777202072538,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.15294117647058825, 0.403921568627451, 0.28627450980392155, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "892K",
 ContentX: 374.25,
 ContentY: 310.04999999999995,
 ContentWidth: 15.75,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableRow,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderBottomWidth = 0.75
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableRow
		s.BoxSizing = border-box
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 ContentX: 219.375,
 ContentY: 322.49999999999994,
 ContentWidth: 170.625,
 ContentHeight: 15.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.PaddingTop = 3
		s.PaddingBottom = 3
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 3,
 Bottom: 3,
},
 ContentX: 219.375,
 ContentY: 325.49999999999994,
 ContentWidth: 121.1172279792746,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
	}),
 Text: "Midlands",
 ContentX: 219.375,
 ContentY: 325.49999999999994,
 ContentWidth: 30.73828125,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.15294117647058825, 0.403921568627451, 0.28627450980392155, 1)
		s.PaddingTop = 3
		s.PaddingBottom = 3
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Padding: layouter_domain.BoxEdges{
Top: 3,
 Bottom: 3,
},
 ContentX: 340.4922279792746,
 ContentY: 325.49999999999994,
 ContentWidth: 49.50777202072538,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.15294117647058825, 0.403921568627451, 0.28627450980392155, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "634K",
 ContentX: 374.25,
 ContentY: 325.49999999999994,
 ContentWidth: 15.75,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableRow,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderBottomWidth = 0.75
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableRow
		s.BoxSizing = border-box
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 ContentX: 219.375,
 ContentY: 337.94999999999993,
 ContentWidth: 170.625,
 ContentHeight: 15.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.PaddingTop = 3
		s.PaddingBottom = 3
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 3,
 Bottom: 3,
},
 ContentX: 219.375,
 ContentY: 340.94999999999993,
 ContentWidth: 121.1172279792746,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
	}),
 Text: "North West",
 ContentX: 219.375,
 ContentY: 340.94999999999993,
 ContentWidth: 38.53125,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.15294117647058825, 0.403921568627451, 0.28627450980392155, 1)
		s.PaddingTop = 3
		s.PaddingBottom = 3
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Padding: layouter_domain.BoxEdges{
Top: 3,
 Bottom: 3,
},
 ContentX: 340.4922279792746,
 ContentY: 340.94999999999993,
 ContentWidth: 49.50777202072538,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.15294117647058825, 0.403921568627451, 0.28627450980392155, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "418K",
 ContentX: 374.25,
 ContentY: 340.94999999999993,
 ContentWidth: 15.75,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableRow,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableRow
		s.BoxSizing = border-box
	}),
 ContentX: 219.375,
 ContentY: 353.4,
 ContentWidth: 170.625,
 ContentHeight: 15.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.PaddingTop = 3
		s.PaddingBottom = 3
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 3,
 Bottom: 3,
},
 ContentX: 219.375,
 ContentY: 356.4,
 ContentWidth: 121.1172279792746,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17647058823529413, 0.21568627450980393, 0.2823529411764706, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
	}),
 Text: "Scotland",
 ContentX: 219.375,
 ContentY: 356.4,
 ContentWidth: 29.05078125,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.15294117647058825, 0.403921568627451, 0.28627450980392155, 1)
		s.PaddingTop = 3
		s.PaddingBottom = 3
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Padding: layouter_domain.BoxEdges{
Top: 3,
 Bottom: 3,
},
 ContentX: 340.4922279792746,
 ContentY: 356.4,
 ContentWidth: 49.50777202072538,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.15294117647058825, 0.403921568627451, 0.28627450980392155, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "396K",
 ContentX: 374.25,
 ContentY: 356.4,
 ContentWidth: 15.75,
 ContentHeight: 9.45,
},
},
},
},
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
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.MarginTop = layouter_domain.DimensionPt(9)
		s.FontSize = 5.25
		s.LineHeight = 7.35
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 ContentX: 15,
 ContentY: 414.9,
 ContentWidth: 386.25,
 ContentHeight: 7.35,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 5.25
		s.LineHeight = 7.35
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "Confidential - For internal use only. Generated by Northgate Reporting Platform v4.2",
 ContentX: 104.279296875,
 ContentY: 414.9,
 ContentWidth: 207.69140625,
 ContentHeight: 7.35,
},
},
},
},
},
},
}
}()
