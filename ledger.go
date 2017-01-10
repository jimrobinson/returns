package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// skip_payee lists strings to search for in the Payee field,
// any entry that matches one of these search strings will
// be skipped because they pertain to growth or losses of
// a fund distinct from contributions or withdrawals.
var skip_payee = []string{
	"Dividend",
	"Gain",
	"Fee",
	"Adjustment",
}

// balance returns the account balances of the
// of the specified start (inclusive) and stop
// dates (exclusive)
func balance(start, stop time.Time) (Accounts, error) {
	accounts := make(Accounts)

	cmd := exec.Command(
		"ledger", "balance",
		"-e", start.Format("2006-01-02"),
		"-C", "-V", "--flat", "--no-total",
		"^assets:", "and", "expr", `commodity != "$"`)

	balance := new(bytes.Buffer)
	cmd.Stdin = os.Stdin
	cmd.Stdout = balance
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	r := bufio.NewReader(balance)
	for {
		var line []byte
		var part bool

		line, part, err = r.ReadLine()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return nil, err
		}

		buf.Write(line)
		if part {
			continue
		}

		tok := strings.SplitN(strings.TrimSpace(buf.String()), " ", 2)
		if len(tok) == 2 {
			amount := tok[0]
			name := strings.TrimSpace(tok[1])

			entry := new(Entry)
			entry.Time = start
			entry.Amount, err = strconv.ParseFloat(
				strings.Replace(strings.Replace(amount, "$", "", 1), ",", "", -1), 64)
			if err != nil {
				return nil, err
			}
			entry.Amount = entry.Amount
			entry.Payee = "Opening Balance"

			account := new(Account)
			account.Name = name
			account.Opening = entry
			accounts[account.Name] = account
		}

		buf.Reset()
	}

	cmd = exec.Command(
		"ledger", "balance",
		"-e", stop.Format("2006-01-02"),
		"-C", "-V", "--flat", "--no-total",
		"^assets:", "and", "expr", `commodity != "$"`)

	balance = new(bytes.Buffer)
	cmd.Stdin = os.Stdin
	cmd.Stdout = balance
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	buf.Reset()
	r = bufio.NewReader(balance)
	for {
		var line []byte
		var part bool

		line, part, err = r.ReadLine()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return nil, err
		}

		buf.Write(line)
		if part {
			continue
		}

		tok := strings.SplitN(strings.TrimSpace(buf.String()), " ", 2)
		if len(tok) == 2 {
			amount := tok[0]
			name := strings.TrimSpace(tok[1])

			entry := new(Entry)
			entry.Time = stop
			entry.Amount, err = strconv.ParseFloat(
				strings.Replace(strings.Replace(amount, "$", "", 1), ",", "", -1), 64)
			if err != nil {
				return nil, err
			}
			entry.Amount = -entry.Amount
			entry.Payee = "Closing Balance"

			account, ok := accounts[name]
			if !ok {
				account = new(Account)
				account.Name = name
				accounts[account.Name] = account
			}

			account.Closing = entry
		}

		buf.Reset()
	}

	return accounts, nil
}

// accounts returns the commodity Accounts history and ending value
// as produced by ledger(1) for a specified date range.  It relies on the
// ledger dataset only having either '$' or stock/bond symbols for
// commodities (i.e., no foreigh currency transactions).
func accounts(start, stop time.Time) (accounts Accounts, err error) {
	accounts, err = balance(start, stop)
	if err != nil {
		return nil, err
	}

	// generate a '!' separated list of the date, dollar amount,
	// account name, and payee memo for all transactions where the
	// commodity was not in the US Dollar (in this particular
	// dataset that works out to all transactions using a
	// stock/bond ticker symbols)
	cmd := exec.Command(
		"ledger", "register",
		"-b", start.Format("2006-01-02"), "-e", stop.Format("2006-01-02"),
		"-C", "-B", "^assets:", "and", "expr", `commodity != "$"`,
		"--format", `%(format_date(date))|%(display_amount)|%(account)|%(payee)\n`)

	register := new(bytes.Buffer)

	cmd.Stdin = os.Stdin
	cmd.Stdout = register
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	r := bufio.NewReader(register)
	for {
		var line []byte
		var part bool

		line, part, err = r.ReadLine()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return nil, err
		}

		buf.Write(line)
		if part {
			continue
		}

		tok := strings.Split(strings.TrimSpace(buf.String()), "|")
		buf.Reset()

		if len(tok) != 4 {
			log.Printf("unexpected token count: %d: %s\n", len(tok), buf.String())
			continue
		}

		date, amount, name, payee := tok[0], tok[1], tok[2], tok[3]
		if date == "" || amount == "" || payee == "" {
			continue
		}

		var account *Account
		if account = accounts[name]; account == nil {
			continue
		}

		// skip over the entry if its payee field
		// matches an entry in skip_payee
		skip := false
		for _, v := range skip_payee {
			if strings.Index(payee, v) != -1 {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		entry := new(Entry)
		entry.Time, err = time.Parse(fmt_day, date)
		if err != nil {
			log.Println(err)
			continue
		}
		entry.Amount, err = strconv.ParseFloat(
			strings.Replace(strings.Replace(amount, "$", "", 1), ",", "", -1), 64)
		if err != nil {
			log.Println(err)
			continue
		}
		entry.Payee = payee

		account.Cashflow = append(account.Cashflow, entry)
	}

	for _, account := range accounts {
		if account.Closing == nil && len(account.Cashflow) != 0 {
			n := len(account.Cashflow) - 1
			account.Closing = account.Cashflow[n]
			account.Cashflow = account.Cashflow[0:n]
		}
	}

	return accounts, nil
}
