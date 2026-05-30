import argparse
import sys
import time

TREE_DEPTH = 15
FOLD_MASK = 0xFFFFFFFF
FOLD_MULTIPLIER = 31


class Node:
    __slots__ = ("value", "left", "right")

    def __init__(self, value):
        self.value = value
        self.left = None
        self.right = None


def make_tree(depth, value):
    node = Node(value)
    if depth > 0:
        node.left = make_tree(depth - 1, (value * 2) & FOLD_MASK)
        node.right = make_tree(depth - 1, (value * 2 + 1) & FOLD_MASK)
    return node


def count_nodes(t):
    if t is None:
        return 0
    return 1 + count_nodes(t.left) + count_nodes(t.right)


def invert(t):
    if t is None:
        return
    t.left, t.right = t.right, t.left
    invert(t.left)
    invert(t.right)


def inorder_fold(t, state):
    if t is None:
        return state
    state = inorder_fold(t.left, state)
    state = (state * FOLD_MULTIPLIER + t.value) & FOLD_MASK
    state = inorder_fold(t.right, state)
    return state


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


def do_invert_binary_tree():
    tree = make_tree(TREE_DEPTH, 1)
    pre_count = count_nodes(tree)
    pre_fold = inorder_fold(tree, 0)
    invert(tree)
    post_count = count_nodes(tree)
    post_fold = inorder_fold(tree, 0)
    return (
        int_to_decimal(pre_count) + ","
        + int_to_decimal(pre_fold) + ","
        + int_to_decimal(post_count) + ","
        + int_to_decimal(post_fold)
    )


def run():
    return do_invert_binary_tree()


def run_inner(k):
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_invert_binary_tree()
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
