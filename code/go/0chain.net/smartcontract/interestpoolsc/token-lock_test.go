package interestpoolsc

import (
	"reflect"
	"testing"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

func Test_tokenLock_IsLocked(t *testing.T) {
	type fields struct {
		StartTime common.Timestamp
		Duration  time.Duration
		Owner     datastore.Key
	}
	type args struct {
		entity interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "true",
			fields: fields{
				StartTime: common.Now(),
				Duration:  1 * time.Second,
				Owner:     owner,
			},
			args: args{
				entity: "Second",
			},
			want: true,
		},
		{
			name: "false",
			fields: fields{
				StartTime: common.Now(),
				Duration:  1 * time.Second,
				Owner:     owner,
			},
			args: args{
				entity: time.Now().Add(5 * time.Second),
			},
			want: false,
		},
		{
			name: "true",
			fields: fields{
				StartTime: common.Now(),
				Duration:  10 * time.Second,
				Owner:     owner,
			},
			args: args{
				entity: time.Now(),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := tokenLock{
				StartTime: tt.fields.StartTime,
				Duration:  tt.fields.Duration,
				Owner:     tt.fields.Owner,
			}
			if got := tl.IsLocked(tt.args.entity); got != tt.want {
				t.Errorf("IsLocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_tokenLock_LockStats(t *testing.T) {
	type fields struct {
		StartTime common.Timestamp
		Duration  time.Duration
		Owner     datastore.Key
	}
	type args struct {
		entity interface{}
	}

	var timeNow = time.Now()
	var commonNow = common.Now()

	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		{
			name:   "nil",
			fields: fields{},
			args:   args{entity: "Any type of entity"},
			want:   nil,
		},
		{
			name: "locked true",
			fields: fields{
				StartTime: commonNow,
				Duration:  5 * time.Second,
				Owner:     owner,
			},
			args: args{entity: timeNow},
			want: (&poolStat{
				StartTime: commonNow,
				Duartion:  5 * time.Second,
				TimeLeft:  (5*time.Second - timeNow.Sub(common.ToTime(commonNow))),
				Locked:    true,
			}).encode(),
		},
		{
			name: "locked false",
			fields: fields{
				StartTime: commonNow,
				Duration:  5 * time.Second,
				Owner:     owner,
			},
			args: args{entity: timeNow.Add(50 * time.Second)},
			want: (&poolStat{
				StartTime: commonNow,
				Duartion:  5 * time.Second,
				TimeLeft:  (5*time.Second - timeNow.Add(50*time.Second).Sub(common.ToTime(commonNow))),
				Locked:    false,
			}).encode(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := tokenLock{
				StartTime: tt.fields.StartTime,
				Duration:  tt.fields.Duration,
				Owner:     tt.fields.Owner,
			}
			if got := tl.LockStats(tt.args.entity); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LockStats() = %v, want %v", got, tt.want)
			}
		})
	}
}
