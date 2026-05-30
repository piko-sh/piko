import argparse
import sys
import time

EXPRESSION_COUNT = 10000
RESULT_MASK = 0xFFFFFFFF
LCG_MASK = 0xFFFFFFFF

TOKEN_INTEGER = 0
TOKEN_PLUS = 1
TOKEN_MINUS = 2
TOKEN_MULTIPLY = 3
TOKEN_PAREN_LEFT = 4
TOKEN_PAREN_RIGHT = 5


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


def generate_term(output: list[str], state: int, depth_budget: int) -> int:
    state = (state * 1664525 + 1013904223) & LCG_MASK
    if depth_budget <= 0 or state & 0x3 == 0:
        value = state & 0xFF
        output.append(int_to_decimal_string(value))
        return state
    output.append("(")
    state = generate_term(output, state, depth_budget - 1)
    state = (state * 1664525 + 1013904223) & LCG_MASK
    operator_index = state & 0x3
    if operator_index == 1:
        output.append("-")
    elif operator_index == 2:
        output.append("*")
    else:
        output.append("+")
    state = generate_term(output, state, depth_budget - 1)
    output.append(")")
    return state


def generate_expression(state: int) -> tuple[str, int]:
    output: list[str] = []
    state = (state * 1664525 + 1013904223) & LCG_MASK
    depth_budget = (state & 0x7) + 4
    state = generate_term(output, state, depth_budget)
    return "".join(output), state


def tokenise(source: str) -> list[tuple[int, int]]:
    tokens: list[tuple[int, int]] = []
    position = 0
    source_length = len(source)
    while position < source_length:
        current = source[position]
        if "0" <= current <= "9":
            value = 0
            while position < source_length and "0" <= source[position] <= "9":
                value = value * 10 + (ord(source[position]) - ord("0"))
                position += 1
            tokens.append((TOKEN_INTEGER, value))
        elif current == "+":
            tokens.append((TOKEN_PLUS, 0))
            position += 1
        elif current == "-":
            tokens.append((TOKEN_MINUS, 0))
            position += 1
        elif current == "*":
            tokens.append((TOKEN_MULTIPLY, 0))
            position += 1
        elif current == "(":
            tokens.append((TOKEN_PAREN_LEFT, 0))
            position += 1
        elif current == ")":
            tokens.append((TOKEN_PAREN_RIGHT, 0))
            position += 1
        else:
            position += 1
    return tokens


def precedence_for(kind: int) -> int:
    if kind == TOKEN_MULTIPLY:
        return 20
    if kind == TOKEN_PLUS or kind == TOKEN_MINUS:
        return 10
    return 0


class Evaluator:
    __slots__ = ("tokens", "position")

    def __init__(self, tokens: list[tuple[int, int]]) -> None:
        self.tokens = tokens
        self.position = 0

    def parse_expression(self, minimum_precedence: int) -> int:
        left_value = self.parse_atom()
        while self.position < len(self.tokens):
            operator_kind, _ = self.tokens[self.position]
            operator_precedence = precedence_for(operator_kind)
            if operator_precedence == 0 or operator_precedence < minimum_precedence:
                break
            self.position += 1
            right_value = self.parse_expression(operator_precedence + 1)
            if operator_kind == TOKEN_PLUS:
                left_value = left_value + right_value
            elif operator_kind == TOKEN_MINUS:
                left_value = left_value - right_value
            elif operator_kind == TOKEN_MULTIPLY:
                left_value = left_value * right_value
        return left_value

    def parse_atom(self) -> int:
        if self.position >= len(self.tokens):
            return 0
        kind, value = self.tokens[self.position]
        if kind == TOKEN_INTEGER:
            self.position += 1
            return value
        if kind == TOKEN_PAREN_LEFT:
            self.position += 1
            value = self.parse_expression(0)
            if self.position < len(self.tokens) and self.tokens[self.position][0] == TOKEN_PAREN_RIGHT:
                self.position += 1
            return value
        return 0


def do_expression_eval() -> str:
    state = 99887766
    running_sum = 0
    for _ in range(EXPRESSION_COUNT):
        source, state = generate_expression(state)
        evaluator = Evaluator(tokenise(source))
        value = evaluator.parse_expression(0)
        running_sum = (running_sum + (value & RESULT_MASK)) & RESULT_MASK
    return int_to_decimal_string(running_sum)


def run() -> str:
    return do_expression_eval()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_expression_eval()
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
