package main

const (
	operationCount   = 70
	keyLength        = 12
	idRangeSize      = 32768
	idMask           = idRangeSize - 1
	initialCapacity  = 64
	hashtableLcgMask = 0xFFFFFFFF
	fnvOffsetBasis   = 0x811C9DC5
	fnvPrime         = 0x01000193
	slotEmpty        = 0
	slotOccupied     = 1
	slotTombstone    = 2
	opSeed           = 0xDEADBEEF
)

type hashTable struct {
	keys     []string
	values   []int
	state    []byte
	capacity int
	size     int
	load     int
}

func newHashTable() *hashTable {
	return &hashTable{
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

func (table *hashTable) put(key string, value int) {
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

func (table *hashTable) get(key string) (int, bool) {
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

func (table *hashTable) deleteKey(key string) bool {
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

func (table *hashTable) grow() {
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

func (table *hashTable) insertNoGrow(key string, value int) {
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

func run() int {
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
	return table.size*100000 + hitCount
}
