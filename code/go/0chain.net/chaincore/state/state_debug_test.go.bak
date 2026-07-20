package state

import "testing"

func init() {
	SetDebugLevel(DebugLevelChain)
}

func TestDebug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want bool
	}{
		{
			name: "TRUE",
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := Debug(); got != tt.want {
				t.Errorf("Debug() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDebugChain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want bool
	}{
		{
			name: "TRUE",
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := DebugChain(); got != tt.want {
				t.Errorf("DebugChain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDebugBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want bool
	}{
		{
			name: "FALSE",
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := DebugBlock(); got != tt.want {
				t.Errorf("DebugBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDebugTxn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want bool
	}{
		{
			name: "FALSE",
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := DebugTxn(); got != tt.want {
				t.Errorf("DebugTxn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDebugNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want bool
	}{
		{
			name: "FALSE",
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := DebugNode(); got != tt.want {
				t.Errorf("DebugNode() = %v, want %v", got, tt.want)
			}
		})
	}
}
