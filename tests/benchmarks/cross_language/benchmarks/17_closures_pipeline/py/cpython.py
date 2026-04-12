import argparse
import sys
import time

INPUT_COUNT = 100000
PIPELINE_LCG_MASK = 0xFFFFFFFF
PIPELINE_SEED = 0xCAFEBABE
FOLD_ROTATE_SHIFT = 5


def make_multiply(factor):

    def multiply(value):
        return (value * factor) & PIPELINE_LCG_MASK
    return multiply


def make_divisor_filter(divisor):

    def predicate(value):
        return value % divisor != 0
    return predicate


def make_add(offset):

    def add(value):
        return (value + offset) & PIPELINE_LCG_MASK
    return add


def make_threshold_filter(threshold):

    def predicate(value):
        return value > threshold
    return predicate


def make_fold(shift):

    def fold(acc, value):
        rotated = ((value << shift) | (value >> (32 - shift))) & PIPELINE_LCG_MASK
        return (acc ^ rotated) & PIPELINE_LCG_MASK
    return fold


def apply_map(input_list, fn):
    length = len(input_list)
    output = [0] * length
    for index in range(length):
        output[index] = fn(input_list[index])
    return output


def apply_filter(input_list, predicate):
    output = []
    length = len(input_list)
    for index in range(length):
        current = input_list[index]
        if predicate(current):
            output.append(current)
    return output


def apply_reduce(input_list, initial, fn):
    accumulator = initial
    length = len(input_list)
    for index in range(length):
        accumulator = fn(accumulator, input_list[index])
    return accumulator


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


def do_closures_pipeline():
    state = PIPELINE_SEED
    input_list = [0] * INPUT_COUNT
    for index in range(INPUT_COUNT):
        state = (state * 1664525 + 1013904223) & PIPELINE_LCG_MASK
        input_list[index] = state

    multiplier = make_multiply(3)
    divisor_filter = make_divisor_filter(7)
    adder = make_add(1234)
    threshold_filter = make_threshold_filter(1000)
    folder = make_fold(FOLD_ROTATE_SHIFT)

    stage1 = apply_map(input_list, multiplier)
    stage2 = apply_filter(stage1, divisor_filter)
    stage3 = apply_map(stage2, adder)
    stage4 = apply_filter(stage3, threshold_filter)
    result = apply_reduce(stage4, 0, folder)
    return int_to_decimal(result)


def run():
    return do_closures_pipeline()


def run_inner(k):
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_closures_pipeline()
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
