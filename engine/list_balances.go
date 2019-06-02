package engine

type listBalances map[int64]int64

func (b listBalances) Inc(id int64, amount int64) {
	b[id] += amount
}

func (b listBalances) Dec(id int64, amount int64) {
	b[id] -= amount
}
