package encryption

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

// setupBLS0ChainData prepares signature schemes and test data for benchmarking
func setupBLS0ChainData(n int) ([]SignatureScheme, []string, []string) {
	schemes := make([]SignatureScheme, n)
	signatures := make([]string, n)
	hashes := make([]string, n)

	for i := 0; i < n; i++ {
		// Get BLS signature scheme
		schemes[i] = GetSignatureScheme("bls0chain")

		// Generate keys for the scheme
		err := schemes[i].GenerateKeys()
		if err != nil {
			panic(fmt.Sprintf("Failed to generate keys: %v", err))
		}

		// Create unique message and hash
		message := fmt.Sprintf("benchmark-message-%d", i)
		hashes[i] = Hash(message)

		// Sign the hash
		sig, err := schemes[i].Sign(hashes[i])
		if err != nil {
			panic(fmt.Sprintf("Failed to sign: %v", err))
		}
		signatures[i] = sig
	}

	return schemes, signatures, hashes
}

// BenchmarkBLS0ChainAggregateAndVerify benchmarks the process of aggregating and verifying signatures
func BenchmarkBLS0ChainAggregateAndVerify(b *testing.B) {
	// Print CPU count information
	numCPU := runtime.NumCPU()
	b.Logf("Running benchmark with %d CPUs available", numCPU)

	// Setup test data outside benchmark timing
	n := 100
	schemes, signatures, hashes := setupBLS0ChainData(n)

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create aggregate signature scheme
		aggScheme := GetAggregateSignatureScheme("bls0chain", n, n)

		// Aggregate all signatures
		for j := 0; j < n; j++ {
			err := aggScheme.Aggregate(schemes[j], j, signatures[j], hashes[j])
			if err != nil {
				b.Fatalf("Failed to aggregate signature %d: %v", j, err)
			}
		}

		// Verify the aggregated signature
		valid, err := aggScheme.Verify()
		if err != nil {
			b.Fatalf("Failed to verify aggregated signature: %v", err)
		}
		if !valid {
			b.Fatalf("Aggregated signature verification failed")
		}
	}
}

// BenchmarkBLS0ChainAggregationOnly benchmarks just the aggregation operation
func BenchmarkBLS0ChainAggregationOnly(b *testing.B) {
	// Setup test data outside benchmark timing
	n := 100
	schemes, signatures, hashes := setupBLS0ChainData(n)

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create aggregate signature scheme
		aggScheme := GetAggregateSignatureScheme("bls0chain", n, n)

		// Aggregate all signatures
		for j := 0; j < n; j++ {
			err := aggScheme.Aggregate(schemes[j], j, signatures[j], hashes[j])
			if err != nil {
				b.Fatalf("Failed to aggregate signature %d: %v", j, err)
			}
		}
	}
}

// BenchmarkBLS0ChainVerificationOnly benchmarks just the verification operation
func BenchmarkBLS0ChainVerificationOnly(b *testing.B) {
	// Setup test data outside benchmark timing
	n := 100
	schemes, signatures, hashes := setupBLS0ChainData(n)

	// Create and populate the aggregate signature scheme
	aggScheme := GetAggregateSignatureScheme("bls0chain", n, n)
	for j := 0; j < n; j++ {
		err := aggScheme.Aggregate(schemes[j], j, signatures[j], hashes[j])
		if err != nil {
			b.Fatalf("Failed to aggregate signature %d: %v", j, err)
		}
	}

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Verify the aggregated signature
		valid, err := aggScheme.Verify()
		if err != nil {
			b.Fatalf("Failed to verify aggregated signature: %v", err)
		}
		if !valid {
			b.Fatalf("Aggregated signature verification failed")
		}
	}
}

