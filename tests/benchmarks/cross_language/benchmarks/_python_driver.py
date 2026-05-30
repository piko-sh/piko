import argparse
import sys
import time
import types

WORKLOAD_MODULE_NAME = "__cross_lang_workload__"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("workload", help="path to the workload .py file")
    parser.add_argument("--mode", default="endtoend", choices=["endtoend", "inner"])
    parser.add_argument("--k", type=int, default=0)
    arguments, _unknown = parser.parse_known_args()

    with open(arguments.workload, "r", encoding="utf-8") as handle:
        source = handle.read()

    compile_start = time.perf_counter_ns()
    code = compile(source, arguments.workload, "exec")
    compile_nanos = time.perf_counter_ns() - compile_start

    workload_module = types.ModuleType(WORKLOAD_MODULE_NAME)
    workload_module.__file__ = arguments.workload
    sys.modules[WORKLOAD_MODULE_NAME] = workload_module
    exec(code, workload_module.__dict__)

    if not hasattr(workload_module, "run"):
        print(
            f"driver: workload {arguments.workload!r} does not define `run`",
            file=sys.stderr,
        )
        sys.exit(2)
    run_function = workload_module.run

    if arguments.mode == "inner":
        last = ""
        inner_start = time.perf_counter_ns()
        for _ in range(arguments.k):
            last = run_function()
        inner_nanos = time.perf_counter_ns() - inner_start
        print(last)
        print(f"COMPILE_NANOS={compile_nanos}", file=sys.stderr)
        print(f"INNER_ELAPSED_NS={inner_nanos}", file=sys.stderr)
    else:
        result = run_function()
        print(result)
        print(f"COMPILE_NANOS={compile_nanos}", file=sys.stderr)

if __name__ == "__main__":
    main()
