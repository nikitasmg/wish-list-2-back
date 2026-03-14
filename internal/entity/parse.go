package entity

type ParseResult struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Price       *float64 `json:"price"`
	ImageURL    string   `json:"image_url"`
	Category    string   `json:"category"`
	Brand       string   `json:"brand"`
	Source      string   `json:"source"`
}
