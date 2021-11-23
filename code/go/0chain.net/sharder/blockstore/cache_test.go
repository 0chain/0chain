package blockstore

import (
	"reflect"
	"testing"

	"0chain.net/core/viper"
)

func Test_cacheInit(t *testing.T) {
	type args struct {
		cViper *viper.Viper
	}
	tests := []struct {
		name      string
		args      args
		want      *cacheTier
		wantPanic bool
	}{
		{
			name: "Nil_Volume_Configs_Panic",
			args: args{
				cViper: viper.New(),
			},
			wantPanic: true,
		},
		{
			name: "Unsupported_Write_Policy_Panic",
			args: args{
				cViper: mockConfig(t, map[string]interface{}{
					"write_policy": "unsupported",
					"volumes":      []interface{}{},
				}),
			},
			wantPanic: true,
		},
		{
			name: "Unsupported_Volume_Strategy_Panic",
			args: args{
				cViper: mockConfig(t, map[string]interface{}{}),
			},
			wantPanic: true,
		},
		{
			name: "OK",
			args: args{
				cViper: mockConfig(t, simpleConfigMap(t)),
			},
		}, // Is broken because of broken cacheInit and startCacheVolumes functions
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func(t *testing.T) {
				if r := recover(); (r != nil) != tt.wantPanic {
					t.Errorf("cacheInit() want panic: %v; but got %v", tt.wantPanic, r != nil)
				}
			}(t)

			if got := cacheInit(tt.args.cViper); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cacheInit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_startCacheVolumes(t *testing.T) {
	type args struct {
		mVolumes []map[string]interface{}
		chTier   *cacheTier
	}
	tests := []struct {
		name      string
		args      args
		wantPanic bool
	}{
		{
			name: "No_Path_Panic",
			args: args{
				mVolumes: []map[string]interface{}{{}},
			},
			wantPanic: true,
		},
		{
			name: "Invalid_Path_Panic",
			args: args{
				mVolumes: []map[string]interface{}{
					{
						"path": "",
					},
				},
			},
			wantPanic: true,
		},
		{
			name: "OK",
			args: args{
				mVolumes: []map[string]interface{}{
					mockVolume(t).convertToMap(),
					mockVolume(t).convertToMap(),
				},
				chTier: &cacheTier{
					Volumes: make([]*cacheVolume, 0),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func(t *testing.T) {
				if r := recover(); (r != nil) != tt.wantPanic {
					t.Errorf("tartCacheVolumes() want panic: %v; but got %v", tt.wantPanic, r != nil)
				}
			}(t)

			startCacheVolumes(tt.args.mVolumes, tt.args.chTier)
		})
	}
}
