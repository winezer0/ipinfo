package webapi

import (
	"github.com/winezer0/ipinfo/pkg/asninfo"
	"github.com/winezer0/ipinfo/pkg/iplocate"
)

// ConvertIPLocate 将IP定位查询结果转换为响应数据结构
func ConvertIPLocate(result *iplocate.IPLocate) *IPLocateData {
	if result == nil {
		return nil
	}

	return &IPLocateData{
		IP:       result.IP,
		Version:  result.Version,
		Location: result.Location,
		Country:  result.Country,
		Province: result.Province,
		City:     result.City,
		Area:     result.Area,
		Street:   result.Street,
		ISP:      result.ISP,
	}
}

// ConvertASN 将ASN查询结果转换为响应数据结构
func ConvertASN(result *asninfo.ASNInfo) *ASNData {
	if result == nil {
		return nil
	}

	return &ASNData{
		IP:             result.IP,
		IPVersion:      result.IPVersion,
		FoundASN:       result.FoundASN,
		ASNumber:       result.OrganisationNumber,
		ASOrganisation: result.OrganisationName,
	}
}
