package webapi

// ASNData ASN查询响应数据结构
type ASNData struct {
	IP             string `json:"ip"`
	IPVersion      int    `json:"ip_version"`
	FoundASN       bool   `json:"found_asn"`
	ASNumber       uint64 `json:"as_number"`
	ASOrganisation string `json:"as_organisation"`
}

// IPLocateData IP定位查询响应数据结构
type IPLocateData struct {
	IP       string `json:"ip"`
	Version  int    `json:"version"`
	Location string `json:"location"`
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
	Area     string `json:"area"`
	Street   string `json:"street"`
	ISP      string `json:"isp"`
}

// AllData 完整查询响应数据结构
type AllData struct {
	IP       string        `json:"ip"`
	Location *IPLocateData `json:"location,omitempty"`
	ASN      *ASNData      `json:"asn,omitempty"`
}

// FullResultItem 批量查询中单个IP的结果
type FullResultItem struct {
	IP       string        `json:"ip"`
	Location *IPLocateData `json:"location,omitempty"`
	ASN      *ASNData      `json:"asn,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// FullData 批量查询响应数据结构
type FullData struct {
	Total     int              `json:"total"`
	Success   int              `json:"success"`
	Failed    int              `json:"failed"`
	ClientIPs *ClientIPInfo    `json:"client_ips,omitempty"`
	Results   []FullResultItem `json:"results"`
}
