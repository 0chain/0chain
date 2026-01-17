package common

import "testing"

func TestLookup_GetCode(t *testing.T) {
	t.Parallel()

	type fields struct {
		Code  string
		Value string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "Test_Lookup_GetCode_OK",
			fields: fields{Code: "code"},
			want:   "code",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := &Lookup{
				Code:  tt.fields.Code,
				Value: tt.fields.Value,
			}
			if got := l.GetCode(); got != tt.want {
				t.Errorf("GetCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLookup_GetValue(t *testing.T) {
	t.Parallel()

	type fields struct {
		Code  string
		Value string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "Test_Lookup_GetValue_OK",
			fields: fields{Value: "value"},
			want:   "value",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := &Lookup{
				Code:  tt.fields.Code,
				Value: tt.fields.Value,
			}
			if got := l.GetValue(); got != tt.want {
				t.Errorf("GetValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
