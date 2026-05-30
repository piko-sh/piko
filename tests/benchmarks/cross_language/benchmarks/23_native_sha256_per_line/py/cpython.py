import argparse
import hashlib
import sys
import time

DIGEST_BYTES = 32


def load_corpus():
    candidates = [
        "testdata/corpus.txt",
        "benchmarks/23_native_sha256_per_line/testdata/corpus.txt",
        "tests/benchmarks/cross_language/benchmarks/23_native_sha256_per_line/testdata/corpus.txt",
    ]
    for candidate in candidates:
        try:
            with open(candidate, "rb") as handle:
                return handle.read()
        except FileNotFoundError:
            continue
    raise RuntimeError("corpus not found at any candidate path")


def do_line_hashing(corpus):
    accumulator = bytearray(DIGEST_BYTES)
    line_start = 0
    length = len(corpus)
    newline = 0x0A
    for index in range(length):
        if corpus[index] != newline:
            continue
        digest = hashlib.sha256(corpus[line_start:index]).digest()
        for digest_index in range(DIGEST_BYTES):
            accumulator[digest_index] ^= digest[digest_index]
        line_start = index + 1
    if line_start < length:
        digest = hashlib.sha256(corpus[line_start:]).digest()
        for digest_index in range(DIGEST_BYTES):
            accumulator[digest_index] ^= digest[digest_index]
    return bytes_to_hex(accumulator)


def bytes_to_hex(input_bytes):
    hex_chars = "0123456789abcdef"
    out = []
    for index in range(len(input_bytes)):
        value = input_bytes[index]
        out.append(hex_chars[(value >> 4) & 0x0F])
        out.append(hex_chars[value & 0x0F])
    return "".join(out)


def run():
    corpus = load_corpus()
    return do_line_hashing(corpus)


def run_inner(k):
    corpus = load_corpus()
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_line_hashing(corpus)
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
