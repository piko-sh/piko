import argparse
import sys
import time

DENSE_INPUTS = 256
DENSE_OUTPUTS = 256


def build_weights() -> list[list[float]]:
    return [
        [float(((i + j * 7) % 31) - 15) * 0.01 for j in range(DENSE_INPUTS)]
        for i in range(DENSE_OUTPUTS)
    ]


def build_input() -> list[float]:
    return [float(((j * 13) % 17) - 8) * 0.1 for j in range(DENSE_INPUTS)]


def build_bias() -> list[float]:
    return [float(((i * 5) % 11) - 5) * 0.1 for i in range(DENSE_OUTPUTS)]


def dense_forward(
    weights: list[list[float]],
    input_vector: list[float],
    bias: list[float],
    output: list[float],
) -> None:
    for i in range(len(output)):
        row = weights[i]
        accumulator = 0.0
        for j in range(len(row)):
            accumulator += row[j] * input_vector[j]
        accumulator += bias[i]
        if accumulator < 0:
            accumulator = 0
        output[i] = accumulator


def summarise_output(output: list[float]) -> str:
    total = 0.0
    for i in range(len(output)):
        total += output[i]
    scaled = int(total * 1000.0)
    return int64_to_decimal_string(scaled)


def int64_to_decimal_string(value: int) -> str:
    if value == 0:
        return "0"
    negative = value < 0
    if negative:
        value = -value
    digits = []
    while value > 0:
        digits.append(chr(ord("0") + value % 10))
        value //= 10
    if negative:
        digits.append("-")
    digits.reverse()
    return "".join(digits)


def run() -> str:
    weights = build_weights()
    input_vector = build_input()
    bias = build_bias()
    output = [0.0] * DENSE_OUTPUTS
    dense_forward(weights, input_vector, bias, output)
    return summarise_output(output)


def run_inner(k: int) -> tuple[str, int]:
    weights = build_weights()
    input_vector = build_input()
    bias = build_bias()
    output = [0.0] * DENSE_OUTPUTS

    start_nanos = time.perf_counter_ns()
    for _ in range(k):
        dense_forward(weights, input_vector, bias, output)
    elapsed_nanos = time.perf_counter_ns() - start_nanos
    return summarise_output(output), elapsed_nanos


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
