package pairlist

import "sort"

type Pair struct {
	Key   string
	Value int
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func RankByValue(freqs map[string]int, reverse bool) PairList {
	pl := make(PairList, len(freqs))
	i := 0
	for k, v := range freqs {
		pl[i] = Pair{k, v}
		i++
	}
	if reverse {
		sort.Sort(sort.Reverse(pl))
	} else {
		sort.Sort(pl)
	}
	return pl
}
