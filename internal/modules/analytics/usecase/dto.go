package usecase

import (
	"github.com/UmedjonQurbonov/CRM/internal/modules/analytics/domain"
)

// AnalyticsSummaryDTO represents the API response payload for financial P&L.
type AnalyticsSummaryDTO struct {
	PeriodFrom       string `json:"period_from" example:"2026-09-01"`
	PeriodTo         string `json:"period_to" example:"2026-09-22"`
	Revenue          string `json:"revenue" example:"12500.00"`
	CostOfGoodsSold  string `json:"cost_of_goods_sold" example:"7500.00"`
	GrossProfit      string `json:"gross_profit" example:"5000.00"`
	TotalCommissions string `json:"total_commissions" example:"850.00"`
	TotalExpenses    string `json:"total_expenses" example:"1200.00"`
	NetProfit        string `json:"net_profit" example:"2950.00"`
	TotalOrders      int    `json:"total_orders" example:"120"`
}

// SellerRankingDTO represents individual seller performance within the store.
type SellerRankingDTO struct {
	SellerID               string `json:"seller_id" example:"9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"`
	SellerName             string `json:"seller_name" example:"Umedjon Qurbonov"`
	TotalOrders            int    `json:"total_orders" example:"45"`
	TotalRevenue           string `json:"total_revenue" example:"6500.00"`
	CommissionEarned       string `json:"commission_earned" example:"552.50"`
	RevenueSharePercentage string `json:"revenue_share_percentage" example:"52.00%"`
}

// TopProductDTO represents a best-selling product aggregated for a period.
type TopProductDTO struct {
	ProductID         string `json:"product_id" example:"70cf0d01-fca2-45ff-bf86-99602ea12f39"`
	ProductName       string `json:"product_name" example:"Classic White T-Shirt"`
	SKU               string `json:"sku" example:"TSHIRT-WHT-001"`
	TotalQuantitySold int    `json:"total_quantity_sold" example:"85"`
	TotalRevenue      string `json:"total_revenue" example:"12750.00"`
}

// SellerEarningsDTO represents personal earnings metrics for a seller.
type SellerEarningsDTO struct {
	SellerID              string `json:"seller_id" example:"9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"`
	SellerName            string `json:"seller_name" example:"Umedjon Qurbonov"`
	CommissionRate        string `json:"commission_rate" example:"8.50%"`
	TotalSalesAmount      string `json:"total_sales_amount" example:"6500.00"`
	TotalCommissionEarned string `json:"total_commission_earned" example:"552.50"`
	OrdersCount           int    `json:"orders_count" example:"45"`
	PeriodFrom            string `json:"period_from" example:"2026-09-01"`
	PeriodTo              string `json:"period_to" example:"2026-09-22"`
}

// ToAnalyticsSummaryDTO converts domain.AnalyticsSummary into AnalyticsSummaryDTO.
func ToAnalyticsSummaryDTO(s *domain.AnalyticsSummary, fromStr, toStr string) AnalyticsSummaryDTO {
	return AnalyticsSummaryDTO{
		PeriodFrom:       fromStr,
		PeriodTo:         toStr,
		Revenue:          s.Revenue.StringFixed(2),
		CostOfGoodsSold:  s.CostOfGoodsSold.StringFixed(2),
		GrossProfit:      s.GrossProfit.StringFixed(2),
		TotalCommissions: s.TotalCommissions.StringFixed(2),
		TotalExpenses:    s.TotalExpenses.StringFixed(2),
		NetProfit:        s.NetProfit.StringFixed(2),
		TotalOrders:      s.TotalOrders,
	}
}

// ToSellerRankingDTO converts domain.SellerRanking into SellerRankingDTO.
func ToSellerRankingDTO(r *domain.SellerRanking) SellerRankingDTO {
	return SellerRankingDTO{
		SellerID:               r.SellerID.String(),
		SellerName:             r.SellerName,
		TotalOrders:            r.TotalOrders,
		TotalRevenue:           r.TotalRevenue.StringFixed(2),
		CommissionEarned:       r.CommissionEarned.StringFixed(2),
		RevenueSharePercentage: r.RevenueSharePercentage.StringFixed(2) + "%",
	}
}

// ToTopProductDTO converts domain.TopProduct into TopProductDTO.
func ToTopProductDTO(p *domain.TopProduct) TopProductDTO {
	return TopProductDTO{
		ProductID:         p.ProductID.String(),
		ProductName:       p.ProductName,
		SKU:               p.SKU,
		TotalQuantitySold: p.TotalQuantitySold,
		TotalRevenue:      p.TotalRevenue.StringFixed(2),
	}
}

// ToSellerEarningsDTO converts domain.SellerEarningsSummary into SellerEarningsDTO.
func ToSellerEarningsDTO(e *domain.SellerEarningsSummary, fromStr, toStr string) SellerEarningsDTO {
	return SellerEarningsDTO{
		SellerID:              e.SellerID.String(),
		SellerName:            e.SellerName,
		CommissionRate:        e.CommissionRate.StringFixed(2) + "%",
		TotalSalesAmount:      e.TotalSalesAmount.StringFixed(2),
		TotalCommissionEarned: e.TotalCommissionEarned.StringFixed(2),
		OrdersCount:           e.OrdersCount,
		PeriodFrom:            fromStr,
		PeriodTo:              toStr,
	}
}
