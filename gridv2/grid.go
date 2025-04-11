package gridv2

import (
	"fmt"
	"math"
	"strconv"

	"github.com/banbox/banbot/core"
	"github.com/banbox/banbot/strat"
	"github.com/banbox/banexg/log"
	"go.uber.org/zap"
)

// import (
// 	"fmt"
// 	"strconv"
// 	"strings"
// 	"time"

// 	// "github.com/banbox/banbot/pkg/fixedpoint"
// 	// "github.com/c9s/bbgo/pkg/types"
// )

func NewGridState(startPrice, upperPrice, lowerPrice float64, gridNum int64, quoteAmount float64, symbol string) *GridState {
	log.Info("NewGridState", zap.Float64("startPrice", startPrice), zap.Float64("upperPrice", upperPrice), zap.Float64("lowerPrice", lowerPrice), zap.Int64("gridNum", gridNum), zap.Float64("quoteAmount", quoteAmount), zap.String("symbol", symbol))
	gs := &GridState{}
	gs.GenerateGridInfos(startPrice, upperPrice, lowerPrice, gridNum, quoteAmount, symbol)
	return gs
}

func (gs *GridState) GenerateGridInfos(startPrice, upperPrice, lowerPrice float64, gridNum int64, quoteAmount float64, symbol string) error {
	err := gs.GenerateGridInfoPirce(startPrice, upperPrice, lowerPrice, gridNum, 0)
	if err != nil {
		return err
	}
	err = gs.GenerateGridInfoQty(quoteAmount, 0)
	if err != nil {
		return err
	}
	// err = gs.GenerateGridInfoOrders(quoteAmount, symbol, market, groupID)
	// if err != nil {
	// 	return err
	// }

	return nil
}

// GenerateGridInfoPirce 生成网格基本价格和买卖方向的信息
func (gs *GridState) GenerateGridInfoPirce(startPrice, upperPrice, lowerPrice float64, gridNum int64, side int) error {
	if gs.GridInfos == nil {
		gs.GridInfos = make(map[int64]*GridInfo)
	}

	// 计算价格区间和网格间距
	priceRange := upperPrice - lowerPrice
	numGrids := gridNum
	gridSpread := priceRange / float64(numGrids-1)

	if gridSpread == 0 {
		return fmt.Errorf("网格间距为零，请检查价格区间和网格数量")
	}

	// 使用Round替代Floor进行四舍五入
	gridIndex := math.Round((startPrice - lowerPrice) / gridSpread)
	alignedPrice := lowerPrice + gridIndex*gridSpread

	// 打印对齐信息
	log.Info("网格信息:")
	log.Info("网格间距:", zap.Float64("gridSpread", gridSpread))
	log.Info("原始价格:", zap.Float64("startPrice", startPrice))
	log.Info("对齐后价格:", zap.Float64("alignedPrice", alignedPrice))
	log.Info("对齐差值:", zap.Float64("alignedPrice-startPrice", alignedPrice-startPrice))
	log.Info("网格索引:", zap.Float64("gridIndex", gridIndex))

	switch side {
	case 1:
		// 从对齐价格向下生成买网格
		// 调整起始index，使其相对于gridIndex
		index := int64(-1)
		for price := alignedPrice - gridSpread; lowerPrice <= price; price = price - gridSpread {
			gs.GridInfos[index] = &GridInfo{
				Price:  price,
				Short:  false,
				Status: GridStatusNotPlaced,
			}
			index--
		}

	case -1:
		// 从对齐价格向上生成卖网格
		// 调整起始index，使其相对于gridIndex
		index := int64(1)
		for price := alignedPrice + gridSpread; price <= upperPrice; price = price + gridSpread {
			gs.GridInfos[index] = &GridInfo{
				Price:  price,
				Short:  true,
				Status: GridStatusNotPlaced,
			}
			index++
		}

	case 0:
		// 使用同一个对齐价格生成买卖网格
		if err := gs.GenerateGridInfoPirce(alignedPrice, upperPrice, lowerPrice, gridNum, 1); err != nil {
			return err
		}
		if err := gs.GenerateGridInfoPirce(alignedPrice, upperPrice, lowerPrice, gridNum, -1); err != nil {
			return err
		}
		// //一般来说一开始不要下起始（0）位置的附近的网格
		// gs.GridInfos[0].Status = GridStatusRecentlyFilled
		isShort := false
		if alignedPrice > startPrice {
			isShort = true
		}
		gs.GridInfos[0] = &GridInfo{
			Price:  alignedPrice,
			Short:  isShort,
			Status: GridStatusNotNeeded,
		}
	}

	return nil
}

