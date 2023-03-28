package round

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundStartingStore(t *testing.T) {

	type checkEntity struct {
		Name string
	}
	entity0 := &checkEntity{Name: "0"}
	entity1 := &checkEntity{Name: "1"}
	entity2 := &checkEntity{Name: "2"}
	entity3 := &checkEntity{Name: "3"}
	entity4 := &checkEntity{Name: "4"}

	t.Run("put", func(t *testing.T) {
		storage := NewRoundStartingStorage()
		assert.Equal(t, 0, storage.Count())
		assert.Empty(t, storage.GetRounds())

		//Check put
		err := storage.Put(entity1, 5)
		assert.NoError(t, err)

		err = storage.Put(entity1, 5)
		assert.NoError(t, err)

		err = storage.Put(entity2, 151)
		assert.NoError(t, err)
		assert.EqualValues(t, []int64{5, 151}, storage.GetRounds())

		err = storage.Put(entity3, 51)
		assert.NoError(t, err)

		err = storage.Put(entity4, 251)
		assert.NoError(t, err)

		assert.Equal(t, 4, storage.Count())
		assert.EqualValues(t, []int64{5, 51, 151, 251}, storage.GetRounds())
	})

	t.Run("putCheckOrder", func(t *testing.T) {
		storage := NewRoundStartingStorage()
		assert.Equal(t, 0, storage.Count())
		assert.Empty(t, storage.GetRounds())

		//Check put
		err := storage.Put(entity1, 1001)
		assert.NoError(t, err)

		err = storage.Put(entity2, 501)
		assert.NoError(t, err)

		err = storage.Put(entity3, 2001)
		assert.NoError(t, err)

		assert.Equal(t, 3, storage.Count())
		assert.EqualValues(t, []int64{501, 1001, 2001}, storage.GetRounds())
		assert.Equal(t, int64(501), storage.GetRounds()[0])
		assert.Equal(t, int64(1001), storage.GetRounds()[1])
		assert.Equal(t, int64(2001), storage.GetRounds()[2])

	})

	t.Run("CountAndRounds", func(t *testing.T) {
		storage := NewRoundStartingStorage()
		assert.Equal(t, 0, storage.Count())
		assert.Empty(t, storage.GetRounds())
		want := make([]int64, 0)
		for i := 1; i <= 10; i++ {
			err := storage.Put(entity1, int64(i*10))
			want = append(want, int64(i*10))
			assert.NoError(t, err)
		}
		assert.Equal(t, 10, storage.Count())
		assert.EqualValues(t, want, storage.GetRounds())

		for i := 0; i < 10; i++ {
			got := storage.GetRound(i)
			assert.Equal(t, want[i], got)
		}
	})

	t.Run("Get", func(t *testing.T) {
		storage := NewRoundStartingStorage()
		assert.Nil(t, storage.GetLatest())
		assert.Nil(t, storage.Get(0))
		assert.Nil(t, storage.Get(10))

		err := storage.Put(entity0, 0)
		require.NoError(t, err)
		err = storage.Put(entity3, 151)
		require.NoError(t, err)
		err = storage.Put(entity1, 5)
		require.NoError(t, err)
		err = storage.Put(entity4, 251)
		require.NoError(t, err)
		err = storage.Put(entity2, 51)
		require.NoError(t, err)
		assert.Equal(t, entity4, storage.GetLatest())

		// 0,5,51,151,251
		assert.Nil(t, storage.Get(-1))
		assert.Equal(t, entity0, storage.Get(0))
		assert.Equal(t, entity0, storage.Get(2))
		assert.Equal(t, entity1, storage.Get(5))
		assert.Equal(t, entity1, storage.Get(50))
		assert.Equal(t, entity2, storage.Get(51))
		assert.Equal(t, entity2, storage.Get(52))
		assert.Equal(t, entity2, storage.Get(150))
		assert.Equal(t, entity3, storage.Get(151))
		assert.Equal(t, entity3, storage.Get(250))
		assert.Equal(t, entity4, storage.Get(251))
		assert.Equal(t, entity4, storage.Get(252))
		assert.Equal(t, entity4, storage.Get(1000))
		assert.Equal(t, entity4, storage.Get(100000000))
	})

	t.Run("Prune", func(t *testing.T) {
		storage := NewRoundStartingStorage()
		err := storage.Put(entity0, 0)
		require.NoError(t, err)
		err = storage.Put(entity3, 151)
		require.NoError(t, err)
		err = storage.Put(entity1, 5)
		require.NoError(t, err)
		err = storage.Put(entity4, 251)
		require.NoError(t, err)
		err = storage.Put(entity2, 51)
		require.NoError(t, err)

		err = storage.Prune(150)
		assert.EqualError(t, err, ErrRoundEntityNotFound.Error())

		err = storage.Prune(0)
		assert.NoError(t, err)
		assert.Equal(t, 4, storage.Count())
		assert.EqualValues(t, []int64{5, 51, 151, 251}, storage.GetRounds())

		err = storage.Prune(0)
		assert.EqualError(t, err, ErrRoundEntityNotFound.Error())

		got := storage.Get(0)
		assert.Nil(t, got)

		got = storage.Get(5)
		assert.NotNil(t, got)

		err = storage.Prune(151)
		assert.NoError(t, err)
		assert.Equal(t, 1, storage.Count())
		assert.EqualValues(t, []int64{251}, storage.GetRounds())

		err = storage.Prune(251)
		assert.NoError(t, err)
		assert.Equal(t, 0, storage.Count())
		assert.Empty(t, storage.GetRounds())

	})
}

func Test_roundStartingStorage_FindRoundIndex(t *testing.T) {
	t.Parallel()

	type fields struct {
		max    int64
		items  map[int64]RoundStorageEntity
		rounds []int64
	}
	type args struct {
		round int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "Round_Greater_Than_Max_OK",
			fields: fields{
				max:    1,
				rounds: make([]int64, 5),
			},
			args: args{round: 5},
			want: 4,
		},
		{
			name: "OK",
			fields: fields{
				max: 3,
				rounds: []int64{
					2,
					3,
				},
			},
			args: args{round: 2},
			want: 0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &roundStartingStorage{
				mu:     &sync.RWMutex{},
				max:    tt.fields.max,
				items:  tt.fields.items,
				rounds: tt.fields.rounds,
			}
			if got := s.FindRoundIndex(tt.args.round); got != tt.want {
				t.Errorf("FindRoundIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}
