/*
 Navicat MySQL Data Transfer

 Source Server         : 127.0.0.1_3306
 Source Server Type    : MySQL
 Source Server Version : 80042
 Source Host           : 127.0.0.1:3306
 Source Schema         : one_adventure_commerce

 Target Server Type    : MySQL
 Target Server Version : 80042
 File Encoding         : 65001

 Date: 16/10/2026 09:15:53
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for orders
-- ----------------------------
DROP TABLE IF EXISTS `orders`;
CREATE TABLE `orders` (
  `order_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `product_id` bigint unsigned NOT NULL,
  `quantity` int NOT NULL,
  `amount` decimal(12,2) NOT NULL COMMENT '价格',
  `currency_type` tinyint NOT NULL COMMENT '货币类型',
  `order_type` tinyint NOT NULL COMMENT '订单类型',
  `status` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL COMMENT 'PENDING_PAY	待支付	订单创建成功，等待用户付款\nPAID	已支付	支付成功，等待履约\nPROCESSING	处理中	支付完成，订单正在处理（可选）\nSHIPPED	已发货	物流已发出\nCOMPLETED	已完成	交易结束\nCANCELLED	已取消	订单关闭\nREFUNDING	退款中	正在退款\nREFUNDED	已退款	退款完成\nCLOSED	已关闭	订单生命周期结束',
  `promotion_id` bigint unsigned NOT NULL COMMENT '商品所属活动',
  `create_time` datetime NOT NULL,
  `update_time` datetime NOT NULL,
  `request_id` varchar(100) NOT NULL,
  `order_no` varchar(36) NOT NULL COMMENT '订单号',
  PRIMARY KEY (`order_id`),
  UNIQUE KEY `orders_unique` (`request_id`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of orders
-- ----------------------------
BEGIN;
INSERT INTO `orders` VALUES (4, 1, 910001, 1, 100.00, 1, 1, 'CLOSED', 920001, '2026-09-04 08:10:41', '2026-09-04 08:10:41', '7e40c7be-88ce-45b2-9a27-5dad2ec26674', '0af5c8b0-fc9a-4b83-bf09-17db21dc960d');
COMMIT;

-- ----------------------------
-- Table structure for payment_order
-- ----------------------------
DROP TABLE IF EXISTS `payment_order`;
CREATE TABLE `payment_order` (
  `id` bigint NOT NULL COMMENT '主键ID',
  `payment_no` varchar(64) NOT NULL COMMENT '支付单号',
  `order_no` varchar(64) NOT NULL COMMENT '业务订单号',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `amount` decimal(12,2) NOT NULL COMMENT '支付金额',
  `currency` varchar(16) NOT NULL DEFAULT 'CNY' COMMENT '币种',
  `status` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL COMMENT 'INIT	初始化	支付单刚创建\nWAIT_PAY	等待支付	等待用户付款\nPAYING	支付中	已发起第三方支付\nSUCCESS	支付成功	支付完成\nFAILED	支付失败	支付失败\nCLOSED	已关闭	支付单失效\nREFUNDING	退款中	退款处理中\nREFUNDED	已退款	退款完成',
  `subject` varchar(128) DEFAULT NULL COMMENT '支付标题',
  `expire_time` datetime DEFAULT NULL COMMENT '支付过期时间',
  `paid_time` datetime DEFAULT NULL COMMENT '支付完成时间',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_payment_no` (`payment_no`),
  KEY `idx_order_no` (`order_no`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='支付订单表';

-- ----------------------------
-- Table structure for payment_refund
-- ----------------------------
DROP TABLE IF EXISTS `payment_refund`;
CREATE TABLE `payment_refund` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `refund_no` varchar(64) NOT NULL COMMENT '退款单号，系统内部唯一',
  `order_id` bigint NOT NULL COMMENT '业务订单ID',
  `payment_order_id` bigint NOT NULL COMMENT '支付单ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `refund_amount` bigint NOT NULL COMMENT '退款金额，单位：分',
  `reason` varchar(255) DEFAULT NULL COMMENT '退款原因',
  `status` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL COMMENT '退款状态\nSUCCESS 成功\nRUNNING 执行中\nSTOP        终止',
  `third_refund_no` varchar(128) DEFAULT NULL COMMENT '第三方退款单号',
  `retry_count` int NOT NULL DEFAULT '0' COMMENT '退款执行/查询重试次数',
  `next_retry_at` datetime DEFAULT NULL COMMENT '下一次重试时间',
  `last_error` varchar(500) DEFAULT NULL COMMENT '最近一次错误信息',
  `refund_time` datetime DEFAULT NULL COMMENT '退款成功时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_refund_no` (`refund_no`),
  KEY `idx_payment_order_id` (`payment_order_id`),
  KEY `idx_order_id` (`order_id`),
  KEY `idx_status_retry` (`status`,`next_retry_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='支付退款单';

-- ----------------------------
-- Table structure for payment_transaction
-- ----------------------------
DROP TABLE IF EXISTS `payment_transaction`;
CREATE TABLE `payment_transaction` (
  `id` bigint NOT NULL COMMENT '主键ID',
  `payment_no` varchar(64) NOT NULL COMMENT '支付单号',
  `order_no` varchar(64) NOT NULL COMMENT '业务订单号(冗余)',
  `transaction_no` varchar(128) DEFAULT NULL COMMENT '第三方交易号',
  `channel` varchar(32) NOT NULL COMMENT '支付渠道',
  `request_amount` decimal(12,2) NOT NULL COMMENT '请求支付金额',
  `paid_amount` decimal(12,2) DEFAULT NULL COMMENT '实际支付金额',
  `status` tinyint NOT NULL COMMENT '交易状态\nCREATED	已创建	准备请求第三方\nPROCESSING	处理中	第三方处理中\nSUCCESS	成功	第三方确认成功\nFAILED	失败	交易失败\nCLOSED	关闭	交易关闭\nREFUNDING	退款中	退款处理中\nREFUNDED	退款成功	资金已退回',
  `callback_time` datetime DEFAULT NULL COMMENT '第三方回调时间',
  `callback_content` text COMMENT '第三方回调原始内容',
  `request_content` text COMMENT '请求第三方参数',
  `response_content` text COMMENT '第三方响应内容',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_transaction_no` (`transaction_no`),
  KEY `idx_payment_no` (`payment_no`),
  KEY `idx_order_no` (`order_no`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='支付交易流水表';

-- ----------------------------
-- Table structure for product
-- ----------------------------
DROP TABLE IF EXISTS `product`;
CREATE TABLE `product` (
  `product_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(128) NOT NULL,
  `product_type` tinyint NOT NULL COMMENT '产品类型',
  `price` bigint NOT NULL COMMENT '价格',
  `currency_type` tinyint NOT NULL COMMENT '货币类型',
  `status` tinyint NOT NULL,
  `create_time` datetime NOT NULL,
  `update_time` datetime NOT NULL,
  PRIMARY KEY (`product_id`)
) ENGINE=InnoDB AUTO_INCREMENT=910004 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of product
-- ----------------------------
BEGIN;
INSERT INTO `product` VALUES (910001, 'Test Gold Chest Product', 1, 199, 1, 1, '2026-09-03 17:02:54', '2026-09-03 17:02:54');
INSERT INTO `product` VALUES (910002, 'Test Energy Pack Product', 1, 499, 1, 1, '2026-09-03 17:02:54', '2026-09-03 17:02:54');
INSERT INTO `product` VALUES (910003, 'Test Adventure Pass Product', 2, 2999, 1, 1, '2026-09-03 17:02:54', '2026-09-03 17:02:54');
COMMIT;

-- ----------------------------
-- Table structure for product_item
-- ----------------------------
DROP TABLE IF EXISTS `product_item`;
CREATE TABLE `product_item` (
  `product_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `item_template_id` bigint NOT NULL COMMENT '物品id',
  `quantity` int NOT NULL COMMENT '数量',
  `create_time` datetime NOT NULL,
  `update_time` datetime NOT NULL,
  PRIMARY KEY (`product_id`,`item_template_id`)
) ENGINE=InnoDB AUTO_INCREMENT=910004 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of product_item
-- ----------------------------
BEGIN;
INSERT INTO `product_item` VALUES (910001, 910001, 10, '2026-09-03 17:02:54', '2026-09-03 17:02:54');
INSERT INTO `product_item` VALUES (910001, 910002, 5, '2026-09-03 17:02:54', '2026-09-03 17:02:54');
INSERT INTO `product_item` VALUES (910002, 910002, 20, '2026-09-03 17:02:54', '2026-09-03 17:02:54');
INSERT INTO `product_item` VALUES (910003, 910001, 30, '2026-09-03 17:02:54', '2026-09-03 17:02:54');
INSERT INTO `product_item` VALUES (910003, 910003, 1, '2026-09-03 17:02:54', '2026-09-03 17:02:54');
COMMIT;

-- ----------------------------
-- Table structure for promotion
-- ----------------------------
DROP TABLE IF EXISTS `promotion`;
CREATE TABLE `promotion` (
  `promotion_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(128) NOT NULL,
  `type` tinyint NOT NULL,
  `status` tinyint NOT NULL COMMENT '1:启用 0:停止',
  `start_time` datetime NOT NULL,
  `end_time` datetime NOT NULL,
  `create_time` datetime NOT NULL,
  `update_time` datetime NOT NULL,
  PRIMARY KEY (`promotion_id`)
) ENGINE=InnoDB AUTO_INCREMENT=920004 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='促销活动列表';

-- ----------------------------
-- Records of promotion
-- ----------------------------
BEGIN;
INSERT INTO `promotion` VALUES (920001, 'Test New Player Sale', 1, 1, '2026-01-01 00:00:00', '2030-12-31 23:59:59', '2026-09-03 17:02:54', '2026-09-03 17:02:54');
INSERT INTO `promotion` VALUES (920002, 'Test Limited Stock Sale', 1, 1, '2026-01-01 00:00:00', '2030-12-31 23:59:59', '2026-09-03 17:02:54', '2026-09-03 17:02:54');
INSERT INTO `promotion` VALUES (920003, 'Test Disabled Sale', 1, 0, '2026-01-01 00:00:00', '2030-12-31 23:59:59', '2026-09-03 17:02:54', '2026-09-03 17:02:54');
COMMIT;

-- ----------------------------
-- Table structure for promotion_inventory
-- ----------------------------
DROP TABLE IF EXISTS `promotion_inventory`;
CREATE TABLE `promotion_inventory` (
  `product_id` bigint unsigned NOT NULL,
  `stock` int NOT NULL,
  `locked` int NOT NULL,
  `promotion_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`product_id`,`promotion_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='库存表';

-- ----------------------------
-- Records of promotion_inventory
-- ----------------------------
BEGIN;
INSERT INTO `promotion_inventory` VALUES (910001, 1000, 0, 920001);
INSERT INTO `promotion_inventory` VALUES (910001, 100, 0, 920003);
INSERT INTO `promotion_inventory` VALUES (910002, 500, 0, 920001);
INSERT INTO `promotion_inventory` VALUES (910003, 20, 0, 920002);
COMMIT;

-- ----------------------------
-- Table structure for promotion_product
-- ----------------------------
DROP TABLE IF EXISTS `promotion_product`;
CREATE TABLE `promotion_product` (
  `promotion_id` bigint unsigned NOT NULL,
  `product_id` bigint unsigned NOT NULL,
  `price` bigint NOT NULL,
  `stock` int NOT NULL,
  `currency_type` tinyint NOT NULL COMMENT '货币类型',
  `limit_type` tinyint NOT NULL COMMENT '限购类型 0:不限购 1:针对用户的数量限购',
  `limit_num` int NOT NULL COMMENT '限购数量',
  `create_time` datetime NOT NULL,
  `update_time` datetime NOT NULL,
  PRIMARY KEY (`promotion_id`,`product_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of promotion_product
-- ----------------------------
BEGIN;
INSERT INTO `promotion_product` VALUES (920001, 910001, 100, 1000, 1, 1, 1, '2026-09-03 17:02:54', '2026-09-03 17:02:54');
INSERT INTO `promotion_product` VALUES (920001, 910002, 299, 500, 1, 1, 3, '2026-09-03 17:02:54', '2026-09-03 17:02:54');
INSERT INTO `promotion_product` VALUES (920002, 910003, 1999, 20, 1, 0, 0, '2026-09-03 17:02:54', '2026-09-03 17:02:54');
INSERT INTO `promotion_product` VALUES (920003, 910001, 50, 100, 1, 0, 0, '2026-09-03 17:02:54', '2026-09-03 17:02:54');
COMMIT;

SET FOREIGN_KEY_CHECKS = 1;
