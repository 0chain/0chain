package event

import (
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestUpdateBuilder_build(t *testing.T) {
	type update struct {
		key       string
		condition string
		val       interface{}
	}

	type condition struct {
		key string
		val interface{}
	}

	type fields struct {
		ids     interface{}
		updates []update
		idParts []condition
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "simple",
			fields: fields{
				ids:     interface{}([]string{"1"}),
				updates: []update{{val: []string{"c1"}, key: "column"}},
			},
			want: "UPDATE table SET column = t.column FROM (SELECT unnest(?::text[]) AS id, unnest(?::text[]) AS column) AS t WHERE table.id = t.id",
		},
		{
			name: "simple int ids",
			fields: fields{
				ids:     []int{1},
				updates: []update{{val: []string{"c1"}, key: "column"}},
			},
			want: "UPDATE table SET column = t.column FROM (SELECT unnest(?::bigint[]) AS id, unnest(?::text[]) AS column) AS t WHERE table.id = t.id",
		},
		{
			name: "several updates",
			fields: fields{
				ids: []string{"1"},
				updates: []update{
					{val: []string{"c11", "c12", "c13"}, key: "column1"},
					{val: []string{"c21", "c22", "c23"}, key: "column2"},
					{val: []string{"c31", "c32", "c33"}, key: "column3"},
					{val: []string{"c41", "c42", "c43"}, key: "column4"},
				},
			},
			want: "UPDATE table SET column1 = t.column1, column2 = t.column2, column3 = t.column3, column4 = t.column4 FROM (SELECT unnest(?::text[]) AS id, unnest(?::text[]) AS column1, unnest(?::text[]) AS column2, unnest(?::text[]) AS column3, unnest(?::text[]) AS column4) AS t WHERE table.id = t.id",
		},
		{
			name: "several updates with different values",
			fields: fields{
				ids: []string{"1"},
				updates: []update{
					{val: []string{"c11", "c12", "c13"}, key: "column1"},
					{val: []int{1, 2, 3}, key: "column2"},
					{val: [][]byte{[]byte("c31"), []byte("c32"), []byte("c33")}, key: "column3"},
					{val: []float64{1, 2, 3}, key: "column4"},
				},
			},
			want: "UPDATE table SET column1 = t.column1, column2 = t.column2, column3 = t.column3, column4 = t.column4 FROM (SELECT unnest(?::text[]) AS id, unnest(?::text[]) AS column1, unnest(?::bigint[]) AS column2, unnest(?::bytea[]) AS column3, unnest(?::decimal[]) AS column4) AS t WHERE table.id = t.id",
		},
		{
			name: "several updates with condition",
			fields: fields{
				ids: []string{"1"},
				updates: []update{
					{val: []string{"c11", "c12", "c13"}, key: "column1", condition: "column1 + t.column1"},
					{val: []string{"c21", "c22", "c23"}, key: "column2", condition: "column2 - t.column2"},
					{val: []string{"c31", "c32", "c33"}, key: "column3", condition: "t.column3"},
					{val: []string{"c41", "c42", "c43"}, key: "column4", condition: "column4 * t.column4"},
				},
			},
			want: "UPDATE table SET column1 = column1 + t.column1, column2 = column2 - t.column2, column3 = t.column3, column4 = column4 * t.column4 FROM (SELECT unnest(?::text[]) AS id, unnest(?::text[]) AS column1, unnest(?::text[]) AS column2, unnest(?::text[]) AS column3, unnest(?::text[]) AS column4) AS t WHERE table.id = t.id",
		},
		{
			name: "several updates with different values and conditions",
			fields: fields{
				ids: []string{"1"},
				updates: []update{
					{val: []string{"c11", "c12", "c13"}, key: "column1", condition: "column1 + t.column1"},
					{val: []int{1, 2, 3}, key: "column2", condition: "column2 - t.column2"},
					{val: [][]byte{[]byte("c31"), []byte("c32"), []byte("c33")}, key: "column3", condition: "t.column3"},
					{val: []float64{1, 2, 3}, key: "column4", condition: "column4 * t.column4"},
				},
			},
			want: "UPDATE table SET column1 = column1 + t.column1, column2 = column2 - t.column2, column3 = t.column3, column4 = column4 * t.column4 FROM (SELECT unnest(?::text[]) AS id, unnest(?::text[]) AS column1, unnest(?::bigint[]) AS column2, unnest(?::bytea[]) AS column3, unnest(?::decimal[]) AS column4) AS t WHERE table.id = t.id",
		},
		{
			name: "multiple id parts",
			fields: fields{
				ids: []string{"1","2","3"},
				updates: []update{
					{val: []string{"c11", "c12", "c13"}, key: "column1"},
				},
				idParts: []condition{
					{key: "id2", val: []string{"1_2", "2_2", "3_2"}},
					{key: "id3", val: []string{"1_3", "2_3", "3_3"}},
				},
			},
			want: "UPDATE table SET column1 = t.column1 FROM (SELECT unnest(?::text[]) AS id, unnest(?::text[]) AS id2, unnest(?::text[]) AS id3, unnest(?::text[]) AS column1) AS t WHERE table.id = t.id AND table.id2 = t.id2 AND table.id3 = t.id3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toTest := CreateBuilder("table", "id", tt.fields.ids)
			var vals []interface{}
			vals = append(vals, []interface{}{pq.Array(tt.fields.ids)})

			for _, c := range tt.fields.idParts {
				toTest.AddIdPart(c.key, c.val)
				vals = append(vals, []interface{}{pq.Array(c.val)})
			}

			for _, u := range tt.fields.updates {
				if len(u.condition) > 0 {
					toTest.AddUpdate(u.key, u.val, u.condition)
				} else {
					toTest.AddUpdate(u.key, u.val)
				}
				vals = append(vals, []interface{}{pq.Array(u.val)})
			}
			assert.Equalf(t, &Query{tt.want, vals}, toTest.build(), "build()")
		})
	}
}
