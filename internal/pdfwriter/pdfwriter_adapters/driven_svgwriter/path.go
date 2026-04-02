// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package driven_svgwriter

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

const (
	// quadraticToCubicFactor holds the 2/3 scaling factor for converting
	// quadratic Bezier control points to cubic Bezier control points.
	quadraticToCubicFactor = 2.0 / 3.0

	// degreesToRadians holds the conversion factor from degrees to radians.
	degreesToRadians = math.Pi / 180.0

	// cubicArgCount holds the number of arguments for an SVG cubic Bezier command.
	cubicArgCount = 6

	// quadArgCount holds the number of arguments for an SVG quadratic Bezier command.
	quadArgCount = 4

	// arcArgCount holds the number of arguments for an SVG arc command.
	arcArgCount = 7

	// arcEndXIndex holds the index of the end-point X coordinate within arc arguments.
	arcEndXIndex = 5

	// arcEndYIndex holds the index of the end-point Y coordinate within arc arguments.
	arcEndYIndex = 6

	// asciiCaseDiff holds the difference between lowercase and uppercase ASCII letters.
	asciiCaseDiff = 32

	// cubicCP1XIndex holds the index of the second control point X within cubic arguments.
	cubicCP1XIndex = 2

	// cubicCP1YIndex holds the index of the second control point Y within cubic arguments.
	cubicCP1YIndex = 3

	// cubicEndXIndex holds the index of the end-point X within cubic arguments.
	cubicEndXIndex = 4

	// cubicEndYIndex holds the index of the end-point Y within cubic arguments.
	cubicEndYIndex = 5

	// arcAlphaDivisor holds the divisor used in the arc-to-cubic alpha calculation.
	arcAlphaDivisor = 3
)

// PathCommand represents a single SVG path command with absolute coordinates.
type PathCommand struct {
	// Args holds the numeric arguments for the command.
	Args []float64

	// Type holds the single-letter command type (M, L, C, Z, etc.).
	Type byte
}

// pathState tracks the current position and control point state during path parsing.
type pathState struct {
	// commands holds the accumulated absolute-coordinate path commands.
	commands []PathCommand

	// cx holds the current X position.
	cx float64

	// cy holds the current Y position.
	cy float64

	// sx holds the X position of the current sub-path start.
	sx float64

	// sy holds the Y position of the current sub-path start.
	sy float64

	// lastCPX holds the X position of the last control point for smooth curves.
	lastCPX float64

	// lastCPY holds the Y position of the last control point for smooth curves.
	lastCPY float64

	// lastCmd holds the last command type for smooth curve reflection.
	lastCmd byte
}

// ParsePathData parses an SVG path d attribute into absolute-coordinate commands.
//
// Relative commands are converted to absolute. H/V->L, S->C, T->Q, and Q->C
// since PDF only supports cubic Bezier.
//
// Takes d (string) which specifies the SVG path d attribute value.
//
// Returns []PathCommand which holds the parsed absolute-coordinate commands.
//
// Returns error when the path data contains invalid syntax or unexpected tokens.
func ParsePathData(d string) ([]PathCommand, error) {
	tokens, err := tokenizePath(d)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, nil
	}

	state := &pathState{}
	i := 0

	for i < len(tokens) {
		tok := tokens[i]
		if !tok.isCommand {
			return nil, fmt.Errorf("svg: expected command at position %d, got %q", i, tok.value)
		}
		cmd := tok.value[0]
		i++

		var parseErr error
		i, parseErr = processPathCommand(state, tokens, i, cmd)
		if parseErr != nil {
			return nil, parseErr
		}
	}

	return state.commands, nil
}

