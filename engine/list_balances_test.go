package engine

import (
	"reflect"
	"testing"
)

func Test_listBalances_Inc(t *testing.T) {
	tests := []struct {
		name   string
		b      listBalances
		id     int64
		amount int64
		want   listBalances
	}{
		{
			"ExistsKey",
			listBalances{1: 100, 2: 200},
			1,
			100,
			listBalances{1: 200, 2: 200},
		},

		{
			"NotExistsKey",
			listBalances{2: 200},
			1,
			200,
			listBalances{1: 200, 2: 200},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.b.Inc(tt.id, tt.amount)
			if !reflect.DeepEqual(tt.b, tt.want) {
				t.Error("not equal")
			}
		})
	}
}

func Test_listBalances_Dec(t *testing.T) {
	tests := []struct {
		name   string
		b      listBalances
		id     int64
		amount int64
		want   listBalances
	}{
		{
			"ExistsKey",
			listBalances{1: 100, 2: 200},
			1,
			100,
			listBalances{1: 0, 2: 200},
		},

		{
			"NotExistsKey",
			listBalances{2: 200},
			1,
			200,
			listBalances{1: -200, 2: 200},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.b.Dec(tt.id, tt.amount)
			if !reflect.DeepEqual(tt.b, tt.want) {
				t.Error("not equal")
			}
		})
	}
}