// BenchmarkBLS0ChainScaling benchmarks with different numbers of signatures
func BenchmarkBLS0ChainScaling(b *testing.B) {
	sizes := []int{10, 50, 100, 200}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size-%d", size), func(b *testing.B) {
			// Setup test data outside benchmark timing
			schemes, signatures, hashes := setupBLS0ChainData(size)

			// Reset timer to exclude setup time
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Create aggregate signature scheme
				aggScheme := GetAggregateSignatureScheme("bls0chain", size, size)

				// Aggregate all signatures
				for j := 0; j < size; j++ {
					err := aggScheme.Aggregate(schemes[j], j, signatures[j], hashes[j])
					if err != nil {
						b.Fatalf("Failed to aggregate signature %d: %v", j, err)
					}
				}

				// Verify the aggregated signature
				valid, err := aggScheme.Verify()
				if err != nil {
					b.Fatalf("Failed to verify aggregated signature: %v", err)
				}
				if !valid {
					b.Fatalf("Aggregated signature verification failed")
				}
			}
		})
	}
}

// BenchmarkBLS0ChainIndividualVerification benchmarks verifying each signature individually without aggregation
func BenchmarkBLS0ChainIndividualVerification(b *testing.B) {
	// Setup test data outside benchmark timing
	n := 100
	schemes, signatures, hashes := setupBLS0ChainData(n)

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Verify each signature individually
		for j := 0; j < n; j++ {
			valid, err := schemes[j].Verify(signatures[j], hashes[j])
			if err != nil {
				b.Fatalf("Failed to verify individual signature %d: %v", j, err)
			}
			if !valid {
				b.Fatalf("Individual signature %d verification failed", j)
			}
		}
	}
}

// BenchmarkBLS0ChainConcurrentVerification benchmarks verifying signatures individually using goroutines
func BenchmarkBLS0ChainConcurrentVerification(b *testing.B) {
	// Print CPU count information
	numCPU := runtime.NumCPU()
	b.Logf("Running benchmark with %d CPUs available", numCPU)

	// Setup test data outside benchmark timing
	n := 100
	schemes, signatures, hashes := setupBLS0ChainData(n)

	// Get number of available CPUs
	numCPU = runtime.NumCPU()

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		errChan := make(chan error, n)
		failChan := make(chan int, n)

		// Create a task channel
		tasks := make(chan int, n)

		// Launch worker goroutines (limited by CPU count)
		for w := 0; w < numCPU; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Process tasks from the channel
				for idx := range tasks {
					valid, err := schemes[idx].Verify(signatures[idx], hashes[idx])
					if err != nil {
						errChan <- fmt.Errorf("failed to verify signature %d: %v", idx, err)
						continue
					}
					if !valid {
						failChan <- idx
					}
				}
			}()
		}

		// Send tasks to workers
		for j := 0; j < n; j++ {
			tasks <- j
		}
		close(tasks)

		// Wait for all verifications to complete
		wg.Wait()
		close(errChan)
		close(failChan)

		// Check for any errors
		for err := range errChan {
			b.Fatalf("%v", err)
		}

		// Check for any failed verifications
		for idx := range failChan {
			b.Fatalf("Signature %d verification failed", idx)
		}
	}
}