// processPathCommand processes a single path command and its arguments from the
// token stream, updating the path state.
//
// Takes state (*pathState) which specifies the current path parsing state.
// Takes tokens ([]pathToken) which specifies the full token stream. Takes i
// (int) which specifies the current position in the token stream. Takes cmd
// (byte) which specifies the command letter to process.
//
// Returns int which holds the updated token stream position.
// Returns error when the arguments are missing or malformed.
func processPathCommand(state *pathState, tokens []pathToken, i int, cmd byte) (int, error) {
	argCount := commandArgCount(cmd)
	isRelative := cmd >= 'a' && cmd <= 'z'
	absCmd := toUpper(cmd)

	if absCmd == 'Z' {
		state.commands = append(state.commands, PathCommand{Type: 'Z'})
		state.cx, state.cy = state.sx, state.sy
		state.lastCPX, state.lastCPY = state.cx, state.cy
		state.lastCmd = 'Z'
		return i, nil
	}

	if argCount == 0 {
		return i, nil
	}

	first := true
	for i < len(tokens) && !tokens[i].isCommand {
		args, newI, err := consumeArgs(tokens, i, argCount)
		if err != nil {
			return i, fmt.Errorf("svg: error parsing args for %c: %w", cmd, err)
		}
		i = newI

		if isRelative {
			makeAbsolute(absCmd, args, state.cx, state.cy)
		}

		applyPathCommand(state, absCmd, args, first)
		state.lastCmd = absCmd
		first = false
	}

	return i, nil
}

// applyPathCommand dispatches a single absolute command to the appropriate
// handler, updating the path state.
//
// Takes state (*pathState) which specifies the current path parsing state.
// Takes absCmd (byte) which specifies the uppercase command letter. Takes args
// ([]float64) which specifies the command arguments. Takes first (bool) which
// specifies whether this is the first argument group for the command.
func applyPathCommand(state *pathState, absCmd byte, args []float64, first bool) {
	switch absCmd {
	case 'M':
		applyMoveCommand(state, args, first)
	case 'L':
		applyLineCommand(state, args)
	case 'H':
		state.commands = append(state.commands, PathCommand{Type: 'L', Args: []float64{args[0], state.cy}})
		state.cx = args[0]
		state.lastCPX, state.lastCPY = state.cx, state.cy
	case 'V':
		state.commands = append(state.commands, PathCommand{Type: 'L', Args: []float64{state.cx, args[0]}})
		state.cy = args[0]
		state.lastCPX, state.lastCPY = state.cx, state.cy
	case 'C':
		applyCubicCommand(state, args)
	case 'S':
		applySmoothCubicCommand(state, args)
	case 'Q':
		applyQuadCommand(state, args)
	case 'T':
		applySmoothQuadCommand(state, args)
	case 'A':
		applyArcCommand(state, args)
	}
}

// applyMoveCommand applies a move-to command, treating subsequent coordinate
// pairs as implicit line-to commands.
//
// Takes state (*pathState) which specifies the current path parsing state.
// Takes args ([]float64) which specifies the X and Y coordinates. Takes first
// (bool) which specifies whether this is the initial move or an implicit line.
func applyMoveCommand(state *pathState, args []float64, first bool) {
	if first {
		state.commands = append(state.commands, PathCommand{Type: 'M', Args: []float64{args[0], args[1]}})
		state.sx, state.sy = args[0], args[1]
	} else {
		state.commands = append(state.commands, PathCommand{Type: 'L', Args: []float64{args[0], args[1]}})
	}
	state.cx, state.cy = args[0], args[1]
	state.lastCPX, state.lastCPY = state.cx, state.cy
}

// applyLineCommand applies a line-to command and updates the current position.
//
// Takes state (*pathState) which specifies the current path parsing state.
// Takes args ([]float64) which specifies the X and Y coordinates.
func applyLineCommand(state *pathState, args []float64) {
	state.commands = append(state.commands, PathCommand{Type: 'L', Args: []float64{args[0], args[1]}})
	state.cx, state.cy = args[0], args[1]
	state.lastCPX, state.lastCPY = state.cx, state.cy
}

// applyCubicCommand applies a cubic Bezier command and updates the current
// position and last control point.
//
// Takes state (*pathState) which specifies the current path parsing state.
// Takes args ([]float64) which specifies the six cubic Bezier arguments.
func applyCubicCommand(state *pathState, args []float64) {
	state.commands = append(state.commands, PathCommand{Type: 'C', Args: []float64{
		args[0], args[1], args[cubicCP1XIndex], args[cubicCP1YIndex], args[cubicEndXIndex], args[cubicEndYIndex],
	}})
	state.lastCPX, state.lastCPY = args[cubicCP1XIndex], args[cubicCP1YIndex]
	state.cx, state.cy = args[cubicEndXIndex], args[cubicEndYIndex]
}

