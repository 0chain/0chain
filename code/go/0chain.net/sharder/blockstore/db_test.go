package blockstore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func BenchmarkInitMetaRecordDB(b *testing.B) {
	bmrDir, qmrDir := b.TempDir(), b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InitMetaRecordDB(bmrDir, qmrDir, true)
	}
}

func BenchmarkBlockWhereRecord_AddOrUpdate(b *testing.B) {
	InitMetaRecordDB(b.TempDir(), b.TempDir(), true)
	bwr := mockBlockWhereRecord()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := bwr.AddOrUpdate(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetBlockWhereRecord(b *testing.B) {
	InitMetaRecordDB(b.TempDir(), b.TempDir(), true)
	bwr := mockBlockWhereRecord()
	err := bwr.AddOrUpdate()
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := GetBlockWhereRecord(bwr.Hash); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeleteBlockWhereRecord(b *testing.B) {
	InitMetaRecordDB(b.TempDir(), b.TempDir(), true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		bwr := mockBlockWhereRecord()
		if err := bwr.AddOrUpdate(); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		if _, err := GetBlockWhereRecord(bwr.Hash); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmovedBlockRecord_Add(b *testing.B) {
	InitMetaRecordDB(b.TempDir(), b.TempDir(), true)
	ubr := mockUnmovedBlockRecord()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ubr.Add(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmovedBlockRecord_Delete(b *testing.B) {
	InitMetaRecordDB(b.TempDir(), b.TempDir(), true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ubr := mockUnmovedBlockRecord()
		if err := ubr.Add(); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		if err := ubr.Delete(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetUnmovedBlock(b *testing.B) {
	InitMetaRecordDB(b.TempDir(), b.TempDir(), true)
	prevUbr := mockUnmovedBlockRecord()
	if err := prevUbr.Add(); err != nil {
		b.Fatal(err)
	}
	ubr := mockUnmovedBlockRecord()
	if err := ubr.Add(); err != nil {
		b.Fatal(err)
	}
	nextUbr := mockUnmovedBlockRecord()
	if err := ubr.Add(); err != nil {
		b.Fatal(err)
	}

	key := []byte(ubr.CreatedAt.Format(time.RFC3339))
	nextKey := []byte(nextUbr.CreatedAt.Format(time.RFC3339))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := GetUnmovedBlock(key, nextKey); err != nil {
			b.Fatal(err)
		}
	}
}
