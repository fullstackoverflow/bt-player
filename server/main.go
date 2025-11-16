package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"server/torrent"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed assets/*
var assetsFS embed.FS

type PlayRequest struct {
	MagnetLink string `json:"magnetLink" binding:"required"`
}

func getContentType(extension string) string {
	switch extension {
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mkv":
		return "video/x-matroska"
	case ".avi":
		return "video/x-msvideo"
	default:
		return "application/octet-stream"
	}
}

func main() {
	torrent.InitTorrentClient()

	r := gin.Default()
	
	// 使用嵌入的文件系统
	assetsSubFS, _ := fs.Sub(assetsFS, "assets")
	r.StaticFS("/assets", http.FS(assetsSubFS))

	r.GET("/torrent/play/:session/:stream", func(c *gin.Context) {
		session := c.Param("session")
		stream, err := strconv.Atoi(c.Param("stream"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stream parameter"})
			return
		}

		// 从缓存获取 torrent
		t, ok := torrent.GetTorrentFromCache(session)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "torrent not found in cache"})
			return
		}

		if stream >= len(t.Files()) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stream index"})
			return
		}

		file := t.Files()[stream]
		fileName := file.DisplayPath()
		extension := strings.ToLower(filepath.Ext(fileName))

		log.Printf("Streaming file: %s (type: %s)", fileName, extension)
		
		// 使用 Gin 的 ServeContent 支持 Range 请求
		reader := file.NewReader()
		defer reader.Close()
		
		// torrent reader 已经实现了 io.ReadSeeker
		c.Header("Accept-Ranges", "bytes")
		c.Header("Content-Type", getContentType(extension))
		
		// ServeContent 自动处理 Range 请求
		http.ServeContent(c.Writer, c.Request, fileName, time.Now(), reader)
	})

	r.GET("/torrent/add", func(c *gin.Context) {
		link := c.Query("link")
		if link == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No magnet link provided"})
			return
		}
		log.Printf("MagnetLink: %s", link)
		client := torrent.GetGlobalTorrentClient()
		torrent_content, err := torrent.DownloadTorrent(link)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to download torrent"})
			return
		}
		magnet_link, err := torrent.TransTorrentFileToMagnetLink(torrent_content)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to convert torrent to magnet link"})
			return
		}
		log.Printf("Magnet Link: %s", magnet_link)
		t, err := client.AddMagnet(magnet_link)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid magnet url"})
			return
		}
		log.Printf("等待获取 torrent 元数据...")

		select {
		case <-t.GotInfo():
			log.Printf("成功获取 torrent 元数据")
		case <-time.After(2 * time.Minute):
			// 超时则移除该 torrent
			t.Drop()
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "获取 torrent 元数据超时，可能所有 tracker 都不可用"})
			return
		}

		if len(t.Files()) == 0 {
			t.Drop()
			c.JSON(http.StatusBadRequest, gin.H{"error": "No files found in torrent"})
			return
		}

		torrent_session_id := torrent.GetLinkHash(link)
		log.Printf("Torrent Session: %s", torrent_session_id)

		// 添加到缓存
		torrent.AddTorrentToCache(torrent_session_id, t)

		// 构建文件列表
		files := make([]gin.H, len(t.Files()))
		for i, file := range t.Files() {
			files[i] = gin.H{
				"index": i,
				"name":  file.DisplayPath(),
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"torrent_session_id": torrent_session_id,
			"files":              files,
		})
	})

	if err := r.Run(); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
