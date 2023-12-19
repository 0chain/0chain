package model

import (
	"time"
)

// Model a basic GoLang struct which includes the following fields: ID, CreatedAt
// It may be embedded into your model or you may build your own model without it
//
//	type User struct {
//	  model.Model
//	}
type ImmutableModel struct {
	ID        uint      `json:"id" model:"primarykey"`
	CreatedAt time.Time `json:"created_at"`
}

type IdModel struct {
	ID uint `json:"id" model:"primarykey"`
}

type UpdatableModel struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
