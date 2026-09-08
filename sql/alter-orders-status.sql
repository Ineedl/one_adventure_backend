-- The order state PENDING_PAY is 10? Actually 11 characters; widen status
-- before consuming promotion_order_create events.
USE `one_adventure_commerce`;
ALTER TABLE `orders` MODIFY COLUMN `status` VARCHAR(20) NOT NULL;
