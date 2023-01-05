package event

import (
	"fmt"
	"reflect"

	"github.com/0chain/common/core/logging"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const SetTemplate = "%v = t.%v"
const ExprTemplate = "%v = %v"
const UnnestTemplate = "unnest(?::%v[]) AS %v"
const UpdateTemplate = "UPDATE %v SET"
const WhereTemplate = "WHERE %v.%v = t.%v"
const QueryTemplate = "%v %v FROM (SELECT %v) AS t %v"

var typeToSQL = map[reflect.Type]string{
	reflect.TypeOf([]string{}):  "text",
	reflect.TypeOf([]int64{}):   "bigint",
	reflect.TypeOf([]int{}):     "bigint",
	reflect.TypeOf([]byte{}):    "bytea",
	reflect.TypeOf([]float64{}): "decimal",
	reflect.TypeOf([]float32{}): "decimal",
}

type UpdateBuilder struct {
	sets    []string
	unnests []string
	update  string
	where   string
	values  []interface{}
}

func CreateBuilder(table string, idColumn string, idValues interface{}) (b *UpdateBuilder) {
	b = &UpdateBuilder{}
	b.AddUpdate(idColumn, idValues)
	b.sets = nil
	b.update = fmt.Sprintf(UpdateTemplate, table)
	b.where = fmt.Sprintf(WhereTemplate, table, idColumn, idColumn)

	return b
}

func (b *UpdateBuilder) AddUpdate(column string, values interface{}, expr ...string) *UpdateBuilder {
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

	atype, ok := typeToSQL[reflect.TypeOf(values)]

	logging.Logger.Debug("type", zap.String("t", reflect.TypeOf(values).String()))
	if !ok {
		atype = typeToSQL[reflect.TypeOf([]string{})]
	}

	if b.unnests != nil {
		b.unnests = append(b.unnests, ", ")
	}
	b.unnests = append(b.unnests, fmt.Sprintf(UnnestTemplate, atype, column))

	b.values = append(b.values, []interface{}{pq.Array(values)})

	return b
}

type Query struct {
	Q string
	V []interface{}
}

func (b *UpdateBuilder) Build() *Query {
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

func (b *UpdateBuilder) Exec(db *EventDb) *gorm.DB {
	q := b.Build()
	return db.Store.Get().Exec(q.Q, q.V...)
}