func (gs *GridState) UpdateGridInfoSide(Id int64, side bool) {
	gs.GridInfos[Id].Short = side
}

// 根据成交属于哪个网格ID，更新网格的买卖方向信息
func (gs *GridState) UpdateGridInfoByFilledId(gridId int64) error {
	gridInfo := gs.GridInfos[gridId]
	if gridInfo == nil {
		return fmt.Errorf("网格ID %d 不存在", gridId)
	}

	gridInfo.Status = GridStatusRecentlyFilled
	//这里方向是不确定的，因为如果上涨下跌的话这里还是buy
	// gridInfo.Side = types.SideTypeSell
	//检查上一个网格是否需要挂单
	uppperGridInfo, ok := gs.GridInfos[gridId+1]
	if ok {
		if !uppperGridInfo.Short && uppperGridInfo.Status != GridStatusRecentlyFilled {
			return fmt.Errorf("网格ID %d 的下一个网格ID %d 的方向不正确", gridId, gridId+1)
		}
		if uppperGridInfo.Status == GridStatusNotNeeded || uppperGridInfo.Status == GridStatusRecentlyFilled {
			uppperGridInfo.Status = GridStatusNotPlaced
			uppperGridInfo.Short = true
		}
	}
	//检查下一个网格是否需要挂单
	lowerGridInfo, ok := gs.GridInfos[gridId-1]
	if ok {
		if lowerGridInfo.Short && lowerGridInfo.Status != GridStatusNotNeeded {
			return fmt.Errorf("网格ID %d 的下一个网格ID %d 的方向不正确,网格状态为%s", gridId, gridId-1, lowerGridInfo.Status)
		}
		if lowerGridInfo.Status == GridStatusNotNeeded || lowerGridInfo.Status == GridStatusRecentlyFilled {
			lowerGridInfo.Status = GridStatusNotPlaced
			lowerGridInfo.Short = false
		}
	}

	return nil
}

// 根据网格基本价格和买卖方向信息，生成网格订单信息
// TODO: 马丁模式
func (gs *GridState) GenerateGridInfoQty(quoteAmount float64, amount float64) error {
	for _, gridInfo := range gs.GridInfos {
		if quoteAmount != 0 {
			gridInfo.QuoteAmount = quoteAmount
		} else if amount != 0 {
			gridInfo.Amount = amount
		} else {
			return fmt.Errorf("quoteAmount 和 amount 不能同时为0")
		}
	}
	return nil
}

