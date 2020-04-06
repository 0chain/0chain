package round

import (
	"github.com/stretchr/testify/assert"
	"testing"
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

		storage.Put(entity0, 0)
		storage.Put(entity3, 151)
		storage.Put(entity1, 5)
		storage.Put(entity4, 251)
		storage.Put(entity2, 51)
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
		storage.Put(entity0, 0)
		storage.Put(entity3, 151)
		storage.Put(entity1, 5)
		storage.Put(entity4, 251)
		storage.Put(entity2, 51)

		err := storage.Prune(150)
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
