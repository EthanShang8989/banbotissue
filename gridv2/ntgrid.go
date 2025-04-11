package gridv2

import (
	"strconv"

	"github.com/banbox/banbot/config"
	"github.com/banbox/banbot/core"
	"github.com/banbox/banbot/orm/ormo"
	"github.com/banbox/banbot/strat"
	"github.com/banbox/banexg/log"
	"go.uber.org/zap"
)

/*
NeutralGrid 中性网格策略
在指定价格区间内创建均匀分布的网格，上涨卖出，下跌买入
适合震荡市场，不适合单边趋势市场
*/

func init() {
	strat.AddStratGroup("gridv2", map[string]strat.FuncMakeStrat{
		"ntgrid": NtGrid,
	})
}

// Grid 网格结构体
type Grid struct {
	// 网格基础配置
	UpperPrice float64 // 网格上限价格
	LowerPrice float64 // 网格下限价格
	GridNum    int     // 网格数量
	GridGap    float64 // 网格间距

	// // 仓位管理
	// InitAmount   float64 // 初始开仓数量
	// MaxPositions int     // 最大持仓数量

	// // 网格状态
	// IsActive   bool              // 网格是否激活
	// GridPrices []float64         // 网格价格数组
	// GridOrders map[float64]int64 // 网格价格对应的订单ID
	GridState *GridState

	// 止损设置
	StopLossRate  float64 // 止损比例
	StopLossPrice float64 // 止损价格
}

// 创建新的网格实例
func NewGrid(upperPrice, lowerPrice float64, gridNum int, initQuoteAmount float64, maxPositions int) *Grid {
	gridGap := (upperPrice - lowerPrice) / float64(gridNum-1)

	return &Grid{
		UpperPrice: upperPrice,
		LowerPrice: lowerPrice,
		GridNum:    gridNum,
		GridGap:    gridGap,
	}
}

