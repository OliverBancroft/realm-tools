package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// 为测试创建临时目录
func setupTestDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "realm-config-test")
	if err != nil {
		t.Fatalf("无法创建临时测试目录: %v", err)
	}
	return tempDir
}

// 清理测试目录
func cleanupTestDir(t *testing.T, dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		t.Errorf("清理测试目录失败: %v", err)
	}
}

// 创建示例配置文件用于测试
func createSampleConfigFile(t *testing.T, dir string) string {
	config := RealmConfig{
		Log: LogConfig{
			Level:  "info",
			Output: "/var/log/test.log",
		},
		Endpoints: []*Endpoint{
			{
				Listen: "0.0.0.0:1234",
				Remote: "example.com:5678",
			},
			{
				Listen: "0.0.0.0:4321",
				Remote: "test.example.org:8765",
			},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("无法序列化测试配置: %v", err)
	}

	filename := filepath.Join(dir, "test_config.json")
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		t.Fatalf("无法写入测试配置文件: %v", err)
	}

	return filename
}

// 测试拆分配置功能
func TestSplitConfig(t *testing.T) {
	// 设置测试环境
	testDir := setupTestDir(t)
	defer cleanupTestDir(t, testDir)

	// 保存当前工作目录
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("无法获取当前工作目录: %v", err)
	}

	// 切换到测试目录
	err = os.Chdir(testDir)
	if err != nil {
		t.Fatalf("无法切换到测试目录: %v", err)
	}
	defer os.Chdir(originalDir)

	// 创建测试配置
	configFile := createSampleConfigFile(t, testDir)

	// 执行拆分操作
	err = splitConfig(configFile)
	if err != nil {
		t.Fatalf("拆分配置失败: %v", err)
	}

	// 验证结果
	// 检查配置目录是否创建
	configDirPath := filepath.Join(testDir, configDir)
	if _, err := os.Stat(configDirPath); os.IsNotExist(err) {
		t.Errorf("配置目录未创建: %s", configDirPath)
	}

	// 检查日志配置文件
	logFilePath := filepath.Join(configDirPath, "log.yaml")
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		t.Errorf("日志配置文件未创建: %s", logFilePath)
	}

	// 检查端点配置文件
	files, err := filepath.Glob(filepath.Join(configDirPath, "endpoint_*.yaml"))
	if err != nil {
		t.Errorf("无法查找端点配置文件: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("生成的端点配置文件数量不正确，预期: 2, 实际: %d", len(files))
	}
}

// 测试合并配置功能
func TestMergeConfig(t *testing.T) {
	// 设置测试环境
	testDir := setupTestDir(t)
	defer cleanupTestDir(t, testDir)

	// 保存当前工作目录
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("无法获取当前工作目录: %v", err)
	}

	// 切换到测试目录
	err = os.Chdir(testDir)
	if err != nil {
		t.Fatalf("无法切换到测试目录: %v", err)
	}
	defer os.Chdir(originalDir)

	// 创建测试配置
	configFile := createSampleConfigFile(t, testDir)

	// 先拆分配置
	err = splitConfig(configFile)
	if err != nil {
		t.Fatalf("拆分配置失败: %v", err)
	}

	// 修改日志配置，模拟用户编辑
	logConfigPath := filepath.Join(testDir, configDir, "log.yaml")
	logData := []byte("# 这是测试日志配置\nlevel: debug\noutput: /var/log/modified.log\n")
	err = os.WriteFile(logConfigPath, logData, 0644)
	if err != nil {
		t.Fatalf("无法修改日志配置文件: %v", err)
	}

	// 创建合并输出的文件路径
	mergedConfigFile := filepath.Join(testDir, "merged_config.json")

	// 执行合并操作
	err = mergeConfig(mergedConfigFile)
	if err != nil {
		t.Fatalf("合并配置失败: %v", err)
	}

	// 验证结果
	// 检查合并后的配置文件是否存在
	if _, err := os.Stat(mergedConfigFile); os.IsNotExist(err) {
		t.Errorf("合并后的配置文件未创建: %s", mergedConfigFile)
	}

	// 读取并解析合并后的配置
	data, err := os.ReadFile(mergedConfigFile)
	if err != nil {
		t.Fatalf("无法读取合并后的配置文件: %v", err)
	}

	var mergedConfig RealmConfig
	err = json.Unmarshal(data, &mergedConfig)
	if err != nil {
		t.Fatalf("无法解析合并后的配置: %v", err)
	}

	// 验证日志配置是否被正确修改
	if mergedConfig.Log.Level != "debug" {
		t.Errorf("日志级别未被正确修改，预期: debug, 实际: %s", mergedConfig.Log.Level)
	}

	if mergedConfig.Log.Output != "/var/log/modified.log" {
		t.Errorf("日志输出路径未被正确修改，预期: /var/log/modified.log, 实际: %s", mergedConfig.Log.Output)
	}

	// 验证端点数量是否正确
	if len(mergedConfig.Endpoints) != 2 {
		t.Errorf("合并后的端点数量不正确，预期: 2, 实际: %d", len(mergedConfig.Endpoints))
	}
}

// 测试完整的拆分-合并流程，验证数据完整性
func TestSplitMergeIntegrity(t *testing.T) {
	// 设置测试环境
	testDir := setupTestDir(t)
	defer cleanupTestDir(t, testDir)

	// 保存当前工作目录
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("无法获取当前工作目录: %v", err)
	}

	// 切换到测试目录
	err = os.Chdir(testDir)
	if err != nil {
		t.Fatalf("无法切换到测试目录: %v", err)
	}
	defer os.Chdir(originalDir)

	// 创建原始测试配置
	configFile := createSampleConfigFile(t, testDir)

	// 读取原始配置
	originalData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("无法读取原始配置文件: %v", err)
	}

	var originalConfig RealmConfig
	err = json.Unmarshal(originalData, &originalConfig)
	if err != nil {
		t.Fatalf("无法解析原始配置: %v", err)
	}

	// 执行拆分
	err = splitConfig(configFile)
	if err != nil {
		t.Fatalf("拆分配置失败: %v", err)
	}

	// 执行合并
	mergedConfigFile := filepath.Join(testDir, "merged_config.json")
	err = mergeConfig(mergedConfigFile)
	if err != nil {
		t.Fatalf("合并配置失败: %v", err)
	}

	// 读取合并后的配置
	mergedData, err := os.ReadFile(mergedConfigFile)
	if err != nil {
		t.Fatalf("无法读取合并后的配置文件: %v", err)
	}

	var mergedConfig RealmConfig
	err = json.Unmarshal(mergedData, &mergedConfig)
	if err != nil {
		t.Fatalf("无法解析合并后的配置: %v", err)
	}

	// 验证日志配置是否保持一致
	if !reflect.DeepEqual(originalConfig.Log, mergedConfig.Log) {
		t.Errorf("日志配置不一致，原始: %+v, 合并后: %+v", originalConfig.Log, mergedConfig.Log)
	}

	// 验证端点配置是否保持一致
	if len(originalConfig.Endpoints) != len(mergedConfig.Endpoints) {
		t.Errorf("端点数量不一致，原始: %d, 合并后: %d",
			len(originalConfig.Endpoints), len(mergedConfig.Endpoints))
	} else {
		for i := range originalConfig.Endpoints {
			if !reflect.DeepEqual(originalConfig.Endpoints[i], mergedConfig.Endpoints[i]) {
				t.Errorf("端点 #%d 配置不一致，原始: %+v, 合并后: %+v",
					i, originalConfig.Endpoints[i], mergedConfig.Endpoints[i])
			}
		}
	}
}
