package gridv2

import (
	"fmt"
	"testing"
)

func TestGenerateGridInfoPirce(t *testing.T) {
	gs := &GridState{}
	gs.GenerateGridInfoPirce(62624, 90000, 50000, 20, 0)
	gs.GenGridInfoOrders(true)
	err := gs.UpdateGridInfoByFilledId(-1)
	if err != nil {
		t.Fatal(err)
	}
	gs.GenGridInfoOrders(true)
	err = gs.UpdateGridInfoByFilledId(-2)
	if err != nil {
		t.Fatal(err)
	}
	gs.GenGridInfoOrders(false)
	grids := MapToSlice(gs.GridInfos)
	// spew.Dump(grids)

	// 按序输出
	fmt.Println("按价格排序后的网格信息：")
	fmt.Println("网格序号\t价格\t\t方向")
	fmt.Println("----------------------------------------")
	for _, grid := range grids {
		fmt.Printf("%d\t%v\t%v\t%v\n", grid.Index, grid.Info.Price, grid.Info.Short, grid.Info.Status)
	}
}

// }
// func TestGenerateGridInfos(t *testing.T) {
// 	gs := &GridState{}
// 	// gs.GenerateGridInfos(fixedpoint.NewFromInt(9990), fixedpoint.NewFromInt(11000), fixedpoint.NewFromInt(9000), 20, types.SideTypeBoth)
// 	gs.GenerateGridInfos(fixedpoint.NewFromInt(10010), fixedpoint.NewFromInt(11000), fixedpoint.NewFromInt(9000), 20, fixedpoint.NewFromInt(1000), "BTCUSDT", types.Market{
// 		Symbol:        "BTCUSDT",
// 		BaseCurrency:  "BTC",
// 		QuoteCurrency: "USDT",
// 	}, 1)
// 	grids := MapToSlice(gs.GridInfos)
// 	// spew.Dump(grids)

// 	// // 按序输出
// 	// fmt.Println("按价格排序后的网格信息：")
// 	// fmt.Println("网格序号\t价格\t\t方向")
// 	// fmt.Println("----------------------------------------")
// 	// for _, grid := range grids {
// 	// 	fmt.Printf("%d\t%s\t%s\n",
// 	// 		grid.Index,
// 	// 		grid.Info.Price.String(),
// 	// 		grid.Info.Side)
// 	// }
// 	spew.Dump(grids)
// 	orders := gs.GetNeedToSubmitOrders(true)
// 	spew.Dump(orders)
// }

// func TestUpdateGridInfoByFilledId(t *testing.T) {
// 	gs := &GridState{}
// 	// gs.GenerateGridInfos(fixedpoint.NewFromInt(9990), fixedpoint.NewFromInt(11000), fixedpoint.NewFromInt(9000), 20, types.SideTypeBoth)
// 	gs.GenerateGridInfos(fixedpoint.NewFromInt(10010), fixedpoint.NewFromInt(11000), fixedpoint.NewFromInt(9000), 20, fixedpoint.NewFromInt(1000), "BTCUSDT", types.Market{
// 		Symbol:        "BTCUSDT",
// 		BaseCurrency:  "BTC",
// 		QuoteCurrency: "USDT",
// 	}, 1)
// 	gs.GetNeedToSubmitOrders(true)
// 	gs.UpdateGridInfoByFilledId(-1)
// 	gs.GetNeedToSubmitOrders(true)
// 	gs.UpdateGridInfoByFilledId(-2)
// 	// spew.Dump(orders)
// 	grids := MapToSlice(gs.GridInfos)
// 	// spew.Dump(grids)

// 	// 按序输出
// 	fmt.Println("按价格排序后的网格信息：")
// 	fmt.Println("网格序号\t价格\t\t方向")
// 	fmt.Println("----------------------------------------")
// 	for _, grid := range grids {
// 		fmt.Printf("%d\t%s\t%s\t%s\t%s\t%s\n",
// 			grid.Index,
// 			grid.Info.Price.String(),
// 			grid.Info.Status,
// 			grid.Info.Side,
// 			grid.Info.order.Side,
// 			grid.Info.order.ClientOrderID,
// 		)
// 	}
// 	order := gs.GetNeedToSubmitOrders(true)
// 	spew.Dump(order)
// 	grids = MapToSlice(gs.GridInfos)

// 	// 按序输出
// 	fmt.Println("按价格排序后的网格信息：")
// 	fmt.Println("网格序号\t价格\t\t方向")
// 	fmt.Println("----------------------------------------")
// 	for _, grid := range grids {
// 		fmt.Printf("%d\t%s\t%s\t%s\t%s\t%s\n",
// 			grid.Index,
// 			grid.Info.Price.String(),
// 			grid.Info.Status,
// 			grid.Info.Side,
// 			grid.Info.order.Side,
// 			grid.Info.order.ClientOrderID,
// 		)
// 	}
// }
// func TestGetGridIdFromCid(t *testing.T) {
// 	cid := "BTCUSDT-G-2-1744021943367007858"
// 	gridId, err := GetGridIdFromCid(cid)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	spew.Dump(gridId)
// }
