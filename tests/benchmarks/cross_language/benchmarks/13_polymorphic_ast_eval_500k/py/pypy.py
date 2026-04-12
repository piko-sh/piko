import argparse
import sys
import time

LCG_MASK = 0xFFFFFFFF
RESULT_MASK = 0xFFFFFFFF
TREE_DEPTH_BUDGET = 12
WALKS_PER_EVAL = 100
ENV_SLOT_COUNT = 8
TREE_SEED = 12345678
ENV_SEED = 0x517CC1B7


class AddNode:
    __slots__ = ("left", "right")

    def __init__(self, left, right):
        self.left = left
        self.right = right

    def eval(self, env):
        return (self.left.eval(env) + self.right.eval(env)) & RESULT_MASK


class SubNode:
    __slots__ = ("left", "right")

    def __init__(self, left, right):
        self.left = left
        self.right = right

    def eval(self, env):
        return (self.left.eval(env) - self.right.eval(env)) & RESULT_MASK


class MulNode:
    __slots__ = ("left", "right")

    def __init__(self, left, right):
        self.left = left
        self.right = right

    def eval(self, env):
        return (self.left.eval(env) * self.right.eval(env)) & RESULT_MASK


class ModNode:
    __slots__ = ("left", "right")

    def __init__(self, left, right):
        self.left = left
        self.right = right

    def eval(self, env):
        right_value = self.right.eval(env)
        if right_value == 0:
            return 0
        return self.left.eval(env) % right_value


class MinNode:
    __slots__ = ("left", "right")

    def __init__(self, left, right):
        self.left = left
        self.right = right

    def eval(self, env):
        left_value = self.left.eval(env)
        right_value = self.right.eval(env)
        if left_value < right_value:
            return left_value
        return right_value


class MaxNode:
    __slots__ = ("left", "right")

    def __init__(self, left, right):
        self.left = left
        self.right = right

    def eval(self, env):
        left_value = self.left.eval(env)
        right_value = self.right.eval(env)
        if left_value > right_value:
            return left_value
        return right_value


class IfPosNode:
    __slots__ = ("condition", "then_branch", "else_branch")

    def __init__(self, condition, then_branch, else_branch):
        self.condition = condition
        self.then_branch = then_branch
        self.else_branch = else_branch

    def eval(self, env):
        if self.condition.eval(env) != 0:
            return self.then_branch.eval(env)
        return self.else_branch.eval(env)


class ConstNode:
    __slots__ = ("value",)

    def __init__(self, value):
        self.value = value

    def eval(self, env):
        return self.value


class VarNode:
    __slots__ = ("slot",)

    def __init__(self, slot):
        self.slot = slot

    def eval(self, env):
        return env[self.slot]


def build_tree(state: int, depth_budget: int):
    state = (state * 1664525 + 1013904223) & LCG_MASK
    if depth_budget <= 0 or (state & 0x7) == 0:
        state = (state * 1664525 + 1013904223) & LCG_MASK
        if (state & 0x1) == 0:
            return ConstNode((state >> 1) & 0xFF), state
        return VarNode((state >> 1) & 0x7), state
    kind = state & 0x7
    if kind == 7:
        condition, state = build_tree(state, depth_budget - 1)
        then_branch, state = build_tree(state, depth_budget - 1)
        else_branch, state = build_tree(state, depth_budget - 1)
        return IfPosNode(condition, then_branch, else_branch), state
    left, state = build_tree(state, depth_budget - 1)
    right, state = build_tree(state, depth_budget - 1)
    if kind == 0:
        return AddNode(left, right), state
    if kind == 1:
        return SubNode(left, right), state
    if kind == 2:
        return MulNode(left, right), state
    if kind == 3:
        return ModNode(left, right), state
    if kind == 4:
        return MinNode(left, right), state
    if kind == 5:
        return MaxNode(left, right), state
    return AddNode(left, right), state


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


def do_ast_eval() -> str:
    tree, _ = build_tree(TREE_SEED, TREE_DEPTH_BUDGET)
    env_state = ENV_SEED
    env = [0] * ENV_SLOT_COUNT
    accumulator = 0
    for _ in range(WALKS_PER_EVAL):
        for slot in range(ENV_SLOT_COUNT):
            env_state = (env_state * 1664525 + 1013904223) & LCG_MASK
            env[slot] = env_state
        accumulator = (accumulator + tree.eval(env)) & RESULT_MASK
    return int_to_decimal_string(accumulator)


def run() -> str:
    return do_ast_eval()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_ast_eval()
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
