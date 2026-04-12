import argparse
import sys
import time

TRAINING_TOKENS = 50000
OUTPUT_TOKENS = 10000
RESULT_MASK = 0xFFFFFFFF
LCG_MASK = 0xFFFFFFFF
VOCABULARY_MASK = 0x3F

VOCABULARY = [
    "the", "of", "and", "to", "in", "a", "is", "it", "you", "that",
    "he", "was", "for", "on", "are", "with", "as", "I", "his", "they",
    "be", "at", "one", "have", "this", "from", "or", "had", "by", "hot",
    "word", "but", "what", "some", "we", "can", "out", "other", "were", "all",
    "there", "when", "up", "use", "your", "how", "said", "an", "each", "she",
    "which", "do", "their", "time", "if", "will", "way", "about", "many", "then",
    "them", "write", "would", "like",
]


def generate_corpus(token_count: int) -> list[str]:
    state = 31415927
    output = []
    for _ in range(token_count):
        state = (state * 1664525 + 1013904223) & LCG_MASK
        output.append(VOCABULARY[state & VOCABULARY_MASK])
    return output


def build_bigram_table(corpus: list[str]) -> dict[str, list[str]]:
    transitions: dict[str, list[str]] = {}
    for token_index in range(len(corpus) - 1):
        previous = corpus[token_index]
        next_token = corpus[token_index + 1]
        if previous in transitions:
            transitions[previous].append(next_token)
        else:
            transitions[previous] = [next_token]
    return transitions


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


def do_markov() -> str:
    corpus = generate_corpus(TRAINING_TOKENS)
    transitions = build_bigram_table(corpus)
    state = 42100100
    fold_hash = 0
    previous = corpus[0]
    for _ in range(OUTPUT_TOKENS):
        next_candidates = transitions.get(previous)
        if not next_candidates:
            next_token = corpus[0]
        else:
            state = (state * 1664525 + 1013904223) & LCG_MASK
            next_token = next_candidates[state % len(next_candidates)]
        for character in next_token:
            fold_hash = ((fold_hash * 31) + ord(character)) & RESULT_MASK
        previous = next_token
    return int_to_decimal_string(fold_hash)


def run() -> str:
    return do_markov()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_markov()
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
