package engine

type listBalances map[int64]int64

func (b listBalances) get(id int64) int64 {
	return b[id]
}

func (b listBalances) inc(id int64, amount int64) {
	b[id] += amount
}

func (b listBalances) dec(id int64, amount int64) {
	b[id] -= amount
}