// 根据网格id和当前位置生成对应的订单
func (gs *GridState) GenGridInfoOrders(updateStatus bool) ([]*strat.EnterReq, []*strat.ExitReq, error) {
	// gridInfo := gs.GridInfos[gridId]
	// if gridInfo == nil {
	// 	return nil, nil, fmt.Errorf("网格ID %d 不存在", gridId)
	// }
	openPosOrders := make([]*strat.EnterReq, 0)
	closePosOrders := make([]*strat.ExitReq, 0)
	//因为是双向持仓模式 所以需要根据当前仓位和具体需要下单的情况来决定需要的出入场订单
	//因为上面生成的网格信息0和0以上是卖网格，0以下是买网格 现在默认是空仓起步，所以默认两边都是开仓的。后续根据仓位需求转化买卖单。
	// recentGridId, err := gs.getRecentGridId()

	//刚开始两边都下开仓单
	if gs.NeedToInit() {
		for gridId, gridInfo := range gs.GridInfos {
			if gridInfo.Status != GridStatusNotNeeded && gridInfo.Status != GridStatusRecentlyFilled {
				order := &strat.EnterReq{}
				order.Tag = strconv.FormatInt(gridId, 10)
				order.Limit = gridInfo.Price
				order.LegalCost = gridInfo.QuoteAmount
				order.OrderType = core.OrderTypeLimit
				order.Short = gridInfo.Short
				order.Leverage = 2
				openPosOrders = append(openPosOrders, order)
				if updateStatus {
					gridInfo.Status = GridStatusPlaced
				}
			}
		}
	} else {
		//中途更新了订单状态需要找到还需要挂单的网格\
		for gridId, gridInfo := range gs.GridInfos {
			if gridInfo.Status == GridStatusNotPlaced {
				//网格上方卖是开单 买是平仓
				if gridId > 0 {
					if gridInfo.Short {
						openPosOrders = append(openPosOrders, &strat.EnterReq{
							Leverage:  2,
							Tag:       strconv.FormatInt(gridId, 10),
							Limit:     gridInfo.Price,
							LegalCost: gridInfo.QuoteAmount,
						})
					} else {
						closePosOrders = append(closePosOrders, &strat.ExitReq{
							Dirt:       core.OdDirtLong,
							Tag:        strconv.FormatInt(gridId, 10),
							Limit:      gridInfo.Price,
							Amount:     gridInfo.QuoteAmount / gridInfo.Price,
							FilledOnly: true,
						})
					}
				} else if gridId == 0 {
					//0号网格是当作初始价格 所以需要全部平仓
					var dirt int
					recentGridId, err := gs.getRecentGridId()
					if err != nil {
						log.Error("获取最近成交的网格ID失败", zap.Error(err))
					}
					if recentGridId > 0 {
						dirt = core.OdDirtLong
					} else {
						dirt = core.OdDirtShort
					}
					closePosOrders = append(closePosOrders, &strat.ExitReq{
						Dirt:       dirt,
						Tag:        strconv.FormatInt(gridId, 10),
						Limit:      gridInfo.Price,
						ExitRate:   1,
						FilledOnly: true,
					})
				} else {
					//初始价格下方买是开单 卖是平仓
					if gridInfo.Short {
						closePosOrders = append(closePosOrders, &strat.ExitReq{
							Dirt:       core.OdDirtShort,
							Tag:        strconv.FormatInt(gridId, 10),
							Limit:      gridInfo.Price,
							Amount:     gridInfo.QuoteAmount / gridInfo.Price,
							FilledOnly: true,
							Force:      true,
						})
					} else {
						openPosOrders = append(openPosOrders, &strat.EnterReq{
							Leverage:  2,
							Tag:       strconv.FormatInt(gridId, 10),
							Limit:     gridInfo.Price,
							LegalCost: gridInfo.QuoteAmount,
						})
					}
				}
				if updateStatus {
					gridInfo.Status = GridStatusPlaced
				}
			}

		}
	}
	return openPosOrders, closePosOrders, nil
}

func (gs *GridState) NeedToInit() bool {
	//如果发现一个都没有挂单的话说明还在初始化中
	init := true
	for _, gridInfo := range gs.GridInfos {
		if gridInfo.Status == GridStatusPlaced {
			init = false
		}
	}
	return init
}

// 获得最近成交的网格ID
func (gs *GridState) getRecentGridId() (int64, error) {
	for gridId, gridInfo := range gs.GridInfos {
		if gridInfo.Status == GridStatusRecentlyFilled {
			return gridId, nil
		}
	}
	return 0, fmt.Errorf("最近成交的网格ID不存在")
}

