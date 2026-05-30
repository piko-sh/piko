import argparse
import sys
import time

IMAGE_WIDTH = 200
IMAGE_HEIGHT = 200
MAX_ITERATIONS = 80
VIEW_REAL_MIN = -2.0
VIEW_REAL_MAX = 1.0
VIEW_IMAG_MIN = -1.5
VIEW_IMAG_MAX = 1.5
ESCAPE_THRESHOLD_SQUARED = 4.0


def int64_to_decimal(value: int) -> str:
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


def do_mandelbrot_render() -> str:
    real_step = (VIEW_REAL_MAX - VIEW_REAL_MIN) / IMAGE_WIDTH
    imag_step = (VIEW_IMAG_MAX - VIEW_IMAG_MIN) / IMAGE_HEIGHT
    total_iterations = 0
    for pixel_y in range(IMAGE_HEIGHT):
        c_im = VIEW_IMAG_MIN + pixel_y * imag_step
        for pixel_x in range(IMAGE_WIDTH):
            c_re = VIEW_REAL_MIN + pixel_x * real_step
            z_re = 0.0
            z_im = 0.0
            iterations = 0
            while iterations < MAX_ITERATIONS:
                z_re_squared = z_re * z_re
                z_im_squared = z_im * z_im
                if z_re_squared + z_im_squared > ESCAPE_THRESHOLD_SQUARED:
                    break
                z_im = 2.0 * z_re * z_im + c_im
                z_re = z_re_squared - z_im_squared + c_re
                iterations += 1
            total_iterations += iterations
    return int64_to_decimal(total_iterations)


def run() -> str:
    return do_mandelbrot_render()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_mandelbrot_render()
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
