package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AnalyticsSummary represents the aggregated financial Profit & Loss statement for a period.
type AnalyticsSummary struct {
	Revenue          decimal.Decimal
	CostOfGoodsSold  decimal.Decimal
	GrossProfit      decimal.Decimal
	TotalCommissions decimal.Decimal
	TotalExpenses    decimal.Decimal
	NetProfit        decimal.Decimal
	TotalOrders      int
}

// CalculatePL computes Gross Profit and Net Profit according to business rules:
// GP = Revenue - CostOfGoodsSold
// NP = GrossProfit - TotalExpenses - TotalCommissions
func (s *AnalyticsSummary) CalculatePL() {
	s.GrossProfit = s.Revenue.Sub(s.CostOfGoodsSold)
	s.NetProfit = s.GrossProfit.Sub(s.TotalExpenses).Sub(s.TotalCommissions)
}

// SellerRanking represents sales volume, revenue share, and earnings for a seller in a period.
type SellerRanking struct {
	SellerID               uuid.UUID
	SellerName             string
	TotalOrders            int
	TotalRevenue           decimal.Decimal
	CommissionEarned       decimal.Decimal
	RevenueSharePercentage decimal.Decimal
}

// TopProduct represents a best-selling product aggregated across completed orders.
type TopProduct struct {
	ProductID         uuid.UUID
	ProductName       string
	SKU               string
	TotalQuantitySold int
	TotalRevenue      decimal.Decimal
}

// SellerEarningsSummary represents the personal earnings and performance metrics of a seller.
type SellerEarningsSummary struct {
	SellerID              uuid.UUID
	SellerName            string
	CommissionRate        decimal.Decimal
	TotalSalesAmount      decimal.Decimal
	TotalCommissionEarned decimal.Decimal
	OrdersCount           int
}
