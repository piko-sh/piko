package main

import "time"

const nodeCount = 10000

const averageDegree = 6

const maxEdgeWeight = 1000

const resultMask = 0xFFFFFFFF

const lcgMask = 0xFFFFFFFF

const unreachable = 1 << 30

func Run() string {
	return doDijkstra()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doDijkstra()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doDijkstra() string {
	edgeHeads, edgeTargets, edgeWeights := generateGraph()
	distances := runDijkstra(edgeHeads, edgeTargets, edgeWeights)
	var runningSum uint32
	for _, distance := range distances {
		if distance < unreachable {
			runningSum = (runningSum + uint32(distance)) & resultMask
		}
	}
	return intToDecimalString(int(runningSum))
}

func generateGraph() ([]int, []int, []int) {
	totalEdgesUpperBound := nodeCount * averageDegree * 2
	edgeTargetsBuffer := make([]int, 0, totalEdgesUpperBound)
	edgeWeightsBuffer := make([]int, 0, totalEdgesUpperBound)
	edgeHeads := make([]int, nodeCount+1)
	state := uint32(11223344)
	for nodeIndex := 0; nodeIndex < nodeCount; nodeIndex++ {
		edgeHeads[nodeIndex] = len(edgeTargetsBuffer)
		state = (state*1664525 + 1013904223) & lcgMask
		outgoingDegree := int(state%uint32(averageDegree*2)) + 1
		for edgeIndex := 0; edgeIndex < outgoingDegree; edgeIndex++ {
			state = (state*1664525 + 1013904223) & lcgMask
			target := int(state % uint32(nodeCount))
			state = (state*1664525 + 1013904223) & lcgMask
			weight := int(state%uint32(maxEdgeWeight)) + 1
			edgeTargetsBuffer = append(edgeTargetsBuffer, target)
			edgeWeightsBuffer = append(edgeWeightsBuffer, weight)
		}
	}
	edgeHeads[nodeCount] = len(edgeTargetsBuffer)
	return edgeHeads, edgeTargetsBuffer, edgeWeightsBuffer
}

func runDijkstra(edgeHeads, edgeTargets, edgeWeights []int) []int {
	distances := make([]int, nodeCount)
	for nodeIndex := 0; nodeIndex < nodeCount; nodeIndex++ {
		distances[nodeIndex] = unreachable
	}
	distances[0] = 0

	heap := make([]int, 0, nodeCount*2)
	heapDistances := make([]int, 0, nodeCount*2)
	heap = append(heap, 0)
	heapDistances = append(heapDistances, 0)

	for len(heap) > 0 {
		currentNode := heap[0]
		currentDistance := heapDistances[0]
		heapPopFront(&heap, &heapDistances)
		if currentDistance > distances[currentNode] {
			continue
		}
		edgeRangeStart := edgeHeads[currentNode]
		edgeRangeEnd := edgeHeads[currentNode+1]
		for edgeIndex := edgeRangeStart; edgeIndex < edgeRangeEnd; edgeIndex++ {
			neighbour := edgeTargets[edgeIndex]
			candidate := currentDistance + edgeWeights[edgeIndex]
			if candidate < distances[neighbour] {
				distances[neighbour] = candidate
				heapPush(&heap, &heapDistances, neighbour, candidate)
			}
		}
	}
	return distances
}

func heapPush(heap, heapDistances *[]int, node, distance int) {
	*heap = append(*heap, node)
	*heapDistances = append(*heapDistances, distance)
	siftUp(heap, heapDistances, len(*heap)-1)
}

func heapPopFront(heap, heapDistances *[]int) {
	lastIndex := len(*heap) - 1
	(*heap)[0] = (*heap)[lastIndex]
	(*heapDistances)[0] = (*heapDistances)[lastIndex]
	*heap = (*heap)[:lastIndex]
	*heapDistances = (*heapDistances)[:lastIndex]
	if len(*heap) > 0 {
		siftDown(heap, heapDistances, 0)
	}
}

func siftUp(heap, heapDistances *[]int, position int) {
	for position > 0 {
		parent := (position - 1) / 2
		if (*heapDistances)[parent] <= (*heapDistances)[position] {
			break
		}
		(*heap)[parent], (*heap)[position] = (*heap)[position], (*heap)[parent]
		(*heapDistances)[parent], (*heapDistances)[position] = (*heapDistances)[position], (*heapDistances)[parent]
		position = parent
	}
}

func siftDown(heap, heapDistances *[]int, position int) {
	length := len(*heap)
	for {
		leftChild := 2*position + 1
		rightChild := 2*position + 2
		smallest := position
		if leftChild < length && (*heapDistances)[leftChild] < (*heapDistances)[smallest] {
			smallest = leftChild
		}
		if rightChild < length && (*heapDistances)[rightChild] < (*heapDistances)[smallest] {
			smallest = rightChild
		}
		if smallest == position {
			break
		}
		(*heap)[position], (*heap)[smallest] = (*heap)[smallest], (*heap)[position]
		(*heapDistances)[position], (*heapDistances)[smallest] = (*heapDistances)[smallest], (*heapDistances)[position]
		position = smallest
	}
}

func intToDecimalString(value int) string {
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
