package chain

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"0chain.net/core/datastore"
	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func genMiners(n int) map[datastore.Key]struct{} {
	mp := make(map[datastore.Key]struct{}, n)
	for i := 0; i < n; i++ {
		mp[strconv.Itoa(i)] = struct{}{}
	}

	return mp
}

func makeVersionsCount(t *testing.T, vcounts map[string]int) map[datastore.Key]semver.Version {
	rand.Seed(time.Now().UnixNano())
	ret := make(map[datastore.Key]semver.Version)
	for vstr, count := range vcounts {
		v, err := semver.Make(vstr)
		require.NoError(t, err)

		// generate random miner id
		for i := 0; i < count; i++ {
			id := strconv.FormatUint(rand.Uint64(), 10)
			ret[id] = v
		}
	}

	return ret
}

func makeVersions(t *testing.T, n int) []semver.Version {
	vs := make([]semver.Version, n)
	for i := 0; i < n; i++ {
		v, err := semver.Make(fmt.Sprintf("%d.0.0", i+1))
		require.NoError(t, err)
		vs[i] = v
	}

	return vs
}

func makeVersion(t *testing.T, vstr string) *semver.Version {
	v, err := semver.New(vstr)
	require.NoError(t, err)
	return v
}

func Test_scVersions_GetConsensusVersion(t *testing.T) {
	type fields struct {
		miners           map[datastore.Key]struct{}
		versions         map[datastore.Key]semver.Version
		thresholdPercent int
	}

	tests := []struct {
		name   string
		fields fields
		want   *semver.Version
	}{
		// TODO: Add test cases.
		{
			name: "ok",
			fields: fields{
				miners: genMiners(10),
				versions: makeVersionsCount(t, map[string]int{
					"1.0.0": 10,
				}),
				thresholdPercent: 80, // 80 percent
			},
			want: makeVersion(t, "1.0.0"),
		},
		{
			name: "versions_num < threshold",
			fields: fields{
				miners: genMiners(10),
				versions: makeVersionsCount(t, map[string]int{
					"1.0.0": 5,
				}),
				thresholdPercent: 80, // 80 percent
			},
			want: nil,
		},
		{
			name: "versions_num == threshold",
			fields: fields{
				miners: genMiners(10),
				versions: makeVersionsCount(t, map[string]int{
					"1.0.0": 8,
				}),
				thresholdPercent: 80, // 80 percent
			},
			want: makeVersion(t, "1.0.0"),
		},
		{
			name: "no conensus, 50/50",
			fields: fields{
				miners: genMiners(100),
				versions: makeVersionsCount(t, map[string]int{
					"1.0.0": 50,
					"2.0.0": 50,
				}),
				thresholdPercent: 80, // 80 percent
			},
			want: nil,
		},
		{
			name: "no conensus, 30/70",
			fields: fields{
				miners: genMiners(100),
				versions: makeVersionsCount(t, map[string]int{
					"1.0.0": 30,
					"2.0.0": 70,
				}),
				thresholdPercent: 80, // 80 percent
			},
			want: nil,
		},
		{
			name: "no conensus, 21/79",
			fields: fields{
				miners: genMiners(100),
				versions: makeVersionsCount(t, map[string]int{
					"1.0.0": 21,
					"2.0.0": 79,
				}),
				thresholdPercent: 80, // 80 percent
			},
			want: nil,
		},
		{
			name: "conensus, 20/80",
			fields: fields{
				miners: genMiners(100),
				versions: makeVersionsCount(t, map[string]int{
					"1.0.0": 20,
					"2.0.0": 80,
				}),
				thresholdPercent: 80, // 80 percent
			},
			want: makeVersion(t, "2.0.0"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scv := &versionsConsensus{
				nodes:            tt.fields.miners,
				versions:         tt.fields.versions,
				thresholdPercent: tt.fields.thresholdPercent,
			}
			assert.Equalf(t, tt.want, scv.GetConsensusVersion(), "GetConsensusVersion()")
		})
	}
}

