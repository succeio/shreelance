package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	GitHubID        int64          `gorm:"column:github_id;uniqueIndex" json:"github_id"`
	GitHubToken     string         `gorm:"size:255" json:"github_token"`
	GitLabID        int64          `gorm:"column:gitlab_id;uniqueIndex" json:"gitlab_id"`
	GitLabToken     string         `gorm:"size:255" json:"gitlab_token"`
	GitLabUsername  string         `gorm:"size:255" json:"gitlab_username"`
	Username        string         `gorm:"size:255;not null" json:"username"`
	Email           string         `gorm:"size:255;uniqueIndex" json:"email"`
	PasswordHash    string         `gorm:"size:255" json:"-"`
	AvatarURL       string         `gorm:"size:1024" json:"avatar_url"`
	Stack           string         `gorm:"type:text" json:"stack"` // comma-separated stack, e.g. "Go, TypeScript, Python"
	ExperienceYears int            `gorm:"default:0" json:"experience_years"`
	GitHubCreatedAt time.Time      `json:"github_created_at"`
	GitLabCreatedAt time.Time      `json:"gitlab_created_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	Orders []Order `gorm:"foreignKey:CustomerID" json:"orders,omitempty"`
	Bids   []Bid   `gorm:"foreignKey:FreelancerID" json:"bids,omitempty"`
}

type Order struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Title        string         `gorm:"size:255;not null" json:"title"`
	Description  string         `gorm:"type:text;not null" json:"description"`
	Budget       float64        `gorm:"type:decimal(10,2);not null" json:"budget"`
	Category     string         `gorm:"size:100;default:'';not null" json:"category"`       // e.g. frontend, backend, fullstack, devops, ML, etc.
	RequiredTech string         `gorm:"type:text;default:'';not null" json:"required_tech"` // e.g. "Go, React, Docker"
	Status       string         `gorm:"size:50;default:'open';not null" json:"status"`      // open, in_progress, completed, cancelled
	CustomerID   uint           `gorm:"not null" json:"customer_id"`
	Customer     User           `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	FreelancerID *uint          `json:"freelancer_id,omitempty"`
	Freelancer   *User          `gorm:"foreignKey:FreelancerID" json:"freelancer,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	Bids []Bid `gorm:"foreignKey:OrderID" json:"bids,omitempty"`
}

type Bid struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	OrderID      uint           `gorm:"not null" json:"order_id"`
	Order        Order          `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	FreelancerID uint           `gorm:"not null" json:"freelancer_id"`
	Freelancer   User           `gorm:"foreignKey:FreelancerID" json:"freelancer,omitempty"`
	Price        float64        `gorm:"type:decimal(10,2);not null" json:"price"`
	Comment      string         `gorm:"type:text;not null" json:"comment"`
	Status       string         `gorm:"size:50;default:'pending';not null" json:"status"` // pending, accepted, rejected
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
