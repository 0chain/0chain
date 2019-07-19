package encryption

//SplittableSignatureScheme - a signature scheme that supports splitting the primary key into parts
type SplittableSignatureScheme interface {
	SignatureScheme
	GenerateSplitKeys(numSplits int) ([]SignatureScheme, error)
	AggregateSignatures(signatures []string) (string, error)
}
