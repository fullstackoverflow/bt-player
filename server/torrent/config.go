package torrent

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Config 存储配置
type Config struct {
	mu            sync.RWMutex
	EnableSeeding bool   `json:"enable_seeding"` // 是否做种
	StorageLimit  int64  `json:"storage_limit"`  // 存储上限（GB，0表示不限制）
	DataPath      string `json:"data_path"`       // 数据存储路径
	
	stopCleanup   chan struct{} // 停止清理协程的信号
}

var (
	config     *Config
	configOnce sync.Once
)

const (
	configFile = "./torrent-config.json"
)

// InitConfig 初始化配置
func InitConfig() *Config {
	configOnce.Do(func() {
		config = loadConfigFromFile()
		if config == nil {
			// 默认配置
			config = &Config{
				EnableSeeding: true,
				StorageLimit:  0,              // 默认不限制
				DataPath:      "./torrent-data", // 默认存储路径
			}
			config.saveToFile()
		}
		config.stopCleanup = make(chan struct{})
		log.Printf("配置已加载: 做种=%v, 存储上限=%dGB, 存储路径=%s", config.EnableSeeding, config.StorageLimit, config.DataPath)
		
		// 启动时立即检查并清理（仅在设置了存储限制时）
		if config.StorageLimit > 0 {
			config.checkAndCleanStorage()
		}
		
		// 启动定期清理协程（每10分钟检查一次）
		go config.startPeriodicCleanup()
	})
	return config
}

// GetConfig 获取配置
func GetConfig() *Config {
	if config == nil {
		return InitConfig()
	}
	return config
}

// loadConfigFromFile 从文件加载配置
func loadConfigFromFile() *Config {
	data, err := os.ReadFile(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("读取配置文件失败: %v", err)
		}
		return nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("解析配置文件失败: %v", err)
		return nil
	}

	return &cfg
}

// saveToFile 保存配置到文件
func (c *Config) saveToFile() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

// SetSeeding 设置是否做种
func (c *Config) SetSeeding(enable bool) {
	c.mu.Lock()
	c.EnableSeeding = enable
	c.mu.Unlock()

	c.saveToFile()
	log.Printf("设置做种: %v", enable)
}

// GetSeeding 获取是否做种
func (c *Config) GetSeeding() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.EnableSeeding
}

// SetStorageLimit 设置存储上限（GB，0表示不限制）
func (c *Config) SetStorageLimit(limitGB int64) {
	c.mu.Lock()
	c.StorageLimit = limitGB
	c.mu.Unlock()

	c.saveToFile()
	log.Printf("设置存储上限: %d GB", limitGB)
}

// GetStorageLimit 获取存储上限（GB）
func (c *Config) GetStorageLimit() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.StorageLimit
}

// SetDataPath 设置数据存储路径
func (c *Config) SetDataPath(path string) {
	c.mu.Lock()
	c.DataPath = path
	c.mu.Unlock()

	c.saveToFile()
	log.Printf("设置存储路径: %s", path)
}

// GetDataPath 获取数据存储路径
func (c *Config) GetDataPath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.DataPath
}

// FileInfo 文件信息
type FileInfo struct {
	Path       string
	Size       int64
	CreateTime time.Time
}

// startPeriodicCleanup 定期清理存储空间（每10分钟检查一次）
func (c *Config) startPeriodicCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// 仅在设置了存储限制时执行清理
			if c.GetStorageLimit() > 0 {
				c.checkAndCleanStorage()
			}
		case <-c.stopCleanup:
			log.Println("停止定期清理协程")
			return
		}
	}
}

// checkAndCleanStorage 检查并清理存储空间
func (c *Config) checkAndCleanStorage() {
	c.mu.RLock()
	limitGB := c.StorageLimit
	dataPath := c.DataPath
	c.mu.RUnlock()
	
	// 0 表示不限制
	if limitGB == 0 {
		return
	}
	
	// 扫描所有文件
	files, totalSize, err := c.scanDataPath(dataPath)
	if err != nil {
		log.Printf("扫描数据目录失败: %v", err)
		return
	}
	
	limitBytes := limitGB * 1024 * 1024 * 1024
	
	log.Printf("当前存储使用: %.2f GB / %d GB", float64(totalSize)/(1024*1024*1024), limitGB)
	
	if totalSize <= limitBytes {
		return
	}
	
	// 需要清理，按创建时间排序（最早的在前）
	sort.Slice(files, func(i, j int) bool {
		return files[i].CreateTime.Before(files[j].CreateTime)
	})
	
	// 删除文件直到总大小低于限制
	for _, file := range files {
		if totalSize <= limitBytes {
			break
		}
		
		log.Printf("删除文件: %s (%.2f MB, 创建于 %s)", 
			file.Path, 
			float64(file.Size)/(1024*1024), 
			file.CreateTime.Format("2006-01-02 15:04:05"))
		
		if err := os.RemoveAll(file.Path); err != nil {
			log.Printf("删除文件失败: %v", err)
			continue
		}
		
		totalSize -= file.Size
	}
	
	log.Printf("清理完成，当前存储: %.2f GB", float64(totalSize)/(1024*1024*1024))
}

// scanDataPath 扫描数据目录，返回所有文件信息（排除 .torrent.bolt.db）
func (c *Config) scanDataPath(dataPath string) ([]FileInfo, int64, error) {
	var files []FileInfo
	var totalSize int64
	
	// 确保目录存在
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		return files, 0, nil
	}
	
	err := filepath.WalkDir(dataPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		// 跳过目录
		if d.IsDir() {
			return nil
		}
		
		// 跳过 .torrent.bolt.db 文件
		if strings.HasSuffix(path, ".torrent.bolt.db") {
			return nil
		}
		
		info, err := d.Info()
		if err != nil {
			log.Printf("获取文件信息失败 %s: %v", path, err)
			return nil
		}
		
		files = append(files, FileInfo{
			Path:       path,
			Size:       info.Size(),
			CreateTime: info.ModTime(), // 使用修改时间作为创建时间
		})
		
		totalSize += info.Size()
		
		return nil
	})
	
	return files, totalSize, err
}

// StopCleanup 停止清理协程（用于优雅关闭）
func (c *Config) StopCleanup() {
	if c.stopCleanup != nil {
		close(c.stopCleanup)
	}
}
