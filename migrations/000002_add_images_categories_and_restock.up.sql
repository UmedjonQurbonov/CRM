-- Migration: 000002_add_images_categories_and_restock.up.sql

-- 1. Add image_url and category to products
ALTER TABLE products ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN IF NOT EXISTS category VARCHAR(100) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);

-- 2. Expense categories table for user-managed categories
CREATE TABLE IF NOT EXISTS expense_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_expense_categories_name ON expense_categories(name);

-- 3. Seed standard expense categories
INSERT INTO expense_categories (name)
VALUES 
    ('Коммунальные услуги'),
    ('Аренда помещения'),
    ('Зарплаты и авансы'),
    ('Доставка и логистика'),
    ('Расходные материалы'),
    ('Реклама и маркетинг'),
    ('Прочие расходы')
ON CONFLICT (name) DO NOTHING;
