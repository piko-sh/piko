import argparse
import sys
import time

TYPE_SWITCH_MASK = 0xFFFFFFFF
VALUE_STREAM_LENGTH = 100000
TYPE_SWITCH_SEED = 0xC0FFEE
FNV_OFFSET_BASIS_32 = 0x811C9DC5
FNV_PRIME_32 = 0x01000193


class IntVal:
    __slots__ = ("v",)

    def __init__(self, v):
        self.v = v


class StringVal:
    __slots__ = ("v",)

    def __init__(self, v):
        self.v = v


class BytesVal:
    __slots__ = ("v",)

    def __init__(self, v):
        self.v = v


class BoolVal:
    __slots__ = ("v",)

    def __init__(self, v):
        self.v = v


class FloatVal:
    __slots__ = ("v",)

    def __init__(self, v):
        self.v = v


def fnv1a32(key):
    hash_value = FNV_OFFSET_BASIS_32
    for index in range(len(key)):
        hash_value ^= ord(key[index])
        hash_value = (hash_value * FNV_PRIME_32) & TYPE_SWITCH_MASK
    return hash_value


def dispatch_by_type(v):
    if isinstance(v, IntVal):
        return (v.v + 1234) & TYPE_SWITCH_MASK
    if isinstance(v, StringVal):
        return fnv1a32(v.v)
    if isinstance(v, BytesVal):
        sum_value = 0
        data = v.v
        for index in range(len(data)):
            sum_value = (sum_value + data[index]) & TYPE_SWITCH_MASK
        return sum_value
    if isinstance(v, BoolVal):
        return 1 if v.v else 0
    if isinstance(v, FloatVal):
        return int(v.v) & TYPE_SWITCH_MASK
    return 0


def build_value_stream():
    state = TYPE_SWITCH_SEED
    values = [None] * VALUE_STREAM_LENGTH
    shared_string = "abcdefghij"
    shared_bytes = bytes([0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08])
    for index in range(VALUE_STREAM_LENGTH):
        state = (state * 1664525 + 1013904223) & TYPE_SWITCH_MASK
        kind = state & 0x7
        if kind == 0 or kind == 1 or kind == 2:
            values[index] = IntVal(state)
        elif kind == 3:
            values[index] = StringVal(shared_string)
        elif kind == 4:
            values[index] = BytesVal(shared_bytes)
        elif kind == 5:
            values[index] = BoolVal((state & 1) == 1)
        elif kind == 6:
            values[index] = FloatVal(float(state))
        else:
            values[index] = IntVal(state)
    return values


def walk_and_dispatch(values):
    accumulator = 0
    length = len(values)
    for index in range(length):
        accumulator = (accumulator ^ dispatch_by_type(values[index])) & TYPE_SWITCH_MASK
    return int_to_decimal(accumulator)


def int_to_decimal(value):
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


def do_type_switches():
    values = build_value_stream()
    return walk_and_dispatch(values)


def run():
    return do_type_switches()


def run_inner(k):
    values = build_value_stream()
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = walk_and_dispatch(values)
    elapsed_nanos = time.perf_counter_ns() - start_nanos
    return last, elapsed_nanos


def _main():
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