// BenchmarkBLS0ChainComparison allows direct comparison between approaches
func BenchmarkBLS0ChainComparison(b *testing.B) {
	// Print CPU count information
	numCPU := runtime.NumCPU()
	b.Logf("Running benchmark with %d CPUs available", numCPU)

	b.Run("Individual-10", func(b *testing.B) {
		// Setup test data outside benchmark timing
		n := 10
		schemes, signatures, hashes := setupBLS0ChainData(n)

		// Reset timer to exclude setup time
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Verify each signature individually
			for j := 0; j < n; j++ {
				valid, err := schemes[j].Verify(signatures[j], hashes[j])
				if err != nil {
					b.Fatalf("Failed to verify individual signature %d: %v", j, err)
				}
				if !valid {
					b.Fatalf("Individual signature %d verification failed", j)
				}
			}
		}
	})

	b.Run("Aggregated-10", func(b *testing.B) {
		// Setup test data outside benchmark timing
		n := 10
		schemes, signatures, hashes := setupBLS0ChainData(n)

		// Reset timer to exclude setup time
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Create aggregate signature scheme
			aggScheme := GetAggregateSignatureScheme("bls0chain", n, n)

			// Aggregate all signatures
			for j := 0; j < n; j++ {
				err := aggScheme.Aggregate(schemes[j], j, signatures[j], hashes[j])
				if err != nil {
					b.Fatalf("Failed to aggregate signature %d: %v", j, err)
				}
			}

			// Verify the aggregated signature
			valid, err := aggScheme.Verify()
			if err != nil {
				b.Fatalf("Failed to verify aggregated signature: %v", err)
			}
			if !valid {
				b.Fatalf("Aggregated signature verification failed")
			}
		}
	})

	b.Run("Concurrent-10", func(b *testing.B) {
		// Setup test data outside benchmark timing
		n := 10
		schemes, signatures, hashes := setupBLS0ChainData(n)

		// Get number of available CPUs
		numCPU := runtime.NumCPU()
		// For small n, use at most n workers
		workerCount := numCPU
		if n < numCPU {
			workerCount = n
		}

		// Reset timer to exclude setup time
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			errChan := make(chan error, n)
			failChan := make(chan int, n)

			// Create a task channel
			tasks := make(chan int, n)

			// Launch worker goroutines (limited by CPU count)
			for w := 0; w < workerCount; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					// Process tasks from the channel
					for idx := range tasks {
						valid, err := schemes[idx].Verify(signatures[idx], hashes[idx])
						if err != nil {
							errChan <- fmt.Errorf("failed to verify signature %d: %v", idx, err)
							continue
						}
						if !valid {
							failChan <- idx
						}
					}
				}()
			}

			// Send tasks to workers
			for j := 0; j < n; j++ {
				tasks <- j
			}
			close(tasks)

			// Wait for all verifications to complete
			wg.Wait()
			close(errChan)
			close(failChan)

			// Check for any errors
			for err := range errChan {
				b.Fatalf("%v", err)
			}

			// Check for any failed verifications
			for idx := range failChan {
				b.Fatalf("Signature %d verification failed", idx)
			}
		}
	})
}

// BenchmarkBLS0ChainFullComparison benchmarks different approaches at multiple scales
func BenchmarkBLS0ChainFullComparison(b *testing.B) {
	// Print CPU count information
	numCPU := runtime.NumCPU()
	b.Logf("Running benchmark with %d CPUs available", numCPU)

	testSizes := []int{10, 50, 100, 200}

	for _, n := range testSizes {
		// Individual sequential verification
		b.Run(fmt.Sprintf("Sequential-%d", n), func(b *testing.B) {
			schemes, signatures, hashes := setupBLS0ChainData(n)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				for j := 0; j < n; j++ {
					valid, err := schemes[j].Verify(signatures[j], hashes[j])
					if err != nil {
						b.Fatalf("Failed to verify signature %d: %v", j, err)
					}
					if !valid {
						b.Fatalf("Signature %d verification failed", j)
					}
				}
			}
		})

		// Individual concurrent verification
		b.Run(fmt.Sprintf("Concurrent-%d", n), func(b *testing.B) {
			schemes, signatures, hashes := setupBLS0ChainData(n)

			// Get number of available CPUs
			numCPU := runtime.NumCPU()
			// For small n, use at most n workers
			workerCount := numCPU
			if n < numCPU {
				workerCount = n
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				errChan := make(chan error, n)
				failChan := make(chan int, n)

				// Create a task channel
				tasks := make(chan int, n)

				// Launch worker goroutines (limited by CPU count)
				for w := 0; w < workerCount; w++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						// Process tasks from the channel
						for idx := range tasks {
							valid, err := schemes[idx].Verify(signatures[idx], hashes[idx])
							if err != nil {
								errChan <- fmt.Errorf("failed to verify signature %d: %v", idx, err)
								continue
							}
							if !valid {
								failChan <- idx
							}
						}
					}()
				}

				// Send tasks to workers
				for j := 0; j < n; j++ {
					tasks <- j
				}
				close(tasks)

				// Wait for all verifications to complete
				wg.Wait()
				close(errChan)
				close(failChan)

				// Check for any errors
				for err := range errChan {
					b.Fatalf("%v", err)
				}

				// Check for any failed verifications
				for idx := range failChan {
					b.Fatalf("Signature %d verification failed", idx)
				}
			}
		})

		// Aggregated verification
		b.Run(fmt.Sprintf("Aggregated-%d", n), func(b *testing.B) {
			schemes, signatures, hashes := setupBLS0ChainData(n)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Create aggregate signature scheme
				aggScheme := GetAggregateSignatureScheme("bls0chain", n, n)

				// Aggregate all signatures
				for j := 0; j < n; j++ {
					err := aggScheme.Aggregate(schemes[j], j, signatures[j], hashes[j])
					if err != nil {
						b.Fatalf("Failed to aggregate signature %d: %v", j, err)
					}
				}

				// Verify the aggregated signature
				valid, err := aggScheme.Verify()
				if err != nil {
					b.Fatalf("Failed to verify aggregated signature: %v", err)
				}
				if !valid {
					b.Fatalf("Aggregated signature verification failed")
				}
			}
		})
	}
}