func Test_scVersions_Set(t *testing.T) {
	scv := versionsConsensus{
		nodes:    genMiners(5),
		versions: map[datastore.Key]semver.Version{},
	}

	v1, err := semver.Make("1.0.0")
	require.NoError(t, err)
	err = scv.Add("1", v1)
	require.NoError(t, err)

	v, ok := scv.versions["1"]
	require.True(t, ok)
	require.Equal(t, v1, v)

	// expect miner deos not exist error
	err = scv.Add("11", v1)
	require.EqualError(t, err, "miner_not_exist_in_mb: miner does not exist in magic block, id: 11")
}

func Test_scVersions_UpdateMinersList(t *testing.T) {
	scVersions := makeVersions(t, 10)

	type want struct {
		miners   map[datastore.Key]struct{}
		versions map[datastore.Key]semver.Version
	}

	tt := []struct {
		name            string
		scVersionCreate func(t *testing.T) *versionsConsensus
		miners          map[datastore.Key]struct{}
		want            want
	}{
		{
			name: "no changes - different versions",
			scVersionCreate: func(t *testing.T) *versionsConsensus {
				scv := newVersionsConsensus(genMiners(3), 80)
				for i := 0; i < 3; i++ {
					err := scv.Add(strconv.Itoa(i), scVersions[i])
					require.NoError(t, err)
				}

				return scv
			},
			miners: genMiners(3),
			want: want{
				miners: genMiners(3),
				versions: map[datastore.Key]semver.Version{
					"0": scVersions[0],
					"1": scVersions[1],
					"2": scVersions[2],
				},
			},
		},
		{
			name: "no changes - same versions",
			scVersionCreate: func(t *testing.T) *versionsConsensus {
				scv := newVersionsConsensus(genMiners(3), 80)
				for i := 0; i < 3; i++ {
					err := scv.Add(strconv.Itoa(i), scVersions[0])
					require.NoError(t, err)
				}

				return scv
			},
			miners: genMiners(3),
			want: want{
				miners: genMiners(3),
				versions: map[datastore.Key]semver.Version{
					"0": scVersions[0],
					"1": scVersions[0],
					"2": scVersions[0],
				},
			},
		},
		{
			name: "add new miners",
			scVersionCreate: func(t *testing.T) *versionsConsensus {
				scv := newVersionsConsensus(genMiners(3), 80)
				for i := 0; i < 3; i++ {
					err := scv.Add(strconv.Itoa(i), scVersions[0])
					require.NoError(t, err)
				}

				return scv
			},
			miners: genMiners(6),
			want: want{
				miners: genMiners(6),
				versions: map[datastore.Key]semver.Version{
					"0": scVersions[0],
					"1": scVersions[0],
					"2": scVersions[0],
				},
			},
		},
		{
			name: "remove miners",
			scVersionCreate: func(t *testing.T) *versionsConsensus {
				scv := newVersionsConsensus(genMiners(3), 80)
				for i := 0; i < 3; i++ {
					err := scv.Add(strconv.Itoa(i), scVersions[0])
					require.NoError(t, err)
				}

				return scv
			},
			miners: genMiners(2),
			want: want{
				miners: genMiners(2),
				versions: map[datastore.Key]semver.Version{
					"0": scVersions[0],
					"1": scVersions[0],
				},
			},
		},
		{
			name: "both add and remove miners",
			scVersionCreate: func(t *testing.T) *versionsConsensus {
				scv := newVersionsConsensus(genMiners(3), 80)
				for i := 0; i < 3; i++ {
					err := scv.Add(strconv.Itoa(i), scVersions[0])
					require.NoError(t, err)
				}

				return scv
			},
			miners: func(t *testing.T) map[datastore.Key]struct{} {
				ms := genMiners(6)
				// delete miner 0, 1
				delete(ms, "0")
				delete(ms, "1")

				return ms
			}(t),
			want: want{
				miners: map[datastore.Key]struct{}{
					"2": struct{}{},
					"3": struct{}{},
					"4": struct{}{},
					"5": struct{}{},
				},
				versions: map[datastore.Key]semver.Version{
					"2": scVersions[0],
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			scv := tc.scVersionCreate(t)
			scv.UpdateNodesList(tc.miners)

			require.Equal(t, len(tc.want.miners), len(scv.nodes))
			for k := range tc.want.miners {
				_, ok := scv.nodes[k]
				require.True(t, ok)
			}

			require.Equal(t, len(tc.want.versions), len(scv.versions))
			for k, v := range tc.want.versions {
				require.Equal(t, v, scv.versions[k])
			}
		})
	}
}