// applySmoothCubicCommand applies a smooth cubic Bezier command by reflecting
// the previous control point, then emitting a full cubic command.
//
// Takes state (*pathState) which specifies the current path parsing state.
// Takes args ([]float64) which specifies the four smooth cubic arguments.
func applySmoothCubicCommand(state *pathState, args []float64) {
	reflectedX, reflectedY := state.cx, state.cy
	if state.lastCmd == 'C' || state.lastCmd == 'S' {
		reflectedX = 2*state.cx - state.lastCPX
		reflectedY = 2*state.cy - state.lastCPY
	}
	state.commands = append(state.commands, PathCommand{Type: 'C', Args: []float64{
		reflectedX, reflectedY, args[0], args[1], args[cubicCP1XIndex], args[cubicCP1YIndex],
	}})
	state.lastCPX, state.lastCPY = args[0], args[1]
	state.cx, state.cy = args[cubicCP1XIndex], args[cubicCP1YIndex]
}

// applyQuadCommand converts a quadratic Bezier to a cubic Bezier and appends
// the resulting command.
//
// Takes state (*pathState) which specifies the current path parsing state.
// Takes args ([]float64) which specifies the four quadratic Bezier arguments.
func applyQuadCommand(state *pathState, args []float64) {
	quadX, quadY := args[0], args[1]
	endX, endY := args[cubicCP1XIndex], args[cubicCP1YIndex]
	cp1x := state.cx + quadraticToCubicFactor*(quadX-state.cx)
	cp1y := state.cy + quadraticToCubicFactor*(quadY-state.cy)
	cp2x := endX + quadraticToCubicFactor*(quadX-endX)
	cp2y := endY + quadraticToCubicFactor*(quadY-endY)
	state.commands = append(state.commands, PathCommand{Type: 'C', Args: []float64{
		cp1x, cp1y, cp2x, cp2y, endX, endY,
	}})
	state.lastCPX, state.lastCPY = quadX, quadY
	state.cx, state.cy = endX, endY
}

// applySmoothQuadCommand applies a smooth quadratic Bezier by reflecting the
// previous control point, converting to cubic, and appending the result.
//
// Takes state (*pathState) which specifies the current path parsing state.
// Takes args ([]float64) which specifies the two smooth quadratic arguments.
func applySmoothQuadCommand(state *pathState, args []float64) {
	reflectedX, reflectedY := state.cx, state.cy
	if state.lastCmd == 'Q' || state.lastCmd == 'T' {
		reflectedX = 2*state.cx - state.lastCPX
		reflectedY = 2*state.cy - state.lastCPY
	}
	endX, endY := args[0], args[1]
	cp1x := state.cx + quadraticToCubicFactor*(reflectedX-state.cx)
	cp1y := state.cy + quadraticToCubicFactor*(reflectedY-state.cy)
	cp2x := endX + quadraticToCubicFactor*(reflectedX-endX)
	cp2y := endY + quadraticToCubicFactor*(reflectedY-endY)
	state.commands = append(state.commands, PathCommand{Type: 'C', Args: []float64{
		cp1x, cp1y, cp2x, cp2y, endX, endY,
	}})
	state.lastCPX, state.lastCPY = reflectedX, reflectedY
	state.cx, state.cy = endX, endY
}

// applyArcCommand converts an SVG arc to cubic Bezier segments and appends
// them to the path state.
//
// Takes state (*pathState) which specifies the current path parsing state.
// Takes args ([]float64) which specifies the seven arc arguments.
func applyArcCommand(state *pathState, args []float64) {
	endX, endY := args[arcEndXIndex], args[arcEndYIndex]
	cubics := ArcToCubics(state.cx, state.cy, args[0], args[1], args[2],
		args[3] != 0, args[4] != 0, endX, endY)
	state.commands = append(state.commands, cubics...)
	state.cx, state.cy = endX, endY
	state.lastCPX, state.lastCPY = state.cx, state.cy
}

