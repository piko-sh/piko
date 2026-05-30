import argparse
import sys
import time

sys.setrecursionlimit(1_000_000)

TARGET_BYTES = 100 * 1024
LCG_MASK = 0xFFFFFFFF


def write_json_value(output: bytearray, state: int, depth: int) -> int:
    state = (state * 1664525 + 1013904223) & LCG_MASK
    if len(output) >= TARGET_BYTES:
        output.append(ord("0"))
        return state
    pick = state & 0x7
    if depth > 6:
        pick = pick & 0x3
    if pick in (0, 1):
        return write_json_object(output, state, depth)
    if pick == 2:
        return write_json_array(output, state, depth)
    if pick in (3, 4):
        return write_json_string(output, state)
    if pick in (5, 6):
        return write_json_number(output, state)
    state = (state * 1664525 + 1013904223) & LCG_MASK
    if state & 1 == 0:
        output.extend(b"true")
    else:
        output.extend(b"null")
    return state


def write_json_object(output: bytearray, state: int, depth: int) -> int:
    state = (state * 1664525 + 1013904223) & LCG_MASK
    entry_count = (state & 0x7) + 1
    output.append(ord("{"))
    for entry_index in range(entry_count):
        if entry_index > 0:
            output.append(ord(","))
        state = write_json_string(output, state)
        output.append(ord(":"))
        state = write_json_value(output, state, depth + 1)
        if len(output) >= TARGET_BYTES:
            break
    output.append(ord("}"))
    return state


def write_json_array(output: bytearray, state: int, depth: int) -> int:
    state = (state * 1664525 + 1013904223) & LCG_MASK
    entry_count = (state & 0x7) + 1
    output.append(ord("["))
    for entry_index in range(entry_count):
        if entry_index > 0:
            output.append(ord(","))
        state = write_json_value(output, state, depth + 1)
        if len(output) >= TARGET_BYTES:
            break
    output.append(ord("]"))
    return state


def write_json_string(output: bytearray, state: int) -> int:
    state = (state * 1664525 + 1013904223) & LCG_MASK
    length = (state & 0x7) + 1
    output.append(ord('"'))
    for _ in range(length):
        state = (state * 1664525 + 1013904223) & LCG_MASK
        output.append(ord("a") + (state % 26))
    output.append(ord('"'))
    return state


def write_json_number(output: bytearray, state: int) -> int:
    state = (state * 1664525 + 1013904223) & LCG_MASK
    value = state & 0xFFFF
    for character in int_to_decimal_string(value):
        output.append(ord(character))
    return state


def generate_document() -> bytes:
    output = bytearray()
    output.append(ord("{"))
    state = 31415926
    entry_index = 0
    while len(output) < TARGET_BYTES:
        if entry_index > 0:
            output.append(ord(","))
        state = write_json_string(output, state)
        output.append(ord(":"))
        state = write_json_value(output, state, 1)
        entry_index += 1
    output.append(ord("}"))
    return bytes(output)


class JsonParser:
    __slots__ = ("source", "position", "keys_observed")

    def __init__(self, source: bytes) -> None:
        self.source = source
        self.position = 0
        self.keys_observed = 0

    def parse_value(self) -> None:
        self.skip_whitespace()
        if self.position >= len(self.source):
            return
        current = self.source[self.position]
        if current == ord("{"):
            self.parse_object()
        elif current == ord("["):
            self.parse_array()
        elif current == ord('"'):
            self.parse_string()
        elif current == ord("t"):
            self.position += 4
        elif current == ord("f"):
            self.position += 5
        elif current == ord("n"):
            self.position += 4
        else:
            self.parse_number()

    def parse_object(self) -> None:
        self.position += 1
        self.skip_whitespace()
        while self.position < len(self.source) and self.source[self.position] != ord("}"):
            self.skip_whitespace()
            self.parse_string()
            self.keys_observed += 1
            self.skip_whitespace()
            self.position += 1
            self.parse_value()
            self.skip_whitespace()
            if self.position < len(self.source) and self.source[self.position] == ord(","):
                self.position += 1
        if self.position < len(self.source):
            self.position += 1

    def parse_array(self) -> None:
        self.position += 1
        self.skip_whitespace()
        while self.position < len(self.source) and self.source[self.position] != ord("]"):
            self.parse_value()
            self.skip_whitespace()
            if self.position < len(self.source) and self.source[self.position] == ord(","):
                self.position += 1
        if self.position < len(self.source):
            self.position += 1

    def parse_string(self) -> None:
        self.position += 1
        while self.position < len(self.source) and self.source[self.position] != ord('"'):
            self.position += 1
        if self.position < len(self.source):
            self.position += 1

    def parse_number(self) -> None:
        while self.position < len(self.source):
            current = self.source[self.position]
            if current < ord("0") or current > ord("9"):
                break
            self.position += 1

    def skip_whitespace(self) -> None:
        whitespace = (ord(" "), ord("\t"), ord("\n"), ord("\r"))
        while self.position < len(self.source) and self.source[self.position] in whitespace:
            self.position += 1


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


def do_parse() -> str:
    document = generate_document()
    parser = JsonParser(document)
    parser.parse_value()
    return int_to_decimal_string(parser.keys_observed)


def run() -> str:
    return do_parse()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_parse()
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
