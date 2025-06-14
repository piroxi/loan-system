package entity

type User struct {
	DBCommon
	Username string `gorm:"uniqueIndex" json:"username"`
	Role     uint   `json:"role"`
}
