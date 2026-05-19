package pg

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/martketplace-vkr/analytics/domain"
)

const moneyScale = 100

type Repository struct {
	analyticsDB *sqlx.DB
	orderDB     *sqlx.DB
	catalogDB   *sqlx.DB
}

func New(analyticsDB *sqlx.DB, orderDB *sqlx.DB, catalogDB *sqlx.DB) *Repository {
	return &Repository{
		analyticsDB: analyticsDB,
		orderDB:     orderDB,
		catalogDB:   catalogDB,
	}
}

type sourceOrder struct {
	ID          int64     `db:"id"`
	VendorID    int64     `db:"vendor_id"`
	Status      string    `db:"status"`
	ProductID   int64     `db:"product_id"`
	ProductName string    `db:"product_name"`
	Quantity    int64     `db:"quantity"`
	UnitPrice   string    `db:"unit_price"`
	TotalPrice  string    `db:"total_price"`
	CreatedAt   time.Time `db:"created_at"`
}

type sourceProduct struct {
	ID           int64  `db:"id"`
	VendorID     int64  `db:"vendor_id"`
	CategoryID   int64  `db:"category_id"`
	Name         string `db:"name"`
	Price        string `db:"price"`
	StockCount   int64  `db:"stock_count"`
	CategoryName string `db:"category_name"`
}

type costRow struct {
	VendorID  int64  `db:"vendor_id"`
	ProductID int64  `db:"product_id"`
	CostPrice string `db:"cost_price"`
}

type productAgg struct {
	Day                string
	VendorID           int64
	ProductID          int64
	CategoryID         int64
	OrdersCount        int64
	SalesCount         int64
	DemandUnits        int64
	SoldUnits          int64
	SoldUnitsWithCost  int64
	RevenueCents       int64
	CostCoveredRevenue int64
	CostTotalCents     int64
	GrossProfitCents   int64
}

type nicheAgg struct {
	Day            string
	CategoryID     int64
	OrdersCount    int64
	SalesCount     int64
	DemandUnits    int64
	SoldUnits      int64
	RevenueCents   int64
	ActiveProducts int64
	StockCount     int64
	VendorsCount   int64
}

