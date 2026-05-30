import argparse
import sys
import time

PAIR_COUNT = 1000
ALPHABET = "abcdefghijklmnopqrstuvwxyz012345"
ALPHABET_MASK = 0x1F
MIN_LENGTH = 8
LENGTH_RANGE_MASK = 0x1F
LCG_MASK = 0xFFFFFFFF
MAX_LENGTH = MIN_LENGTH + LENGTH_RANGE_MASK


def generate_string(state: int) -> tuple[str, int]:
    state = (state * 1664525 + 1013904223) & LCG_MASK
    length = MIN_LENGTH + (state & LENGTH_RANGE_MASK)
    output_chars = []
    for _ in range(length):
        state = (state * 1664525 + 1013904223) & LCG_MASK
        output_chars.append(ALPHABET[state & ALPHABET_MASK])
    return "".join(output_chars), state


def edit_distance(left: str, right: str, dp_row: list[int], dp_next: list[int]) -> int:
    left_length = len(left)
    right_length = len(right)
    for column_index in range(right_length + 1):
        dp_row[column_index] = column_index
    for row_index in range(1, left_length + 1):
        dp_next[0] = row_index
        left_character = left[row_index - 1]
        for column_index in range(1, right_length + 1):
            substitution_cost = 0 if left_character == right[column_index - 1] else 1
            deletion = dp_row[column_index] + 1
            insertion = dp_next[column_index - 1] + 1
            substitution = dp_row[column_index - 1] + substitution_cost
            candidate = deletion
            if insertion < candidate:
                candidate = insertion
            if substitution < candidate:
                candidate = substitution
            dp_next[column_index] = candidate
        for swap_index in range(right_length + 1):
            dp_row[swap_index] = dp_next[swap_index]
    return dp_row[right_length]


def do_levenshtein() -> str:
    state = 2718281828
    total_distance = 0
    dp_row = [0] * (MAX_LENGTH + 2)
    dp_next = [0] * (MAX_LENGTH + 2)
    for _ in range(PAIR_COUNT):
        left_string, state = generate_string(state)
        right_string, state = generate_string(state)
        total_distance += edit_distance(left_string, right_string, dp_row, dp_next)
    return int_to_decimal_string(total_distance)


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


def run() -> str:
    return do_levenshtein()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_levenshtein()
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
