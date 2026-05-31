package domain

import "time"

type DateRange struct {
	From time.Time
	To   time.Time
}

type KPI struct {
	DemandUnits         int64
	SoldUnits           int64
	OrdersCount         int64
	SalesCount          int64
	Revenue             string
	GrossProfit         string
	MarketplaceFee      string
	NetProfit           string
	MarginPercent       float64
	NetMarginPercent    float64
	CostCoveragePercent float64
}

type DailyTrendPoint struct {
	Day            string
	DemandUnits    int64
	SoldUnits      int64
	Revenue        string
	GrossProfit    string
	MarketplaceFee string
	NetProfit      string
}

type ProductDailyTrend struct {
	ProductID   int64
	ProductName string
	Points      []ProductDailyTrendPoint
}

type ProductDailyTrendPoint struct {
	Day       string
	SoldUnits int64
}

type Overview struct {
	KPI           KPI
	Trend         []DailyTrendPoint
	ProductTrends []ProductDailyTrend
	Tariff        Tariff
}

type NicheMetric struct {
	CategoryID             int64
	CategoryName           string
	MarketDemandUnits      int64
	MarketSoldUnits        int64
	MarketRevenue          string
	ActiveProducts         int64
	StockCount             int64
	VendorsCount           int64
	OpportunityScore       float64
	VendorDemandUnits      int64
	VendorSoldUnits        int64
	VendorRevenue          string
	VendorGrossProfit      string
	VendorMarketplaceFee   string
	VendorNetProfit        string
	VendorMarginPercent    float64
	VendorNetMarginPercent float64
}

type ProductMetric struct {
	ProductID        int64
	ProductName      string
	CategoryID       int64
	CategoryName     string
	StockCount       int64
	Price            string
	CostPrice        string
	HasCost          bool
	DemandUnits      int64
	SoldUnits        int64
	Revenue          string
	GrossProfit      string
	MarketplaceFee   string
	NetProfit        string
	MarginPercent    float64
	NetMarginPercent float64
}

type ProductCost struct {
	VendorID  int64
	ProductID int64
	CostPrice string
	Currency  string
}

type Tariff struct {
	ID                int64
	Name              string
	CommissionPercent string
	IsDefault         bool
	AssignedVendors   int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type VendorTariffAssignment struct {
	VendorID int64
	Tariff   Tariff
}

type SalesReportRow struct {
	Day              string
	OrderID          int64
	Status           string
	ProductID        int64
	ProductName      string
	CategoryName     string
	Quantity         int64
	UnitPrice        string
	TotalPrice       string
	CostPrice        string
	GrossProfit      string
	MarketplaceFee   string
	NetProfit        string
	MarginPercent    float64
	NetMarginPercent float64
}

type SalesReport struct {
	Content     []byte
	Filename    string
	ContentType string
}
