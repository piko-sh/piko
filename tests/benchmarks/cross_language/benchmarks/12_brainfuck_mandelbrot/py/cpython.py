import argparse
import sys
import time

BRAINFUCK_PROGRAM = (
    "++++++++++++>++++++++>"
    "<<["
    ">[->+>+<<]"
    ">>[-<<+>>]"
    "<<<-]"
    "++++++++[>++++++++<-]>."
    "<++++++++++[->+++<]>++++."
)

MEMORY_SIZE = 30000
RESULT_MASK = 0xFFFFFFFF


def build_jump_table(source: str) -> list[int]:
    table = [0] * len(source)
    stack: list[int] = []
    for index in range(len(source)):
        character = source[index]
        if character == "[":
            stack.append(index)
            continue
        if character == "]":
            open_index = stack.pop()
            table[open_index] = index
            table[index] = open_index
    return table


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


def do_brainfuck() -> str:
    memory = bytearray(MEMORY_SIZE)
    pointer = 0
    jump_table = build_jump_table(BRAINFUCK_PROGRAM)
    instruction_index = 0
    fold_hash = 0
    source_length = len(BRAINFUCK_PROGRAM)
    while instruction_index < source_length:
        instruction = BRAINFUCK_PROGRAM[instruction_index]
        if instruction == ">":
            pointer += 1
        elif instruction == "<":
            pointer -= 1
        elif instruction == "+":
            memory[pointer] = (memory[pointer] + 1) & 0xFF
        elif instruction == "-":
            memory[pointer] = (memory[pointer] - 1) & 0xFF
        elif instruction == ".":
            fold_hash = ((fold_hash * 31) + memory[pointer]) & RESULT_MASK
        elif instruction == ",":
            memory[pointer] = 0
        elif instruction == "[":
            if memory[pointer] == 0:
                instruction_index = jump_table[instruction_index]
        elif instruction == "]":
            if memory[pointer] != 0:
                instruction_index = jump_table[instruction_index]
        instruction_index += 1
    return int_to_decimal_string(fold_hash)


def run() -> str:
    return do_brainfuck()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_brainfuck()
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
