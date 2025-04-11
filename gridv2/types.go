package gridv2

const (
	//单个网格信息
	GridStatusNotPlaced      = "未挂单"
	GridStatusPlaced         = "已挂单"
	GridStatusNotNeeded      = "不需要挂单"
	GridStatusRecentlyFilled = "最近已成交的 不需要挂单"

	//整体网格状态信息
	GridStatusInit = "初始化中"
	GridStatusOver = "超出网格"
)

// GridInfo 表示在对应价位需要执行数量或者价值
type GridInfo struct {
	Price       float64 // 挂单价格
	Short       bool    // 表示数量或者价值
	Status      string  // 当前状态：未挂单，已挂单，不需要挂单。
	Amount      float64 // 挂单数量 和QuoteAmount 只能有一个不为0
	QuoteAmount float64 // 挂单价值 和Amount 只能有一个不为0
}

// GridState 用于管理整个网格状态
type GridState struct {
	// 使用 map 来存储网格信息，网格初始价格的网格为0，价格递增的网格为1，价格递减的网格为-1
	GridInfos map[int64]*GridInfo
}
