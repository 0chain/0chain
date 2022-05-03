package smartcontractinterface

import (
	"context"
	"net/url"
	"reflect"
	"testing"

	"github.com/rcrowley/go-metrics"
)

func TestSmartContract_HandlerStats(t *testing.T) {
	t.Parallel()

	resp := "<!DOCTYPE html><html><body><style>\n" +
		".number { text-align: right; }\n" +
		".menu li { list-style-type: none; }\n" +
		"table, td, th { border: 1px solid black;  border-collapse: collapse;}\n" +
		"tr.header { background-color: #E0E0E0;  }\n" +
		".inactive { background-color: #F44336; }\n" +
		".warning { background-color: #FFEB3B; }\n" +
		".optimal { color: #1B5E20; }\n" +
		".slow { font-style: italic; }\n" +
		".bold {font-weight:bold;}" +
		"</style>" +
		"<table width='100%'><tr><td><h2>1</h2><table width='100%'>" +
		"<tr><td class='sheader' colspan=2'>Metrics</td></tr>" +
		"<tr><td>Count</td><td>0</td></tr>" +
		"<tr><td class='sheader' colspan='2'>Metric Value</td></tr>" +
		"<tr><td>Min</td><td>0.00</td></tr>" +
		"<tr><td>Mean</td><td>0.00 &plusmn;0.00</td></tr>" +
		"<tr><td>Max</td><td>0.00</td></tr>" +
		"<tr><td>50.00%</td><td>0.00</td></tr>" +
		"<tr><td>90.00%</td><td>0.00</td></tr>" +
		"<tr><td>95.00%</td><td>0.00</td></tr>" +
		"<tr><td>99.00%</td><td>0.00</td></tr>" +
		"<tr><td>99.90%</td><td>0.00</td></tr>" +
		"</table></td><td><h2>2</h2><table width='100%'>" +
		"<tr><td class='sheader' colspan=2'>Metrics</td></tr>" +
		"<tr><td>Count</td><td>0</td></tr>" +
		"<tr><td class='sheader' colspan='2'>Time taken</td></tr>" +
		"<tr><td>Min</td><td>0.00 ms</td></tr>" +
		"<tr><td>Mean</td><td>0.00 &plusmn;0.00 ms</td></tr>" +
		"<tr><td>Max</td><td>0.00 ms</td></tr>" +
		"<tr><td>50.00%</td><td>0.00 ms</td></tr>" +
		"<tr><td>90.00%</td><td>0.00 ms</td></tr><" +
		"tr><td>95.00%</td><td>0.00 ms</td></tr>" +
		"<tr><td>99.00%</td><td>0.00 ms</td></tr>" +
		"<tr><td>99.90%</td><td>0.00 ms</td></tr>" +
		"<tr><td class='sheader' colspan='2'>Rate per second</td></tr>" +
		"<tr><td>Last 1-min rate</td><td>0.00</td></tr>" +
		"<tr><td>Last 5-min rate</td><td>0.00</td></tr>" +
		"<tr><td>Last 15-min rate</td><td>0.00</td></tr>" +
		"<tr><td>Overall mean rate</td><td>0.00</td></tr>" +
		"</table></td></tr><tr><td><h2>3</h2>\n" +
		"\t<table width='100%'>\n" +
		"\t\t<tr><td>Count</td><td>0</td></tr>\n" +
		"\t</table>\n" +
		"</td><td><h2>4</h2>This is wrong. You should not be seeing this</td></tr></body></html>"

	type fields struct {
		ID                          string
		RestHandlers                map[string]SmartContractRestHandler
		SmartContractExecutionStats map[string]interface{}
	}
	type args struct {
		ctx    context.Context
		params url.Values
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				SmartContractExecutionStats: map[string]interface{}{
					"1": metrics.NewHistogram(metrics.NewUniformSample(5)),
					"2": metrics.NewTimer(),
					"3": metrics.NewCounter(),
					"4": "default stats",
				},
			},
			want: resp,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := &SmartContract{
				ID:                          tt.fields.ID,
				SmartContractExecutionStats: tt.fields.SmartContractExecutionStats,
			}
			got, err := sc.HandlerStats(tt.args.ctx, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandlerStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandlerStats() got = %v, want %v", got, tt.want)
			}
		})
	}
}
