package main

type Cashflow []*Entry

func (c Cashflow) Len() int {
	return len(c)
}
func (c Cashflow) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c Cashflow) Less(i, j int) bool {
	return c[i].Time.Before(c[j].Time)
}
