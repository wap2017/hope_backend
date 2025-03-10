package api

// Response is a general response structure
// type Response struct {
// 	Success bool        `json:"success"`
// 	Message string      `json:"message,omitempty"`
// 	Data    interface{} `json:"data,omitempty"`
// }

// Response represents the standard API response structure
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Total   int64       `json:"total,omitempty"`
	Page    int         `json:"page,omitempty"`
	Size    int         `json:"size,omitempty"`
}
