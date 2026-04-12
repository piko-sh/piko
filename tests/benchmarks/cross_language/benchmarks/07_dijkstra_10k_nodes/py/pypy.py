import argparse
import sys
import time

NODE_COUNT = 10000
AVERAGE_DEGREE = 6
MAX_EDGE_WEIGHT = 1000
RESULT_MASK = 0xFFFFFFFF
LCG_MASK = 0xFFFFFFFF
UNREACHABLE = 1 << 30


def generate_graph() -> tuple[list[int], list[int], list[int]]:
    edge_targets: list[int] = []
    edge_weights: list[int] = []
    edge_heads = [0] * (NODE_COUNT + 1)
    state = 11223344
    for node_index in range(NODE_COUNT):
        edge_heads[node_index] = len(edge_targets)
        state = (state * 1664525 + 1013904223) & LCG_MASK
        outgoing_degree = (state % (AVERAGE_DEGREE * 2)) + 1
        for _ in range(outgoing_degree):
            state = (state * 1664525 + 1013904223) & LCG_MASK
            target = state % NODE_COUNT
            state = (state * 1664525 + 1013904223) & LCG_MASK
            weight = (state % MAX_EDGE_WEIGHT) + 1
            edge_targets.append(target)
            edge_weights.append(weight)
    edge_heads[NODE_COUNT] = len(edge_targets)
    return edge_heads, edge_targets, edge_weights


def sift_up(heap: list[int], heap_distances: list[int], position: int) -> None:
    while position > 0:
        parent = (position - 1) // 2
        if heap_distances[parent] <= heap_distances[position]:
            break
        heap[parent], heap[position] = heap[position], heap[parent]
        heap_distances[parent], heap_distances[position] = heap_distances[position], heap_distances[parent]
        position = parent


def sift_down(heap: list[int], heap_distances: list[int], position: int) -> None:
    length = len(heap)
    while True:
        left_child = 2 * position + 1
        right_child = 2 * position + 2
        smallest = position
        if left_child < length and heap_distances[left_child] < heap_distances[smallest]:
            smallest = left_child
        if right_child < length and heap_distances[right_child] < heap_distances[smallest]:
            smallest = right_child
        if smallest == position:
            break
        heap[position], heap[smallest] = heap[smallest], heap[position]
        heap_distances[position], heap_distances[smallest] = heap_distances[smallest], heap_distances[position]
        position = smallest


def heap_push(heap: list[int], heap_distances: list[int], node: int, distance: int) -> None:
    heap.append(node)
    heap_distances.append(distance)
    sift_up(heap, heap_distances, len(heap) - 1)


def heap_pop_front(heap: list[int], heap_distances: list[int]) -> tuple[int, int]:
    head_node = heap[0]
    head_distance = heap_distances[0]
    last_index = len(heap) - 1
    heap[0] = heap[last_index]
    heap_distances[0] = heap_distances[last_index]
    heap.pop()
    heap_distances.pop()
    if heap:
        sift_down(heap, heap_distances, 0)
    return head_node, head_distance


def run_dijkstra(edge_heads: list[int], edge_targets: list[int], edge_weights: list[int]) -> list[int]:
    distances = [UNREACHABLE] * NODE_COUNT
    distances[0] = 0
    heap: list[int] = [0]
    heap_distances: list[int] = [0]

    while heap:
        current_node, current_distance = heap_pop_front(heap, heap_distances)
        if current_distance > distances[current_node]:
            continue
        edge_range_start = edge_heads[current_node]
        edge_range_end = edge_heads[current_node + 1]
        for edge_index in range(edge_range_start, edge_range_end):
            neighbour = edge_targets[edge_index]
            candidate = current_distance + edge_weights[edge_index]
            if candidate < distances[neighbour]:
                distances[neighbour] = candidate
                heap_push(heap, heap_distances, neighbour, candidate)
    return distances


def do_dijkstra() -> str:
    edge_heads, edge_targets, edge_weights = generate_graph()
    distances = run_dijkstra(edge_heads, edge_targets, edge_weights)
    running_sum = 0
    for distance in distances:
        if distance < UNREACHABLE:
            running_sum = (running_sum + distance) & RESULT_MASK
    return int_to_decimal_string(running_sum)


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


def run() -> str:
    return do_dijkstra()


def run_inner(k: int) -> tuple[str, int]:
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_dijkstra()
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
