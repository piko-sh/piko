import argparse
import multiprocessing
import os
import sys
import time

NUMBER_OF_WORKERS = 16
TOP_RESULTS_COUNT = 50


def locate_corpus() -> str:
    candidates = [
        "testdata/corpus.txt",
        "benchmarks/16_parallel_word_count_montecristo/testdata/corpus.txt",
        "tests/benchmarks/cross_language/benchmarks/16_parallel_word_count_montecristo/testdata/corpus.txt",
    ]
    for candidate in candidates:
        if os.path.exists(candidate):
            return candidate
    raise FileNotFoundError("corpus not found at any candidate path")


def load_corpus() -> bytes:
    with open(locate_corpus(), "rb") as corpus_handle:
        return corpus_handle.read()


def is_letter_byte(value: int) -> bool:
    if 0x61 <= value <= 0x7A:
        return True
    if 0x41 <= value <= 0x5A:
        return True
    return False


def split_corpus_into_chunks(corpus: bytes, chunk_count: int) -> list[bytes]:
    if len(corpus) == 0 or chunk_count <= 1:
        return [corpus]
    chunk_size = len(corpus) // chunk_count
    chunks: list[bytes] = []
    cursor = 0
    for _ in range(chunk_count - 1):
        boundary = cursor + chunk_size
        if boundary > len(corpus):
            boundary = len(corpus)
        while boundary < len(corpus) and is_letter_byte(corpus[boundary]):
            boundary += 1
        chunks.append(corpus[cursor:boundary])
        cursor = boundary
    chunks.append(corpus[cursor:])
    return chunks


def tokenise_and_count(chunk: bytes) -> dict[str, int]:
    counts: dict[str, int] = {}
    token_start = -1
    length = len(chunk)
    for index in range(length):
        current = chunk[index]
        if is_letter_byte(current):
            if token_start < 0:
                token_start = index
        elif token_start >= 0:
            emit_word(counts, chunk, token_start, index)
            token_start = -1
    if token_start >= 0:
        emit_word(counts, chunk, token_start, length)
    return counts


def emit_word(counts: dict[str, int], chunk: bytes, start: int, end: int) -> None:
    lowered = bytearray(end - start)
    for index in range(end - start):
        current = chunk[start + index]
        if 0x41 <= current <= 0x5A:
            current += 32
        lowered[index] = current
    key = lowered.decode("ascii")
    if key in counts:
        counts[key] = counts[key] + 1
    else:
        counts[key] = 1


def merge_maps(maps: list[dict[str, int]]) -> dict[str, int]:
    merged: dict[str, int] = {}
    for local in maps:
        for key, value in local.items():
            if key in merged:
                merged[key] = merged[key] + value
            else:
                merged[key] = value
    return merged


def beats(left: tuple[str, int], right: tuple[str, int]) -> bool:
    left_word, left_count = left
    right_word, right_count = right
    if left_count != right_count:
        return left_count > right_count
    return left_word < right_word


def insert_into_top_k(top: list[tuple[str, int]], entry: tuple[str, int], top_count: int) -> None:
    current_length = len(top)
    if current_length < top_count:
        position = current_length
        while position > 0 and beats(entry, top[position - 1]):
            position -= 1
        top.append(("", 0))
        for shift in range(current_length, position, -1):
            top[shift] = top[shift - 1]
        top[position] = entry
        return
    if not beats(entry, top[top_count - 1]):
        return
    position = top_count - 1
    while position > 0 and beats(entry, top[position - 1]):
        position -= 1
    for shift in range(top_count - 1, position, -1):
        top[shift] = top[shift - 1]
    top[position] = entry


def render_top_k(top: list[tuple[str, int]]) -> str:
    lines = []
    for word, count in top:
        lines.append(word + "\t" + int_to_decimal(count))
    return "\n".join(lines)


def int_to_decimal(value: int) -> str:
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


def format_top_k(counts: dict[str, int], top_count: int) -> str:
    top: list[tuple[str, int]] = []
    for word, count in counts.items():
        insert_into_top_k(top, (word, count), top_count)
    return render_top_k(top)


def do_parallel_word_count(corpus: bytes, pool: "multiprocessing.pool.Pool") -> str:
    chunks = split_corpus_into_chunks(corpus, NUMBER_OF_WORKERS)
    results = pool.map(tokenise_and_count, chunks)
    merged = merge_maps(results)
    return format_top_k(merged, TOP_RESULTS_COUNT)


def run() -> str:
    corpus = load_corpus()
    with multiprocessing.Pool(NUMBER_OF_WORKERS) as pool:
        return do_parallel_word_count(corpus, pool)


def run_inner(k: int) -> tuple[str, int]:
    corpus = load_corpus()
    with multiprocessing.Pool(NUMBER_OF_WORKERS) as pool:
        start_nanos = time.perf_counter_ns()
        last = ""
        for _ in range(k):
            last = do_parallel_word_count(corpus, pool)
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
    multiprocessing.set_start_method("fork", force=True)
    _main()
