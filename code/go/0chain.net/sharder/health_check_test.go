package sharder_test

import (
	"testing"

	"0chain.net/sharder"
)

func TestHealthCheckScan_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		e         sharder.HealthCheckScan
		want      string
		wantPanic bool
	}{
		{
			name: "Test_HealthCheckScan_String_Deep_OK",
			e:    0,
			want: "Deep.....",
		},
		{
			name: "Test_HealthCheckScan_String_Proximity_OK",
			e:    1,
			want: "Proximity",
		},
		{
			name:      "Test_HealthCheckScan_String_Proximity_PANIC",
			e:         2, // e > 1 will panic
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic, but it is not")
					}
				}()
			}

			if got := tt.e.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
