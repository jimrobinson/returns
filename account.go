package main

import (
	"fmt"
	"sort"
	"time"
)

var maxPosNegPoints = int(1000)
var precision = float64(0.000001)
var maxDepth = int(100)

type Accounts map[string]*Account

type Account struct {
	Name     string
	Cashflow Cashflow
	Opening  *Entry
	Closing  *Entry
}

// Entries returns a sorted set of Entry, excluding any entry that is
// after the Closing value entry or that has a zero Amount.
func (account *Account) Entries() (arr Cashflow) {
	arr = make(Cashflow, 0, len(account.Cashflow)+2)
	if account.Opening != nil {
		arr = append(arr, account.Opening)
	}
	for _, c := range account.Cashflow {
		if c.Time.After(account.Closing.Time) {
			continue
		}
		if c.Amount == 0 {
			continue
		}
		arr = append(arr, c)
	}
	arr = append(arr, account.Closing)
	sort.Sort(arr)
	return
}

// xirr uses the logic contributed in the Perl Finance::Math:IRR module
// by Erwan Lemonnier to compute the XIRR for an account cashflow.
func (account *Account) xirr() (irr float64, err error) {

	var startDt time.Time
	if account.Opening != nil {
		startDt = account.Opening.Time
	} else if len(account.Cashflow) != 0 {
		startDt = account.Cashflow[0].Time
	} else {
		startDt = account.Closing.Time
	}

	entries := account.Entries()
	coefficient := make(map[float64]float64)
	for _, entry := range entries {
		ddays := entry.Time.Sub(startDt).Hours() / 24.0
		coefficient[ddays/365.0] += entry.Amount
	}

	for k, v := range coefficient {
		if v == 0 {
			delete(coefficient, k)
		}
	}

	if len(coefficient) >= 2 {
		poly := NewPolynom(coefficient)
		var root float64

		root, err = poly.Secant(0.5, 1.0, precision, maxDepth)
		if err != nil {
			i := 1
			for ((poly.xneg == nil) || (poly.xpos == nil)) && i <= maxPosNegPoints {
				poly.Eval(float64(i))
				poly.Eval(-1 + 10/(float64(i)+9))
				i++
			}

			if (poly.xneg == nil) || (poly.xpos == nil) {
				err = fmt.Errorf("failed to find an interval on which polynomial is >0 and <0 at the boundaries")
				return
			}

			a := *poly.xneg
			b := *poly.xpos
			root, err = poly.Brent(a, b, precision, maxDepth)
		}

		if root == 0 {
			irr = 0
			err = fmt.Errorf("got root of 0, meaning infinite IRR")
			return
		}

		irr = -1 + 1/root
	}
	return
}