// arcParams bundles all parameters for an SVG arc-to-cubic conversion.
type arcParams struct {
	// startX holds the starting X coordinate of the arc.
	startX float64

	// startY holds the starting Y coordinate of the arc.
	startY float64

	// rx holds the X radius of the ellipse.
	rx float64

	// ry holds the Y radius of the ellipse.
	ry float64

	// xAxisRotation holds the rotation of the ellipse X axis in degrees.
	xAxisRotation float64

	// largeArc holds whether the large arc flag is set.
	largeArc bool

	// sweep holds whether the sweep flag is set.
	sweep bool

	// endX holds the ending X coordinate of the arc.
	endX float64

	// endY holds the ending Y coordinate of the arc.
	endY float64
}

// ArcToCubics converts an SVG arc to cubic Bezier segments.
//
// Takes startX (float64) which specifies the starting X coordinate. Takes
// startY (float64) which specifies the starting Y coordinate. Takes rx
// (float64) which specifies the X radius. Takes ry (float64) which specifies
// the Y radius. Takes xAxisRotation (float64) which specifies the ellipse
// rotation in degrees. Takes largeArc (bool) which specifies the large-arc
// flag. Takes sweep (bool) which specifies the sweep flag. Takes endX
// (float64) which specifies the ending X coordinate. Takes endY (float64)
// which specifies the ending Y coordinate.
//
// Returns []PathCommand which holds the cubic Bezier segments approximating the
// arc.
func ArcToCubics(startX, startY, rx, ry, xAxisRotation float64, largeArc, sweep bool, endX, endY float64) []PathCommand { //nolint:revive // arc conversion requires all params
	params := arcParams{
		startX: startX, startY: startY,
		rx: rx, ry: ry,
		xAxisRotation: xAxisRotation,
		largeArc:      largeArc, sweep: sweep,
		endX: endX, endY: endY,
	}
	return arcToCubicsFromParams(params)
}

// arcToCubicsFromParams converts an SVG arc to cubic Bezier segments using
// the bundled arc parameters.
//
// Takes p (arcParams) which specifies the arc geometry.
//
// Returns []PathCommand which holds the cubic Bezier segments, or nil if
// the start and end points coincide.
func arcToCubicsFromParams(p arcParams) []PathCommand {
	if p.startX == p.endX && p.startY == p.endY {
		return nil
	}
	if p.rx == 0 || p.ry == 0 {
		return []PathCommand{{Type: 'C', Args: []float64{p.startX, p.startY, p.endX, p.endY, p.endX, p.endY}}}
	}

	p.rx = math.Abs(p.rx)
	p.ry = math.Abs(p.ry)

	phi := p.xAxisRotation * degreesToRadians
	cosPhi := math.Cos(phi)
	sinPhi := math.Sin(phi)

	centreX, centreY, theta1, dtheta := computeArcCentre(p, cosPhi, sinPhi)
	return emitArcSegments(centreX, centreY, p.rx, p.ry, phi, theta1, dtheta)
}

// computeArcCentre computes the ellipse centre and angle span for an SVG arc
// using the endpoint parameterisation.
//
// Takes p (arcParams) which specifies the arc geometry. Takes cosPhi (float64)
// which specifies the cosine of the X axis rotation. Takes sinPhi (float64)
// which specifies the sine of the X axis rotation.
//
// Returns centreX (float64) which holds the ellipse centre X.
// Returns centreY (float64) which holds the ellipse centre Y.
// Returns theta1 (float64) which holds the start angle in
// radians.
// Returns dtheta (float64) which holds the angular span in
// radians.
func computeArcCentre(p arcParams, cosPhi, sinPhi float64) (centreX, centreY, theta1, dtheta float64) {
	dx := (p.startX - p.endX) / 2
	dy := (p.startY - p.endY) / 2
	x1p := cosPhi*dx + sinPhi*dy
	y1p := -sinPhi*dx + cosPhi*dy

	rxSq := p.rx * p.rx
	rySq := p.ry * p.ry
	x1pSq := x1p * x1p
	y1pSq := y1p * y1p

	lambda := x1pSq/rxSq + y1pSq/rySq
	if lambda > 1 {
		scale := math.Sqrt(lambda)
		p.rx *= scale
		p.ry *= scale
		rxSq = p.rx * p.rx
		rySq = p.ry * p.ry
	}

	num := rxSq*rySq - rxSq*y1pSq - rySq*x1pSq
	den := rxSq*y1pSq + rySq*x1pSq
	sq := 0.0
	if den > 0 {
		sq = math.Sqrt(math.Max(0, num/den))
	}
	if p.largeArc == p.sweep {
		sq = -sq
	}
	cxp := sq * p.rx * y1p / p.ry
	cyp := -sq * p.ry * x1p / p.rx

	centreX = cosPhi*cxp - sinPhi*cyp + (p.startX+p.endX)/2
	centreY = sinPhi*cxp + cosPhi*cyp + (p.startY+p.endY)/2

	theta1 = vecAngle(1, 0, (x1p-cxp)/p.rx, (y1p-cyp)/p.ry)
	dtheta = vecAngle((x1p-cxp)/p.rx, (y1p-cyp)/p.ry, (-x1p-cxp)/p.rx, (-y1p-cyp)/p.ry)

	if !p.sweep && dtheta > 0 {
		dtheta -= 2 * math.Pi
	} else if p.sweep && dtheta < 0 {
		dtheta += 2 * math.Pi
	}

	return centreX, centreY, theta1, dtheta
}

