/*
 Navicat MySQL Data Transfer

 Source Server         : 127.0.0.1_3306
 Source Server Type    : MySQL
 Source Server Version : 80042
 Source Host           : 127.0.0.1:3306
 Source Schema         : one_adventure_server

 Target Server Type    : MySQL
 Target Server Version : 80042
 File Encoding         : 65001

 Date: 16/10/2026 09:16:12
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for channel_info
-- ----------------------------
DROP TABLE IF EXISTS `channel_info`;
CREATE TABLE `channel_info` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(10) NOT NULL,
  `index` int NOT NULL,
  `server_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_server_id_index` (`server_id`,`index`),
  UNIQUE KEY `idx_unique_server_id_name` (`server_id`,`name`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of channel_info
-- ----------------------------
BEGIN;
INSERT INTO `channel_info` VALUES (1, '频道1', 1, 1);
INSERT INTO `channel_info` VALUES (2, '频道2', 2, 1);
INSERT INTO `channel_info` VALUES (3, '频道1', 1, 2);
INSERT INTO `channel_info` VALUES (4, '频道2', 2, 2);
INSERT INTO `channel_info` VALUES (5, '频道3', 3, 2);
COMMIT;

-- ----------------------------
-- Table structure for server_info
-- ----------------------------
DROP TABLE IF EXISTS `server_info`;
CREATE TABLE `server_info` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of server_info
-- ----------------------------
BEGIN;
INSERT INTO `server_info` VALUES (1, '暹罗猫');
INSERT INTO `server_info` VALUES (2, '田园犬');
COMMIT;

SET FOREIGN_KEY_CHECKS = 1;
