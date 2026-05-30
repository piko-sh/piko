package main

import "time"

const lcgMask = 0xFFFFFFFF

const resultMask = 0xFFFFFFFF

const treeDepthBudget = 12

const walksPerEval = 100

const envSlotCount = 8

const treeSeed = 12345678

const envSeed = 0x517cc1b7

type Node interface {
	Eval(env []uint32) uint32
}

type addNode struct {
	left, right Node
}

func (node *addNode) Eval(env []uint32) uint32 {
	return (node.left.Eval(env) + node.right.Eval(env)) & resultMask
}

type subNode struct {
	left, right Node
}

func (node *subNode) Eval(env []uint32) uint32 {
	return (node.left.Eval(env) - node.right.Eval(env)) & resultMask
}

type mulNode struct {
	left, right Node
}

func (node *mulNode) Eval(env []uint32) uint32 {
	return (node.left.Eval(env) * node.right.Eval(env)) & resultMask
}

type modNode struct {
	left, right Node
}

func (node *modNode) Eval(env []uint32) uint32 {
	rightValue := node.right.Eval(env)
	if rightValue == 0 {
		return 0
	}
	return node.left.Eval(env) % rightValue
}

type minNode struct {
	left, right Node
}

func (node *minNode) Eval(env []uint32) uint32 {
	leftValue := node.left.Eval(env)
	rightValue := node.right.Eval(env)
	if leftValue < rightValue {
		return leftValue
	}
	return rightValue
}

type maxNode struct {
	left, right Node
}

func (node *maxNode) Eval(env []uint32) uint32 {
	leftValue := node.left.Eval(env)
	rightValue := node.right.Eval(env)
	if leftValue > rightValue {
		return leftValue
	}
	return rightValue
}

type ifPosNode struct {
	condition, thenBranch, elseBranch Node
}

func (node *ifPosNode) Eval(env []uint32) uint32 {
	if node.condition.Eval(env) != 0 {
		return node.thenBranch.Eval(env)
	}
	return node.elseBranch.Eval(env)
}

type constNode struct {
	value uint32
}

func (node *constNode) Eval(env []uint32) uint32 {
	return node.value
}

type varNode struct {
	slot int
}

func (node *varNode) Eval(env []uint32) uint32 {
	return env[node.slot]
}

func buildTree(state uint32, depthBudget int) (Node, uint32) {
	state = (state*1664525 + 1013904223) & lcgMask
	if depthBudget <= 0 || state&0x7 == 0 {
		state = (state*1664525 + 1013904223) & lcgMask
		if state&0x1 == 0 {
			return &constNode{value: (state >> 1) & 0xFF}, state
		}
		return &varNode{slot: int((state >> 1) & 0x7)}, state
	}
	kind := state & 0x7
	if kind == 7 {
		var condition, thenBranch, elseBranch Node
		condition, state = buildTree(state, depthBudget-1)
		thenBranch, state = buildTree(state, depthBudget-1)
		elseBranch, state = buildTree(state, depthBudget-1)
		return &ifPosNode{condition: condition, thenBranch: thenBranch, elseBranch: elseBranch}, state
	}
	var left, right Node
	left, state = buildTree(state, depthBudget-1)
	right, state = buildTree(state, depthBudget-1)
	switch kind {
	case 0:
		return &addNode{left: left, right: right}, state
	case 1:
		return &subNode{left: left, right: right}, state
	case 2:
		return &mulNode{left: left, right: right}, state
	case 3:
		return &modNode{left: left, right: right}, state
	case 4:
		return &minNode{left: left, right: right}, state
	case 5:
		return &maxNode{left: left, right: right}, state
	}
	return &addNode{left: left, right: right}, state
}

func Run() string {
	return doAstEval()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doAstEval()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doAstEval() string {
	tree, _ := buildTree(uint32(treeSeed), treeDepthBudget)
	envState := uint32(envSeed)
	env := make([]uint32, envSlotCount)
	var accumulator uint32
	for walk := 0; walk < walksPerEval; walk++ {
		for slot := 0; slot < envSlotCount; slot++ {
			envState = (envState*1664525 + 1013904223) & lcgMask
			env[slot] = envState
		}
		accumulator = (accumulator + tree.Eval(env)) & resultMask
	}
	return intToDecimalString(int(accumulator))
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
