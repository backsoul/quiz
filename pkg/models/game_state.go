package models

import "time"

type GameState struct {
	IsActive    bool       `json:"isActive"`
	StartTime   *time.Time `json:"startTime,omitempty"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	Message     string     `json:"message"`
	PlayerCount int        `json:"playerCount"`
}

type GameControl struct {
	Action  string `json:"action"` // "start" o "end"
	AdminID string `json:"adminId,omitempty"`
}
