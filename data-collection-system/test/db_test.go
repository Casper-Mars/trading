package test

import (
	"context"
	"testing"
	"time"

	"data-collection-system/internal/config"
	"data-collection-system/internal/dao"
	"data-collection-system/internal/database"
	"data-collection-system/internal/models"
	"data-collection-system/pkg/logger"
)

// TestDatabaseConnection 测试数据库连接和基础模型
func TestDatabaseConnection(t *testing.T) {
	// 初始化日志
	logger.Init(config.LogConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	})

	// 配置数据库（使用测试配置）
	dbConfig := config.DatabaseConfig{
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "password",
		DBName:   "trading_test",
		Charset:  "utf8mb4",
	}

	// 初始化数据库
	_, err := database.Init(dbConfig)
	if err != nil {
		t.Skipf("Database connection failed (this is expected in CI): %v", err)
		return
	}
	defer database.Close()

	// 自动迁移
	if err := database.AutoMigrate(); err != nil {
		t.Fatalf("Database migration failed: %v", err)
	}

	// 初始化DAO管理器
	db := database.GetDB()
	daoManager := dao.NewDAOManager(db)
	defer daoManager.Close()

	// 测试股票模型
	testStock(t, daoManager)

	// 测试任务模型
	testDataTask(t, daoManager)
}

func testStock(t *testing.T, daoManager dao.DAOManager) {
	ctx := context.Background()
	stockDAO := daoManager.Stock()

	// 创建测试股票
	listDate := time.Date(1991, 4, 3, 0, 0, 0, 0, time.UTC)
	stock := &models.Stock{
		Symbol:    "000001.SZ",
		Name:      "平安银行",
		Exchange:  "SZSE",
		Industry:  "银行",
		ListDate:  &listDate,
		Status:    models.StockStatusActive,
	}

	// 测试创建
	if err := stockDAO.Create(ctx, stock); err != nil {
		t.Fatalf("Failed to create stock: %v", err)
	}

	// 测试查询
	foundStock, err := stockDAO.GetBySymbol(ctx, "000001.SZ")
	if err != nil {
		t.Fatalf("Failed to get stock: %v", err)
	}
	if foundStock == nil {
		t.Fatal("Stock not found")
	}
	if foundStock.Name != "平安银行" {
		t.Errorf("Expected name '平安银行', got '%s'", foundStock.Name)
	}

	// 测试状态检查
	if !foundStock.IsActive() {
		t.Error("Stock should be active")
	}

	// 清理测试数据
	if err := stockDAO.Delete(ctx, "000001.SZ"); err != nil {
		t.Fatalf("Failed to delete stock: %v", err)
	}
}

func testDataTask(t *testing.T, daoManager dao.DAOManager) {
	ctx := context.Background()
	taskDAO := daoManager.DataTask()

	// 创建测试任务
	task := &models.DataTask{
		TaskName:    "test_stock_data_collection",
		TaskType:    models.TaskTypeStockData,
		Description: "测试股票数据采集任务",
		CronExpr:    "0 9 * * 1-5",
		Status:      models.TaskStatusEnabled,
		Config: models.TaskConfig{
			"symbols": []string{"000001.SZ", "000002.SZ"},
			"source":  "tushare",
		},
	}

	// 测试创建
	if err := taskDAO.Create(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// 测试查询
	foundTask, err := taskDAO.GetByName(ctx, "test_stock_data_collection")
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}
	if foundTask == nil {
		t.Fatal("Task not found")
	}
	if foundTask.TaskType != models.TaskTypeStockData {
		t.Errorf("Expected task type '%s', got '%s'", models.TaskTypeStockData, foundTask.TaskType)
	}

	// 测试状态检查
	if !foundTask.IsEnabled() {
		t.Error("Task should be enabled")
	}

	// 测试配置
	symbols, exists := foundTask.GetConfigValue("symbols")
	if !exists {
		t.Error("Config 'symbols' should exist")
	}
	if symbols == nil {
		t.Error("Config 'symbols' should not be nil")
	}

	// 清理测试数据
	if err := taskDAO.Delete(ctx, foundTask.ID); err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}
}