// BasicNeutralGrid 基础中性网格策略
func NtGrid(pol *config.RunPolicyConfig) *strat.TradeStrat {
	// 从策略参数中读取配置
	gridNum := int(pol.Def("gridNum", 5, core.PNorm(3, 20)))
	upperPrice := pol.Def("upperPrice", 10000, core.PNorm(1000, 100000))
	lowerPrice := pol.Def("lowerPrice", 1000, core.PNorm(100, 10000))
	initQuoteAmount := pol.Def("initAmount", 1.0, core.PNorm(0.1, 5))
	maxPositions := int(pol.Def("maxPositions", 5, core.PNorm(1, 10)))
	// stopLossRate := pol.Def("stopLossRate", 0.05, core.PNorm(0.01, 0.2))

	return &strat.TradeStrat{
		Name:      "NeutralGrid",
		WarmupNum: 0,
		// OnPairInfos: func(s *strat.StratJob) []*strat.PairSub {
		// 	return []*strat.PairSub{
		// 		{"_cur_", "15m", 100},
		// 	}
		// },
		OnStartUp: func(s *strat.StratJob) {
			// 初始化基本数据，但是没初始化网格数据，因为不知道起始价格
			log.Info("初始化策略状态")
			s.More = NewGrid(upperPrice, lowerPrice, gridNum, initQuoteAmount, maxPositions)
		},
		OnBar: func(s *strat.StratJob) {
			g, _ := s.More.(*Grid)
			// log.Info("onbar...")
			if g.GridState == nil {
				log.Info("初始化网格状态", zap.String("symbol", s.Symbol.Symbol))
				g.GridState = NewGridState(s.Env.Close.Get(0), upperPrice, lowerPrice, int64(gridNum), initQuoteAmount, s.Symbol.Symbol)
				openOrders, closeOrders, err := g.GridState.GenGridInfoOrders(true)
				grids := MapToSlice(g.GridState.GridInfos)
				// spew.Dump(grids)
				// 按序输出
				// log.Info("网格状态:", zap.Any("gridInfos", g.GridState.GridInfos))
				log.Info("按价格排序后的网格信息：")
				log.Info("网格序号\t价格\t\t方向")
				for _, grid := range grids {
					log.Info("网格信息:", zap.Int64("序号", grid.Index),
						zap.Float64("价格", grid.Info.Price),
						zap.Bool("方向", grid.Info.Short))
				}
				if err != nil {
					log.Error("生成网格订单失败", zap.Error(err))
				}
				log.Info("openOrders", zap.Any("openOrders", openOrders))
				log.Info("closeOrders", zap.Any("closeOrders", closeOrders))
				for _, order := range openOrders {
					err := s.OpenOrder(order)
					if err != nil {
						log.Error("开仓失败", zap.Error(err))
					}
				}
				for _, order := range closeOrders {
					err := s.CloseOrders(order)
					if err != nil {
						log.Error("平仓失败", zap.Error(err))
					}
				}
			}
		},
		OnOrderChange: func(s *strat.StratJob, od *ormo.InOutOrder, chgType int) {
			g, _ := s.More.(*Grid)
			log.Info("onorderchange", zap.Any("od", od), zap.Int("chgType", chgType))
			switch chgType {
			case strat.OdChgEnterFill, strat.OdChgExitFill:
				var filledGridId int64
				var err error
				if chgType == strat.OdChgEnterFill {
					filledGridId, err = strconv.ParseInt(od.EnterTag, 10, 64)
					if err != nil {
						log.Error("转换网格ID失败", zap.Error(err))
					}
				} else {
					filledGridId, err = strconv.ParseInt(od.ExitTag, 10, 64)
					if err != nil {
						log.Error("转换网格ID失败", zap.Error(err))
					}
				}
				g.GridState.UpdateGridInfoByFilledId(filledGridId)
				log.Info("网格订单完成", zap.Any("filledGridId", filledGridId))
				openOrders, closeOrders, err := g.GridState.GenGridInfoOrders(true)
				if err != nil {
					log.Error("生成网格订单失败", zap.Error(err))
				}
				log.Info("新的开仓单", zap.Any("openOrders", openOrders))
				log.Info("新的平仓单", zap.Any("closeOrders", closeOrders))
				for _, order := range openOrders {
					err := s.OpenOrder(order)
					if err != nil {
						log.Error("开仓失败", zap.Error(err))
					}
				}
				for _, order := range closeOrders {
					err := s.CloseOrders(order)
					if err != nil {
						log.Error("平仓失败", zap.Error(err))
					}
				}
			}
		},
	}
}

// // PlaceInitialOrders 放置初始买单和卖单
// func (g *Grid) PlaceInitialOrders(s *strat.StratJob) {
// 	currentPrice := s.Env.Close.Get(0)

// 	// 找到当前价格所在网格
// 	currentIndex := g.FindCurrentGridIndex(currentPrice)

// 	// 为低于当前价格的网格放置买单
// 	for i := 0; i < currentIndex; i++ {
// 		gridPrice := g.GridPrices[i]
// 		g.PlaceBuyOrder(s, gridPrice)
// 	}

// 	// 为高于当前价格的网格放置卖单
// 	for i := currentIndex + 1; i < len(g.GridPrices); i++ {
// 		gridPrice := g.GridPrices[i]
// 		g.PlaceSellOrder(s, gridPrice)
// 	}
// }

// // CheckGridOrders 检查网格订单状态
// func (g *Grid) CheckGridOrders(s *strat.StratJob) {
// 	currentPrice := s.Env.Close.Get(0)

// 	// 检查所有网格价格并填补缺失的订单
// 	for _, price := range g.GridPrices {
// 		if price < currentPrice {
// 			// 应该有买单
// 			if _, exists := g.GridOrders[price]; !exists {
// 				g.PlaceBuyOrder(s, price)
// 			}
// 		} else if price > currentPrice {
// 			// 应该有卖单
// 			if _, exists := g.GridOrders[price]; !exists {
// 				g.PlaceSellOrder(s, price)
// 			}
// 		}
// 	}
// }

