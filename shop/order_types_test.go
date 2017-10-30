package shop

import "testing"

func TestOrderType_Prefix(t *testing.T) {
	tests := []struct {
		name string
		t    OrderType
		want string
	}{
		{
			"",
			OrderType(""),
			"",
		},
		{
			"",
			OrderType("a"),
			"A",
		},
		{
			"",
			OrderType("aBc"),
			"ABC",
		},
		{
			"",
			OrderType("aBcAb"),
			"ABCAB",
		},
		{
			"",
			OrderType("aBcAbC"),
			"ABCAB",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.Prefix(); got != tt.want {
				t.Errorf("OrderType.Prefix() = %v, want %v", got, tt.want)
			}
		})
	}
}
