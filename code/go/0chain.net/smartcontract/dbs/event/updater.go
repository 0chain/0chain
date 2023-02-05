package event

import (
	"fmt"
	"reflect"

	"github.com/0chain/common/core/logging"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

const SetTemplate = "%v = t.%v"
const ExprTemplate = "%v = %v"
const UnnestTemplate = "unnest(?::%v[]) AS %v"
const UpdateTemplate = "UPDATE %v SET"
const WhereTemplate = "WHERE %v.%v = t.%v" // TODO: Remove
const ConditionTemplate = "%v %v %v"
const QueryTemplate = "%v %v FROM (SELECT %v) AS t %v"

var typeToSQL = map[reflect.Type]string{
	reflect.TypeOf([]string{}):  "text",
	reflect.TypeOf([]int64{}):   "bigint",
	reflect.TypeOf([]uint64{}):  "bigint",
	reflect.TypeOf([]int{}):     "bigint",
	reflect.TypeOf([][]byte{}):  "bytea",
	reflect.TypeOf([]float64{}): "decimal",
	reflect.TypeOf([]float32{}): "decimal",
}

// UpdateBuilder helps in building and execution batch updates for postgres sql dialect.
// It uses UPDATE ... FROM postgres construction providing array of values to the db with update query.
// Every row will use corresponding values (matched by id) for appropriate updates.
// Update row will look like this <<id, val1, val2, ... valn>>
// Note that all update rows should have the same number of values, id is required together with at least one update.
type UpdateBuilder struct {
	tableName string
	sets      []string
	unnests   []string
	update    string
	where     string
	values    []interface{}
}

// CreateBuilder receives table name, id column name and set of id values as parameters.
// These id values will be used to select appropriate rows and apply updates to them accordingly
func CreateBuilder(table string, idColumn string, idValues interface{}) (b *UpdateBuilder) {
	b = &UpdateBuilder{}
	b.tableName = table
	b.update = fmt.Sprintf(UpdateTemplate, table)
	b.addWhereConditionFromValues(idColumn).addToUnnests(idColumn, idValues).addToValues(idValues)
	return b
}

// AddCompositeId Add new part of a composite id key
func (b *UpdateBuilder) AddCompositeId(columnName string, values interface{}) *UpdateBuilder {
	return b.addWhereConditionFromValues(columnName).addToUnnests(columnName, values).addToValues(values)
}

// AddUpdate adds column to an update.
// column is a column name to be updated
// values are the values to use for update, len(values) should be the same as len(ids) used to create builder.
// value[i] will be applied to row with id[i], so all the values from different updates on the same index will form the update vector.
// expr is an optional expression to be used in update. By default, *UPDATE table SET column = t.column* will be used.
// If expression is set, then it will be used instead of default, e. g. expr = "table.column =table.column + t.column ",
// which means that current value of column will be increased, so *UPDATE table SET table.column =table.column + t.column* will be generated instead
func (b *UpdateBuilder) AddUpdate(column string, values interface{}, expr ...string) *UpdateBuilder {
	return b.addToSets(column, values, expr...).addToUnnests(column, values).addToValues(values)
}

// AddCondition Add ANDed condition comparing a field to a static value to the update query
func (b *UpdateBuilder) AddCondition(column, operator, value string) *UpdateBuilder {
	return b.addWhereCondition(b.tableName+"."+column, operator, value)
}

type Query struct {
	Q string
	V []interface{}
}

func (b *UpdateBuilder) build() *Query {
	sets := ""
	for _, s := range b.sets {
		sets = sets + s
	}
	unnests := ""
	for _, u := range b.unnests {
		unnests = unnests + u
	}

	return &Query{Q: fmt.Sprintf(QueryTemplate, b.update, sets, unnests, b.where), V: b.values}
}

func (b *UpdateBuilder) addToSets(column string, values interface{}, expr ...string) *UpdateBuilder {
	if b.sets != nil {
		b.sets = append(b.sets, ", ")
	}
	switch len(expr) {
	case 0:
		b.sets = append(b.sets, fmt.Sprintf(SetTemplate, column, column))
	case 1:
		b.sets = append(b.sets, fmt.Sprintf(ExprTemplate, column, expr[0]))
	default:
		logging.Logger.Warn("only one expr is supported, ignoring")
		b.sets = append(b.sets, fmt.Sprintf(ExprTemplate, column, expr[0]))
	}

	return b
}

func (b *UpdateBuilder) addToUnnests(column string, values interface{}) *UpdateBuilder {
	atype, ok := typeToSQL[reflect.TypeOf(values)]

	if !ok {
		atype = typeToSQL[reflect.TypeOf([]string{})]
	}

	if b.unnests != nil {
		b.unnests = append(b.unnests, ", ")
	}
	b.unnests = append(b.unnests, fmt.Sprintf(UnnestTemplate, atype, column))

	return b
}

func (b *UpdateBuilder) addToValues(values interface{}) *UpdateBuilder {
	b.values = append(b.values, []interface{}{pq.Array(values)})
	return b
}

// Add condition in the form tableName.columnName = t.columnName
func (b *UpdateBuilder) addWhereConditionFromValues(column string) *UpdateBuilder {
	return b.addWhereCondition(b.tableName+"."+column, "=", "t."+column)
}

func (b *UpdateBuilder) addWhereCondition(left, operator, right string) *UpdateBuilder {
	b.where += " AND " + fmt.Sprintf(ConditionTemplate, left, operator, right)
	return b
}

// Exec builds and executes the query
func (b *UpdateBuilder) Exec(db *EventDb) *gorm.DB {
	q := b.build()
	return db.Store.Get().Exec(q.Q, q.V...)
}