// // 根据网格基本价格和买卖方向信息，生成网格订单信息
// // TODO: 马丁模式
// func (gs *GridState) GenerateGridInfoOrders(QuoteAmount fixedpoint.Value, symbol string, market types.Market, groupID uint32) error {
// 	for gridID, gridInfo := range gs.GridInfos {
// 		timestamp := time.Now().UnixNano()

// 		// 根据gridID的正负来决定使用G还是GN
// 		var gridPrefix string
// 		var gridNum int64
// 		if gridID < 0 {
// 			gridPrefix = "GN"
// 			gridNum = -gridID // 转为正数
// 		} else {
// 			gridPrefix = "G"
// 			gridNum = gridID
// 		}

// 		// 格式：symbol-G数字-timestamp 或 symbol-GN数字-timestamp
// 		cid := fmt.Sprintf("%s-%s%d-%d",
// 			symbol,
// 			gridPrefix,
// 			gridNum,
// 			timestamp,
// 		)

// 		orders := types.SubmitOrder{
// 			ClientOrderID: cid,
// 			Symbol:        symbol,
// 			Side:          gridInfo.Side,
// 			Type:          types.OrderTypeLimit,
// 			// Type:          types.OrderTypeLimitMaker,
// 			// Market:        market,
// 			Quantity: gridInfo.CalculateQuantity(gridInfo.Price),
// 			Price:    gridInfo.Price,
// 			// TimeInForce:      types.TimeInForceGTC,
// 			// MarginSideEffect: types.SideEffectTypeMarginBuy,
// 			// StopPrice:        100000,
// 			GroupID: groupID,
// 		}
// 		gridInfo.order = orders
// 	}
// 	return nil
// }

// func (gs *GridState) UpdateGridInfoOrder(ID int64, order types.SubmitOrder) error {
// 	gridInfo := gs.GridInfos[ID]
// 	if gridInfo == nil {
// 		return fmt.Errorf("网格ID %d 不存在", ID)
// 	}
// 	gridInfo.order = order
// 	return nil
// }
// func (gs *GridState) GetNeedToSubmitOrders(updateStatus bool) []types.SubmitOrder {
// 	orders := make([]types.SubmitOrder, 0)
// 	for _, gridInfo := range gs.GridInfos {
// 		if gridInfo.Status == GridStatusNotPlaced {
// 			orders = append(orders, gridInfo.order)
// 			if updateStatus {
// 				gridInfo.Status = GridStatusPlaced
// 			}
// 		}
// 	}
// 	return orders
// }
// func GetGridIdFromCid(cid string) (int64, error) {
// 	// 查找"G"的位置
// 	gIndex := strings.Index(cid, "G")
// 	if gIndex == -1 {
// 		return 9999, fmt.Errorf("cid %s 中没有找到'G'", cid)
// 	}

// 	// 判断是否是GN（负数）格式
// 	isNegative := false
// 	startIndex := gIndex + 1
// 	if len(cid) > gIndex+1 && cid[gIndex+1] == 'N' {
// 		isNegative = true
// 		startIndex = gIndex + 2
// 	}

// 	// 找到下一个'-'的位置
// 	dashIndex := strings.Index(cid[startIndex:], "-")
// 	if dashIndex == -1 {
// 		return 9999, fmt.Errorf("cid %s 格式不正确，没有找到分隔符'-'", cid)
// 	}

// 	// 提取数字部分
// 	numStr := cid[startIndex : startIndex+dashIndex]

// 	// 转换为int64
// 	gridId, err := strconv.ParseInt(numStr, 10, 64)
// 	if err != nil {
// 		return 9999, fmt.Errorf("cid %s 中数字格式不正确: %v", cid, err)
// 	}

// 	// 如果是GN格式，转换为负数
// 	if isNegative {
// 		gridId = -gridId
// 	}

// 	return gridId, nil
// }