// BenchmarkBLS0ChainChunkedAggregation benchmarks chunking signatures into groups,
// aggregating each group, and then verifying these aggregated signatures concurrently
func BenchmarkBLS0ChainChunkedAggregation(b *testing.B) {
	// Print CPU count information
	numCPU := runtime.NumCPU()
	b.Logf("Running benchmark with %d CPUs available", numCPU)

	// Setup test data outside benchmark timing
	n := 100
	chunkSize := 10 // Each worker handles 10 signatures

	schemes, signatures, hashes := setupBLS0ChainData(n)

	// Get number of chunks needed
	numChunks := (n + chunkSize - 1) / chunkSize // Ceiling division

	// Get optimal worker count (based on CPU count)
	numCPU = runtime.NumCPU()
	workerCount := numCPU
	if numChunks < numCPU {
		workerCount = numChunks
	}

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		errChan := make(chan error, numChunks)
		failChan := make(chan int, numChunks)

		// Create a task channel for chunks
		tasks := make(chan int, numChunks)

		// Launch worker goroutines (limited by CPU count)
		for w := 0; w < workerCount; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Process chunks from the task channel
				for chunkIndex := range tasks {
					// Calculate start and end indices for this chunk
					startIdx := chunkIndex * chunkSize
					endIdx := startIdx + chunkSize
					if endIdx > n {
						endIdx = n
					}
					chunkSize := endIdx - startIdx

					// Create aggregation scheme for this chunk
					aggScheme := GetAggregateSignatureScheme("bls0chain", chunkSize, chunkSize)

					// Aggregate signatures in this chunk
					for j := startIdx; j < endIdx; j++ {
						err := aggScheme.Aggregate(schemes[j], j-startIdx, signatures[j], hashes[j])
						if err != nil {
							errChan <- fmt.Errorf("failed to aggregate signature %d in chunk %d: %v", j, chunkIndex, err)
							return
						}
					}

					// Verify the aggregated signature for this chunk
					valid, err := aggScheme.Verify()
					if err != nil {
						errChan <- fmt.Errorf("failed to verify aggregated signatures in chunk %d: %v", chunkIndex, err)
						return
					}
					if !valid {
						failChan <- chunkIndex
					}
				}
			}()
		}

		// Send chunk indices to workers
		for j := 0; j < numChunks; j++ {
			tasks <- j
		}
		close(tasks)

		// Wait for all verification tasks to complete
		wg.Wait()
		close(errChan)
		close(failChan)

		// Check for any errors
		for err := range errChan {
			b.Fatalf("%v", err)
		}

		// Check for any failed verifications
		for chunkIdx := range failChan {
			b.Fatalf("Chunk %d verification failed", chunkIdx)
		}
	}
}