func (r *Repository) RefreshAggregates(ctx context.Context) error {
	orders, err := r.selectSourceOrders(ctx)
	if err != nil {
		return err
	}

	products, err := r.selectSourceProducts(ctx, 0)
	if err != nil {
		return err
	}

	costs, err := r.selectCosts(ctx)
	if err != nil {
		return err
	}

	productByID := make(map[int64]sourceProduct, len(products))
	for _, product := range products {
		productByID[product.ID] = product
	}

	costByProduct := make(map[string]int64, len(costs))
	for _, cost := range costs {
		costByProduct[costKey(cost.VendorID, cost.ProductID)] = parseMoneyCents(cost.CostPrice)
	}

	productAggs := make(map[string]*productAgg)
	nicheAggs := make(map[string]*nicheAgg)
	seenDays := make(map[string]struct{})

	for _, order := range orders {
		day := order.CreatedAt.Format("2006-01-02")
		seenDays[day] = struct{}{}

		product := productByID[order.ProductID]
		categoryID := product.CategoryID
		if categoryID == 0 {
			categoryID = -1
		}

		if !isCancelledStatus(order.Status) {
			pAgg := getProductAgg(productAggs, day, order.VendorID, order.ProductID, categoryID)
			pAgg.OrdersCount++
			pAgg.DemandUnits += order.Quantity

			nAgg := getNicheAgg(nicheAggs, day, categoryID)
			nAgg.OrdersCount++
			nAgg.DemandUnits += order.Quantity
		}

		if !isSuccessStatus(order.Status) {
			continue
		}

		totalCents := parseMoneyCents(order.TotalPrice)
		pAgg := getProductAgg(productAggs, day, order.VendorID, order.ProductID, categoryID)
		pAgg.SalesCount++
		pAgg.SoldUnits += order.Quantity
		pAgg.RevenueCents += totalCents

		if costCents, ok := costByProduct[costKey(order.VendorID, order.ProductID)]; ok {
			totalCost := costCents * order.Quantity
			pAgg.SoldUnitsWithCost += order.Quantity
			pAgg.CostCoveredRevenue += totalCents
			pAgg.CostTotalCents += totalCost
			pAgg.GrossProfitCents += totalCents - totalCost
		}

		nAgg := getNicheAgg(nicheAggs, day, categoryID)
		nAgg.SalesCount++
		nAgg.SoldUnits += order.Quantity
		nAgg.RevenueCents += totalCents
	}

	if len(seenDays) == 0 {
		seenDays[time.Now().UTC().Format("2006-01-02")] = struct{}{}
	}

	supply := buildSupply(products)
	for day := range seenDays {
		for categoryID, item := range supply {
			nAgg := getNicheAgg(nicheAggs, day, categoryID)
			nAgg.ActiveProducts = item.activeProducts
			nAgg.StockCount = item.stockCount
			nAgg.VendorsCount = int64(len(item.vendors))
		}
	}

	tx, err := r.analyticsDB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err = tx.ExecContext(ctx, `delete from daily_vendor_product_metrics`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `delete from daily_niche_metrics`); err != nil {
		return err
	}

	for _, agg := range productAggs {
		if _, err = tx.ExecContext(ctx, `
			insert into daily_vendor_product_metrics (
				day, vendor_id, product_id, category_id, orders_count, sales_count,
				demand_units, sold_units, sold_units_with_cost, revenue,
				cost_covered_revenue, cost_total, gross_profit
			)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`,
			agg.Day,
			agg.VendorID,
			agg.ProductID,
			agg.CategoryID,
			agg.OrdersCount,
			agg.SalesCount,
			agg.DemandUnits,
			agg.SoldUnits,
			agg.SoldUnitsWithCost,
			formatCents(agg.RevenueCents),
			formatCents(agg.CostCoveredRevenue),
			formatCents(agg.CostTotalCents),
			formatCents(agg.GrossProfitCents),
		); err != nil {
			return err
		}
	}

	for _, agg := range nicheAggs {
		if _, err = tx.ExecContext(ctx, `
			insert into daily_niche_metrics (
				day, category_id, orders_count, sales_count, demand_units,
				sold_units, revenue, active_products, stock_count, vendors_count
			)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`,
			agg.Day,
			agg.CategoryID,
			agg.OrdersCount,
			agg.SalesCount,
			agg.DemandUnits,
			agg.SoldUnits,
			formatCents(agg.RevenueCents),
			agg.ActiveProducts,
			agg.StockCount,
			agg.VendorsCount,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) GetOverview(ctx context.Context, vendorID int64, dateRange domain.DateRange) (domain.Overview, error) {
	kpiRow := struct {
		DemandUnits        int64  `db:"demand_units"`
		SoldUnits          int64  `db:"sold_units"`
		SoldUnitsWithCost  int64  `db:"sold_units_with_cost"`
		OrdersCount        int64  `db:"orders_count"`
		SalesCount         int64  `db:"sales_count"`
		Revenue            string `db:"revenue"`
		GrossProfit        string `db:"gross_profit"`
		CostCoveredRevenue string `db:"cost_covered_revenue"`
	}{}

	err := r.analyticsDB.GetContext(ctx, &kpiRow, `
		select
			coalesce(sum(demand_units), 0) as demand_units,
			coalesce(sum(sold_units), 0) as sold_units,
			coalesce(sum(sold_units_with_cost), 0) as sold_units_with_cost,
			coalesce(sum(orders_count), 0) as orders_count,
			coalesce(sum(sales_count), 0) as sales_count,
			coalesce(sum(revenue), 0)::text as revenue,
			coalesce(sum(gross_profit), 0)::text as gross_profit,
			coalesce(sum(cost_covered_revenue), 0)::text as cost_covered_revenue
		from daily_vendor_product_metrics
		where vendor_id = $1
			and day between $2 and $3
	`, vendorID, dateRange.From, dateRange.To)
	if err != nil {
		return domain.Overview{}, err
	}

	trendRows := []struct {
		Day         time.Time `db:"day"`
		DemandUnits int64     `db:"demand_units"`
		SoldUnits   int64     `db:"sold_units"`
		Revenue     string    `db:"revenue"`
		GrossProfit string    `db:"gross_profit"`
	}{}
	err = r.analyticsDB.SelectContext(ctx, &trendRows, `
		select
			day,
			coalesce(sum(demand_units), 0) as demand_units,
			coalesce(sum(sold_units), 0) as sold_units,
			coalesce(sum(revenue), 0)::text as revenue,
			coalesce(sum(gross_profit), 0)::text as gross_profit
		from daily_vendor_product_metrics
		where vendor_id = $1
			and day between $2 and $3
		group by day
		order by day
	`, vendorID, dateRange.From, dateRange.To)
	if err != nil {
		return domain.Overview{}, err
	}

	trend := make([]domain.DailyTrendPoint, 0, len(trendRows))
	for _, row := range trendRows {
		trend = append(trend, domain.DailyTrendPoint{
			Day:         row.Day.Format("2006-01-02"),
			DemandUnits: row.DemandUnits,
			SoldUnits:   row.SoldUnits,
			Revenue:     normalizeMoneyText(row.Revenue),
			GrossProfit: normalizeMoneyText(row.GrossProfit),
		})
	}

	revenueWithCost := parseMoneyCents(kpiRow.CostCoveredRevenue)
	grossProfit := parseMoneyCents(kpiRow.GrossProfit)

	return domain.Overview{
		KPI: domain.KPI{
			DemandUnits:         kpiRow.DemandUnits,
			SoldUnits:           kpiRow.SoldUnits,
			OrdersCount:         kpiRow.OrdersCount,
			SalesCount:          kpiRow.SalesCount,
			Revenue:             normalizeMoneyText(kpiRow.Revenue),
			GrossProfit:         normalizeMoneyText(kpiRow.GrossProfit),
			MarginPercent:       percent(grossProfit, revenueWithCost),
			CostCoveragePercent: percent(kpiRow.SoldUnitsWithCost, kpiRow.SoldUnits),
		},
		Trend: trend,
	}, nil
}

func (r *Repository) GetProducts(ctx context.Context, vendorID int64, dateRange domain.DateRange) ([]domain.ProductMetric, error) {
	products, err := r.selectSourceProducts(ctx, vendorID)
	if err != nil {
		return nil, err
	}

	costs, err := r.selectCosts(ctx)
	if err != nil {
		return nil, err
	}

	aggRows := []struct {
		ProductID          int64  `db:"product_id"`
		CategoryID         int64  `db:"category_id"`
		DemandUnits        int64  `db:"demand_units"`
		SoldUnits          int64  `db:"sold_units"`
		Revenue            string `db:"revenue"`
		GrossProfit        string `db:"gross_profit"`
		CostCoveredRevenue string `db:"cost_covered_revenue"`
	}{}
	err = r.analyticsDB.SelectContext(ctx, &aggRows, `
		select
			product_id,
			max(category_id) as category_id,
			coalesce(sum(demand_units), 0) as demand_units,
			coalesce(sum(sold_units), 0) as sold_units,
			coalesce(sum(revenue), 0)::text as revenue,
			coalesce(sum(gross_profit), 0)::text as gross_profit,
			coalesce(sum(cost_covered_revenue), 0)::text as cost_covered_revenue
		from daily_vendor_product_metrics
		where vendor_id = $1
			and day between $2 and $3
		group by product_id
	`, vendorID, dateRange.From, dateRange.To)
	if err != nil {
		return nil, err
	}

	aggByProduct := make(map[int64]domain.ProductMetric, len(aggRows))
	revenueWithCostByProduct := make(map[int64]int64, len(aggRows))
	for _, row := range aggRows {
		grossProfit := normalizeMoneyText(row.GrossProfit)
		revenueWithCost := parseMoneyCents(row.CostCoveredRevenue)
		aggByProduct[row.ProductID] = domain.ProductMetric{
			ProductID:     row.ProductID,
			CategoryID:    row.CategoryID,
			DemandUnits:   row.DemandUnits,
			SoldUnits:     row.SoldUnits,
			Revenue:       normalizeMoneyText(row.Revenue),
			GrossProfit:   grossProfit,
			MarginPercent: percent(parseMoneyCents(grossProfit), revenueWithCost),
		}
		revenueWithCostByProduct[row.ProductID] = revenueWithCost
	}

	costByProduct := make(map[string]string, len(costs))
	for _, cost := range costs {
		costByProduct[costKey(cost.VendorID, cost.ProductID)] = normalizeMoneyText(cost.CostPrice)
	}

	result := make([]domain.ProductMetric, 0, len(products)+len(aggRows))
	seen := make(map[int64]struct{}, len(products))
	for _, product := range products {
		metric := aggByProduct[product.ID]
		metric.ProductID = product.ID
		metric.ProductName = product.Name
		metric.CategoryID = product.CategoryID
		metric.CategoryName = product.CategoryName
		metric.StockCount = product.StockCount
		metric.Price = normalizeMoneyText(product.Price)

		if cost, ok := costByProduct[costKey(vendorID, product.ID)]; ok {
			metric.CostPrice = cost
			metric.HasCost = true
		}

		if metric.Revenue == "" {
			metric.Revenue = "0.00"
		}
		if metric.GrossProfit == "" {
			metric.GrossProfit = "0.00"
		}
		if metric.MarginPercent == 0 && revenueWithCostByProduct[product.ID] > 0 {
			metric.MarginPercent = percent(parseMoneyCents(metric.GrossProfit), revenueWithCostByProduct[product.ID])
		}

		result = append(result, metric)
		seen[product.ID] = struct{}{}
	}

	for _, row := range aggRows {
		if _, ok := seen[row.ProductID]; ok {
			continue
		}

		result = append(result, aggByProduct[row.ProductID])
	}

	sort.Slice(result, func(i, j int) bool {
		left := parseMoneyCents(result[i].Revenue)
		right := parseMoneyCents(result[j].Revenue)
		if left == right {
			return result[i].DemandUnits > result[j].DemandUnits
		}
		return left > right
	})

	return result, nil
}

func (r *Repository) GetNiches(ctx context.Context, vendorID int64, dateRange domain.DateRange, sortKey string, limit int) ([]domain.NicheMetric, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	categories, err := r.selectCategories(ctx)
	if err != nil {
		return nil, err
	}

	marketRows := []struct {
		CategoryID     int64  `db:"category_id"`
		DemandUnits    int64  `db:"demand_units"`
		SoldUnits      int64  `db:"sold_units"`
		Revenue        string `db:"revenue"`
		ActiveProducts int64  `db:"active_products"`
		StockCount     int64  `db:"stock_count"`
		VendorsCount   int64  `db:"vendors_count"`
	}{}
	err = r.analyticsDB.SelectContext(ctx, &marketRows, `
		select
			category_id,
			coalesce(sum(demand_units), 0) as demand_units,
			coalesce(sum(sold_units), 0) as sold_units,
			coalesce(sum(revenue), 0)::text as revenue,
			coalesce(max(active_products), 0) as active_products,
			coalesce(max(stock_count), 0) as stock_count,
			coalesce(max(vendors_count), 0) as vendors_count
		from daily_niche_metrics
		where day between $1 and $2
		group by category_id
	`, dateRange.From, dateRange.To)
	if err != nil {
		return nil, err
	}

	ownRows := []struct {
		CategoryID         int64  `db:"category_id"`
		DemandUnits        int64  `db:"demand_units"`
		SoldUnits          int64  `db:"sold_units"`
		Revenue            string `db:"revenue"`
		GrossProfit        string `db:"gross_profit"`
		CostCoveredRevenue string `db:"cost_covered_revenue"`
	}{}
	err = r.analyticsDB.SelectContext(ctx, &ownRows, `
		select
			category_id,
			coalesce(sum(demand_units), 0) as demand_units,
			coalesce(sum(sold_units), 0) as sold_units,
			coalesce(sum(revenue), 0)::text as revenue,
			coalesce(sum(gross_profit), 0)::text as gross_profit,
			coalesce(sum(cost_covered_revenue), 0)::text as cost_covered_revenue
		from daily_vendor_product_metrics
		where vendor_id = $1
			and day between $2 and $3
		group by category_id
	`, vendorID, dateRange.From, dateRange.To)
	if err != nil {
		return nil, err
	}

	ownByCategory := make(map[int64]domain.NicheMetric, len(ownRows))
	for _, row := range ownRows {
		grossProfit := normalizeMoneyText(row.GrossProfit)
		ownByCategory[row.CategoryID] = domain.NicheMetric{
			CategoryID:          row.CategoryID,
			VendorDemandUnits:   row.DemandUnits,
			VendorSoldUnits:     row.SoldUnits,
			VendorRevenue:       normalizeMoneyText(row.Revenue),
			VendorGrossProfit:   grossProfit,
			VendorMarginPercent: percent(parseMoneyCents(grossProfit), parseMoneyCents(row.CostCoveredRevenue)),
		}
	}

	result := make([]domain.NicheMetric, 0, len(marketRows))
	for _, row := range marketRows {
		own := ownByCategory[row.CategoryID]
		metric := domain.NicheMetric{
			CategoryID:          row.CategoryID,
			CategoryName:        categoryName(categories, row.CategoryID),
			MarketDemandUnits:   row.DemandUnits,
			MarketSoldUnits:     row.SoldUnits,
			MarketRevenue:       normalizeMoneyText(row.Revenue),
			ActiveProducts:      row.ActiveProducts,
			StockCount:          row.StockCount,
			VendorsCount:        row.VendorsCount,
			OpportunityScore:    round2(float64(row.DemandUnits) / float64(row.ActiveProducts+1)),
			VendorDemandUnits:   own.VendorDemandUnits,
			VendorSoldUnits:     own.VendorSoldUnits,
			VendorRevenue:       nonEmptyMoney(own.VendorRevenue),
			VendorGrossProfit:   nonEmptyMoney(own.VendorGrossProfit),
			VendorMarginPercent: own.VendorMarginPercent,
		}
		result = append(result, metric)
	}

	sort.Slice(result, func(i, j int) bool {
		switch sortKey {
		case "demand":
			return result[i].MarketDemandUnits > result[j].MarketDemandUnits
		case "margin":
			return result[i].VendorMarginPercent > result[j].VendorMarginPercent
		default:
			if result[i].OpportunityScore == result[j].OpportunityScore {
				return result[i].MarketDemandUnits > result[j].MarketDemandUnits
			}
			return result[i].OpportunityScore > result[j].OpportunityScore
		}
	})

	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (r *Repository) UpsertProductCost(ctx context.Context, cost domain.ProductCost) error {
	_, err := r.analyticsDB.ExecContext(ctx, `
		insert into vendor_product_costs (vendor_id, product_id, cost_price, currency, updated_at)
		values ($1, $2, $3, $4, now())
		on conflict (vendor_id, product_id)
		do update set
			cost_price = excluded.cost_price,
			currency = excluded.currency,
			updated_at = now()
	`, cost.VendorID, cost.ProductID, cost.CostPrice, cost.Currency)
	return err
}

func (r *Repository) VendorOwnsProduct(ctx context.Context, vendorID int64, productID int64) (bool, error) {
	var exists bool
	err := r.catalogDB.GetContext(ctx, &exists, `
		select exists(
			select 1 from products
			where vendor_id = $1
				and id = $2
		)
	`, vendorID, productID)
	return exists, err
}

func (r *Repository) GetSalesReportRows(ctx context.Context, vendorID int64, dateRange domain.DateRange, limit int) ([]domain.SalesReportRow, error) {
	if limit <= 0 {
		limit = 50000
	}

	orders := []sourceOrder{}
	err := r.orderDB.SelectContext(ctx, &orders, `
		select
			id,
			vendor_id,
			status::text as status,
			product_id,
			product_name,
			quantity,
			unit_price,
			total_price,
			created_at
		from "order"."order"
		where vendor_id = $1
			and created_at::date between $2 and $3
		order by created_at desc, id desc
		limit $4
	`, vendorID, dateRange.From, dateRange.To, limit+1)
	if err != nil {
		return nil, err
	}
	if len(orders) > limit {
		return nil, ErrReportTooLarge
	}

	products, err := r.selectSourceProducts(ctx, vendorID)
	if err != nil {
		return nil, err
	}
	productByID := make(map[int64]sourceProduct, len(products))
	for _, product := range products {
		productByID[product.ID] = product
	}

	costs, err := r.selectCosts(ctx)
	if err != nil {
		return nil, err
	}
	costByProduct := make(map[string]string, len(costs))
	for _, cost := range costs {
		costByProduct[costKey(cost.VendorID, cost.ProductID)] = normalizeMoneyText(cost.CostPrice)
	}

	rows := make([]domain.SalesReportRow, 0, len(orders))
	for _, order := range orders {
		product := productByID[order.ProductID]
		cost := costByProduct[costKey(order.VendorID, order.ProductID)]
		grossProfit := "0.00"
		margin := 0.0

		if isSuccessStatus(order.Status) && cost != "" {
			revenueCents := parseMoneyCents(order.TotalPrice)
			costTotalCents := parseMoneyCents(cost) * order.Quantity
			grossProfit = formatCents(revenueCents - costTotalCents)
			margin = percent(parseMoneyCents(grossProfit), revenueCents)
		}

		rows = append(rows, domain.SalesReportRow{
			Day:           order.CreatedAt.Format("2006-01-02"),
			OrderID:       order.ID,
			Status:        order.Status,
			ProductID:     order.ProductID,
			ProductName:   order.ProductName,
			CategoryName:  product.CategoryName,
			Quantity:      order.Quantity,
			UnitPrice:     normalizeMoneyText(order.UnitPrice),
			TotalPrice:    normalizeMoneyText(order.TotalPrice),
			CostPrice:     cost,
			GrossProfit:   grossProfit,
			MarginPercent: margin,
		})
	}

	return rows, nil
}

func (r *Repository) selectSourceOrders(ctx context.Context) ([]sourceOrder, error) {
	orders := []sourceOrder{}
	err := r.orderDB.SelectContext(ctx, &orders, `
		select
			id,
			vendor_id,
			status::text as status,
			product_id,
			product_name,
			quantity,
			unit_price,
			total_price,
			created_at
		from "order"."order"
	`)
	return orders, err
}

func (r *Repository) selectSourceProducts(ctx context.Context, vendorID int64) ([]sourceProduct, error) {
	products := []sourceProduct{}
	err := r.catalogDB.SelectContext(ctx, &products, `
		select
			p.id,
			p.vendor_id,
			p.category_id,
			p.name,
			p.price::text as price,
			p.stock_count,
			coalesce(c.name, 'Категория ' || p.category_id::text) as category_name
		from products p
		left join categories c on c.id = p.category_id
		where ($1 = 0 or p.vendor_id = $1)
	`, vendorID)
	return products, err
}

func (r *Repository) selectCategories(ctx context.Context) (map[int64]string, error) {
	rows := []struct {
		ID   int64  `db:"id"`
		Name string `db:"name"`
	}{}
	err := r.catalogDB.SelectContext(ctx, &rows, `select id, name from categories`)
	if err != nil {
		return nil, err
	}

	result := make(map[int64]string, len(rows))
	for _, row := range rows {
		result[row.ID] = row.Name
	}
	return result, nil
}

func (r *Repository) selectCosts(ctx context.Context) ([]costRow, error) {
	costs := []costRow{}
	err := r.analyticsDB.SelectContext(ctx, &costs, `
		select vendor_id, product_id, cost_price::text as cost_price
		from vendor_product_costs
	`)
	return costs, err
}

type supplyItem struct {
	activeProducts int64
	stockCount     int64
	vendors        map[int64]struct{}
}

func buildSupply(products []sourceProduct) map[int64]supplyItem {
	result := make(map[int64]supplyItem)
	for _, product := range products {
		if product.StockCount <= 0 {
			continue
		}

		categoryID := product.CategoryID
		if categoryID == 0 {
			categoryID = -1
		}

		item := result[categoryID]
		if item.vendors == nil {
			item.vendors = make(map[int64]struct{})
		}
		item.activeProducts++
		item.stockCount += product.StockCount
		item.vendors[product.VendorID] = struct{}{}
		result[categoryID] = item
	}
	return result
}

func getProductAgg(values map[string]*productAgg, day string, vendorID int64, productID int64, categoryID int64) *productAgg {
	key := fmt.Sprintf("%s:%d:%d", day, vendorID, productID)
	if values[key] == nil {
		values[key] = &productAgg{
			Day:        day,
			VendorID:   vendorID,
			ProductID:  productID,
			CategoryID: categoryID,
		}
	}
	return values[key]
}

func getNicheAgg(values map[string]*nicheAgg, day string, categoryID int64) *nicheAgg {
	key := fmt.Sprintf("%s:%d", day, categoryID)
	if values[key] == nil {
		values[key] = &nicheAgg{
			Day:        day,
			CategoryID: categoryID,
		}
	}
	return values[key]
}

func costKey(vendorID int64, productID int64) string {
	return fmt.Sprintf("%d:%d", vendorID, productID)
}

func isCancelledStatus(status string) bool {
	normalized := strings.ToLower(strings.TrimSpace(status))
	return strings.HasPrefix(normalized, "cancel") || strings.HasPrefix(normalized, "canecel")
}

func isSuccessStatus(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "success")
}

func parseMoneyCents(value string) int64 {
	normalized := strings.TrimSpace(strings.ReplaceAll(value, " ", ""))
	if normalized == "" {
		return 0
	}
	normalized = strings.ReplaceAll(normalized, ",", ".")
	parsed, err := strconv.ParseFloat(normalized, 64)
	if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) {
		return 0
	}
	return int64(math.Round(parsed * moneyScale))
}

func formatCents(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	return fmt.Sprintf("%s%d.%02d", sign, value/moneyScale, value%moneyScale)
}

func normalizeMoneyText(value string) string {
	return formatCents(parseMoneyCents(value))
}

func nonEmptyMoney(value string) string {
	if strings.TrimSpace(value) == "" {
		return "0.00"
	}
	return normalizeMoneyText(value)
}

func percent(numerator int64, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return round2(float64(numerator) / float64(denominator) * 100)
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func categoryName(categories map[int64]string, categoryID int64) string {
	if categoryID == -1 {
		return "Без категории"
	}
	if name := strings.TrimSpace(categories[categoryID]); name != "" {
		return name
	}
	return fmt.Sprintf("Категория %d", categoryID)
}
