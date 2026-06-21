package clipboard

import "time"

const MaxTextLength = 20_000
const DefaultPageSize = 15
const MaxPageSize = 100

type Item struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateItemRequest struct {
	Text string `json:"text"`
}

type ListItemsResponse struct {
	Items       []Item `json:"items"`
	Page        int    `json:"page"`
	PageSize    int    `json:"pageSize"`
	Pages       int    `json:"pages"`
	Total       int    `json:"total"`
	HasNext     bool   `json:"hasNext"`
	HasPrevious bool   `json:"hasPrevious"`
}
