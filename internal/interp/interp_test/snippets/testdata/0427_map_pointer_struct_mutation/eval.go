package main

const cacheCapacity = 4
const totalOperations = 100
const keyspaceSize = cacheCapacity * 4
const lcgMask = 0xFFFFFFFF

type node struct {
	key      int
	value    int
	previous *node
	next     *node
}

type cache struct {
	capacity int
	lookup   map[int]*node
	head     *node
	tail     *node
	size     int
}

func newCache(capacity int) *cache {
	return &cache{capacity: capacity, lookup: map[int]*node{}}
}

func (c *cache) get(key int) (int, bool) {
	n, found := c.lookup[key]
	if !found {
		return 0, false
	}
	c.moveToFront(n)
	return n.value, true
}

func (c *cache) put(key, value int) {
	if existing, found := c.lookup[key]; found {
		existing.value = value
		c.moveToFront(existing)
		return
	}
	n := &node{key: key, value: value}
	c.lookup[key] = n
	c.attachAtFront(n)
	c.size++
	if c.size > c.capacity {
		removed := c.tail
		c.detach(removed)
		delete(c.lookup, removed.key)
		c.size--
	}
}

func (c *cache) moveToFront(n *node) {
	if n == c.head {
		return
	}
	c.detach(n)
	c.attachAtFront(n)
}

func (c *cache) attachAtFront(n *node) {
	n.previous = nil
	n.next = c.head
	if c.head != nil {
		c.head.previous = n
	}
	c.head = n
	if c.tail == nil {
		c.tail = n
	}
}

func (c *cache) detach(n *node) {
	if n.previous != nil {
		n.previous.next = n.next
	} else {
		c.head = n.next
	}
	if n.next != nil {
		n.next.previous = n.previous
	} else {
		c.tail = n.previous
	}
}

func run() int {
	c := newCache(cacheCapacity)
	state := uint32(13579246)
	hitCount := 0
	for index := 0; index < totalOperations; index++ {
		state = (state*1664525 + 1013904223) & lcgMask
		key := int((state >> 8) & uint32(keyspaceSize-1))
		isGet := state>>31 == 0
		if isGet {
			_, found := c.get(key)
			if found {
				hitCount++
			}
		} else {
			value := int(state & 0xFFFF)
			c.put(key, value)
		}
	}
	return hitCount
}
