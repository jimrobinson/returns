package main

import (
	"bytes"
	"strconv"
	"strings"
	"time"
)

type Entry struct {
	Time   time.Time
	Amount float64
	Payee  string
}

func (e Entry) Dollars(positive bool) string {

	var str []string
	if positive && 0 > e.Amount {
		str = strings.Split(strconv.FormatFloat(-1.0*e.Amount, 'f', 2, 64), "")
	} else {
		str = strings.Split(strconv.FormatFloat(e.Amount, 'f', 2, 64), "")
	}

	dec := 0
	dot := false
	rev := make([]string, 0, len(str)*2)
	for i := len(str) - 1; i >= 0; i-- {
		if str[i] == "." {
			dot = true
		}
		rev = append(rev, str[i])
		if dot && dec >= 3 && (dec%3) == 0 {
			if i != 0 && !(str[0] == "-" && i == 1) {
				rev = append(rev, ",")
			}
		}
		if dot {
			dec++
		}
	}

	buf := new(bytes.Buffer)
	buf.WriteRune('$')
	for i := len(rev) - 1; i >= 0; i-- {
		buf.WriteString(rev[i])
	}
	return buf.String()
}
