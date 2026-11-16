-- Migration: 000002_add_images_categories_and_restock.down.sql

DROP TABLE IF EXISTS expense_categories;

DROP INDEX IF EXISTS idx_products_category;
ALTER TABLE products DROP COLUMN IF EXISTS category;
ALTER TABLE products DROP COLUMN IF EXISTS image_url;
