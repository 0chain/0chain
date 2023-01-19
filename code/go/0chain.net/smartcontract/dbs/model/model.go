package model

import (
	"time"
)

// Model a basic GoLang struct which includes the following fields: ID, CreatedAt
// It may be embedded into your model or you may build your own model without it
//    type User struct {
//      model.Model
//    }
type ImmutableModel struct {
	ID        uint `model:"primarykey"`
	CreatedAt time.Time
}

type IdModel struct {
	ID uint `model:"primarykey"`
}

type UpdatableModel struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
