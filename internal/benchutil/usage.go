package benchutil

// Usage describes the current process only; CPU is cumulative user+kernel time.
type Usage struct {
	CPUSeconds   float64 `json:"cpu_seconds"`
	RSSBytes     uint64  `json:"rss_bytes"`
	PeakRSSBytes uint64  `json:"peak_rss_bytes"`
}