// emitArcSegments splits an arc into segments of at most 90 degrees and
// converts each to a cubic Bezier command.
//
// Takes centreX (float64) which specifies the ellipse centre X. Takes centreY
// (float64) which specifies the ellipse centre Y. Takes rx (float64) which
// specifies the X radius. Takes ry (float64) which specifies the Y radius.
// Takes phi (float64) which specifies the X axis rotation in radians. Takes
// theta1 (float64) which specifies the start angle. Takes dtheta (float64)
// which specifies the angular span.
//
// Returns []PathCommand which holds the cubic Bezier segments.
func emitArcSegments(centreX, centreY, rx, ry, phi, theta1, dtheta float64) []PathCommand {
	segments := int(math.Ceil(math.Abs(dtheta) / (math.Pi / 2)))
	segments = max(segments, 1)
	segAngle := dtheta / float64(segments)

	result := make([]PathCommand, 0, segments)
	for i := range segments {
		t1 := theta1 + float64(i)*segAngle
		t2 := t1 + segAngle
		result = append(result, arcSegmentToCubic(centreX, centreY, rx, ry, phi, t1, t2))
	}
	return result
}

// arcSegmentToCubic converts a single arc segment into a cubic Bezier
// PathCommand.
//
// Takes cx (float64) which specifies the ellipse centre X. Takes cy (float64)
// which specifies the ellipse centre Y. Takes rx (float64) which specifies the
// X radius. Takes ry (float64) which specifies the Y radius. Takes phi
// (float64) which specifies the X axis rotation in radians. Takes theta1
// (float64) which specifies the segment start angle. Takes theta2 (float64)
// which specifies the segment end angle.
//
// Returns PathCommand which holds the cubic Bezier approximation of the segment.
func arcSegmentToCubic(cx, cy, rx, ry, phi, theta1, theta2 float64) PathCommand {
	alpha := math.Sin(theta2-theta1) * (math.Sqrt(4+arcAlphaDivisor*math.Pow(math.Tan((theta2-theta1)/2), 2)) - 1) / arcAlphaDivisor

	cos1 := math.Cos(theta1)
	sin1 := math.Sin(theta1)
	cos2 := math.Cos(theta2)
	sin2 := math.Sin(theta2)
	cosPhi := math.Cos(phi)
	sinPhi := math.Sin(phi)

	px := func(cosT, sinT float64) float64 {
		return cosPhi*rx*cosT - sinPhi*ry*sinT + cx
	}
	py := func(cosT, sinT float64) float64 {
		return sinPhi*rx*cosT + cosPhi*ry*sinT + cy
	}
	dpx := func(cosT, sinT float64) float64 {
		return -cosPhi*rx*sinT - sinPhi*ry*cosT
	}
	dpy := func(cosT, sinT float64) float64 {
		return -sinPhi*rx*sinT + cosPhi*ry*cosT
	}

	x1 := px(cos1, sin1)
	y1 := py(cos1, sin1)
	dx1 := dpx(cos1, sin1)
	dy1 := dpy(cos1, sin1)
	x2 := px(cos2, sin2)
	y2 := py(cos2, sin2)
	dx2 := dpx(cos2, sin2)
	dy2 := dpy(cos2, sin2)

	_ = x1
	_ = y1

	return PathCommand{
		Type: 'C',
		Args: []float64{
			x1 + alpha*dx1, y1 + alpha*dy1,
			x2 - alpha*dx2, y2 - alpha*dy2,
			x2, y2,
		},
	}
}

