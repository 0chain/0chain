package event

func (edb *EventDb) GetQueryData(fields string, table interface{}) ([]interface{}, error) {
	var result []interface{}
	err := edb.Get().Model(&table).Select(fields).Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}
