package models

// Pizza represents a pizza with its properties
type Pizza struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Ingredients []string `json:"ingredients"`
	Price      float64  `json:"price"`
}
