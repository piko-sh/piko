import argparse
import sys
import time

OPERATION_COUNT = 100000
KEY_LENGTH = 12
ID_RANGE_SIZE = 32768
ID_MASK = ID_RANGE_SIZE - 1
INITIAL_CAPACITY = 64
HASHTABLE_LCG_MASK = 0xFFFFFFFF
FNV_OFFSET_BASIS = 0x811C9DC5
FNV_PRIME = 0x01000193
SLOT_EMPTY = 0
SLOT_OCCUPIED = 1
SLOT_TOMBSTONE = 2
OP_SEED = 0xDEADBEEF


class HashTable:
    __slots__ = ("keys", "values", "state", "capacity", "size", "load")

    def __init__(self) -> None:
        self.keys: list[str] = [""] * INITIAL_CAPACITY
        self.values: list[int] = [0] * INITIAL_CAPACITY
        self.state: bytearray = bytearray(INITIAL_CAPACITY)
        self.capacity: int = INITIAL_CAPACITY
        self.size: int = 0
        self.load: int = 0

    def put(self, key: str, value: int) -> None:
        if self.load * 10 >= self.capacity * 7:
            self._grow()
        hash_value = fnv1a32(key)
        mask = self.capacity - 1
        index = hash_value & mask
        first_tombstone = -1
        while True:
            slot_state = self.state[index]
            if slot_state == SLOT_EMPTY:
                if first_tombstone >= 0:
                    self.keys[first_tombstone] = key
                    self.values[first_tombstone] = value
                    self.state[first_tombstone] = SLOT_OCCUPIED
                    self.size += 1
                    return
                self.keys[index] = key
                self.values[index] = value
                self.state[index] = SLOT_OCCUPIED
                self.size += 1
                self.load += 1
                return
            if slot_state == SLOT_OCCUPIED and self.keys[index] == key:
                self.values[index] = value
                return
            if slot_state == SLOT_TOMBSTONE and first_tombstone < 0:
                first_tombstone = index
            index = (index + 1) & mask

    def get(self, key: str) -> tuple[int, bool]:
        hash_value = fnv1a32(key)
        mask = self.capacity - 1
        index = hash_value & mask
        while self.state[index] != SLOT_EMPTY:
            if self.state[index] == SLOT_OCCUPIED and self.keys[index] == key:
                return self.values[index], True
            index = (index + 1) & mask
        return 0, False

    def delete_key(self, key: str) -> bool:
        hash_value = fnv1a32(key)
        mask = self.capacity - 1
        index = hash_value & mask
        while self.state[index] != SLOT_EMPTY:
            if self.state[index] == SLOT_OCCUPIED and self.keys[index] == key:
                self.state[index] = SLOT_TOMBSTONE
                self.size -= 1
                return True
            index = (index + 1) & mask
        return False

    def _grow(self) -> None:
        old_keys = self.keys
        old_values = self.values
        old_state = self.state
        old_capacity = self.capacity
        self.capacity = old_capacity * 2
        self.keys = [""] * self.capacity
        self.values = [0] * self.capacity
        self.state = bytearray(self.capacity)
        self.size = 0
        self.load = 0
        for index in range(old_capacity):
            if old_state[index] == SLOT_OCCUPIED:
                self._insert_no_grow(old_keys[index], old_values[index])

    def _insert_no_grow(self, key: str, value: int) -> None:
        hash_value = fnv1a32(key)
        mask = self.capacity - 1
        index = hash_value & mask
        while self.state[index] != SLOT_EMPTY:
            index = (index + 1) & mask
        self.keys[index] = key
        self.values[index] = value
        self.state[index] = SLOT_OCCUPIED
        self.size += 1
        self.load += 1


def fnv1a32(key: str) -> int:
    hash_value = FNV_OFFSET_BASIS
    for index in range(len(key)):
        hash_value ^= ord(key[index])
        hash_value = (hash_value * FNV_PRIME) & HASHTABLE_LCG_MASK
    return hash_value


def generate_key(identifier: int) -> str:
    key_chars = [""] * KEY_LENGTH
    for index in range(KEY_LENGTH - 1, -1, -1):
        key_chars[index] = chr(ord("a") + identifier % 26)
        identifier //= 26
    return "".join(key_chars)


def int_to_decimal(value: int) -> str:
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


def int_pair_to_string(first: int, second: int) -> str:
    return int_to_decimal(first) + "," + int_to_decimal(second)


def do_hashtable_workload() -> str:
    table = HashTable()
    state = OP_SEED
    hit_count = 0
    for _ in range(OPERATION_COUNT):
        state = (state * 1664525 + 1013904223) & HASHTABLE_LCG_MASK
        op_kind = (state >> 28) & 0xF
        identifier = (state >> 4) & ID_MASK
        key = generate_key(identifier)
        if op_kind < 8:
            table.put(key, state)
        elif op_kind < 14:
            _, found = table.get(key)
            if found:
                hit_count += 1
        else:
            table.delete_key(key)
    return int_pair_to_string(table.size, hit_count)


def run() -> str:
    return do_hashtable_workload()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_hashtable_workload()
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
