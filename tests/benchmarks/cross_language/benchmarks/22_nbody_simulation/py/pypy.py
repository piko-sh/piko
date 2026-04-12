import argparse
import math
import sys
import time

NUM_STEPS = 50000
TIME_STEP = 0.01
PI = 3.141592653589793
DAYS_PER_YEAR = 365.24
SOLAR_MASS = 4 * PI * PI
ENERGY_SCALE = 1.0e9


class Body:
    __slots__ = ("x", "y", "z", "vx", "vy", "vz", "mass")

    def __init__(self, x, y, z, vx, vy, vz, mass):
        self.x = x
        self.y = y
        self.z = z
        self.vx = vx
        self.vy = vy
        self.vz = vz
        self.mass = mass


def initial_bodies():
    return [
        Body(0.0, 0.0, 0.0, 0.0, 0.0, 0.0, SOLAR_MASS),
        Body(
            4.84143144246472090e+00,
            -1.16032004402742839e+00,
            -1.03622044471123109e-01,
            1.66007664274403694e-03 * DAYS_PER_YEAR,
            7.69901118419740425e-03 * DAYS_PER_YEAR,
            -6.90460016972063023e-05 * DAYS_PER_YEAR,
            9.54791938424326609e-04 * SOLAR_MASS,
        ),
        Body(
            8.34336671824457987e+00,
            4.12479856412430479e+00,
            -4.03523417114321381e-01,
            -2.76742510726862411e-03 * DAYS_PER_YEAR,
            4.99852801234917238e-03 * DAYS_PER_YEAR,
            2.30417297573763929e-05 * DAYS_PER_YEAR,
            2.85885980666130812e-04 * SOLAR_MASS,
        ),
        Body(
            1.28943695621391310e+01,
            -1.51111514016986312e+01,
            -2.23307578892655734e-01,
            2.96460137564761618e-03 * DAYS_PER_YEAR,
            2.37847173959480950e-03 * DAYS_PER_YEAR,
            -2.96589568540237556e-05 * DAYS_PER_YEAR,
            4.36624404335156298e-05 * SOLAR_MASS,
        ),
        Body(
            1.53796971148509165e+01,
            -2.59193146099879641e+01,
            1.79258772950371181e-01,
            2.68067772490389322e-03 * DAYS_PER_YEAR,
            1.62824170038242295e-03 * DAYS_PER_YEAR,
            -9.51592254519715870e-05 * DAYS_PER_YEAR,
            5.15138902046611451e-05 * SOLAR_MASS,
        ),
    ]


def offset_momentum(bodies):
    px = 0.0
    py = 0.0
    pz = 0.0
    for body in bodies:
        px += body.vx * body.mass
        py += body.vy * body.mass
        pz += body.vz * body.mass
    bodies[0].vx = -px / SOLAR_MASS
    bodies[0].vy = -py / SOLAR_MASS
    bodies[0].vz = -pz / SOLAR_MASS


def advance(bodies, dt):
    n = len(bodies)
    for i in range(n):
        body_i = bodies[i]
        for j in range(i + 1, n):
            body_j = bodies[j]
            dx = body_i.x - body_j.x
            dy = body_i.y - body_j.y
            dz = body_i.z - body_j.z
            distance_squared = dx * dx + dy * dy + dz * dz
            distance = math.sqrt(distance_squared)
            magnitude = dt / (distance_squared * distance)
            i_mass_mag = body_i.mass * magnitude
            j_mass_mag = body_j.mass * magnitude
            body_i.vx -= dx * j_mass_mag
            body_i.vy -= dy * j_mass_mag
            body_i.vz -= dz * j_mass_mag
            body_j.vx += dx * i_mass_mag
            body_j.vy += dy * i_mass_mag
            body_j.vz += dz * i_mass_mag
    for body in bodies:
        body.x += dt * body.vx
        body.y += dt * body.vy
        body.z += dt * body.vz


def compute_energy(bodies):
    energy = 0.0
    n = len(bodies)
    for i in range(n):
        body_i = bodies[i]
        vx = body_i.vx
        vy = body_i.vy
        vz = body_i.vz
        energy += 0.5 * body_i.mass * (vx * vx + vy * vy + vz * vz)
        for j in range(i + 1, n):
            body_j = bodies[j]
            dx = body_i.x - body_j.x
            dy = body_i.y - body_j.y
            dz = body_i.z - body_j.z
            distance = math.sqrt(dx * dx + dy * dy + dz * dz)
            energy -= body_i.mass * body_j.mass / distance
    return energy


def round_to_nearest(value):
    if value >= 0:
        return math.floor(value + 0.5)
    return -math.floor(-value + 0.5)


def int_to_decimal(value):
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


def do_simulate():
    bodies = initial_bodies()
    offset_momentum(bodies)
    energy_before = compute_energy(bodies)
    for _ in range(NUM_STEPS):
        advance(bodies, TIME_STEP)
    energy_after = compute_energy(bodies)
    before_scaled = int(round_to_nearest(energy_before * ENERGY_SCALE))
    after_scaled = int(round_to_nearest(energy_after * ENERGY_SCALE))
    return int_to_decimal(before_scaled) + "," + int_to_decimal(after_scaled)


def run():
    return do_simulate()


def run_inner(k):
    start_nanos = time.perf_counter_ns()
    last = ""
    for _ in range(k):
        last = do_simulate()
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
