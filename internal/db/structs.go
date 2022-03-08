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
	Username string `gorm:"primaryKey"`
	SQLiteCard
}

func (c *Card) TableName() string {
	return "cards"
}

type SQLiteCard struct {
	Id     int     `gorm:"primaryKey"`
	Nid    int     `gorm:"not null" gorm:"index:ix_cards_nid"`
	Did    int     `gorm:"not null" gorm:"index:ix_cards_sched"`
	Ord    int     `gorm:"not null"`
	Mod    int     `gorm:"not null"`
	Usn    int     `gorm:"not null" gorm:"index:ix_cards_usn "`
	Type   int     `gorm:"not null"`
	Queue  int     `gorm:"not null" gorm:"index:ix_cards_sched"`
	Due    int     `gorm:"not null" gorm:"index:ix_cards_sched"`
	Ivl    float64 `gorm:"not null"`
	Factor int     `gorm:"not null"`
	Reps   int     `gorm:"not null"`
	Lapses int     `gorm:"not null"`
	Left   int     `gorm:"not null"`
	Odue   int     `gorm:"not null"`
	Odid   int     `gorm:"not null"`
	Flags  int     `gorm:"not null"`
	Data   string  `gorm:"not null"`
}

func (c *SQLiteCard) TableName() string {
	return "cards"
}

type Col struct {
	Username string `gorm:"primaryKey"`
	SQLiteCol
}

type SQLiteCol struct {
	Id     int    `gorm:"not null"`
	Crt    int    `gorm:"not null"`
	Mod    int    `gorm:"not null"`
	Scm    int    `gorm:"not null"`
	Ver    int    `gorm:"not null"`
	Dty    int    `gorm:"not null"`
	Usn    int    `gorm:"not null"`
	Ls     int    `gorm:"not null"`
	Conf   string `gorm:"not null"`
	Models string `gorm:"not null"`
	Decks  string `gorm:"not null"`
	Dconf  string `gorm:"not null"`
	Tags   string `gorm:"not null"`
}

func (c *Col) TableName() string {
	return "col"
}

func (c *SQLiteCol) TableName() string {
	return "col"
}

type Note struct {
	Username string `gorm:"primaryKey"`
	SQLiteNote
}

type SQLiteNote struct {
	Id    int    `gorm:"primaryKey"`
	Guid  string `gorm:"not null"`
	Mid   int    `gorm:"not null"`
	Mod   int    `gorm:"not null"`
	Usn   int    `gorm:"not null" gorm:"index:ix_notes_usn"`
	Tags  string `gorm:"not null"`
	Flds  string `gorm:"not null"`
	Sfld  string `gorm:"not null"`
	Csum  int    `gorm:"not null" gorm:"index:ix_notes_csum "`
	Flags int    `gorm:"not null"`
	Data  string `gorm:"not null"`
}

func (c *Note) TableName() string {
	return "notes"
}
func (c *SQLiteNote) TableName() string {
	return "notes"
}

type Revlog struct {
	Username string `gorm:"primaryKey"`
	SQLiteRevlog
}

type SQLiteRevlog struct {
	Id      int     `gorm:"primaryKey"`
	Iid     int     `gorm:"not null"`
	Usn     int     `gorm:"not null" gorm:"index:ix_revlog_usn"`
	Ease    int     `gorm:"not null"`
	Ivl     float64 `gorm:"not null"`
	LastIvl int     `gorm:"not null"`
	Factor  int     `gorm:"not null"`
	Time    int     `gorm:"not null"`
	Type    int     `gorm:"not null"`
}

func (c *Revlog) TableName() string {
	return "revlog"
}

func (c *SQLiteRevlog) TableName() string {
	return "revlog"
}

type Media struct {
	Username string `gorm:"primaryKey"`
	SQLiteMedia
}

func (m *Media) TableName() string {
	return "media"
}

type SQLiteMedia struct {
	Fname string `gorm:"primaryKey"`
	Usn   int    `gorm:"not null"`
	Csum  string `gorm:"not null"`
}

func (m *SQLiteMedia) TableName() string {
	return "media"
}
