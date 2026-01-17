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
		assert.EqualValues(t, []int64{5, 151}, storage.GetRounds(), "rounds should be in ascending order")

		err = storage.Put(entity3, 51)
		assert.NoError(t, err)

		err = storage.Put(entity4, 251)
		assert.NoError(t, err)

		assert.Equal(t, 4, storage.Count())
		assert.EqualValues(t, []int64{5, 51, 151, 251}, storage.GetRounds(), "rounds should be in ascending order")
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
		assert.EqualValues(t, []int64{501, 1001, 2001}, storage.GetRounds(), "rounds should be in ascending order")
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
			want = append(want, int64(i*10)) // Append to maintain ascending order
			assert.NoError(t, err)
		}
		assert.Equal(t, 10, storage.Count())
		assert.EqualValues(t, want, storage.GetRounds(), "rounds should be in ascending order")

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

		// Verify rounds are in ascending order: 0,5,51,151,251
		assert.EqualValues(t, []int64{0, 5, 51, 151, 251}, storage.GetRounds(), "rounds should be in ascending order")

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

		// Verify initial ascending order: 0,5,51,151,251
		assert.EqualValues(t, []int64{0, 5, 51, 151, 251}, storage.GetRounds(), "rounds should be in ascending order")

		err = storage.Prune(150)
		assert.EqualError(t, err, ErrRoundEntityNotFound.Error())

		err = storage.Prune(0)
		assert.NoError(t, err)
		assert.Equal(t, 4, storage.Count())
		assert.EqualValues(t, []int64{5, 51, 151, 251}, storage.GetRounds(), "rounds should maintain ascending order after prune")

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

	t.Run("Prune with ascending order", func(t *testing.T) {
		storage := NewRoundStartingStorage()
		// Insert in random order but should maintain ascending order
		err := storage.Put(entity2, 80)
		require.NoError(t, err)
		err = storage.Put(entity4, 100)
		require.NoError(t, err)
		err = storage.Put(entity1, 90)
		require.NoError(t, err)
		err = storage.Put(entity3, 70)
		require.NoError(t, err)

		// Verify ascending order
		assert.EqualValues(t, []int64{70, 80, 90, 100}, storage.GetRounds())

		// Prune at 90 should remove 70, 80, and 90
		err = storage.Prune(90)
		assert.NoError(t, err)
		assert.Equal(t, 1, storage.Count())
		assert.EqualValues(t, []int64{100}, storage.GetRounds())

		// Verify items were removed correctly
		assert.NotNil(t, storage.Get(100))
		assert.Nil(t, storage.Get(90))
		assert.Nil(t, storage.Get(80))
		assert.Nil(t, storage.Get(70))

		// Test pruning the last remaining round
		err = storage.Prune(100)
		assert.NoError(t, err)
		assert.Equal(t, 0, storage.Count())
		assert.Empty(t, storage.GetRounds())
		assert.Nil(t, storage.Get(100))
	})
}

func Test_roundStartingStorage_FindRoundIndex(t *testing.T) {
	t.Run("empty rounds", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{},
			max:    0,
			mu:     &sync.RWMutex{},
		}
		got := s.FindRoundIndex(10)
		require.Equal(t, -1, got)
	})

	t.Run("round greater than max", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{80, 90, 100},
			max:    100,
			mu:     &sync.RWMutex{},
		}
		got := s.FindRoundIndex(150)
		require.Equal(t, 2, got)
	})

	t.Run("exact round match", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{80, 90, 100},
			max:    100,
			mu:     &sync.RWMutex{},
		}
		got := s.FindRoundIndex(90)
		require.Equal(t, 1, got)
	})

	t.Run("round between values", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{80, 90, 100},
			max:    100,
			mu:     &sync.RWMutex{},
		}
		got := s.FindRoundIndex(85)
		require.Equal(t, 0, got)
	})

	t.Run("round lower than all values", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{80, 90, 100},
			max:    100,
			mu:     &sync.RWMutex{},
		}
		got := s.FindRoundIndex(70)
		require.Equal(t, -1, got)
	})

	t.Run("with ascending order multiple cases", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{70, 80, 90, 100},
			max:    100,
			mu:     &sync.RWMutex{},
		}
		testCases := []struct {
			input    int64
			expected int
		}{
			{95, 2},  // should return index of 90
			{90, 2},  // exact match with 90
			{89, 1},  // should return index of 80
			{75, 0},  // should return index of 70
			{70, 0},  // exact match with 70
			{60, -1}, // lower than all values
		}
		for _, tc := range testCases {
			got := s.FindRoundIndex(tc.input)
			require.Equal(t, tc.expected, got, "failed for input %d", tc.input)
		}
	})
}

