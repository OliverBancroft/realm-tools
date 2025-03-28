package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	configDir = "realm_configs"
)

// RealmConfig 表示整个配置文件结构
type RealmConfig struct {
	Log       LogConfig   `json:"log"`
	Endpoints []*Endpoint `json:"endpoints"`
}

// LogConfig 表示日志配置
type LogConfig struct {
	Level  string `json:"level,omitempty" yaml:"level,omitempty"`
	Output string `json:"output,omitempty" yaml:"output,omitempty"`
}

// Endpoint 表示一个端点配置
type Endpoint struct {
	Listen string `json:"listen" yaml:"listen"`
	Remote string `json:"remote" yaml:"remote"`
}

func ensureConfigDir() error {
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		err = os.MkdirAll(configDir, 0755)
		if err != nil {
			return fmt.Errorf("创建配置目录失败: %v", err)
		}
		fmt.Printf("创建配置目录: %s\n", configDir)
	}
	return nil
}

func splitConfig(jsonFile string) error {
	// 读取JSON文件
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config RealmConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析JSON失败: %v", err)
	}

	// 确保配置目录存在
	if err := ensureConfigDir(); err != nil {
		return err
	}

	// 保存日志配置
	logData, err := yaml.Marshal(config.Log)
	if err != nil {
		return fmt.Errorf("序列化日志配置失败: %v", err)
	}
	logFile := filepath.Join(configDir, "log.yaml")
	if err := os.WriteFile(logFile, logData, 0644); err != nil {
		return fmt.Errorf("保存日志配置失败: %v", err)
	}
	fmt.Printf("已保存日志配置到 %s\n", logFile)

	// 分别保存每个端点配置
	for i, endpoint := range config.Endpoints {
		// 生成有意义的文件名
		remote := strings.ReplaceAll(strings.ReplaceAll(endpoint.Remote, ":", "_"), ".", "_")
		filename := fmt.Sprintf("endpoint_%d_%s.yaml", i+1, remote)
		filepath := filepath.Join(configDir, filename)

		// 序列化为YAML
		data, err := yaml.Marshal(endpoint)
		if err != nil {
			return fmt.Errorf("序列化端点配置失败: %v", err)
		}

		// 写入文件
		if err := os.WriteFile(filepath, data, 0644); err != nil {
			return fmt.Errorf("保存端点配置失败: %v", err)
		}
		fmt.Printf("已保存端点配置到 %s\n", filepath)
	}

	fmt.Printf("\n配置已拆分完成！您现在可以在 %s 目录中编辑文件并添加注释\n", configDir)
	fmt.Println("编辑完成后，运行 'realm-config merge' 来重新生成realm.json")
	return nil
}

func mergeConfig(outputFile string) error {
	// 确保配置目录存在
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("错误: 配置目录 %s 不存在", configDir)
	}

	result := RealmConfig{
		Endpoints: []*Endpoint{},
	}

	// 读取日志配置
	logFile := filepath.Join(configDir, "log.yaml")
	if _, err := os.Stat(logFile); err == nil {
		data, err := os.ReadFile(logFile)
		if err != nil {
			return fmt.Errorf("读取日志配置失败: %v", err)
		}

		var logConfig LogConfig
		if err := yaml.Unmarshal(data, &logConfig); err != nil {
			return fmt.Errorf("解析日志配置失败: %v", err)
		}

		result.Log = logConfig
		fmt.Printf("已加载日志配置: %s\n", logFile)
	}

	// 获取所有端点配置文件
	pattern := filepath.Join(configDir, "endpoint_*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("查找端点配置文件失败: %v", err)
	}

	// 排序文件名以保持顺序
	sort.Strings(files)

	// 读取所有端点配置
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("读取端点配置失败: %v", err)
		}

		var endpoint Endpoint
		if err := yaml.Unmarshal(data, &endpoint); err != nil {
			return fmt.Errorf("解析端点配置失败: %v", err)
		}

		result.Endpoints = append(result.Endpoints, &endpoint)
		fmt.Printf("已加载端点配置: %s\n", file)
	}

	// 序列化为JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("生成JSON失败: %v", err)
	}

	// 保存到输出文件
	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		return fmt.Errorf("保存JSON配置失败: %v", err)
	}

	fmt.Printf("\n已成功合并配置到 %s\n", outputFile)
	return nil
}

func printUsage() {
	fmt.Println("用法:")
	fmt.Println("  realm-config split [json文件]  - 将JSON配置拆分为YAML文件")
	fmt.Println("  realm-config merge [json文件]  - 将YAML文件合并为JSON配置")
	fmt.Println("\n示例:")
	fmt.Println("  realm-config split             - 拆分默认的realm.json")
	fmt.Println("  realm-config merge custom.json - 合并配置到custom.json")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := strings.ToLower(os.Args[1])
	filename := "realm.json"
	if len(os.Args) > 2 {
		filename = os.Args[2]
	}

	var err error
	switch command {
	case "split":
		err = splitConfig(filename)
	case "merge":
		err = mergeConfig(filename)
	default:
		fmt.Printf("未知命令: %s\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
