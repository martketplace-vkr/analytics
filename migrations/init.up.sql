create table if not exists vendor_product_costs (
    vendor_id bigint not null,
    product_id bigint not null,
    cost_price numeric(10,2) not null check (cost_price >= 0),
    currency text not null default 'RUB',
    updated_at timestamptz not null default now(),
    constraint pk_vendor_product_costs primary key (vendor_id, product_id)
);

create table if not exists daily_vendor_product_metrics (
    day date not null,
    vendor_id bigint not null,
    product_id bigint not null,
    category_id bigint not null,
    orders_count bigint not null default 0,
    sales_count bigint not null default 0,
    demand_units bigint not null default 0,
    sold_units bigint not null default 0,
    sold_units_with_cost bigint not null default 0,
    revenue numeric(14,2) not null default 0,
    cost_covered_revenue numeric(14,2) not null default 0,
    cost_total numeric(14,2) not null default 0,
    gross_profit numeric(14,2) not null default 0,
    constraint pk_daily_vendor_product_metrics primary key (day, vendor_id, product_id)
);

create index if not exists daily_vendor_product_metrics_day_idx
    on daily_vendor_product_metrics (day);

create index if not exists daily_vendor_product_metrics_vendor_day_idx
    on daily_vendor_product_metrics (vendor_id, day);

create index if not exists daily_vendor_product_metrics_product_idx
    on daily_vendor_product_metrics (product_id);

create index if not exists daily_vendor_product_metrics_category_idx
    on daily_vendor_product_metrics (category_id);

create table if not exists daily_niche_metrics (
    day date not null,
    category_id bigint not null,
    orders_count bigint not null default 0,
    sales_count bigint not null default 0,
    demand_units bigint not null default 0,
    sold_units bigint not null default 0,
    revenue numeric(14,2) not null default 0,
    active_products bigint not null default 0,
    stock_count bigint not null default 0,
    vendors_count bigint not null default 0,
    constraint pk_daily_niche_metrics primary key (day, category_id)
);

create index if not exists daily_niche_metrics_day_idx
    on daily_niche_metrics (day);

create index if not exists daily_niche_metrics_category_idx
    on daily_niche_metrics (category_id);
