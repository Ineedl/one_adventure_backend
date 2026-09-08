-- Test products and promotions for local development.
-- This script is idempotent and can be executed repeatedly.

SET NAMES utf8mb4;

-- ---------------------------------------------------------------------------
-- Products
-- ---------------------------------------------------------------------------
USE `one_adventure_item`;

START TRANSACTION;

INSERT INTO `item_info` (`id`, `name`, `describe`) VALUES
    (900001, 'Test Gold Chest', 'Promotion test product containing game currency'),
    (900002, 'Test Energy Pack', 'Promotion test product containing energy items'),
    (900003, 'Test Adventure Pass', 'Promotion test product for a limited-time pass')
ON DUPLICATE KEY UPDATE
    `name` = VALUES(`name`),
    `describe` = VALUES(`describe`);

INSERT INTO `item_template` (`id`, `item_id`, `type`) VALUES
    (910001, 900001, 'promotion_test'),
    (910002, 900002, 'promotion_test'),
    (910003, 900003, 'promotion_test')
ON DUPLICATE KEY UPDATE
    `item_id` = VALUES(`item_id`),
    `type` = VALUES(`type`);

COMMIT;

-- ---------------------------------------------------------------------------
-- Commerce products, promotions and promotion products
-- Time ranges are deliberately broad so the refresh API can load them.
-- price uses the smallest currency unit (for example, cents).
-- ---------------------------------------------------------------------------
USE `one_adventure_commerce`;

START TRANSACTION;

INSERT INTO `product`
    (`product_id`, `name`, `product_type`, `price`, `currency_type`, `status`, `create_time`, `update_time`)
VALUES
    (910001, 'Test Gold Chest Product', 1, 199, 1, 1, NOW(), NOW()),
    (910002, 'Test Energy Pack Product', 1, 499, 1, 1, NOW(), NOW()),
    (910003, 'Test Adventure Pass Product', 2, 2999, 1, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE
    `name` = VALUES(`name`),
    `product_type` = VALUES(`product_type`),
    `price` = VALUES(`price`),
    `currency_type` = VALUES(`currency_type`),
    `status` = VALUES(`status`),
    `update_time` = NOW();

-- Product contents. A product can be a bundle; these rows define the actual
-- item templates and quantities delivered to the customer after purchase.
INSERT INTO `product_item`
    (`product_id`, `item_template_id`, `quantity`, `create_time`, `update_time`)
VALUES
    (910001, 910001, 10, NOW(), NOW()),
    (910001, 910002, 5, NOW(), NOW()),
    (910002, 910002, 20, NOW(), NOW()),
    (910003, 910001, 30, NOW(), NOW()),
    (910003, 910003, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE
    `quantity` = VALUES(`quantity`),
    `update_time` = NOW();

INSERT INTO `promotion`
    (`promotion_id`, `name`, `type`, `status`, `start_time`, `end_time`, `create_time`, `update_time`)
VALUES
    (920001, 'Test New Player Sale', 1, 1, '2026-01-01 00:00:00', '2030-12-31 23:59:59', NOW(), NOW()),
    (920002, 'Test Limited Stock Sale', 1, 1, '2026-01-01 00:00:00', '2030-12-31 23:59:59', NOW(), NOW()),
    (920003, 'Test Disabled Sale', 1, 0, '2026-01-01 00:00:00', '2030-12-31 23:59:59', NOW(), NOW())
ON DUPLICATE KEY UPDATE
    `name` = VALUES(`name`),
    `type` = VALUES(`type`),
    `status` = VALUES(`status`),
    `start_time` = VALUES(`start_time`),
    `end_time` = VALUES(`end_time`),
    `update_time` = NOW();

INSERT INTO `promotion_product`
    (`promotion_id`, `product_id`, `price`, `stock`, `currency_type`, `limit_type`, `limit_num`, `create_time`, `update_time`)
VALUES
    (920001, 910001, 100, 1000, 1, 1, 1, NOW(), NOW()),
    (920001, 910002, 299, 500, 1, 1, 3, NOW(), NOW()),
    (920002, 910003, 1999, 20, 1, 0, 0, NOW(), NOW()),
    (920003, 910001, 50, 100, 1, 0, 0, NOW(), NOW())
ON DUPLICATE KEY UPDATE
    `price` = VALUES(`price`),
    `stock` = VALUES(`stock`),
    `currency_type` = VALUES(`currency_type`),
    `limit_type` = VALUES(`limit_type`),
    `limit_num` = VALUES(`limit_num`),
    `update_time` = NOW();

INSERT INTO `promotion_inventory` (`product_id`, `stock`, `locked`, `promotion_id`) VALUES
    (910001, 1000, 0, 920001),
    (910002, 500, 0, 920001),
    (910003, 20, 0, 920002),
    (910001, 100, 0, 920003)
ON DUPLICATE KEY UPDATE
    `stock` = VALUES(`stock`),
    `locked` = VALUES(`locked`);

COMMIT;

-- Refresh active promotion stock in Redis after importing:
-- POST /promotion/api/v1/stock-refresh with body {}
