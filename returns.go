package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

var verbose *bool = flag.Bool("v", false, "print cashflows")
var merge *bool = flag.Bool("m", false, "merge cashflows")

var alltime *bool = flag.Bool("a", false, "calculate returns over all time")
var begin *string = flag.String("b", time.Now().Add(-time.Hour*24*30*3).Format("2006-01-02"), "reporting start date")
var end *string = flag.String("e", time.Now().Format("2006-01-02"), "reporting end date")
var period *string = flag.String("p", fmt.Sprintf("%d", time.Now().Format("2001")), "reporting period (default: current year)")

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [-v] [-m] [<substring> ...]\n", os.Args[0])
	flag.PrintDefaults()
}

var fmt_day = "2006-01-02"

func main() {
	flag.Usage = usage
	flag.Parse()

	timeFmt := []string{
		"2006-01-02",
		"2006-01",
		"2006",
	}

	if *alltime {
		*begin = "1900-01-01"
	} else if *period != "" {
		for i, v := range timeFmt {
			dt, err := time.Parse(v, *period)
			if err == nil {
				switch i {
				case 0:
					*begin = dt.Format(v)
					*end = dt.Add(time.Hour * 24).Format("2006-01-02")
				case 1:
					*begin = dt.Format("2006-01-02")
					*end = dt.Add(time.Hour*24*31).Format("2006-01") + "-01"
				case 2:
					*begin = dt.Format("2006-01-02")
					*end = dt.Add(time.Hour*24*366).Format("2006") + "-01-01"
				}
				break
			}
		}
	}

	var start, stop time.Time
	for _, v := range timeFmt {
		dt, err := time.Parse(v, *begin)
		if err == nil {
			start = dt
			break
		}
	}

	for _, v := range timeFmt {
		dt, err := time.Parse(v, *end)
		if err == nil {
			stop = dt
			break
		}
	}

	accounts, err := accounts(start, stop)
	if err != nil {
		log.Fatal(err)
	}

	if accounts == nil {
		log.Fatal("no accounts")
	}

	if *merge {
		account := new(Account)
		account.Closing = new(Entry)

		names := make(sort.StringSlice, 0)
		for _, subaccount := range accounts {
			if len(flag.Args()) == 0 {
				names = append(names, subaccount.Name)

				if subaccount.Opening != nil {
					if account.Opening == nil {
						account.Opening = new(Entry)
						account.Opening.Payee = "Opening Balance"
					}
					if account.Opening.Time.IsZero() || subaccount.Opening.Time.Before(account.Opening.Time) {
						account.Opening.Time = subaccount.Opening.Time
					}
					account.Opening.Amount += subaccount.Opening.Amount
				}

				if account.Closing.Time.IsZero() || subaccount.Closing.Time.After(account.Closing.Time) {
					account.Closing.Time = subaccount.Closing.Time
				}
				account.Closing.Amount += subaccount.Closing.Amount

				for _, entry := range subaccount.Cashflow {
					account.Cashflow = append(account.Cashflow, entry)
				}
			} else {
				for _, s := range flag.Args() {
					if strings.Index(strings.ToLower(subaccount.Name), strings.ToLower(s)) > -1 {
						names = append(names, subaccount.Name)

						if subaccount.Opening != nil {
							if account.Opening == nil {
								account.Opening = new(Entry)
							}
							if account.Opening.Time.IsZero() || subaccount.Opening.Time.Before(account.Opening.Time) {
								account.Opening.Time = subaccount.Opening.Time
							}
							account.Opening.Amount += subaccount.Opening.Amount
						}

						if account.Closing.Time.IsZero() || subaccount.Closing.Time.After(account.Closing.Time) {
							account.Closing.Time = subaccount.Closing.Time
						}
						account.Closing.Amount += subaccount.Closing.Amount

						for _, entry := range subaccount.Cashflow {
							account.Cashflow = append(account.Cashflow, entry)
						}
						break
					}
				}
			}
		}

		if len(flag.Args()) > 0 && len(names) == 0 {
			fmt.Printf("no accounts matched the provided substrings:\n")
			for _, s := range flag.Args() {
				fmt.Printf("\t%s\n", s)
			}
			return
		}

		xirr, err := account.xirr()
		if err != nil {
			fmt.Printf("%s\n", err)
			return
		}


		fmt.Printf("%s - %s:\n", start.Format("2006-01-02"), stop.Format("2006-01-02"))
		fmt.Printf("%12s\t%6.2f%%\t%12s\n",
			account.Opening.Dollars(true),
			xirr*100.0, account.Closing.Dollars(true))
		sort.Sort(names)
		for i := range names {
			fmt.Printf("\t%s\n", names[i])
		}

		if *verbose {
			for _, entry := range account.Entries() {
				fmt.Printf("\t%s\t%12s\t%s\n",
					entry.Time.Format(fmt_day), entry.Dollars(false), entry.Payee)
			}
		}
	} else {
		names := make(sort.StringSlice, 0)
		for name, _ := range accounts {
			if len(flag.Args()) == 0 {
				names = append(names, name)
			} else {
				for _, s := range flag.Args() {
					if strings.Index(strings.ToLower(name), strings.ToLower(s)) > -1 {
						names = append(names, name)
						break
					}
				}
			}
		}
		sort.Sort(names)

		i := 0
		for _, name := range names {
			if *verbose && i > 0 {
				fmt.Print("\n")
			}
			i++

			account := accounts[name]

			xirr, err := account.xirr()
			if err != nil {
				fmt.Printf("%s: %s\n", account.Name, err)
				continue
			}

			if (i == 1) {
				fmt.Printf("%s - %s:\n", start.Format("2006-01-02"), stop.Format("2006-01-02"))
			}
			fmt.Printf("%12s\t%6.2f%%\t%12s\t%s\n",
				account.Opening.Dollars(true),
				xirr*100.0, account.Closing.Dollars(true),
				account.Name)
			if *verbose {
				for _, entry := range account.Entries() {
					fmt.Printf("\t%s\t%12s\t%s\n",
						entry.Time.Format(fmt_day), entry.Dollars(false), entry.Payee)
				}
			}
		}
	}
}
