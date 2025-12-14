package torrent

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	anacrolixLog "github.com/anacrolix/log"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	lru "github.com/hashicorp/golang-lru/v2"
)

var (
	globalClient *torrent.Client
	initOnce     sync.Once
	initErr      error
)

// Torrent 缓存 - 使用 LRU 实现并发安全的读写，自动淘汰最久未使用的项
var (
	torrentCache     *lru.Cache[string, *torrent.Torrent]
	torrentCacheOnce sync.Once
)

const maxCacheSize = 100 // 最多缓存 100 个 torrent

// 初始化 LRU 缓存
func initTorrentCache() {
	torrentCacheOnce.Do(func() {
		var err error
		// 创建带淘汰回调的 LRU 缓存
		torrentCache, err = lru.NewWithEvict(maxCacheSize, func(key string, value *torrent.Torrent) {
			log.Printf("LRU 淘汰: %s", key[:8])
			value.Drop() // 释放 torrent 资源
		})
		if err != nil {
			log.Fatalf("初始化 LRU 缓存失败: %v", err)
		}
	})
}

// AddTorrentToCache 添加 torrent 到缓存 (在 /add 接口中调用)
func AddTorrentToCache(hash string, t *torrent.Torrent) {
	initTorrentCache()
	config := GetConfig()
	if config.GetSeeding() {
		t.AllowDataUpload()
	} else {
		t.DisallowDataUpload()
	}
	torrentCache.Add(hash, t)
	log.Printf("添加到缓存: %s (当前缓存数: %d)", hash[:8], torrentCache.Len())
}

// GetTorrentFromCache 从缓存获取 torrent (在 /play 接口中调用)
func GetTorrentFromCache(hash string) (*torrent.Torrent, bool) {
	initTorrentCache()
	if value, ok := torrentCache.Get(hash); ok {
		log.Printf("缓存命中: %s", hash[:8])
		return value, true
	}
	log.Printf("缓存未命中: %s", hash[:8])
	return nil, false
}

// RemoveTorrentFromCache 从缓存移除 torrent
func RemoveTorrentFromCache(hash string) {
	initTorrentCache()
	if value, ok := torrentCache.Get(hash); ok {
		value.Drop()
		torrentCache.Remove(hash)
		log.Printf("从缓存移除: %s", hash[:8])
	}
}

func GetLinkHash(link string) string {
	hash := md5.Sum([]byte(link))
	return fmt.Sprintf("%x", hash)
}

func TransTorrentFileToMagnetLink(fileBytes []byte) (string, error) {
	mi, err := metainfo.Load(bytes.NewReader(fileBytes))
	if err != nil {
		return "", fmt.Errorf("failed to load metainfo: %w", err)
	}
	log.Println("parsed torrent file successfully")

	infoHash := mi.HashInfoBytes().String()
	magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s", infoHash)

	info, err := mi.UnmarshalInfo()
	if err == nil {
		magnet += fmt.Sprintf("&dn=%s", url.QueryEscape(info.Name))
	}

	for _, tier := range mi.AnnounceList {
		for _, tracker := range tier {
			magnet += fmt.Sprintf("&tr=%s", url.QueryEscape(tracker))
		}
	}
	return magnet, nil
}

func DownloadTorrent(magnetLink string) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(magnetLink)
	if err != nil {
		return nil, fmt.Errorf("failed to download torrent file: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download torrent file, torrent file link return status code: %d", resp.StatusCode)
	}

	fileBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return fileBytes, nil
}

func InitTorrentClient() error {
	initOnce.Do(func() {
		config := torrent.NewDefaultClientConfig()
		dataPath := GetConfig().GetDataPath()
		config.DefaultStorage = storage.NewFile(dataPath)
		port := 10000 + rand.Intn(50000)
		config.ListenPort = port

		// 设置日志级别 - 只显示警告和错误
		// 方式1: 使用 Debug = false (默认就是 false)
		// config.Debug = false

		// 方式2: 设置自定义 Logger，过滤掉 Debug 和 Info 级别
		config.Logger = anacrolixLog.Default.WithFilterLevel(anacrolixLog.Error)

		// 方式3: 完全禁用日志
		// config.Logger = anacrolixLog.Discard

		globalClient, initErr = torrent.NewClient(config)
	})
	return initErr
}

func GetGlobalTorrentClient() *torrent.Client {
	return globalClient
}
