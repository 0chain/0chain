package viper_test

import (
	"os"
	paths "path"
	"strconv"
	"sync"
	"testing"
	"time"

	"0chain.net/core/viper"
)

func Test_Viper_IsThreadSafe(t *testing.T) {
	t.Parallel()

	type (
		testCase struct {
			name    string
			format  string
			threads int
			viper   *viper.Viper
		}
		testList []testCase
	)

	threads := 5
	path := os.TempDir()
	tests := make(testList, len(viper.SupportedExts))

	for _, format := range viper.SupportedExts {
		tests = append(tests, testCase{
			name:    format + "_VIPER_IS_THREAD_SAFE_OK",
			format:  format,
			threads: threads,
			viper:   viper.New(),
		})
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var wg sync.WaitGroup
			for i := 0; i < test.threads; i++ {
				wg.Add(1)

				go func(i int) {
					defer wg.Done()

					flag := i != 0
					iter := strconv.Itoa(i)
					duration := time.Duration(i) * time.Second
					timestamp := time.Now()
					filepath := paths.Join(path, "viper_test_"+iter+"."+test.format)

					test.viper.Set("test.Bool", flag)
					test.viper.Set("test.Duration", duration)
					test.viper.Set("test.Float64", float64(i))
					test.viper.Set("test.Int", i)
					test.viper.Set("test.String", iter)
					test.viper.Set("test.Time", timestamp)
					test.viper.Set("test.Cost", map[string]string{"1": "1", "2": "1", "3": "1", "4": "1"})

					if err := test.viper.WriteConfigFile(filepath); err != nil {
						t.Errorf("can't write file path: '%s'", filepath)
					}
					defer func() { _ = os.Remove(filepath) }()

					if err := test.viper.ReadConfigFile(filepath); err != nil {
						t.Errorf("can't read file path: '%s'", filepath)
					}

					_ = test.viper.GetBool("test.Bool")
					_ = test.viper.GetDuration("test.Duration")
					_ = test.viper.GetFloat64("test.Float64")
					_ = test.viper.GetInt("test.Int")
					_ = test.viper.GetString("test.String")
					_ = test.viper.GetTime("test.Time")
				}(i)
			}
			wg.Wait()
		})
	}
}
