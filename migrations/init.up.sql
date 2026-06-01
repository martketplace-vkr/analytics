create table if not exists vendor_product_costs (
    vendor_id bigint not null,
    product_id bigint not null,
    cost_price numeric(10,2) not null check (cost_price >= 0),
    currency text not null default 'RUB',
    updated_at timestamptz not null default now(),
    constraint pk_vendor_product_costs primary key (vendor_id, product_id)
);

create table if not exists vendor_tariffs (
    id bigserial primary key,
    name text not null,
    commission_percent numeric(5,2) not null check (commission_percent >= 0 and commission_percent <= 100),
    is_default boolean not null default false,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create unique index if not exists vendor_tariffs_single_default_idx
    on vendor_tariffs (is_default)
    where is_default;

insert into vendor_tariffs (name, commission_percent, is_default)
select 'Default', 5.00, true
where not exists (select 1 from vendor_tariffs where is_default);

create table if not exists vendor_tariff_assignments (
    vendor_id bigint primary key,
    tariff_id bigint not null references vendor_tariffs(id),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists vendor_tariff_assignments_tariff_idx
    on vendor_tariff_assignments (tariff_id);

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
    marketplace_fee numeric(14,2) not null default 0,
    net_profit numeric(14,2) not null default 0,
    views_count bigint not null default 0,
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

create table if not exists product_view_events (
    id bigserial primary key,
    product_id bigint not null,
    vendor_id bigint not null,
    category_id bigint not null,
    visitor_id text not null,
    view_day date not null default current_date,
    viewed_at timestamptz not null default now(),
    created_at timestamptz not null default now()
);

create index if not exists product_view_events_vendor_viewed_idx
    on product_view_events (vendor_id, viewed_at);

create index if not exists product_view_events_product_viewed_idx
    on product_view_events (product_id, viewed_at);

create unique index if not exists product_view_events_product_visitor_day_idx
    on product_view_events (product_id, visitor_id, view_day);

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