// // PlaceBuyOrder 放置买单
// func (g *Grid) PlaceBuyOrder(s *strat.StratJob, price float64) {
// 	if s.OrderNum >= g.MaxPositions {
// 		return // 达到最大订单数量
// 	}

// 	// 如果已有此价格的订单，跳过
// 	if _, exists := g.GridOrders[price]; exists {
// 		return
// 	}

// 	// // 创建买单
// 	// req := &strat.EnterReq{
// 	// 	Tag:    fmt.Sprintf("grid_buy_%.4f", price),
// 	// 	Short:  false, // 做多
// 	// 	Limit:  price, // 限价单
// 	// 	Amount: g.InitAmount,
// 	// 	// StopLossPrice: g.StopLossPrice,
// 	// }

// 	// 开单
// 	// orderID := s.OpenOrder(req)
// 	// g.GridOrders[price] = orderID
// 	// fmt.Printf("放置买单: 价格=%.4f, 数量=%.4f, 订单ID=%d\n", price, g.InitAmount, orderID)
// }

// // PlaceSellOrder 放置卖单
// func (g *Grid) PlaceSellOrder(s *strat.StratJob, price float64) {
// 	if s.OrderNum >= g.MaxPositions {
// 		return // 达到最大订单数量
// 	}

// 	// 如果已有此价格的订单，跳过
// 	if _, exists := g.GridOrders[price]; exists {
// 		return
// 	}

// 	// // 创建卖单
// 	// req := &strat.EnterReq{
// 	// 	Tag:    fmt.Sprintf("grid_sell_%.4f", price),
// 	// 	Short:  true,  // 做空
// 	// 	Limit:  price, // 限价单
// 	// 	Amount: g.InitAmount,
// 	// 	// StopLossPrice: math.Max(g.UpperPrice*(1+g.StopLossRate), price*(1+g.StopLossRate)),
// 	// }

// 	// // 开单
// 	// orderID := s.OpenOrder(req)
// 	// // g.GridOrders[price] = orderID
// 	// fmt.Printf("放置卖单: 价格=%.4f, 数量=%.4f, 订单ID=%d\n", price, g.InitAmount, orderID)
// }

// // HandleOrderFilled 处理订单成交
// func (g *Grid) HandleOrderFilled(s *strat.StratJob, od *ormo.InOutOrder, isEnter bool) {
// 	// currentPrice := s.Env.Close.Get(0)

// 	if isEnter {
// 		// 处理入场成交
// 		entryPrice := od.Enter.Average
// 		tag := od.EnterTag
// 		fmt.Printf("订单入场成交: ID=%d, 价格=%.4f, 标签=%s\n", od.ID, entryPrice, tag)

// 		// 从映射中删除此订单
// 		for price, id := range g.GridOrders {
// 			if id == od.ID {
// 				delete(g.GridOrders, price)
// 				break
// 			}
// 		}

// 		// 如果成交的是买单，在上方放置卖单
// 		// 如果成交的是卖单，在下方放置买单
// 		if !od.Short {
// 			// 买单成交，放置卖单
// 			sellPrice := entryPrice + g.GridGap
// 			if sellPrice <= g.UpperPrice {
// 				g.PlaceSellOrder(s, sellPrice)
// 			}
// 		} else {
// 			// 卖单成交，放置买单
// 			buyPrice := entryPrice - g.GridGap
// 			if buyPrice >= g.LowerPrice {
// 				g.PlaceBuyOrder(s, buyPrice)
// 			}
// 		}
// 	} else {
// 		// 处理退出成交
// 		exitPrice := od.Exit.Average
// 		fmt.Printf("订单退出成交: ID=%d, 价格=%.4f, 标签=%s\n", od.ID, exitPrice, od.ExitTag)

// 		// 根据退出价格放置新订单
// 		gridIndex := g.FindCurrentGridIndex(exitPrice)
// 		if gridIndex >= 0 && gridIndex < g.GridNum {
// 			// 检查是否需要在此位置放置新订单
// 			g.CheckGridOrders(s)
// 		}
// 	}
// }
