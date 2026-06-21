package photos

import "time"

const DefaultPageSize = 15
const MaxPageSize = 60
const MaxUploadBytes = 8 << 20

type Photo struct {
	ID          string    `json:"id"`
	FileName    string    `json:"fileName"`
	ContentType string    `json:"contentType"`
	SizeBytes   int64     `json:"sizeBytes"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ListPhotosResponse struct {
	Items       []Photo `json:"items"`
	Page        int     `json:"page"`
	PageSize    int     `json:"pageSize"`
	Pages       int     `json:"pages"`
	Total       int     `json:"total"`
	HasNext     bool    `json:"hasNext"`
	HasPrevious bool    `json:"hasPrevious"`
}
