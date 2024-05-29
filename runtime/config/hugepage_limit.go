package config

type HugepageLimit struct {
	Pagesize string `json:"page_size"`
	Limit    uint64 `json:"limit"`
}
