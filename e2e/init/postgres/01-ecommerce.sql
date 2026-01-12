-- E-commerce Sample Database
-- This schema provides sample data for testing mcp-trino's semantic provider

CREATE SCHEMA IF NOT EXISTS ecommerce;

-- ============================================================================
-- Customers Table
-- ============================================================================
CREATE TABLE ecommerce.customers (
    customer_id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

COMMENT ON TABLE ecommerce.customers IS 'Customer master data';
COMMENT ON COLUMN ecommerce.customers.customer_id IS 'Unique customer identifier';
COMMENT ON COLUMN ecommerce.customers.email IS 'Customer email address - primary contact method';
COMMENT ON COLUMN ecommerce.customers.first_name IS 'Customer first/given name';
COMMENT ON COLUMN ecommerce.customers.last_name IS 'Customer last/family name';
COMMENT ON COLUMN ecommerce.customers.phone IS 'Customer phone number (optional)';
COMMENT ON COLUMN ecommerce.customers.created_at IS 'Account creation timestamp';
COMMENT ON COLUMN ecommerce.customers.updated_at IS 'Last profile update timestamp';
COMMENT ON COLUMN ecommerce.customers.is_active IS 'Whether the account is active';

-- ============================================================================
-- Products Table
-- ============================================================================
CREATE TABLE ecommerce.products (
    product_id SERIAL PRIMARY KEY,
    sku VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    unit_price DECIMAL(10,2) NOT NULL,
    cost_price DECIMAL(10,2),
    stock_quantity INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE ecommerce.products IS 'Product catalog';
COMMENT ON COLUMN ecommerce.products.sku IS 'Stock Keeping Unit - unique product identifier';
COMMENT ON COLUMN ecommerce.products.name IS 'Product display name';
COMMENT ON COLUMN ecommerce.products.description IS 'Product description for customers';
COMMENT ON COLUMN ecommerce.products.category IS 'Product category for grouping';
COMMENT ON COLUMN ecommerce.products.unit_price IS 'Selling price in USD';
COMMENT ON COLUMN ecommerce.products.cost_price IS 'Cost price for margin calculation';
COMMENT ON COLUMN ecommerce.products.stock_quantity IS 'Current inventory level';
COMMENT ON COLUMN ecommerce.products.is_active IS 'Whether the product is available for sale';

-- ============================================================================
-- Orders Table
-- ============================================================================
CREATE TABLE ecommerce.orders (
    order_id SERIAL PRIMARY KEY,
    customer_id INTEGER REFERENCES ecommerce.customers(customer_id),
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'pending',
    total_amount DECIMAL(12,2),
    shipping_address TEXT,
    billing_address TEXT
);

COMMENT ON TABLE ecommerce.orders IS 'Customer orders';
COMMENT ON COLUMN ecommerce.orders.order_id IS 'Unique order identifier';
COMMENT ON COLUMN ecommerce.orders.customer_id IS 'Reference to customer who placed the order';
COMMENT ON COLUMN ecommerce.orders.order_date IS 'When the order was placed';
COMMENT ON COLUMN ecommerce.orders.status IS 'Order status: pending, processing, completed, cancelled';
COMMENT ON COLUMN ecommerce.orders.total_amount IS 'Total order value in USD';
COMMENT ON COLUMN ecommerce.orders.shipping_address IS 'Delivery address for the order';
COMMENT ON COLUMN ecommerce.orders.billing_address IS 'Billing/invoice address';

-- ============================================================================
-- Order Items Table
-- ============================================================================
CREATE TABLE ecommerce.order_items (
    order_item_id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES ecommerce.orders(order_id),
    product_id INTEGER REFERENCES ecommerce.products(product_id),
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10,2) NOT NULL,
    discount_percent DECIMAL(5,2) DEFAULT 0
);

COMMENT ON TABLE ecommerce.order_items IS 'Line items for each order';
COMMENT ON COLUMN ecommerce.order_items.order_item_id IS 'Unique line item identifier';
COMMENT ON COLUMN ecommerce.order_items.order_id IS 'Reference to parent order';
COMMENT ON COLUMN ecommerce.order_items.product_id IS 'Reference to purchased product';
COMMENT ON COLUMN ecommerce.order_items.quantity IS 'Number of units purchased';
COMMENT ON COLUMN ecommerce.order_items.unit_price IS 'Price at time of purchase';
COMMENT ON COLUMN ecommerce.order_items.discount_percent IS 'Discount applied to this line item';

-- ============================================================================
-- Analytics View: Daily Revenue
-- ============================================================================
CREATE VIEW ecommerce.daily_revenue AS
SELECT
    DATE(o.order_date) as revenue_date,
    COUNT(DISTINCT o.order_id) as order_count,
    SUM(oi.quantity * oi.unit_price * (1 - oi.discount_percent/100)) as gross_revenue
FROM ecommerce.orders o
JOIN ecommerce.order_items oi ON o.order_id = oi.order_id
WHERE o.status != 'cancelled'
GROUP BY DATE(o.order_date);

COMMENT ON VIEW ecommerce.daily_revenue IS 'Daily revenue aggregation for reporting';

-- ============================================================================
-- Seed Data: Customers
-- ============================================================================
INSERT INTO ecommerce.customers (email, first_name, last_name, phone) VALUES
('john.doe@example.com', 'John', 'Doe', '+1-555-0101'),
('jane.smith@example.com', 'Jane', 'Smith', '+1-555-0102'),
('bob.wilson@example.com', 'Bob', 'Wilson', NULL),
('alice.johnson@example.com', 'Alice', 'Johnson', '+1-555-0104'),
('charlie.brown@example.com', 'Charlie', 'Brown', '+1-555-0105');

-- ============================================================================
-- Seed Data: Products
-- ============================================================================
INSERT INTO ecommerce.products (sku, name, description, category, unit_price, cost_price, stock_quantity) VALUES
('LAPTOP-001', 'Pro Laptop 15"', 'High-performance laptop for professionals with 16GB RAM and 512GB SSD', 'Electronics', 1299.99, 899.00, 50),
('LAPTOP-002', 'Budget Laptop 14"', 'Affordable laptop for everyday use', 'Electronics', 599.99, 420.00, 100),
('PHONE-001', 'SmartPhone X', 'Latest smartphone with 5G and triple camera', 'Electronics', 999.99, 650.00, 100),
('PHONE-002', 'SmartPhone Lite', 'Mid-range smartphone with great battery life', 'Electronics', 499.99, 320.00, 150),
('HDPH-001', 'Wireless Headphones Pro', 'Noise-cancelling wireless headphones with 30hr battery', 'Electronics', 249.99, 120.00, 200),
('HDPH-002', 'Wired Earbuds', 'High-quality wired earbuds', 'Electronics', 49.99, 15.00, 500),
('SHIRT-001', 'Classic T-Shirt', 'Cotton t-shirt, multiple colors available', 'Apparel', 29.99, 8.00, 500),
('SHIRT-002', 'Premium Polo', 'Breathable polo shirt for casual or business casual', 'Apparel', 59.99, 22.00, 300),
('JEANS-001', 'Slim Fit Jeans', 'Modern slim fit jeans in dark wash', 'Apparel', 79.99, 28.00, 250),
('BOOK-001', 'Data Engineering Guide', 'Comprehensive guide to modern data engineering', 'Books', 49.99, 18.00, 100);

-- ============================================================================
-- Seed Data: Orders
-- ============================================================================
INSERT INTO ecommerce.orders (customer_id, order_date, status, total_amount, shipping_address, billing_address) VALUES
(1, '2024-01-15 10:30:00', 'completed', 1549.98, '123 Main St, City, ST 12345', '123 Main St, City, ST 12345'),
(2, '2024-01-15 14:45:00', 'completed', 999.99, '456 Oak Ave, Town, ST 67890', '456 Oak Ave, Town, ST 67890'),
(1, '2024-01-16 09:15:00', 'completed', 279.98, '123 Main St, City, ST 12345', '123 Main St, City, ST 12345'),
(3, '2024-01-16 11:00:00', 'completed', 599.99, '789 Pine Rd, Village, ST 11111', '789 Pine Rd, Village, ST 11111'),
(4, '2024-01-17 08:30:00', 'completed', 1399.98, '321 Elm Blvd, Metro, ST 22222', '321 Elm Blvd, Metro, ST 22222'),
(2, '2024-01-17 16:20:00', 'completed', 109.98, '456 Oak Ave, Town, ST 67890', '456 Oak Ave, Town, ST 67890'),
(5, '2024-01-18 12:00:00', 'pending', 499.99, '654 Maple Dr, Suburb, ST 33333', '654 Maple Dr, Suburb, ST 33333'),
(1, '2024-01-18 15:30:00', 'processing', 329.97, '123 Main St, City, ST 12345', '123 Main St, City, ST 12345'),
(3, '2024-01-19 10:00:00', 'cancelled', 1299.99, '789 Pine Rd, Village, ST 11111', '789 Pine Rd, Village, ST 11111'),
(4, '2024-01-19 14:15:00', 'completed', 179.97, '321 Elm Blvd, Metro, ST 22222', '321 Elm Blvd, Metro, ST 22222');

-- ============================================================================
-- Seed Data: Order Items
-- ============================================================================
INSERT INTO ecommerce.order_items (order_id, product_id, quantity, unit_price, discount_percent) VALUES
-- Order 1: John buys laptop and headphones
(1, 1, 1, 1299.99, 0),
(1, 5, 1, 249.99, 0),
-- Order 2: Jane buys smartphone
(2, 3, 1, 999.99, 0),
-- Order 3: John buys headphones and t-shirts
(3, 5, 1, 249.99, 0),
(3, 7, 1, 29.99, 0),
-- Order 4: Bob buys budget laptop
(4, 2, 1, 599.99, 0),
-- Order 5: Alice buys laptop and polo shirts
(5, 1, 1, 1299.99, 0),
(5, 8, 2, 59.99, 0),
-- Order 6: Jane buys earbuds and t-shirt
(6, 6, 2, 49.99, 0),
(6, 7, 1, 29.99, 0),
-- Order 7: Charlie buys mid-range phone (pending)
(7, 4, 1, 499.99, 0),
-- Order 8: John buys jeans and books
(8, 9, 2, 79.99, 0),
(8, 10, 1, 49.99, 0),
-- Order 9: Bob tries to buy laptop (cancelled)
(9, 1, 1, 1299.99, 0),
-- Order 10: Alice buys apparel
(10, 7, 3, 29.99, 0),
(10, 8, 1, 59.99, 10),
(10, 9, 1, 79.99, 0);

-- ============================================================================
-- Grant permissions
-- ============================================================================
GRANT ALL PRIVILEGES ON SCHEMA ecommerce TO ecommerce;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA ecommerce TO ecommerce;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA ecommerce TO ecommerce;
