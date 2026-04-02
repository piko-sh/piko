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
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
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
 ContentHeight: 531.3,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BgImages = [{
 [{
rgb(0.10,
 0.21,
 0.36) -1,
} {
rgb(0.17,
 0.42,
 0.69) -1,
}] 135 linear-gradient ellipse,
}]
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.PaddingTop = 15
		s.PaddingRight = 15
		s.PaddingBottom = 15
		s.PaddingLeft = 15
		s.BorderTopLeftRadius = 4.5
		s.BorderTopRightRadius = 4.5
		s.BorderBottomRightRadius = 4.5
		s.BorderBottomLeftRadius = 4.5
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
 ContentX: 30,
 ContentY: 30,
 ContentWidth: 356.25,
 ContentHeight: 45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.FontSize = 16.5
		s.LineHeight = 23.099999999999998
		s.LetterSpacing = 1.5
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 ContentX: 30,
 ContentY: 30,
 ContentWidth: 356.25,
 ContentHeight: 23.099999999999998,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.FontSize = 16.5
		s.LineHeight = 23.099999999999998
		s.LetterSpacing = 1.5
		s.FontWeight = 700
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Text: "NORTHGATE ENGINEERING LTD",
 ContentX: 30,
 ContentY: 30,
 ContentWidth: 292.4765625,
 ContentHeight: 23.099999999999998,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7450980392156863, 0.8901960784313725, 0.9725490196078431, 1)
		s.MarginTop = layouter_domain.DimensionPt(3)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 30,
 ContentY: 56.099999999999994,
 ContentWidth: 356.25,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7450980392156863, 0.8901960784313725, 0.9725490196078431, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "47 Harbour Street, Bristol, BS1 4AH, United Kingdom",
 ContentX: 30,
 ContentY: 56.099999999999994,
 ContentWidth: 166.6171875,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7450980392156863, 0.8901960784313725, 0.9725490196078431, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 30,
 ContentY: 65.55,
 ContentWidth: 356.25,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.7450980392156863, 0.8901960784313725, 0.9725490196078431, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "Tel: +44 117 496 0023 | accounts@northgate-eng.example.com",
 ContentX: 30,
 ContentY: 65.55,
 ContentWidth: 199.86328125,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlex,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.MarginTop = layouter_domain.DimensionPt(13.5)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayFlex
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 103.5,
 ContentWidth: 386.25,
 ContentHeight: 31.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 103.5,
 ContentWidth: 291.52734375,
 ContentHeight: 31.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.10196078431372549, 0.21176470588235294, 0.36470588235294116, 1)
		s.FontSize = 15
		s.LineHeight = 21
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 103.5,
 ContentWidth: 291.52734375,
 ContentHeight: 21,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.10196078431372549, 0.21176470588235294, 0.36470588235294116, 1)
		s.FontSize = 15
		s.LineHeight = 21
		s.FontWeight = 700
	}),
 Text: "INVOICE",
 ContentX: 15,
 ContentY: 103.5,
 ContentWidth: 63.375,
 ContentHeight: 21,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 ContentX: 306.52734375,
 ContentY: 103.5,
 ContentWidth: 94.72265625,
 ContentHeight: 31.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 ContentX: 306.52734375,
 ContentY: 103.5,
 ContentWidth: 94.72265625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "Invoice No: NE-2026-00184",
 ContentX: 306.52734375,
 ContentY: 103.5,
 ContentWidth: 94.72265625,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 ContentX: 306.52734375,
 ContentY: 114,
 ContentWidth: 94.72265625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "Date: 14 March 2026",
 ContentX: 328.8515625,
 ContentY: 114,
 ContentWidth: 72.3984375,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 ContentX: 306.52734375,
 ContentY: 124.5,
 ContentWidth: 94.72265625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "Due: 13 April 2026",
 ContentX: 336.94921875,
 ContentY: 124.5,
 ContentWidth: 64.30078125,
 ContentHeight: 10.5,
},
},
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlex,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.MarginTop = layouter_domain.DimensionPt(12)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayFlex
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 147,
 ContentWidth: 386.25,
 ContentHeight: 84.9,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BorderTopColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderRightColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BackgroundColour = layouter_domain.NewRGBA(0.9686274509803922, 0.9803921568627451, 0.9882352941176471, 1)
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderLeftColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.MarginRight = layouter_domain.DimensionPt(6)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.PaddingTop = 9
		s.PaddingRight = 9
		s.PaddingBottom = 9
		s.PaddingLeft = 9
		s.BorderTopWidth = 0.75
		s.BorderRightWidth = 0.75
		s.BorderBottomWidth = 0.75
		s.BorderLeftWidth = 0.75
		s.BorderTopLeftRadius = 3
		s.BorderTopRightRadius = 3
		s.BorderBottomRightRadius = 3
		s.BorderBottomLeftRadius = 3
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
Top: 9,
 Right: 9,
 Bottom: 9,
 Left: 9,
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
 ContentX: 24.75,
 ContentY: 156.75,
 ContentWidth: 167.625,
 ContentHeight: 65.4,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.MarginBottom = layouter_domain.DimensionPt(4.5)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.LetterSpacing = 0.75
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 4.5,
},
 ContentX: 24.75,
 ContentY: 156.75,
 ContentWidth: 167.625,
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
 Text: "BILL TO",
 ContentX: 24.75,
 ContentY: 156.75,
 ContentWidth: 27.9140625,
 ContentHeight: 8.399999999999999,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 24.75,
 ContentY: 169.65,
 ContentWidth: 167.625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FontWeight = 700
	}),
 Text: "Whitfield Construction Group",
 ContentX: 24.75,
 ContentY: 169.65,
 ContentWidth: 109.58203125,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 24.75,
 ContentY: 180.15,
 ContentWidth: 167.625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
	}),
 Text: "Rebecca Whitfield",
 ContentX: 24.75,
 ContentY: 180.15,
 ContentWidth: 62.63671875,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 24.75,
 ContentY: 190.65,
 ContentWidth: 167.625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
	}),
 Text: "12 Cabot Square",
 ContentX: 24.75,
 ContentY: 190.65,
 ContentWidth: 58.03125,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 24.75,
 ContentY: 201.15,
 ContentWidth: 167.625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
	}),
 Text: "London, E14 4QQ",
 ContentX: 24.75,
 ContentY: 201.15,
 ContentWidth: 61.55859375,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 24.75,
 ContentY: 211.65,
 ContentWidth: 167.625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
	}),
 Text: "United Kingdom",
 ContentX: 24.75,
 ContentY: 211.65,
 ContentWidth: 57.55078125,
 ContentHeight: 10.5,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BorderTopColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderRightColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BackgroundColour = layouter_domain.NewRGBA(0.9686274509803922, 0.9803921568627451, 0.9882352941176471, 1)
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderLeftColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.PaddingTop = 9
		s.PaddingRight = 9
		s.PaddingBottom = 9
		s.PaddingLeft = 9
		s.BorderTopWidth = 0.75
		s.BorderRightWidth = 0.75
		s.BorderBottomWidth = 0.75
		s.BorderLeftWidth = 0.75
		s.BorderTopLeftRadius = 3
		s.BorderTopRightRadius = 3
		s.BorderBottomRightRadius = 3
		s.BorderBottomLeftRadius = 3
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
Top: 9,
 Right: 9,
 Bottom: 9,
 Left: 9,
},
 Border: layouter_domain.BoxEdges{
Top: 0.75,
 Right: 0.75,
 Bottom: 0.75,
 Left: 0.75,
},
 ContentX: 217.875,
 ContentY: 156.75,
 ContentWidth: 173.625,
 ContentHeight: 65.4,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.MarginBottom = layouter_domain.DimensionPt(4.5)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.LetterSpacing = 0.75
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextTransform = layouter_domain.TextTransformUppercase
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 4.5,
},
 ContentX: 217.875,
 ContentY: 156.75,
 ContentWidth: 173.625,
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
 Text: "SHIP TO",
 ContentX: 217.875,
 ContentY: 156.75,
 ContentWidth: 28.8515625,
 ContentHeight: 8.399999999999999,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 217.875,
 ContentY: 169.65,
 ContentWidth: 173.625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FontWeight = 700
	}),
 Text: "Whitfield Construction Group",
 ContentX: 217.875,
 ContentY: 169.65,
 ContentWidth: 109.58203125,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 217.875,
 ContentY: 180.15,
 ContentWidth: 173.625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
	}),
 Text: "Site Office, Plot 7",
 ContentX: 217.875,
 ContentY: 180.15,
 ContentWidth: 59.61328125,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 217.875,
 ContentY: 190.65,
 ContentWidth: 173.625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
	}),
 Text: "Riverside Development",
 ContentX: 217.875,
 ContentY: 190.65,
 ContentWidth: 81.328125,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 217.875,
 ContentY: 201.15,
 ContentWidth: 173.625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
	}),
 Text: "Cardiff, CF10 5AL",
 ContentX: 217.875,
 ContentY: 201.15,
 ContentWidth: 59.71875,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 217.875,
 ContentY: 211.65,
 ContentWidth: 173.625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
	}),
 Text: "United Kingdom",
 ContentX: 217.875,
 ContentY: 211.65,
 ContentWidth: 57.55078125,
 ContentHeight: 10.5,
},
},
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTable,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.Width = layouter_domain.DimensionPct(100)
		s.MarginTop = layouter_domain.DimensionPt(13.5)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.BorderSpacing = 1.5
		s.Display = layouter_domain.DisplayTable
		s.BoxSizing = border-box
		s.BorderCollapse = layouter_domain.BorderCollapseCollapse
	}),
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 15,
 ContentY: 245.4,
 ContentWidth: 386.25,
 ContentHeight: 130.575,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableRow,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BackgroundColour = layouter_domain.NewRGBA(0.10196078431372549, 0.21176470588235294, 0.36470588235294116, 1)
		s.Colour = layouter_domain.ColourWhite
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableRow
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 245.4,
 ContentWidth: 386.25,
 ContentHeight: 21.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignLeft
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 ContentX: 22.5,
 ContentY: 251.4,
 ContentWidth: 223.39328437562304,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.TextAlign = layouter_domain.TextAlignLeft
	}),
 Text: "Description",
 ContentX: 22.5,
 ContentY: 251.4,
 ContentWidth: 38.68359375,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.Width = layouter_domain.DimensionPt(30)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 ContentX: 260.893284375623,
 ContentY: 251.4,
 ContentWidth: 17.857047916528217,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.TextAlign = layouter_domain.TextAlignCentre
	}),
 Text: "Qty",
 ContentX: 263.75735520888713,
 ContentY: 251.4,
 ContentWidth: 12.12890625,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.Width = layouter_domain.DimensionPt(52.5)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 ContentX: 293.75033229215126,
 ContentY: 251.4,
 ContentWidth: 42.499833853924365,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "Unit Price",
 ContentX: 303.3321973960756,
 ContentY: 251.4,
 ContentWidth: 32.91796875,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.Width = layouter_domain.DimensionPt(52.5)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 ContentX: 351.2501661460756,
 ContentY: 251.4,
 ContentWidth: 42.499833853924365,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "Amount",
 ContentX: 366.57421875,
 ContentY: 251.4,
 ContentWidth: 27.17578125,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableRow,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BackgroundColour = layouter_domain.ColourWhite
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableRow
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 266.85,
 ContentWidth: 386.25,
 ContentHeight: 21.825,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 22.5,
 ContentY: 272.85,
 ContentWidth: 223.39328437562304,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "Structural steel beams, Grade S355, 254x146x31 UB, 6m lengths",
 ContentX: 22.5,
 ContentY: 272.85,
 ContentWidth: 202.6640625,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 260.893284375623,
 ContentY: 272.85,
 ContentWidth: 17.857047916528217,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignCentre
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "24",
 ContentX: 265.96633958388713,
 ContentY: 272.85,
 ContentWidth: 7.7109375,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 293.75033229215126,
 ContentY: 272.85,
 ContentWidth: 42.499833853924365,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "485.00",
 ContentX: 315.1681348960756,
 ContentY: 272.85,
 ContentWidth: 21.08203125,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 351.2501661460756,
 ContentY: 272.85,
 ContentWidth: 42.499833853924365,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "11,640.00",
 ContentX: 363.15234375,
 ContentY: 272.85,
 ContentWidth: 30.59765625,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableRow,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BackgroundColour = layouter_domain.NewRGBA(0.9686274509803922, 0.9803921568627451, 0.9882352941176471, 1)
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableRow
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 288.675,
 ContentWidth: 386.25,
 ContentHeight: 21.825,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 22.5,
 ContentY: 294.675,
 ContentWidth: 223.39328437562304,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "High-strength bolts M20x60 Grade 8.8, zinc plated (box of 100)",
 ContentX: 22.5,
 ContentY: 294.675,
 ContentWidth: 197.61328125,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 260.893284375623,
 ContentY: 294.675,
 ContentWidth: 17.857047916528217,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignCentre
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "8",
 ContentX: 267.89407395888713,
 ContentY: 294.675,
 ContentWidth: 3.85546875,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 293.75033229215126,
 ContentY: 294.675,
 ContentWidth: 42.499833853924365,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "62.50",
 ContentX: 319.0236036460756,
 ContentY: 294.675,
 ContentWidth: 17.2265625,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 351.2501661460756,
 ContentY: 294.675,
 ContentWidth: 42.499833853924365,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "500.00",
 ContentX: 372.66796875,
 ContentY: 294.675,
 ContentWidth: 21.08203125,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableRow,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BackgroundColour = layouter_domain.ColourWhite
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableRow
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 310.5,
 ContentWidth: 386.25,
 ContentHeight: 21.825,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 22.5,
 ContentY: 316.5,
 ContentWidth: 223.39328437562304,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "Fabrication and welding labour, certified welder, per hour",
 ContentX: 22.5,
 ContentY: 316.5,
 ContentWidth: 180.69140625,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 260.893284375623,
 ContentY: 316.5,
 ContentWidth: 17.857047916528217,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignCentre
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "36",
 ContentX: 265.96633958388713,
 ContentY: 316.5,
 ContentWidth: 7.7109375,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 293.75033229215126,
 ContentY: 316.5,
 ContentWidth: 42.499833853924365,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "78.00",
 ContentX: 319.0236036460756,
 ContentY: 316.5,
 ContentWidth: 17.2265625,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 351.2501661460756,
 ContentY: 316.5,
 ContentWidth: 42.499833853924365,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "2,808.00",
 ContentX: 367.0078125,
 ContentY: 316.5,
 ContentWidth: 26.7421875,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableRow,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BackgroundColour = layouter_domain.NewRGBA(0.9686274509803922, 0.9803921568627451, 0.9882352941176471, 1)
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableRow
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 332.325,
 ContentWidth: 386.25,
 ContentHeight: 21.825,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 22.5,
 ContentY: 338.325,
 ContentWidth: 223.39328437562304,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "Protective primer coating, intumescent fire rated, per beam",
 ContentX: 22.5,
 ContentY: 338.325,
 ContentWidth: 188.5546875,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 260.893284375623,
 ContentY: 338.325,
 ContentWidth: 17.857047916528217,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignCentre
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "24",
 ContentX: 265.96633958388713,
 ContentY: 338.325,
 ContentWidth: 7.7109375,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 293.75033229215126,
 ContentY: 338.325,
 ContentWidth: 42.499833853924365,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "45.00",
 ContentX: 319.0236036460756,
 ContentY: 338.325,
 ContentWidth: 17.2265625,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 351.2501661460756,
 ContentY: 338.325,
 ContentWidth: 42.499833853924365,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "1,080.00",
 ContentX: 367.0078125,
 ContentY: 338.325,
 ContentWidth: 26.7421875,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableRow,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BackgroundColour = layouter_domain.ColourWhite
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableRow
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 354.15,
 ContentWidth: 386.25,
 ContentHeight: 21.825,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 22.5,
 ContentY: 360.15,
 ContentWidth: 223.39328437562304,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "Delivery and crane offloading, Cardiff site",
 ContentX: 22.5,
 ContentY: 360.15,
 ContentWidth: 131.07421875,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignCentre
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 260.893284375623,
 ContentY: 360.15,
 ContentWidth: 17.857047916528217,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignCentre
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "1",
 ContentX: 267.89407395888713,
 ContentY: 360.15,
 ContentWidth: 3.85546875,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 293.75033229215126,
 ContentY: 360.15,
 ContentWidth: 42.499833853924365,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "750.00",
 ContentX: 315.1681348960756,
 ContentY: 360.15,
 ContentWidth: 21.08203125,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxTableCell,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.PaddingTop = 6
		s.PaddingRight = 7.5
		s.PaddingBottom = 6
		s.PaddingLeft = 7.5
		s.BorderBottomWidth = 0.375
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayTableCell
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 6,
 Right: 7.5,
 Bottom: 6,
 Left: 7.5,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.375,
},
 ContentX: 351.2501661460756,
 ContentY: 360.15,
 ContentWidth: 42.499833853924365,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.TextAlign = layouter_domain.TextAlignRight
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
	}),
 Text: "750.00",
 ContentX: 372.66796875,
 ContentY: 360.15,
 ContentWidth: 21.08203125,
 ContentHeight: 9.45,
},
},
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlex,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.MarginTop = layouter_domain.DimensionPt(9)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayFlex
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 385.35,
 ContentWidth: 386.25,
 ContentHeight: 56.55,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 385.35,
 ContentWidth: 236.25,
 ContentHeight: 56.55,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.Width = layouter_domain.DimensionPt(150)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 251.25,
 ContentY: 385.35,
 ContentWidth: 150,
 ContentHeight: 56.55,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlex,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.796078431372549, 0.8352941176470589, 0.8784313725490196, 1)
		s.PaddingTop = 3
		s.PaddingBottom = 3
		s.BorderBottomWidth = 0.75
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayFlex
		s.BoxSizing = border-box
		s.BorderBottomStyle = layouter_domain.BorderStyleDashed
	}),
 Padding: layouter_domain.BoxEdges{
Top: 3,
 Bottom: 3,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.75,
},
 ContentX: 251.25,
 ContentY: 388.35,
 ContentWidth: 150,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 251.25,
 ContentY: 388.35,
 ContentWidth: 115.734375,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FlexGrow = 1
	}),
 Text: "Subtotal",
 ContentX: 251.25,
 ContentY: 388.35,
 ContentWidth: 29.4609375,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 ContentX: 366.984375,
 ContentY: 388.35,
 ContentWidth: 34.265625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FontWeight = 700
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "16,778.00",
 ContentX: 366.984375,
 ContentY: 388.35,
 ContentWidth: 34.265625,
 ContentHeight: 10.5,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlex,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.796078431372549, 0.8352941176470589, 0.8784313725490196, 1)
		s.PaddingTop = 3
		s.PaddingBottom = 3
		s.BorderBottomWidth = 0.75
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayFlex
		s.BoxSizing = border-box
		s.BorderBottomStyle = layouter_domain.BorderStyleDashed
	}),
 Padding: layouter_domain.BoxEdges{
Top: 3,
 Bottom: 3,
},
 Border: layouter_domain.BoxEdges{
Bottom: 0.75,
},
 ContentX: 251.25,
 ContentY: 405.6,
 ContentWidth: 150,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FlexGrow = 1
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 251.25,
 ContentY: 405.6,
 ContentWidth: 120.0234375,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.44313725490196076, 0.5019607843137255, 0.5882352941176471, 1)
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FlexGrow = 1
	}),
 Text: "VAT (20%)",
 ContentX: 251.25,
 ContentY: 405.6,
 ContentWidth: 33.9140625,
 ContentHeight: 10.5,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 ContentX: 371.2734375,
 ContentY: 405.6,
 ContentWidth: 29.9765625,
 ContentHeight: 10.5,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.FontWeight = 700
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "3,355.60",
 ContentX: 371.2734375,
 ContentY: 405.6,
 ContentWidth: 29.9765625,
 ContentHeight: 10.5,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlex,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BackgroundColour = layouter_domain.NewRGBA(0.10196078431372549, 0.21176470588235294, 0.36470588235294116, 1)
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.MarginTop = layouter_domain.DimensionPt(1.5)
		s.PaddingTop = 4.5
		s.PaddingRight = 6
		s.PaddingBottom = 4.5
		s.PaddingLeft = 6
		s.BorderTopLeftRadius = 2.25
		s.BorderTopRightRadius = 2.25
		s.BorderBottomRightRadius = 2.25
		s.BorderBottomLeftRadius = 2.25
		s.FontSize = 7.5
		s.LineHeight = 10.5
		s.Display = layouter_domain.DisplayFlex
		s.BoxSizing = border-box
	}),
 Padding: layouter_domain.BoxEdges{
Top: 4.5,
 Right: 6,
 Bottom: 4.5,
 Left: 6,
},
 ContentX: 257.25,
 ContentY: 425.85,
 ContentWidth: 138,
 ContentHeight: 11.549999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.FontSize = 8.25
		s.LineHeight = 11.549999999999999
		s.FlexGrow = 1
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 257.25,
 ContentY: 425.85,
 ContentWidth: 100.265625,
 ContentHeight: 11.549999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.FlexBasis = layouter_domain.DimensionPt(0)
		s.FontSize = 8.25
		s.LineHeight = 11.549999999999999
		s.FlexGrow = 1
		s.FontWeight = 700
	}),
 Text: "Total Due",
 ContentX: 257.25,
 ContentY: 425.85,
 ContentWidth: 38.80078125,
 ContentHeight: 11.549999999999999,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxFlexItem,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.FontSize = 8.25
		s.LineHeight = 11.549999999999999
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 ContentX: 357.515625,
 ContentY: 425.85,
 ContentWidth: 37.734375,
 ContentHeight: 11.549999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.ColourWhite
		s.FontSize = 8.25
		s.LineHeight = 11.549999999999999
		s.FontWeight = 700
		s.TextAlign = layouter_domain.TextAlignRight
	}),
 Text: "20,133.60",
 ContentX: 357.515625,
 ContentY: 425.85,
 ContentWidth: 37.734375,
 ContentHeight: 11.549999999999999,
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
		s.BorderTopColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderRightColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BackgroundColour = layouter_domain.NewRGBA(0.9686274509803922, 0.9803921568627451, 0.9882352941176471, 1)
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.BorderBottomColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.BorderLeftColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.MarginTop = layouter_domain.DimensionPt(15)
		s.PaddingTop = 9
		s.PaddingRight = 9
		s.PaddingBottom = 9
		s.PaddingLeft = 9
		s.BorderTopWidth = 0.75
		s.BorderRightWidth = 0.75
		s.BorderBottomWidth = 0.75
		s.BorderLeftWidth = 0.75
		s.BorderTopLeftRadius = 3
		s.BorderTopRightRadius = 3
		s.BorderBottomRightRadius = 3
		s.BorderBottomLeftRadius = 3
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.BorderTopStyle = layouter_domain.BorderStyleSolid
		s.BorderRightStyle = layouter_domain.BorderStyleSolid
		s.BorderBottomStyle = layouter_domain.BorderStyleSolid
		s.BorderLeftStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 9,
 Right: 9,
 Bottom: 9,
 Left: 9,
},
 Border: layouter_domain.BoxEdges{
Top: 0.75,
 Right: 0.75,
 Bottom: 0.75,
 Left: 0.75,
},
 ContentX: 24.75,
 ContentY: 466.65000000000003,
 ContentWidth: 366.75,
 ContentHeight: 31.349999999999998,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.MarginBottom = layouter_domain.DimensionPt(3)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 Margin: layouter_domain.BoxEdges{
Bottom: 3,
},
 ContentX: 24.75,
 ContentY: 466.65000000000003,
 ContentWidth: 366.75,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.FontWeight = 700
	}),
 Text: "Payment Details",
 ContentX: 24.75,
 ContentY: 466.65000000000003,
 ContentWidth: 55.1484375,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 24.75,
 ContentY: 479.1,
 ContentWidth: 366.75,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "Bank: Lloyds Bank | Sort Code: 30-92-18 | Account: 48291057",
 ContentX: 24.75,
 ContentY: 479.1,
 ContentWidth: 193.88671875,
 ContentHeight: 9.45,
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 24.75,
 ContentY: 488.55,
 ContentWidth: 366.75,
 ContentHeight: 9.45,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.17254901960784313, 0.24313725490196078, 0.3137254901960784, 1)
		s.FontSize = 6.75
		s.LineHeight = 9.45
	}),
 Text: "Reference: NE-2026-00184",
 ContentX: 24.75,
 ContentY: 488.55,
 ContentWidth: 83.296875,
 ContentHeight: 9.45,
},
},
},
},
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.BorderTopColour = layouter_domain.NewRGBA(0.8862745098039215, 0.9098039215686274, 0.9411764705882353, 1)
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.MarginTop = layouter_domain.DimensionPt(10.5)
		s.PaddingTop = 7.5
		s.BorderTopWidth = 0.75
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
		s.BorderTopStyle = layouter_domain.BorderStyleSolid
	}),
 Padding: layouter_domain.BoxEdges{
Top: 7.5,
},
 Border: layouter_domain.BoxEdges{
Top: 0.75,
},
 ContentX: 15,
 ContentY: 526.5,
 ContentWidth: 386.25,
 ContentHeight: 19.799999999999997,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxBlock,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.Display = layouter_domain.DisplayBlock
		s.BoxSizing = border-box
	}),
 ContentX: 15,
 ContentY: 526.5,
 ContentWidth: 386.25,
 ContentHeight: 8.399999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
	}),
 Text: "Payment terms: net 30 days. Late payments are subject to statutory interest under the Late Payment of Commercial Debts Act 1998.",
 ContentX: 15,
 ContentY: 526.5,
 ContentWidth: 370.546875,
 ContentHeight: 8.399999999999999,
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
 ContentX: 15,
 ContentY: 537.9,
 ContentWidth: 386.25,
 ContentHeight: 8.399999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.6274509803921569, 0.6823529411764706, 0.7529411764705882, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
	}),
 Text: "Questions about this invoice? Contact us at ",
 ContentX: 15,
 ContentY: 537.9,
 ContentWidth: 122.49609375,
 ContentHeight: 8.399999999999999,
},
 &layouter_domain.LayoutBox{
Type: layouter_domain.BoxInline,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.16862745098039217, 0.4235294117647059, 0.6901960784313725, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.BoxSizing = border-box
		s.TextDecoration = layouter_domain.TextDecorationUnderline
	}),
 ContentX: 137.49609375,
 ContentY: 537.9,
 ContentWidth: 110.98828125,
 ContentHeight: 8.399999999999999,
 Children: []*layouter_domain.LayoutBox{
&layouter_domain.LayoutBox{
Type: layouter_domain.BoxTextRun,
 Style: withStyle(func(s *layouter_domain.ComputedStyle) {
		s.Colour = layouter_domain.NewRGBA(0.16862745098039217, 0.4235294117647059, 0.6901960784313725, 1)
		s.FontSize = 6
		s.LineHeight = 8.399999999999999
		s.TextDecoration = layouter_domain.TextDecorationUnderline
	}),
 Text: "accounts@northgate-eng.example.com",
 ContentX: 137.49609375,
 ContentY: 537.9,
 ContentWidth: 110.98828125,
 ContentHeight: 8.399999999999999,
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
}
}()
