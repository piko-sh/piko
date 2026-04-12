package main

import "time"

const (
	denseInputs = 256

	denseOutputs = 256
)

func Run() string {
	weights := buildWeights()
	input := buildInput()
	bias := buildBias()
	output := make([]float64, denseOutputs)
	denseForward(weights, input, bias, output)
	return summariseOutput(output)
}

func RunInner(k int) (string, int64) {
	weights := buildWeights()
	input := buildInput()
	bias := buildBias()
	output := make([]float64, denseOutputs)

	startNanos := time.Now().UnixNano()
	for index := 0; index < k; index++ {
		denseForward(weights, input, bias, output)
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return summariseOutput(output), elapsedNanos
}

func denseForward(weights [][]float64, input []float64, bias []float64, output []float64) {
	for i := 0; i < len(output); i++ {
		row := weights[i]
		sum := 0.0
		for j := 0; j < len(row); j++ {
			sum += row[j] * input[j]
		}
		sum += bias[i]
		if sum < 0 {
			sum = 0
		}
		output[i] = sum
	}
}

func buildWeights() [][]float64 {
	weights := make([][]float64, denseOutputs)
	for i := 0; i < denseOutputs; i++ {
		row := make([]float64, denseInputs)
		for j := 0; j < denseInputs; j++ {
			row[j] = float64(((i+j*7)%31)-15) * 0.01
		}
		weights[i] = row
	}
	return weights
}

func buildInput() []float64 {
	input := make([]float64, denseInputs)
	for j := 0; j < denseInputs; j++ {
		input[j] = float64(((j*13)%17)-8) * 0.1
	}
	return input
}

func buildBias() []float64 {
	bias := make([]float64, denseOutputs)
	for i := 0; i < denseOutputs; i++ {
		bias[i] = float64(((i*5)%11)-5) * 0.1
	}
	return bias
}

func summariseOutput(output []float64) string {
	total := 0.0
	for i := 0; i < len(output); i++ {
		total += output[i]
	}
	scaled := int64(total * 1000.0)
	return int64ToDecimalString(scaled)
}

func int64ToDecimalString(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	digits := [21]byte{}
	position := len(digits)
	for value > 0 {
		position--
		digits[position] = byte('0' + (value % 10))
		value /= 10
	}
	if negative {
		position--
		digits[position] = '-'
	}
	return string(digits[position:])
}
