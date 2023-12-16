package config

type Pool struct {
	// Read and write buffer size
	Size int `json:"size"`
	// How much memory to cache,
	// make([size]byte,cache)
	Cache int `json:"cache"`
}
