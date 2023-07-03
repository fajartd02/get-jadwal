package main

import "time"

type Schedule struct {
	ScheduleID uint64    `gorm:"primaryKey" json:"id"`
	UserID     uint64    `json:"user_id"`
	Title      string    `json:"title"`
	Day        string    `json:"day"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type User struct {
	ID        uint64     `gorm:"primaryKey" json:"id"`
	Email     string     `json:"email"`
	Schedules []Schedule `gorm:"foreignKey:UserID"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}
