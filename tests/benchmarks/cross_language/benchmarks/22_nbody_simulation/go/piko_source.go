package main

import (
	"math"
	"time"
)

const numSteps = 50000

const timeStep = 0.01

const piConstant = 3.141592653589793

const daysPerYear = 365.24

const solarMass = 4 * piConstant * piConstant

const energyScale = 1.0e9

type Body struct {
	x, y, z float64

	vx, vy, vz float64

	mass float64
}

func Run() string {
	return doSimulate()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doSimulate()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doSimulate() string {
	bodies := initialBodies()
	offsetMomentum(bodies)
	energyBefore := computeEnergy(bodies)
	for step := 0; step < numSteps; step++ {
		advance(bodies, timeStep)
	}
	energyAfter := computeEnergy(bodies)
	beforeScaled := int64(roundToNearest(energyBefore * energyScale))
	afterScaled := int64(roundToNearest(energyAfter * energyScale))
	return int64ToDecimal(beforeScaled) + "," + int64ToDecimal(afterScaled)
}

func initialBodies() []Body {
	bodies := make([]Body, 5)
	bodies[0] = Body{
		x: 0.0, y: 0.0, z: 0.0,
		vx: 0.0, vy: 0.0, vz: 0.0,
		mass: solarMass,
	}
	bodies[1] = Body{
		x:    4.84143144246472090e+00,
		y:    -1.16032004402742839e+00,
		z:    -1.03622044471123109e-01,
		vx:   1.66007664274403694e-03 * daysPerYear,
		vy:   7.69901118419740425e-03 * daysPerYear,
		vz:   -6.90460016972063023e-05 * daysPerYear,
		mass: 9.54791938424326609e-04 * solarMass,
	}
	bodies[2] = Body{
		x:    8.34336671824457987e+00,
		y:    4.12479856412430479e+00,
		z:    -4.03523417114321381e-01,
		vx:   -2.76742510726862411e-03 * daysPerYear,
		vy:   4.99852801234917238e-03 * daysPerYear,
		vz:   2.30417297573763929e-05 * daysPerYear,
		mass: 2.85885980666130812e-04 * solarMass,
	}
	bodies[3] = Body{
		x:    1.28943695621391310e+01,
		y:    -1.51111514016986312e+01,
		z:    -2.23307578892655734e-01,
		vx:   2.96460137564761618e-03 * daysPerYear,
		vy:   2.37847173959480950e-03 * daysPerYear,
		vz:   -2.96589568540237556e-05 * daysPerYear,
		mass: 4.36624404335156298e-05 * solarMass,
	}
	bodies[4] = Body{
		x:    1.53796971148509165e+01,
		y:    -2.59193146099879641e+01,
		z:    1.79258772950371181e-01,
		vx:   2.68067772490389322e-03 * daysPerYear,
		vy:   1.62824170038242295e-03 * daysPerYear,
		vz:   -9.51592254519715870e-05 * daysPerYear,
		mass: 5.15138902046611451e-05 * solarMass,
	}
	return bodies
}

func offsetMomentum(bodies []Body) {
	px := 0.0
	py := 0.0
	pz := 0.0
	for i := 0; i < len(bodies); i++ {
		px += bodies[i].vx * bodies[i].mass
		py += bodies[i].vy * bodies[i].mass
		pz += bodies[i].vz * bodies[i].mass
	}
	bodies[0].vx = -px / solarMass
	bodies[0].vy = -py / solarMass
	bodies[0].vz = -pz / solarMass
}

func advance(bodies []Body, dt float64) {
	n := len(bodies)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			dx := bodies[i].x - bodies[j].x
			dy := bodies[i].y - bodies[j].y
			dz := bodies[i].z - bodies[j].z
			distanceSquared := dx*dx + dy*dy + dz*dz
			distance := math.Sqrt(distanceSquared)
			magnitude := dt / (distanceSquared * distance)
			iMassMag := bodies[i].mass * magnitude
			jMassMag := bodies[j].mass * magnitude
			bodies[i].vx -= dx * jMassMag
			bodies[i].vy -= dy * jMassMag
			bodies[i].vz -= dz * jMassMag
			bodies[j].vx += dx * iMassMag
			bodies[j].vy += dy * iMassMag
			bodies[j].vz += dz * iMassMag
		}
	}
	for i := 0; i < n; i++ {
		bodies[i].x += dt * bodies[i].vx
		bodies[i].y += dt * bodies[i].vy
		bodies[i].z += dt * bodies[i].vz
	}
}

func computeEnergy(bodies []Body) float64 {
	energy := 0.0
	n := len(bodies)
	for i := 0; i < n; i++ {
		vx := bodies[i].vx
		vy := bodies[i].vy
		vz := bodies[i].vz
		energy += 0.5 * bodies[i].mass * (vx*vx + vy*vy + vz*vz)
		for j := i + 1; j < n; j++ {
			dx := bodies[i].x - bodies[j].x
			dy := bodies[i].y - bodies[j].y
			dz := bodies[i].z - bodies[j].z
			distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
			energy -= bodies[i].mass * bodies[j].mass / distance
		}
	}
	return energy
}

func roundToNearest(value float64) float64 {
	if value >= 0 {
		return math.Floor(value + 0.5)
	}
	return -math.Floor(-value + 0.5)
}

func int64ToDecimal(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	digits := [21]byte{}
	position := len(digits)
	for value > 0 {
		position--
		digits[position] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		position--
		digits[position] = '-'
	}
	return string(digits[position:])
}
