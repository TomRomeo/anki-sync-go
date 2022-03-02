package db

import (
	uuid "github.com/satori/go.uuid"
	"gorm.io/gorm"
)

type Auth struct {
	Username string `gorm:"primaryKey" json:"u"`
	Pw       string `json:"p"`
}

type Session struct {
	Username string    `gorm:"not null"`
	Skey     uuid.UUID `gorm:"primaryKey"`
	created  string    `gorm:"not null"`
}

// BeforeCreate will set a UUID rather than numeric ID.
func (sesh *Session) BeforeCreate(tx *gorm.DB) error {
	sesh.Skey = uuid.NewV4()
	return nil
}

type Card struct {
	Username string  `gorm:"primaryKey"`
	Id       int     `gorm:"primaryKey"`
	Nid      int     `gorm:"not null"`
	Did      int     `gorm:"not null"`
	Ord      int     `gorm:"not null"`
	Mod      int     `gorm:"not null"`
	Usn      int     `gorm:"not null"`
	Type     int     `gorm:"not null"`
	Queue    int     `gorm:"not null"`
	Due      int     `gorm:"not null"`
	Ivl      float64 `gorm:"not null"`
	Factor   int     `gorm:"not null"`
	Reps     int     `gorm:"not null"`
	Lapses   int     `gorm:"not null"`
	Left     int     `gorm:"not null"`
	Odue     int     `gorm:"not null"`
	Odid     int     `gorm:"not null"`
	Flags    int     `gorm:"not null"`
	Data     string  `gorm:"not null"`
}

type Col struct {
	Username string `gorm:"primaryKey"`
	Id       int    `gorm:"not null"`
	Crt      int    `gorm:"not null"`
	Mod      int    `gorm:"not null"`
	Scm      int    `gorm:"not null"`
	Ver      int    `gorm:"not null"`
	Dty      int    `gorm:"not null"`
	Usn      int    `gorm:"not null"`
	Ls       int    `gorm:"not null"`
	Conf     string `gorm:"not null"`
	Models   string `gorm:"not null"`
	Decks    string `gorm:"not null"`
	Dconf    string `gorm:"not null"`
	Tags     string `gorm:"not null"`
}

func (c *Col) TableName() string {
	return "col"
}

type Note struct {
	Username string `gorm:"primaryKey"`
	Id       int    `gorm:"primaryKey"`
	Guid     string `gorm:"not null"`
	Mid      int    `gorm:"not null"`
	Mod      int    `gorm:"not null"`
	Usn      int    `gorm:"not null"`
	Tags     string `gorm:"not null"`
	Flds     string `gorm:"not null"`
	Sfld     string `gorm:"not null"`
	Csum     int    `gorm:"not null"`
	Flags    int    `gorm:"not null"`
	Data     string `gorm:"not null"`
}

func (c *Note) TableName() string {
	return "notes"
}

type Revlog struct {
	Username string  `gorm:"primaryKey"`
	Id       int     `gorm:"primaryKey"`
	Iid      int     `gorm:"not null"`
	Usn      int     `gorm:"not null"`
	Ease     int     `gorm:"not null"`
	Ivl      float64 `gorm:"not null"`
	LastIvl  int     `gorm:"not null"`
	Factor   int     `gorm:"not null"`
	Time     int     `gorm:"not null"`
	Type     int     `gorm:"not null"`
}

func (c *Revlog) TableName() string {
	return "revlog"
}