// vecAngle computes the signed angle between two 2D vectors.
//
// Takes ux (float64) which specifies the first vector X component. Takes uy
// (float64) which specifies the first vector Y component. Takes vx (float64)
// which specifies the second vector X component. Takes vy (float64) which
// specifies the second vector Y component.
//
// Returns float64 which holds the signed angle in radians.
func vecAngle(ux, uy, vx, vy float64) float64 {
	dot := ux*vx + uy*vy
	lenU := math.Sqrt(ux*ux + uy*uy)
	lenV := math.Sqrt(vx*vx + vy*vy)
	cos := dot / (lenU * lenV)
	cos = math.Max(-1, math.Min(1, cos))
	angle := math.Acos(cos)
	if ux*vy-uy*vx < 0 {
		angle = -angle
	}
	return angle
}

// pathToken represents a single token from an SVG path data string.
type pathToken struct {
	// value holds the string content of the token.
	value string

	// isCommand holds whether the token is a command letter rather than a number.
	isCommand bool
}

// tokenizePath splits an SVG path data string into command and number tokens.
//
// Takes d (string) which specifies the raw path data string.
//
// Returns []pathToken which holds the parsed tokens.
// Returns error when an unexpected character is encountered.
func tokenizePath(d string) ([]pathToken, error) {
	var tokens []pathToken
	r := strings.NewReader(d)

	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			break
		}
		if unicode.IsSpace(ch) || ch == ',' {
			continue
		}
		if isPathCommand(ch) {
			tokens = append(tokens, pathToken{value: string(ch), isCommand: true})
			continue
		}
		if ch == '+' || ch == '-' || ch == '.' || (ch >= '0' && ch <= '9') {
			num := readNumber(r, ch)
			tokens = append(tokens, pathToken{value: num, isCommand: false})
			continue
		}
		return nil, fmt.Errorf("svg: unexpected character %q in path data", ch)
	}
	return tokens, nil
}

// readNumber reads a complete numeric token from the reader, starting with
// the given first rune.
//
// Takes r (*strings.Reader) which specifies the input reader. Takes first
// (rune) which specifies the first character of the number already consumed.
//
// Returns string which holds the complete numeric token.
func readNumber(r *strings.Reader, first rune) string {
	var buf strings.Builder
	_, _ = buf.WriteRune(first)
	hasDot := first == '.'
	hasExponent := false

	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			break
		}

		if accepted := appendNumberRune(&buf, r, ch, &hasDot, &hasExponent); !accepted {
			_ = r.UnreadRune()
			break
		}
	}
	return buf.String()
}

// appendNumberRune attempts to append a rune to the number buffer, handling
// digits, decimal points, and exponent notation.
//
// Takes buf (*strings.Builder) which specifies the buffer to append to. Takes
// r (*strings.Reader) which specifies the input reader for exponent sign
// lookahead. Takes ch (rune) which specifies the rune to process. Takes hasDot
// (*bool) which specifies whether a decimal point has been seen. Takes
// hasExponent (*bool) which specifies whether an exponent has been seen.
//
// Returns bool which holds true if the rune was accepted as part of the number.
func appendNumberRune(buf *strings.Builder, r *strings.Reader, ch rune, hasDot, hasExponent *bool) bool {
	if ch >= '0' && ch <= '9' {
		_, _ = buf.WriteRune(ch)
		return true
	}
	if ch == '.' && !*hasDot && !*hasExponent {
		*hasDot = true
		_, _ = buf.WriteRune(ch)
		return true
	}
	if (ch == 'e' || ch == 'E') && !*hasExponent {
		*hasExponent = true
		_, _ = buf.WriteRune(ch)
		readExponentSign(buf, r)
		return true
	}
	return false
}

// readExponentSign reads an optional sign character following an exponent
// indicator and appends it to the buffer.
//
// Takes buf (*strings.Builder) which specifies the buffer to append to. Takes
// r (*strings.Reader) which specifies the input reader.
func readExponentSign(buf *strings.Builder, r *strings.Reader) {
	next, _, err := r.ReadRune()
	if err != nil {
		return
	}
	if next == '+' || next == '-' {
		_, _ = buf.WriteRune(next)
	} else {
		_ = r.UnreadRune()
	}
}

