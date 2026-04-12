import argparse
import sys
import time

CACHE_CAPACITY = 1024
TOTAL_OPERATIONS = 100_000
KEYSPACE_SIZE = CACHE_CAPACITY * 4
LCG_MASK = 0xFFFFFFFF


class LRUNode:
    __slots__ = ("key", "value", "previous", "next_node")

    def __init__(self, key: int, value: int) -> None:
        self.key = key
        self.value = value
        self.previous = None
        self.next_node = None


class LRUCache:
    __slots__ = ("capacity", "lookup", "head", "tail", "size")

    def __init__(self, capacity: int) -> None:
        self.capacity = capacity
        self.lookup: dict[int, LRUNode] = {}
        self.head: LRUNode | None = None
        self.tail: LRUNode | None = None
        self.size = 0

    def get(self, key: int) -> tuple[int, bool]:
        node = self.lookup.get(key)
        if node is None:
            return 0, False
        self.move_to_front(node)
        return node.value, True

    def put(self, key: int, value: int) -> None:
        existing = self.lookup.get(key)
        if existing is not None:
            existing.value = value
            self.move_to_front(existing)
            return
        node = LRUNode(key, value)
        self.lookup[key] = node
        self.attach_at_front(node)
        self.size += 1
        if self.size > self.capacity:
            removed = self.tail
            assert removed is not None
            self.detach(removed)
            del self.lookup[removed.key]
            self.size -= 1

    def move_to_front(self, node: LRUNode) -> None:
        if node is self.head:
            return
        self.detach(node)
        self.attach_at_front(node)

    def attach_at_front(self, node: LRUNode) -> None:
        node.previous = None
        node.next_node = self.head
        if self.head is not None:
            self.head.previous = node
        self.head = node
        if self.tail is None:
            self.tail = node

    def detach(self, node: LRUNode) -> None:
        if node.previous is not None:
            node.previous.next_node = node.next_node
        else:
            self.head = node.next_node
        if node.next_node is not None:
            node.next_node.previous = node.previous
        else:
            self.tail = node.previous


def int_to_decimal_string(value: int) -> str:
    if value == 0:
        return "0"
    negative = value < 0
    if negative:
        value = -value
    digits = []
    while value > 0:
        digits.append(chr(ord("0") + value % 10))
        value //= 10
    digits.reverse()
    if negative:
        return "-" + "".join(digits)
    return "".join(digits)


def do_lru() -> str:
    cache = LRUCache(CACHE_CAPACITY)
    state = 13579246
    hit_count = 0
    for _ in range(TOTAL_OPERATIONS):
        state = (state * 1664525 + 1013904223) & LCG_MASK
        key = (state >> 8) & (KEYSPACE_SIZE - 1)
        is_get = state >> 31 == 0
        if is_get:
            _value, found = cache.get(key)
            if found:
                hit_count += 1
        else:
            value = state & 0xFFFF
            cache.put(key, value)
    return int_to_decimal_string(hit_count)


def run() -> str:
    return do_lru()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_lru()
    elapsed_nanos = time.perf_counter_ns() - start_nanos
    return last, elapsed_nanos


def _main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--mode", default="endtoend", choices=["endtoend", "inner"])
    parser.add_argument("--k", type=int, default=0)
    arguments, _unknown = parser.parse_known_args()

    if arguments.mode == "inner":
        result, elapsed = run_inner(arguments.k)
        print(result)
        print(f"INNER_ELAPSED_NS={elapsed}", file=sys.stderr)
        return

    print(run())

if __name__ == "__main__":
    _main()
