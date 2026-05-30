package main

import "time"

const imageWidth = 200

const imageHeight = 200

const maxIterations = 80

const (
	viewRealMin = -2.0

	viewRealMax = 1.0

	viewImagMin = -1.5

	viewImagMax = 1.5
)

const escapeThresholdSquared = 4.0

func Run() string {
	return doMandelbrotRender()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doMandelbrotRender()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doMandelbrotRender() string {
	realStep := (viewRealMax - viewRealMin) / float64(imageWidth)
	imagStep := (viewImagMax - viewImagMin) / float64(imageHeight)
	var totalIterations int64
	for pixelY := 0; pixelY < imageHeight; pixelY++ {
		cIm := viewImagMin + float64(pixelY)*imagStep
		for pixelX := 0; pixelX < imageWidth; pixelX++ {
			cRe := viewRealMin + float64(pixelX)*realStep
			zRe := 0.0
			zIm := 0.0
			iterations := 0
			for iterations < maxIterations {
				zReSquared := zRe * zRe
				zImSquared := zIm * zIm
				if zReSquared+zImSquared > escapeThresholdSquared {
					break
				}
				zIm = 2.0*zRe*zIm + cIm
				zRe = zReSquared - zImSquared + cRe
				iterations++
			}
			totalIterations += int64(iterations)
		}
	}
	return int64ToDecimal(totalIterations)
}

func int64ToDecimal(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	digits := [20]byte{}
	position := len(digits)
	for value > 0 {
		position--
		digits[position] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		position--
		digits[position] = '-'
	}
	return string(digits[position:])
}
