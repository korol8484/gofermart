package domain

import "testing"

func TestConvertToCurrencyUnit(t *testing.T) {
	tests := []struct {
		name string
		want float64
	}{
		{name: "want_123", want: 123},
		{name: "want_123.22", want: 123.22},
		{name: "want_123.222", want: 123.22},
		{name: "want_1", want: 1},
		{name: "want_0", want: 0},
		{name: "want_0.1", want: 0.1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if ConvertFromCurrencyUnit(ConvertToCurrencyUnit(test.want)) != test.want {
				t.Errorf("can't convert")
			}
		})
	}
}
