import argparse
import sys
import time

PUZZLE_COUNT = 100
CELLS_TO_REMOVE = 40
RESULT_MASK = 0xFFFFFFFF
LCG_MASK = 0xFFFFFFFF

SEED_GRID = [
    [5, 3, 4, 6, 7, 8, 9, 1, 2],
    [6, 7, 2, 1, 9, 5, 3, 4, 8],
    [1, 9, 8, 3, 4, 2, 5, 6, 7],
    [8, 5, 9, 7, 6, 1, 4, 2, 3],
    [4, 2, 6, 8, 5, 3, 7, 9, 1],
    [7, 1, 3, 9, 2, 4, 8, 5, 6],
    [9, 6, 1, 5, 3, 7, 2, 8, 4],
    [2, 8, 7, 4, 1, 9, 6, 3, 5],
    [3, 4, 5, 2, 8, 6, 1, 7, 9],
]


def generate_puzzle(board: list[int], state: int) -> int:
    for row_index in range(9):
        for column_index in range(9):
            board[row_index * 9 + column_index] = SEED_GRID[row_index][column_index]
    for _ in range(CELLS_TO_REMOVE):
        state = (state * 1664525 + 1013904223) & LCG_MASK
        position = state % 81
        board[position] = 0
    return state


def is_valid_placement(board: list[int], row_index: int, column_index: int, candidate: int) -> bool:
    for index in range(9):
        if board[row_index * 9 + index] == candidate:
            return False
        if board[index * 9 + column_index] == candidate:
            return False
    box_row = (row_index // 3) * 3
    box_column = (column_index // 3) * 3
    for offset_row in range(3):
        for offset_column in range(3):
            if board[(box_row + offset_row) * 9 + (box_column + offset_column)] == candidate:
                return False
    return True


def solve_sudoku(board: list[int], position: int) -> bool:
    while position < 81 and board[position] != 0:
        position += 1
    if position == 81:
        return True
    row_index = position // 9
    column_index = position % 9
    for candidate in range(1, 10):
        if is_valid_placement(board, row_index, column_index, candidate):
            board[position] = candidate
            if solve_sudoku(board, position + 1):
                return True
    board[position] = 0
    return False


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


def do_sudoku() -> str:
    state = 76543210
    fold_hash = 0
    board = [0] * 81
    for _ in range(PUZZLE_COUNT):
        state = generate_puzzle(board, state)
        solve_sudoku(board, 0)
        for cell_index in range(81):
            fold_hash = ((fold_hash * 31) + board[cell_index]) & RESULT_MASK
    return int_to_decimal_string(fold_hash)


def run() -> str:
    return do_sudoku()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_sudoku()
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
