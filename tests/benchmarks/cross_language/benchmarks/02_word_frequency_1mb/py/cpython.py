import argparse
import sys
import time

CORPUS_BYTES = 1024 * 1024
TOP_K = 10

VOCABULARY = [
    "the", "of", "and", "to", "in", "a", "is", "it", "you", "that",
    "he", "was", "for", "on", "are", "with", "as", "I", "his", "they",
    "be", "at", "one", "have", "this", "from", "or", "had", "by", "hot",
    "word", "but", "what", "some", "we", "can", "out", "other", "were", "all",
    "there", "when", "up", "use", "your", "how", "said", "an", "each", "she",
    "which", "do", "their", "time", "if", "will", "way", "about", "many", "then",
    "them", "write", "would", "like",
]
VOCABULARY_MASK = 0x3F
LCG_MASK = 0xFFFFFFFF


def generate_corpus() -> bytes:
    output = bytearray()
    state = 987654321
    while len(output) < CORPUS_BYTES:
        state = (state * 1664525 + 1013904223) & LCG_MASK
        word = VOCABULARY[state & VOCABULARY_MASK]
        if output:
            output.append(ord(" "))
        for character in word:
            output.append(ord(character))
    return bytes(output)


def count_words_manual(corpus: bytes) -> dict[str, int]:
    counts: dict[str, int] = {}
    token_start = 0
    corpus_length = len(corpus)
    for position in range(corpus_length + 1):
        if position == corpus_length or corpus[position] == ord(" "):
            if position > token_start:
                word = corpus[token_start:position].decode("ascii")
                counts[word] = counts.get(word, 0) + 1
            token_start = position + 1
    return counts


def less_for_top(left: tuple[str, int], right: tuple[str, int]) -> bool:
    if left[1] != right[1]:
        return left[1] > right[1]
    return left[0] < right[0]


def top_k_by_insertion_sort(counts: dict[str, int], k: int) -> list[tuple[str, int]]:
    heap: list[tuple[str, int]] = []
    for word, count in counts.items():
        entry = (word, count)
        insert_position = len(heap)
        while insert_position > 0 and less_for_top(entry, heap[insert_position - 1]):
            insert_position -= 1
        heap.insert(insert_position, entry)
        if len(heap) > k:
            heap.pop()
    return heap


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


def format_top_k_output(top: list[tuple[str, int]]) -> str:
    pieces = []
    for line_index, (word, count) in enumerate(top):
        if line_index > 0:
            pieces.append("\n")
        pieces.append(word)
        pieces.append("\t")
        pieces.append(int_to_decimal_string(count))
    return "".join(pieces)


def do_word_frequency() -> str:
    corpus = generate_corpus()
    counts = count_words_manual(corpus)
    top = top_k_by_insertion_sort(counts, TOP_K)
    return format_top_k_output(top)


def run() -> str:
    return do_word_frequency()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_word_frequency()
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
