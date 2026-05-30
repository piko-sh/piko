import argparse
import sys
import time

DICTIONARY_WORD_COUNT = 50000
QUERY_COUNT = 50000
MIN_LENGTH = 4
LENGTH_RANGE_MASK = 0xF
ALPHABET = "abcdefghijklmnop"
ALPHABET_MASK = 0xF
LCG_MASK = 0xFFFFFFFF
RESULT_MASK = 0xFFFFFFFF


class TrieNode:
    __slots__ = ("children", "terminal")

    def __init__(self) -> None:
        self.children: list[TrieNode | None] = [None] * 16
        self.terminal = False


def generate_word(state: int) -> tuple[str, int]:
    state = (state * 1664525 + 1013904223) & LCG_MASK
    length = MIN_LENGTH + (state & LENGTH_RANGE_MASK)
    output_chars = []
    for _ in range(length):
        state = (state * 1664525 + 1013904223) & LCG_MASK
        output_chars.append(ALPHABET[state & ALPHABET_MASK])
    return "".join(output_chars), state


def trie_insert(root: TrieNode, word: str) -> None:
    node = root
    for character in word:
        slot = ord(character) - ord(ALPHABET[0])
        child = node.children[slot]
        if child is None:
            child = TrieNode()
            node.children[slot] = child
        node = child
    node.terminal = True


def trie_longest_prefix_length(root: TrieNode, query: str) -> int:
    node = root
    longest_terminal = 0
    for char_index in range(len(query)):
        slot = ord(query[char_index]) - ord(ALPHABET[0])
        next_node = node.children[slot]
        if next_node is None:
            break
        node = next_node
        if node.terminal:
            longest_terminal = char_index + 1
    return longest_terminal


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


def do_trie() -> str:
    root = TrieNode()
    state = 98765432
    for _ in range(DICTIONARY_WORD_COUNT):
        word, state = generate_word(state)
        trie_insert(root, word)
    query_state = 13579
    running_sum = 0
    for _ in range(QUERY_COUNT):
        query, query_state = generate_word(query_state)
        running_sum = (running_sum + trie_longest_prefix_length(root, query)) & RESULT_MASK
    return int_to_decimal_string(running_sum)


def run() -> str:
    return do_trie()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_trie()
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
