package fpu_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/jetsetilly/test7800/hardware/arm/fpu"
	"github.com/jetsetilly/test7800/test"
)

const (
	iterations = 10000
	integerMax = 15000000
)

func TestArithmetic_random(t *testing.T) {
	var fp fpu.FPU

	fpscr := fp.StandardFPSCRValue()
	fpscr.SetRMode(fpu.FPRoundNearest)

	f := func(t *testing.T) {
		var v, w float64
		var c, d uint64
		v = rand.Float64()
		c = fp.FPRound(v, 64, fpscr)
		w = rand.Float64()
		d = fp.FPRound(w, 64, fpscr)

		var r, s uint64

		// addition
		r = fp.FPAdd(c, d, 64, false)
		s = math.Float64bits(v + w)
		test.ExpectEquality(t, r, s)

		// subtraction
		r = fp.FPSub(c, d, 64, false)
		s = math.Float64bits(v - w)
		test.ExpectEquality(t, r, s)

		// multiplication
		r = fp.FPMul(c, d, 64, false)
		s = math.Float64bits(v * w)
		test.ExpectEquality(t, r, s)

		// division
		r = fp.FPDiv(c, d, 64, false)
		s = math.Float64bits(v / w)
		test.ExpectEquality(t, r, s)

		var q uint64

		// mutliplication and add
		r = fp.FPRound(2.5, 32, fpscr)
		s = fp.FPRound(-3.1, 32, fpscr)
		q = fp.FPRound(100, 32, fpscr)
		q = fp.FPMulAdd(q, r, s, 32, false)
		_, _, f := fp.FPUnpack(q, 32, fpscr)
		test.ExpectEquality(t, f, (2.5*-3.1)+100)
	}

	for range iterations {
		t.Run("arithmetic", f)
	}
}

func TestNegation_random(t *testing.T) {
	var fp fpu.FPU

	f := func(t *testing.T) {
		var v float64
		var c uint32
		var d uint32

		v = rand.Float64()
		c = math.Float32bits(float32(v))
		d = math.Float32bits(float32(-v))

		// the two values should be unequal at this point
		test.ExpectInequality(t, c, d)

		// negate one of the values. the two value will now be equal
		d = uint32(fp.FPNeg(uint64(d), 32))
		test.ExpectEquality(t, c, d)

		// negate again to make the values unequal
		d = uint32(fp.FPNeg(uint64(d), 32))
		test.ExpectInequality(t, c, d)

		// and again to make them equal again
		d = uint32(fp.FPNeg(uint64(d), 32))
		test.ExpectEquality(t, c, d)
	}

	for range iterations {
		t.Run("negation", f)
	}
}

func TestAbsolute_random(t *testing.T) {
	var fp fpu.FPU

	f := func(t *testing.T) {
		var v float64
		var c uint32
		var d uint32

		v = rand.Float64()
		c = math.Float32bits(float32(v))
		d = math.Float32bits(float32(-v))

		// the two values should be unequal at this point
		test.ExpectInequality(t, c, d)

		var r uint32

		// force the negative value to be positive
		r = uint32(fp.FPAbs(uint64(d), 32))
		test.ExpectEquality(t, r, c)

		// forcing a positive value has no effect
		r = uint32(fp.FPAbs(uint64(c), 32))
		test.ExpectEquality(t, r, c)
	}

	for range iterations {
		t.Run("absolute", f)
	}
}

func TestRound_random(t *testing.T) {
	var fp fpu.FPU

	fpscr := fp.StandardFPSCRValue()
	fpscr.SetRMode(fpu.FPRoundNearest)

	f := func(t *testing.T) {
		var v float64
		var b uint64
		var c uint32
		v = rand.Float64()
		b = fp.FPRound(v, 32, fpscr)
		c = math.Float32bits(float32(v))
		test.ExpectEquality(t, uint32(b), c)
	}

	for range iterations {
		t.Run("rounding", f)
	}
}

func TestFixedToFP_random(t *testing.T) {
	var fp fpu.FPU

	f := func(t *testing.T) {
		var v uint64
		var c uint64

		v = rand.Uint64N(integerMax)

		// 32 bit
		c = fp.FixedToFP(v, 32, 0, false, true, true)
		test.ExpectEquality(t, c, uint64(math.Float32bits(float32(v))))

		// 64 bit
		c = fp.FixedToFP(v, 64, 0, false, true, true)
		test.ExpectEquality(t, c, math.Float64bits(float64(v)))
	}

	for range iterations {
		t.Run("fixed to fp", f)
	}
}

func TestFPToFixed_random(t *testing.T) {
	var fp fpu.FPU

	f := func(t *testing.T) {
		var v uint64
		var c uint64
		var d uint64

		// 32 bit
		v = rand.Uint64N(integerMax)
		c = fp.FixedToFP(v, 32, 0, false, true, true)
		d = fp.FPToFixed(c, 32, 0, false, true, true)
		test.ExpectEquality(t, d, v)

		// 64 bit
		c = fp.FixedToFP(v, 64, 0, false, true, true)
		d = fp.FPToFixed(c, 64, 0, false, true, true)
		test.ExpectEquality(t, d, v)
	}

	for range iterations {
		t.Run("fp to fixed", f)
	}
}
