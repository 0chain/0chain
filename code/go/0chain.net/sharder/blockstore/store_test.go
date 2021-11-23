package blockstore

import (
	"context"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/core/viper"
	"github.com/stretchr/testify/require"
)

func BenchmarkBlockStore_Write(b *testing.B) {
	err := InitializeStore(mockHotConfig(b), context.Background())
	require.NoError(b, err)

	bl := mockBlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := Store.Write(bl); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBlockStore_ReadWithBlockSummary(b *testing.B) {
	err := InitializeStore(mockHotConfig(b), context.Background())
	require.NoError(b, err)

	bl := mockBlock()
	err = Store.Write(bl)
	require.NoError(b, err)

	bs := block.BlockSummary{Round: bl.Round, Hash: bl.Hash}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Store.ReadWithBlockSummary(&bs); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBlockStore_Read(b *testing.B) {
	err := InitializeStore(mockHotConfig(b), context.Background())
	require.NoError(b, err)

	bl := mockBlock()
	err = Store.Write(bl)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Store.Read(bl.Hash, bl.Round); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInitializeStore(b *testing.B) {
	type (
		tests struct {
			name string
			cfg  *viper.Viper
		}
	)
	testsList := []tests{
		{
			name: "Hot_Only",
			cfg:  mockHotConfig(b),
		},
		{
			name: "Warm_Only",
			cfg: mockConfig(
				b,
				map[string]interface{}{
					"warm":            simpleConfigMap(b),
					storageTypeCfgKey: int(WarmOnly),
				},
			),
		},
		// TODO cache and cold tiers are broken, so any combinations with that tiers are not working
		//{
		//	name: "Cache_And_Warm",
		//	cfg: mockConfig(
		//		b,
		//		map[string]interface{}{
		//			"cache": simpleConfigMap(b),
		//			"warm": simpleConfigMap(b),
		//		},
		//	),
		//	tiering: CacheAndWarm,
		//},
		{
			name: "Hot_And_Cold",
			cfg:  mockHotAndColdConfig(b),
		},
		{
			name: "Warm_And_Cold",
			cfg: mockConfig(
				b,
				map[string]interface{}{
					"warm": simpleConfigMap(b),
					"cold": map[string]interface{}{
						"storage": map[string]interface{}{
							"type": "disk",
							"disk": map[string]interface{}{
								"strategy": RoundRobin,
								"volumes":  mockVolumes(b, 2),
							},
						},
					},
					storageTypeCfgKey: int(WarmAndCold),
				},
			),
		},
	}

	for _, test := range testsList {
		b.Run(test.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := InitializeStore(test.cfg, context.Background()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func Benchmark_getBlockData(b *testing.B) {
	bl := mockBlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := getBlockData(bl); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_readFromDiskTier(b *testing.B) {
	err := InitializeStore(mockHotConfig(b), context.Background())
	require.NoError(b, err)

	bl := mockBlock()
	err = Store.Write(bl)
	require.NoError(b, err)

	bwr, err := GetBlockWhereRecord(bl.Hash)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := readFromDiskTier(bwr, false); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_readFromColdTier(b *testing.B) {
	err := InitializeStore(mockHotAndColdConfig(b), context.Background())
	require.NoError(b, err)

	bl := mockBlock()
	err = Store.Write(bl)
	require.NoError(b, err)

	bwr, err := GetBlockWhereRecord(bl.Hash)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := readFromColdTier(bwr, false); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetStore(b *testing.B) {
	err := InitializeStore(mockHotConfig(b), context.Background())
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetStore()
	}
}
