package datastore

type InstanceProvider func() Entity

var entityMetadataMap = make(map[string]EntityMetadata)

/*RegisterEntityProvider - keep track of a list of entity providers. An entity can be registered with multiple names
* as long as two entities don't use the same name
 */
func RegisterEntityMetadata(entityName string, entityMetadata EntityMetadata) {
	entityMetadataMap[entityName] = entityMetadata
}

/*GetEntityMetadata - return an instance of the entity */
func GetEntityMetadata(entityName string) EntityMetadata {
	return entityMetadataMap[entityName]
}

/*GetEntity - return an instance of the entity */
func GetEntity(entityName string) Entity {
	return GetEntityMetadata(entityName).Instance()
}

type EntityMetadata interface {
	GetName() string
	GetMemoryDB() string
	Instance() Entity
}

type EntityMetadataImpl struct {
	Name     string
	MemoryDB string
	Provider InstanceProvider
}

func (em *EntityMetadataImpl) GetName() string {
	return em.Name
}

func (em *EntityMetadataImpl) GetMemoryDB() string {
	return em.MemoryDB
}

func (em *EntityMetadataImpl) Instance() Entity {
	return em.Provider()
}