// Add chunked aggregation to the comprehensive comparison
func BenchmarkBLS0ChainChunkedComparison(b *testing.B) {
	// Print CPU count information
	numCPU := runtime.NumCPU()
	b.Logf("Running benchmark with %d CPUs available", numCPU)

	testSizes := []int{100, 500, 1000}
	chunkSizes := []int{10, 50, 100}

	for _, n := range testSizes {
		// Full aggregation - one verification for all signatures
		b.Run(fmt.Sprintf("FullAgg-%d", n), func(b *testing.B) {
			schemes, signatures, hashes := setupBLS0ChainData(n)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Create aggregate signature scheme
				aggScheme := GetAggregateSignatureScheme("bls0chain", n, n)

				// Aggregate all signatures
				for j := 0; j < n; j++ {
					err := aggScheme.Aggregate(schemes[j], j, signatures[j], hashes[j])
					if err != nil {
						b.Fatalf("Failed to aggregate signature %d: %v", j, err)
					}
				}

				// Verify the aggregated signature
				valid, err := aggScheme.Verify()
				if err != nil {
					b.Fatalf("Failed to verify aggregated signature: %v", err)
				}
				if !valid {
					b.Fatalf("Aggregated signature verification failed")
				}
			}
		})

		// Test different chunk sizes for the same total number of signatures
		for _, chunkSize := range chunkSizes {
			if chunkSize > n {
				continue // Skip if chunk size is larger than total size
			}

			b.Run(fmt.Sprintf("ChunkedAgg-%d-Chunk%d", n, chunkSize), func(b *testing.B) {
				schemes, signatures, hashes := setupBLS0ChainData(n)

				// Get number of chunks needed
				numChunks := (n + chunkSize - 1) / chunkSize // Ceiling division

				// Get optimal worker count
				numCPU := runtime.NumCPU()
				workerCount := numCPU
				if numChunks < numCPU {
					workerCount = numChunks
				}

				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					var wg sync.WaitGroup
					errChan := make(chan error, numChunks)
					failChan := make(chan int, numChunks)

					// Create a task channel for chunks
					tasks := make(chan int, numChunks)

					// Launch worker goroutines (limited by CPU count)
					for w := 0; w < workerCount; w++ {
						wg.Add(1)
						go func() {
							defer wg.Done()

							// Process chunks from the task channel
							for chunkIndex := range tasks {
								// Calculate start and end indices for this chunk
								startIdx := chunkIndex * chunkSize
								endIdx := startIdx + chunkSize
								if endIdx > n {
									endIdx = n
								}
								chunkSize := endIdx - startIdx

								// Create aggregation scheme for this chunk
								aggScheme := GetAggregateSignatureScheme("bls0chain", chunkSize, chunkSize)

								// Aggregate signatures in this chunk
								for j := startIdx; j < endIdx; j++ {
									err := aggScheme.Aggregate(schemes[j], j-startIdx, signatures[j], hashes[j])
									if err != nil {
										errChan <- fmt.Errorf("failed to aggregate signature %d in chunk %d: %v", j, chunkIndex, err)
										return
									}
								}

								// Verify the aggregated signature for this chunk
								valid, err := aggScheme.Verify()
								if err != nil {
									errChan <- fmt.Errorf("failed to verify aggregated signatures in chunk %d: %v", chunkIndex, err)
									return
								}
								if !valid {
									failChan <- chunkIndex
								}
							}
						}()
					}

					// Send chunk indices to workers
					for j := 0; j < numChunks; j++ {
						tasks <- j
					}
					close(tasks)

					// Wait for all verification tasks to complete
					wg.Wait()
					close(errChan)
					close(failChan)

					// Check for any errors
					for err := range errChan {
						b.Fatalf("%v", err)
					}

					// Check for any failed verifications
					for chunkIdx := range failChan {
						b.Fatalf("Chunk %d verification failed", chunkIdx)
					}
				}
			})
		}

		// Keep the standard concurrent verification for comparison
		b.Run(fmt.Sprintf("Concurrent-%d", n), func(b *testing.B) {
			schemes, signatures, hashes := setupBLS0ChainData(n)

			// Get number of available CPUs
			numCPU := runtime.NumCPU()
			workerCount := numCPU

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				errChan := make(chan error, n)
				failChan := make(chan int, n)

				// Create a task channel
				tasks := make(chan int, n)

				// Launch worker goroutines
				for w := 0; w < workerCount; w++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for idx := range tasks {
							valid, err := schemes[idx].Verify(signatures[idx], hashes[idx])
							if err != nil {
								errChan <- fmt.Errorf("failed to verify signature %d: %v", idx, err)
								continue
							}
							if !valid {
								failChan <- idx
							}
						}
					}()
				}

				// Send tasks to workers
				for j := 0; j < n; j++ {
					tasks <- j
				}
				close(tasks)

				// Wait for all verifications to complete
				wg.Wait()
				close(errChan)
				close(failChan)

				// Check for any errors
				for err := range errChan {
					b.Fatalf("%v", err)
				}

				// Check for any failed verifications
				for idx := range failChan {
					b.Fatalf("Signature %d verification failed", idx)
				}
			}
		})
	}
}