// isPathCommand reports whether the rune is a valid SVG path command letter.
//
// Takes ch (rune) which specifies the character to test.
//
// Returns bool which holds true if the character is a recognised path command.
func isPathCommand(ch rune) bool {
	switch ch {
	case 'M', 'm', 'L', 'l', 'H', 'h', 'V', 'v',
		'C', 'c', 'S', 's', 'Q', 'q', 'T', 't',
		'A', 'a', 'Z', 'z':
		return true
	}
	return false
}

// commandArgCount returns the number of numeric arguments expected for the
// given SVG path command.
//
// Takes cmd (byte) which specifies the command letter (case insensitive).
//
// Returns int which holds the expected argument count.
func commandArgCount(cmd byte) int {
	switch toUpper(cmd) {
	case 'M', 'L', 'T':
		return 2
	case 'H', 'V':
		return 1
	case 'C':
		return cubicArgCount
	case 'S', 'Q':
		return quadArgCount
	case 'A':
		return arcArgCount
	case 'Z':
		return 0
	}
	return 0
}

// toUpper converts a lowercase ASCII command letter to uppercase.
//
// Takes cmd (byte) which specifies the command letter to convert.
//
// Returns byte which holds the uppercase command letter, or the original if
// already uppercase.
func toUpper(cmd byte) byte {
	if cmd >= 'a' && cmd <= 'z' {
		return cmd - asciiCaseDiff
	}
	return cmd
}

// consumeArgs reads the specified number of numeric arguments from the token
// stream starting at the given position.
//
// Takes tokens ([]pathToken) which specifies the full token stream. Takes pos
// (int) which specifies the starting position. Takes argCount (int) which
// specifies the number of arguments to consume.
//
// Returns []float64 which holds the parsed argument values.
// Returns int which holds the updated position.
// Returns error when there are insufficient tokens or
// invalid numbers.
func consumeArgs(tokens []pathToken, pos, argCount int) ([]float64, int, error) {
	args := make([]float64, argCount)
	for j := range argCount {
		if pos >= len(tokens) {
			return nil, pos, fmt.Errorf("expected %d args, got %d", argCount, j)
		}
		if tokens[pos].isCommand {
			return nil, pos, fmt.Errorf("expected number, got command %q", tokens[pos].value)
		}
		v, err := parsePathFloat(tokens[pos].value)
		if err != nil {
			return nil, pos, fmt.Errorf("invalid number %q: %w", tokens[pos].value, err)
		}
		args[j] = v
		pos++
	}
	return args, pos, nil
}

// parsePathFloat parses a numeric string into a float64 using the %g format.
//
// Takes s (string) which specifies the numeric string to parse.
//
// Returns float64 which holds the parsed value.
// Returns error when the string is not a valid number.
func parsePathFloat(s string) (float64, error) {
	var result float64
	_, err := fmt.Sscanf(s, "%g", &result)
	return result, err
}

// makeAbsolute converts relative command arguments to absolute by adding the
// current position offsets.
//
// Takes absCmd (byte) which specifies the uppercase command type. Takes args
// ([]float64) which specifies the arguments to convert in place. Takes cx
// (float64) which specifies the current X position. Takes cy (float64) which
// specifies the current Y position.
func makeAbsolute(absCmd byte, args []float64, cx, cy float64) {
	switch absCmd {
	case 'M', 'L', 'T':
		args[0] += cx
		args[1] += cy
	case 'H':
		args[0] += cx
	case 'V':
		args[0] += cy
	case 'C':
		args[0] += cx
		args[1] += cy
		args[cubicCP1XIndex] += cx
		args[cubicCP1YIndex] += cy
		args[cubicEndXIndex] += cx
		args[arcEndXIndex] += cy
	case 'S', 'Q':
		args[0] += cx
		args[1] += cy
		args[cubicCP1XIndex] += cx
		args[cubicCP1YIndex] += cy
	case 'A':
		args[arcEndXIndex] += cx
		args[arcEndYIndex] += cy
	}
}
