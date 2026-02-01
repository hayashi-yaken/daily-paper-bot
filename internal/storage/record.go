package storage

type PostedEntry struct {
	Date  string `json:"date"`
	Venue string `json:"venue"`
}

type PostedData struct {
	Posted map[string]PostedEntry `json:"posted"`
}
