package magmasc

import (
	"reflect"
	"testing"
)

const (
	testCode = "test_code"
	testText = "test text"
	wrapCode = "wrap_code"
	wrapText = "wrap text"
)

func Test_errWrapper_Error(t *testing.T) {
	t.Parallel()

	tests := [1]struct {
		name string
		err  error
		want string
	}{
		{
			name: "OK",
			err:  errWrap(wrapCode, wrapText, errNew(testCode, testText)),
			want: wrapCode + errDelim + wrapText + errDelim + testCode + errDelim + testText,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.err.Error(); got != test.want {
				t.Errorf("Error() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_errWrapper_Unwrap(t *testing.T) {

	err := errNew(testCode, testText)

	tests := [1]struct {
		name    string
		wrapper *errWrapper
		want    error
	}{
		{
			name:    "OK",
			wrapper: errWrap(wrapCode, wrapText, err),
			want:    err,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.wrapper.Unwrap(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Unwrap() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_errWrapper_WrapErr(t *testing.T) {
	t.Parallel()

	err := errNew(testCode, testText)

	tests := [1]struct {
		name    string
		error   error
		wrapper *errWrapper
		want    *errWrapper
	}{
		{
			name:    "OK",
			error:   errNew(testCode, testText),
			wrapper: errNew(wrapCode, wrapText),
			want:    &errWrapper{code: wrapCode, text: wrapText + errDelim + err.Error(), wrap: err},
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.wrapper.WrapErr(test.error); !reflect.DeepEqual(got, test.want) {
				t.Errorf("WrapErr() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_errIs(t *testing.T) {
	t.Parallel()

	testErr := errNew(testCode, testText)
	wrapErr := errWrap(wrapCode, wrapText, testErr)

	tests := [2]struct {
		name    string
		testErr error
		wrapErr error
		want    bool
	}{
		{
			name:    "TRUE",
			testErr: testErr,
			wrapErr: wrapErr,
			want:    true,
		},
		{
			name:    "FALSE",
			testErr: testErr,
			wrapErr: errWrap(wrapCode, wrapText, errNew(testCode, testText)),
			want:    false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := errIs(test.wrapErr, test.testErr); got != test.want {
				t.Errorf("errIs() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_errNew(t *testing.T) {
	t.Parallel()

	tests := [1]struct {
		name string
		code string
		text string
		want *errWrapper
	}{
		{
			name: "Equal",
			code: testCode,
			text: testText,
			want: &errWrapper{code: testCode, text: testText},
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := errNew(test.code, test.text); !reflect.DeepEqual(got, test.want) {
				t.Errorf("errNew() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_errWrap(t *testing.T) {
	t.Parallel()

	tests := [1]struct {
		name string
		code string
		text string
		wrap error
		want string
	}{
		{
			name: "EQUAL",
			code: wrapCode,
			text: wrapText,
			wrap: errNew(testCode, testText),
			want: wrapCode + errDelim + wrapText + errDelim + testCode + errDelim + testText,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := errWrap(test.code, test.text, test.wrap).Error(); got != test.want {
				t.Errorf("errWrap() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}
