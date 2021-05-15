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

	path := os.TempDir()
	viper.AddConfigPath(path)
	viper.SetConfigType("yaml")

	tests := []struct {
		name  string
		iters int
		viper *viper.Viper
	}{
		{
			name:  "Viper is thread safe",
			iters: 10,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var wg sync.WaitGroup

			for i := 0; i < test.iters; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()

					viper.Set("test.Bool", false)
					viper.Set("test.Duration", time.Second)
					viper.Set("test.Float64", 1.00)
					viper.Set("test.Int", 1)
					viper.Set("test.String", "string")
					viper.Set("test.Time", time.Now())

					_ = viper.GetBool("test.Bool")
					_ = viper.GetDuration("test.Duration")
					_ = viper.GetFloat64("test.Float64")
					_ = viper.GetInt("test.Int")
					_ = viper.GetString("test.String")
					_ = viper.GetTime("test.Time")

					filename := "viper_test_" + strconv.Itoa(i)
					filepath := paths.Join(path, filename+".yaml")

					if err := viper.WriteConfigAs(filepath); err != nil {
						t.Errorf("can't write '%s.yaml' in path: '%s'", filename, path)
					}
					defer func() { _ = os.Remove(filepath) }()

					file, err := os.Open(filepath)
					if err != nil {
						t.Errorf("can't open '%s.yaml' in path: '%s'", filename, path)
					}
					defer func() { _ = file.Close() }()

					if err = viper.ReadConfig(file); err != nil {
						t.Errorf("can't read '%s.yaml' in path: '%s'", filename, path)
					}
				}(i)
			}
			wg.Wait()
		})
	}
}
