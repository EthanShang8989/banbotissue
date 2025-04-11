package gridv2

import "sort"

type GridWithIndex struct {
	Index int64
	Info  *GridInfo
}

func MapToSlice(m map[int64]*GridInfo) []GridWithIndex {
	// 将map转换为切片
	grids := make([]GridWithIndex, 0, len(m))
	for index, grid := range m {
		grids = append(grids, GridWithIndex{Index: index, Info: grid})
	}

	// 按照价格排序
	sort.Slice(grids, func(i, j int) bool {
		return grids[i].Info.Price < grids[j].Info.Price
	})

	return grids
}
