import argparse
import sys
import time

GRID_SIZE = 200
GENERATION_COUNT = 1000
LCG_MASK = 0xFFFFFFFF


def advance_generation(current: list[int], next_grid: list[int]) -> None:
    for row_index in range(GRID_SIZE):
        row_up = row_index - 1
        if row_up < 0:
            row_up = GRID_SIZE - 1
        row_down = row_index + 1
        if row_down >= GRID_SIZE:
            row_down = 0
        row_offset = row_index * GRID_SIZE
        row_up_offset = row_up * GRID_SIZE
        row_down_offset = row_down * GRID_SIZE
        for column_index in range(GRID_SIZE):
            column_left = column_index - 1
            if column_left < 0:
                column_left = GRID_SIZE - 1
            column_right = column_index + 1
            if column_right >= GRID_SIZE:
                column_right = 0
            neighbours = (
                current[row_up_offset + column_left]
                + current[row_up_offset + column_index]
                + current[row_up_offset + column_right]
                + current[row_offset + column_left]
                + current[row_offset + column_right]
                + current[row_down_offset + column_left]
                + current[row_down_offset + column_index]
                + current[row_down_offset + column_right]
            )
            alive = current[row_offset + column_index] == 1
            if alive:
                if neighbours == 2 or neighbours == 3:
                    next_grid[row_offset + column_index] = 1
                else:
                    next_grid[row_offset + column_index] = 0
            else:
                if neighbours == 3:
                    next_grid[row_offset + column_index] = 1
                else:
                    next_grid[row_offset + column_index] = 0


def do_life() -> str:
    cell_count = GRID_SIZE * GRID_SIZE
    current = [0] * cell_count
    next_grid = [0] * cell_count
    state = 20260511
    for cell_index in range(cell_count):
        state = (state * 1664525 + 1013904223) & LCG_MASK
        if (state >> 30) & 0x3 == 0:
            current[cell_index] = 1
    for _ in range(GENERATION_COUNT):
        advance_generation(current, next_grid)
        current, next_grid = next_grid, current
    live_count = 0
    for cell_value in current:
        live_count += cell_value
    return int_to_decimal_string(live_count)


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
    return do_life()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_life()
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
