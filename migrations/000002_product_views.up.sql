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

alter table daily_vendor_product_metrics
    add column if not exists views_count bigint not null default 0;
