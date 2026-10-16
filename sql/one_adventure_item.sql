/*
 Navicat MySQL Data Transfer

 Source Server         : 127.0.0.1_3306
 Source Server Type    : MySQL
 Source Server Version : 80042
 Source Host           : 127.0.0.1:3306
 Source Schema         : one_adventure_item

 Target Server Type    : MySQL
 Target Server Version : 80042
 File Encoding         : 65001

 Date: 16/10/2026 09:16:03
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for inventory_conf
-- ----------------------------
DROP TABLE IF EXISTS `inventory_conf`;
CREATE TABLE `inventory_conf` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `item_num` int unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of inventory_conf
-- ----------------------------
BEGIN;
INSERT INTO `inventory_conf` VALUES (1, 1, 10);
COMMIT;

-- ----------------------------
-- Table structure for inventory_item
-- ----------------------------
DROP TABLE IF EXISTS `inventory_item`;
CREATE TABLE `inventory_item` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `template_id` bigint unsigned NOT NULL,
  `instance_id` varchar(20) NOT NULL,
  `index` int NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_index_user_id` (`index`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for item_info
-- ----------------------------
DROP TABLE IF EXISTS `item_info`;
CREATE TABLE `item_info` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `describe` varchar(100) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=900004 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of item_info
-- ----------------------------
BEGIN;
INSERT INTO `item_info` VALUES (900001, 'Test Gold Chest', 'Promotion test product containing game currency');
INSERT INTO `item_info` VALUES (900002, 'Test Energy Pack', 'Promotion test product containing energy items');
INSERT INTO `item_info` VALUES (900003, 'Test Adventure Pass', 'Promotion test product for a limited-time pass');
COMMIT;

-- ----------------------------
-- Table structure for item_instance
-- ----------------------------
DROP TABLE IF EXISTS `item_instance`;
CREATE TABLE `item_instance` (
  `id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `template_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for item_template
-- ----------------------------
DROP TABLE IF EXISTS `item_template`;
CREATE TABLE `item_template` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `item_id` bigint NOT NULL,
  `type` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL COMMENT '物品属性',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_item_id` (`item_id`)
) ENGINE=InnoDB AUTO_INCREMENT=910004 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of item_template
-- ----------------------------
BEGIN;
INSERT INTO `item_template` VALUES (910001, 900001, 'promotion_test');
INSERT INTO `item_template` VALUES (910002, 900002, 'promotion_test');
INSERT INTO `item_template` VALUES (910003, 900003, 'promotion_test');
COMMIT;

SET FOREIGN_KEY_CHECKS = 1;
