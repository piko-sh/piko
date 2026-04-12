import argparse
import sys
import time

GENERIC_LCG_MASK = 0xFFFFFFFF
SLICES_PER_TYPE = 500
SLICE_SIZE = 100
GENERIC_SEED = 0xCAFEBABE


def generic_sum(xs):
    accumulator = 0
    length = len(xs)
    for index in range(length):
        accumulator = accumulator + xs[index]
    return accumulator


def generic_max(xs):
    if len(xs) == 0:
        return 0
    maximum = xs[0]
    length = len(xs)
    for index in range(1, length):
        if xs[index] > maximum:
            maximum = xs[index]
    return maximum


def generic_min(xs):
    if len(xs) == 0:
        return 0
    minimum = xs[0]
    length = len(xs)
    for index in range(1, length):
        if xs[index] < minimum:
            minimum = xs[index]
    return minimum


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


def do_generic_pipeline():
    state = GENERIC_SEED
    accumulator = 0

    for _ in range(SLICES_PER_TYPE):
        slice_data = [0] * SLICE_SIZE
        for element_index in range(SLICE_SIZE):
            state = (state * 1664525 + 1013904223) & GENERIC_LCG_MASK
            slice_data[element_index] = state
        sum_value = generic_sum(slice_data) & GENERIC_LCG_MASK
        max_value = generic_max(slice_data) & GENERIC_LCG_MASK
        min_value = generic_min(slice_data) & GENERIC_LCG_MASK
        accumulator = (accumulator ^ sum_value ^ max_value ^ min_value) & GENERIC_LCG_MASK

    for _ in range(SLICES_PER_TYPE):
        slice_data = [0] * SLICE_SIZE
        for element_index in range(SLICE_SIZE):
            state = (state * 1664525 + 1013904223) & GENERIC_LCG_MASK
            slice_data[element_index] = state
        sum_value = generic_sum(slice_data) & GENERIC_LCG_MASK
        max_value = generic_max(slice_data) & GENERIC_LCG_MASK
        min_value = generic_min(slice_data) & GENERIC_LCG_MASK
        accumulator = (accumulator ^ sum_value ^ max_value ^ min_value) & GENERIC_LCG_MASK

    return int_to_decimal(accumulator)


def run():
    return do_generic_pipeline()


def run_inner(k):
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_generic_pipeline()
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
