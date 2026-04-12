import argparse
import sys
import time

FIB_TERMS_TO_COMPUTE = 100000
FIB_MODULUS_BITMASK = (1 << 64) - 1


def compute_fib(n: int) -> int:
    previous = 0
    current = 1
    for _ in range(n):
        next_value = (previous + current) & FIB_MODULUS_BITMASK
        previous = current
        current = next_value
    return current


def uint64_to_decimal_string(value: int) -> str:
    if value == 0:
        return "0"
    digits = []
    while value > 0:
        digits.append(chr(ord("0") + value % 10))
        value //= 10
    digits.reverse()
    return "".join(digits)


def run() -> str:
    return uint64_to_decimal_string(compute_fib(FIB_TERMS_TO_COMPUTE))


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = 0
    for _ in range(k):
        last = compute_fib(FIB_TERMS_TO_COMPUTE)
    elapsed_nanos = time.perf_counter_ns() - start_nanos
    return uint64_to_decimal_string(last), elapsed_nanos


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
