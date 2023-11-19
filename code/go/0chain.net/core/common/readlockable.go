package common

// ReadLockable - an interface that expects locking before reading
type ReadLockable interface {
	DoReadLock()
	DoReadUnlock()
}
