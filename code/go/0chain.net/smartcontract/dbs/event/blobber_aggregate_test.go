package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_paginate(t *testing.T) {
	type args struct {
		round      int64
		pageAmount int64
		count      int64
	}
	tests := []struct {
		name              string
		args              args
		size              int64
		currentPageNumber int64
		subpageCount      int
	}{
		{name: "1", args: struct {
			round      int64
			pageAmount int64
			count      int64
		}{
			round: 1, pageAmount: 7, count: 7,
		}, size: 1, currentPageNumber: 1, subpageCount: 1},
		{name: "2", args: struct {
			round      int64
			pageAmount int64
			count      int64
		}{
			round: 13, pageAmount: 7, count: 7,
		}, size: 1, currentPageNumber: 6, subpageCount: 1},
		{name: "3", args: struct {
			round      int64
			pageAmount int64
			count      int64
		}{
			round: 13, pageAmount: 7, count: 68,
		}, size: 10, currentPageNumber: 6, subpageCount: 1},
		{name: "4", args: struct {
			round      int64
			pageAmount int64
			count      int64
		}{
			round: 13, pageAmount: 7, count: 695,
		}, size: 100, currentPageNumber: 6, subpageCount: 2},
		{name: "5", args: struct {
			round      int64
			pageAmount int64
			count      int64
		}{
			round: 13, pageAmount: 7, count: 650,
		}, size: 93, currentPageNumber: 6, subpageCount: 2},
		{name: "6", args: struct {
			round      int64
			pageAmount int64
			count      int64
		}{
			round: 12, pageAmount: 7, count: 650,
		}, size: 93, currentPageNumber: 5, subpageCount: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := paginate(tt.args.round, tt.args.pageAmount, tt.args.count, 50)
			assert.Equalf(t, tt.size, got, "paginate(%v, %v, %v)", tt.args.round, tt.args.pageAmount, tt.args.count)
			assert.Equalf(t, tt.currentPageNumber, got1, "paginate(%v, %v, %v)", tt.args.round, tt.args.pageAmount, tt.args.count)
			assert.Equalf(t, tt.subpageCount, got2, "paginate(%v, %v, %v)", tt.args.round, tt.args.pageAmount, tt.args.count)
		})
	}
}
