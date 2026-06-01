alter table daily_vendor_product_metrics
    drop column if exists views_count;

drop table if exists product_view_events;