func Test_roundStartingStorage_putToSlice(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{},
			mu:     &sync.RWMutex{},
		}
		s.putToSlice(100)
		require.Equal(t, []int64{100}, s.rounds)
	})

	t.Run("insert at beginning - smallest number", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{80, 90, 100},
			mu:     &sync.RWMutex{},
		}
		s.putToSlice(70)
		require.Equal(t, []int64{70, 80, 90, 100}, s.rounds)
	})

	t.Run("insert in middle", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{70, 80, 100},
			mu:     &sync.RWMutex{},
		}
		s.putToSlice(90)
		require.Equal(t, []int64{70, 80, 90, 100}, s.rounds)
	})

	t.Run("insert at end - largest number", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{70, 80, 90},
			mu:     &sync.RWMutex{},
		}
		s.putToSlice(100)
		require.Equal(t, []int64{70, 80, 90, 100}, s.rounds)
	})

	t.Run("insert with duplicate values", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{70, 80, 100},
			mu:     &sync.RWMutex{},
		}
		s.putToSlice(80)
		require.Equal(t, []int64{70, 80, 80, 100}, s.rounds)
	})

	t.Run("multiple inserts maintain order", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{},
			mu:     &sync.RWMutex{},
		}
		rounds := []int64{50, 100, 75, 25, 150}
		for _, r := range rounds {
			s.putToSlice(r)
		}
		require.Equal(t, []int64{25, 50, 75, 100, 150}, s.rounds)
	})
}

func Test_roundStartingStorage_calcNearestRound(t *testing.T) {
	t.Run("empty rounds", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{},
			max:    0,
			mu:     &sync.RWMutex{},
		}
		got := s.calcNearestRound(10)
		require.Equal(t, int64(-1), got)
	})

	t.Run("round greater than max", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{80, 90, 100},
			max:    100,
			mu:     &sync.RWMutex{},
		}
		got := s.calcNearestRound(150)
		require.Equal(t, int64(100), got)
	})

	t.Run("exact round match", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{80, 90, 100},
			max:    100,
			mu:     &sync.RWMutex{},
		}
		got := s.calcNearestRound(90)
		require.Equal(t, int64(90), got)
	})

	t.Run("round between values", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{80, 90, 100},
			max:    100,
			mu:     &sync.RWMutex{},
		}
		got := s.calcNearestRound(85)
		require.Equal(t, int64(80), got, "should return 80 as it's the nearest lower round")
	})

	t.Run("round lower than all values", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{80, 90, 100},
			max:    100,
			mu:     &sync.RWMutex{},
		}
		got := s.calcNearestRound(70)
		require.Equal(t, int64(-1), got, "should return -1 as there's no lower round")
	})

	t.Run("with ascending order", func(t *testing.T) {
		s := &roundStartingStorage{
			rounds: []int64{70, 80, 90, 100},
			max:    100,
			mu:     &sync.RWMutex{},
		}
		testCases := []struct {
			input    int64
			expected int64
		}{
			{95, 90},
			{90, 90},
			{89, 80},
			{75, 70},
			{70, 70},
			{60, -1},
		}
		for _, tc := range testCases {
			got := s.calcNearestRound(tc.input)
			require.Equal(t, tc.expected, got, "failed for input %d", tc.input)
		}
	})
}
