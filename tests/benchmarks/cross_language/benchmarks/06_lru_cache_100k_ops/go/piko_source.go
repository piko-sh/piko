package main

import "time"

const cacheCapacity = 1024

const totalOperations = 100000

const keyspaceSize = cacheCapacity * 4

const lcgMask = 0xFFFFFFFF

func Run() string {
	return doLRU()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doLRU()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doLRU() string {
	cache := newLRUCache(cacheCapacity)
	state := uint32(13579246)
	var hitCount int
	for operationIndex := 0; operationIndex < totalOperations; operationIndex++ {
		state = (state*1664525 + 1013904223) & lcgMask
		key := int((state >> 8) & uint32(keyspaceSize-1))
		isGet := state>>31 == 0
		if isGet {
			_, found := cache.get(key)
			if found {
				hitCount++
			}
		} else {
			value := int(state & 0xFFFF)
			cache.put(key, value)
		}
	}
	return intToDecimalString(hitCount)
}

type lruNode struct {
	key int

	value int

	previous *lruNode

	next *lruNode
}

type lruCache struct {
	capacity int

	lookup map[int]*lruNode

	head *lruNode

	tail *lruNode

	size int
}

func newLRUCache(capacity int) *lruCache {
	return &lruCache{capacity: capacity, lookup: map[int]*lruNode{}}
}

func (cache *lruCache) get(key int) (int, bool) {
	node, found := cache.lookup[key]
	if !found {
		return 0, false
	}
	cache.moveToFront(node)
	return node.value, true
}

func (cache *lruCache) put(key, value int) {
	if existing, found := cache.lookup[key]; found {
		existing.value = value
		cache.moveToFront(existing)
		return
	}
	node := &lruNode{key: key, value: value}
	cache.lookup[key] = node
	cache.attachAtFront(node)
	cache.size++
	if cache.size > cache.capacity {
		removed := cache.tail
		cache.detach(removed)
		delete(cache.lookup, removed.key)
		cache.size--
	}
}

func (cache *lruCache) moveToFront(node *lruNode) {
	if node == cache.head {
		return
	}
	cache.detach(node)
	cache.attachAtFront(node)
}

func (cache *lruCache) attachAtFront(node *lruNode) {
	node.previous = nil
	node.next = cache.head
	if cache.head != nil {
		cache.head.previous = node
	}
	cache.head = node
	if cache.tail == nil {
		cache.tail = node
	}
}

func (cache *lruCache) detach(node *lruNode) {
	if node.previous != nil {
		node.previous.next = node.next
	} else {
		cache.head = node.next
	}
	if node.next != nil {
		node.next.previous = node.previous
	} else {
		cache.tail = node.previous
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
