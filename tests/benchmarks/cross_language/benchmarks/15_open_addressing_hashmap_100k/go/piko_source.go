package main

import "time"

const operationCount = 100000

const keyLength = 12

const idRangeSize = 32768

const idMask = idRangeSize - 1

const initialCapacity = 64

const hashtableLcgMask = 0xFFFFFFFF

const fnvOffsetBasis = 0x811C9DC5
const fnvPrime = 0x01000193

const (
	slotEmpty = 0

	slotOccupied = 1

	slotTombstone = 2
)

const opSeed = 0xDEADBEEF

type HashTable struct {
	keys []string

	values []int

	state []byte

	capacity int

	size int

	load int
}

func newHashTable() *HashTable {
	return &HashTable{
		keys:     make([]string, initialCapacity),
		values:   make([]int, initialCapacity),
		state:    make([]byte, initialCapacity),
		capacity: initialCapacity,
	}
}

func fnv1a32(key string) uint32 {
	hash := uint32(fnvOffsetBasis)
	for index := 0; index < len(key); index++ {
		hash ^= uint32(key[index])
		hash = (hash * fnvPrime) & hashtableLcgMask
	}
	return hash
}

func (table *HashTable) put(key string, value int) {
	if table.load*10 >= table.capacity*7 {
		table.grow()
	}
	hash := fnv1a32(key)
	mask := uint32(table.capacity - 1)
	index := int(hash & mask)
	firstTombstone := -1
	for {
		slotState := table.state[index]
		if slotState == slotEmpty {
			if firstTombstone >= 0 {
				table.keys[firstTombstone] = key
				table.values[firstTombstone] = value
				table.state[firstTombstone] = slotOccupied
				table.size++
				return
			}
			table.keys[index] = key
			table.values[index] = value
			table.state[index] = slotOccupied
			table.size++
			table.load++
			return
		}
		if slotState == slotOccupied && table.keys[index] == key {
			table.values[index] = value
			return
		}
		if slotState == slotTombstone && firstTombstone < 0 {
			firstTombstone = index
		}
		index = int(uint32(index+1) & mask)
	}
}

func (table *HashTable) get(key string) (int, bool) {
	hash := fnv1a32(key)
	mask := uint32(table.capacity - 1)
	index := int(hash & mask)
	for table.state[index] != slotEmpty {
		if table.state[index] == slotOccupied && table.keys[index] == key {
			return table.values[index], true
		}
		index = int(uint32(index+1) & mask)
	}
	return 0, false
}

func (table *HashTable) deleteKey(key string) bool {
	hash := fnv1a32(key)
	mask := uint32(table.capacity - 1)
	index := int(hash & mask)
	for table.state[index] != slotEmpty {
		if table.state[index] == slotOccupied && table.keys[index] == key {
			table.state[index] = slotTombstone
			table.size--
			return true
		}
		index = int(uint32(index+1) & mask)
	}
	return false
}

func (table *HashTable) grow() {
	oldKeys := table.keys
	oldValues := table.values
	oldState := table.state
	oldCapacity := table.capacity
	table.capacity = oldCapacity * 2
	table.keys = make([]string, table.capacity)
	table.values = make([]int, table.capacity)
	table.state = make([]byte, table.capacity)
	table.size = 0
	table.load = 0
	for index := 0; index < oldCapacity; index++ {
		if oldState[index] == slotOccupied {
			table.insertNoGrow(oldKeys[index], oldValues[index])
		}
	}
}

func (table *HashTable) insertNoGrow(key string, value int) {
	hash := fnv1a32(key)
	mask := uint32(table.capacity - 1)
	index := int(hash & mask)
	for table.state[index] != slotEmpty {
		index = int(uint32(index+1) & mask)
	}
	table.keys[index] = key
	table.values[index] = value
	table.state[index] = slotOccupied
	table.size++
	table.load++
}

func generateKey(id uint32) string {
	keyBytes := [keyLength]byte{}
	for index := keyLength - 1; index >= 0; index-- {
		keyBytes[index] = byte('a' + id%26)
		id /= 26
	}
	return string(keyBytes[:])
}

func Run() string {
	return doHashtableWorkload()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doHashtableWorkload()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doHashtableWorkload() string {
	table := newHashTable()
	state := uint32(opSeed)
	hitCount := 0
	for operationIndex := 0; operationIndex < operationCount; operationIndex++ {
		state = (state*1664525 + 1013904223) & hashtableLcgMask
		opKind := (state >> 28) & 0xF
		id := (state >> 4) & idMask
		key := generateKey(id)
		if opKind < 8 {
			table.put(key, int(state))
		} else if opKind < 14 {
			_, found := table.get(key)
			if found {
				hitCount++
			}
		} else {
			table.deleteKey(key)
		}
	}
	return intPairToString(table.size, hitCount)
}

func intPairToString(first int, second int) string {
	return intToDecimal(first) + "," + intToDecimal(second)
}

func intToDecimal(value int) string {
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
