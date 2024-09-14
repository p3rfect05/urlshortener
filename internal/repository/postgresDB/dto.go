package postgresDB

import "time"

type User struct {
	Email        string
	PasswordHash string
	UrlsLeft     int
}

type Link struct {
	ShortUrl  string
	LongUrl   string
	UserEmail string
	ExpiresAt time.Time
}

type Subscription struct {
	Id        int
	Name      string
	TotalUrls int
}
