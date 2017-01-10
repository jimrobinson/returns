package main

import (
	"fmt"
	"math"
)

// Polynom emulates the logic for approximating the root of a
// polynomial using the secant or Brent's methods in the perl
// Math::Polynomial module by Erwan Lemonnier
type Polynom struct {
	polynom    map[float64]float64
	xpos, xneg *float64
	iterations int
}

func NewPolynom(polynom map[float64]float64) *Polynom {
	poly := new(Polynom)
	poly.polynom = polynom
	return poly
}

func (poly *Polynom) Secant(p0, p1, precision float64, max_depth int) (root float64, err error) {
	q0 := poly.Eval(p0)
	q1 := poly.Eval(p1)

	if math.IsNaN(q0) || math.IsNaN(q1) {
		err = fmt.Errorf("one or both or q0 and q1 are not a real number in first evaluation in secant")
		return
	}

	var p float64
	for depth := 1; depth <= max_depth; depth++ {
		poly.iterations = depth

		if (q1 - q0) == 0 {
			err = fmt.Errorf("division by zero with p0=%f, p1=%f, q1=q0=%f", p0, p1, q1)
			return
		}

		p = (q1*p0 - p1*q0) / (q1 - q0)
		if math.IsNaN(p) {
			err = fmt.Errorf("p is not a real number when p1=%f, p0=%f, q1=%f, q0=%f", p1, p0, q1, q0)
			return
		}

		p0 = p1
		q0 = q1
		q1 = poly.Eval(p)
		if math.IsNaN(q1) {
			err = fmt.Errorf("q1 is not a real number when p = %f", p)
			return
		}

		if (q1 == 0) || math.Abs(p-p1) <= precision {
			root = p
			if !poly.isRoot(p) {
				err = fmt.Errorf("secant converges toward %f but that doesn't appear to be a root.", p)
			}
			return
		}

		p1 = p
	}

	return
}

func (poly *Polynom) Brent(a, b, precision float64, max_depth int) (root float64, err error) {
	poly.iterations = 0

	if len(poly.polynom) == 0 {
		err = fmt.Errorf("cannot find the root of an empty polynomial")
		return
	}

	f_a := poly.Eval(a)
	f_b := poly.Eval(b)
	if math.IsNaN(f_a) || math.IsNaN(f_b) {
		err = fmt.Errorf("polynomial is not defined on interval [a=%f, b=%f] in brent", a, b)
		return
	}

	if f_a == 0 {
		root = a
		return
	}
	if f_b == 0 {
		root = b
		return
	}

	if f_a*f_b > 0 {
		err = fmt.Errorf("polynomial does not have opposite signs at a=%f and b=%f in brent", a, b)
	}

	if math.Abs(f_a) < math.Abs(f_b) {
		a, f_a, b, f_b = b, f_b, a, f_a
	}

	c := a
	f_c := f_a

	mflag := 1

	for f_b != 0 && math.Abs(b-a) > precision {

		if poly.iterations > max_depth {
			err = fmt.Errorf("reached maximum iterations %d without getting close enough to the root",
				poly.iterations)
			return
		}

		if poly.iterations != 0 {
			f_a = poly.Eval(a)
			f_b = poly.Eval(b)
			f_c = poly.Eval(c)

			if math.IsNaN(f_a) {
				err = fmt.Errorf("polynomial leads to an imaginary number on a=$a in brent()")
				return
			}
			if math.IsNaN(f_b) {
				err = fmt.Errorf("polynomial leads to an imaginary number on b=$b in brent()")
				return
			}
			if math.IsNaN(f_c) {
				err = fmt.Errorf("polynomial leads to an imaginary number on c=$c in brent()")
				return
			}
		}

		var s, f_s, d float64
		if f_a == f_b {
			err = fmt.Errorf("got same values for polynomial at a=%f and b=%f:\n", a, b)
			return
		} else if (f_a != f_c) && (f_b != f_c) {
			s = (a*f_b*f_c)/((f_a-f_b)*(f_a-f_c)) +
				(b*f_a*f_c)/((f_b-f_a)*(f_b-f_c)) +
				(c*f_a*f_b)/((f_c-f_a)*(f_c-f_b))
		} else {
			s = b - f_b*(b-a)/(f_b-f_a)
		}

		if (s < (3*a+b)/4) && (s > b) ||
			((mflag == 1) && (math.Abs(s-b) >= (math.Abs(b-c) / 2))) ||
			((mflag == 0) && (math.Abs(s-b) >= (math.Abs(c-d) / 2))) {
			s = (a + b) / 2
			mflag = 1
		} else {
			mflag = 0
		}

		f_s = poly.Eval(s)
		if math.IsNaN(f_s) {
			err = fmt.Errorf("polynomial leads to an imaginary number on s=%f in brent", s)
			return
		}

		d = c
		c = b
		f_c = f_b

		if f_a*f_s <= 0 {
			b = s
			f_b = f_s
		} else {
			a = s
			f_a = f_s
		}

		if math.Abs(f_a) < math.Abs(f_b) {
			a, b, f_a, f_b = b, a, f_b, f_a
		}

		poly.iterations++
	}

	root = b
	if !poly.isRoot(root) {
		err = fmt.Errorf("brent converges toward %f but that doesn't appear to be a root", b)
	}

	return
}

func (poly *Polynom) Eval(x float64) (r float64) {
	for power, coef := range poly.polynom {
		r += coef * math.Pow(x, power)
	}
	if !math.IsNaN(r) {
		if poly.xpos == nil && r > 0 {
			poly.xpos = new(float64)
			*poly.xpos = x
		} else if poly.xneg == nil && r < 0 {
			poly.xneg = new(float64)
			*poly.xneg = x
		}
	}
	return
}

func (poly *Polynom) isRoot(x float64) bool {
	return math.Abs(poly.Eval(x)) < 1
}
