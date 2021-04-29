package datastore

type (
	// InstanceProvider describes the instance function signature.
	InstanceProvider func() Entity

	// EntityMetadata describes the interface of the metadata entity.
	EntityMetadata interface {
		GetName() string
		GetDB() string
		Instance() Entity
		GetStore() Store
		GetIDColumnName() string
	}

	// EntityMetadataImpl implements EntityMetadata interface.
	EntityMetadataImpl struct {
		Name         string
		DB           string
		Store        Store
		Provider     InstanceProvider
		IDColumnName string
	}
)

// MetadataProvider constructs entity metadata instance.
func MetadataProvider() *EntityMetadataImpl {
	return &EntityMetadataImpl{
		IDColumnName: "id",
	}
}

// GetName implements EntityMetadata.GetName method of interface.
func (em *EntityMetadataImpl) GetName() string {
	return em.Name
}

// GetDB implements EntityMetadata.GetDB method of interface.
func (em *EntityMetadataImpl) GetDB() string {
	return em.DB
}

// Instance implements EntityMetadata.Instance method of interface.
func (em *EntityMetadataImpl) Instance() Entity {
	return em.Provider()
}

// GetStore implements EntityMetadata.GetStore method of interface.
func (em *EntityMetadataImpl) GetStore() Store {
	return em.Store
}

// GetIDColumnName implements EntityMetadata.GetIDColumnName method of interface.
func (em *EntityMetadataImpl) GetIDColumnName() string {
	return em.IDColumnName
}
