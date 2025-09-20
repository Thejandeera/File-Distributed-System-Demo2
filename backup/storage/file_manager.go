package storage

import (
	"distributedfs/config"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileManager handles file operations with enhanced features
type FileManager struct {
	storagePath string
	fileLocks   map[string]*sync.RWMutex
	locksMu     sync.Mutex
	quotaLimit  int64
}

// NewFileManager creates a new file manager
func NewFileManager(storagePath string, quotaLimit int64) *FileManager {
	// Ensure storage directory exists
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		if err := os.MkdirAll(storagePath, os.ModePerm); err != nil {
			log.Printf("❌ Failed to create storage directory %s: %v", storagePath, err)
		} else {
			log.Printf("📂 Created storage directory: %s", storagePath)
		}
	} else {
		log.Printf("📂 Using existing storage directory: %s", storagePath)
	}

	return &FileManager{
		storagePath: storagePath,
		fileLocks:   make(map[string]*sync.RWMutex),
		quotaLimit:  quotaLimit,
	}
}

// getFileLock returns a lock for a specific file
func (fm *FileManager) getFileLock(filename string) *sync.RWMutex {
	fm.locksMu.Lock()
	defer fm.locksMu.Unlock()

	if lock, exists := fm.fileLocks[filename]; exists {
		return lock
	}

	lock := &sync.RWMutex{}
	fm.fileLocks[filename] = lock
	return lock
}

// UploadFile handles file uploads with enhanced validation
func (fm *FileManager) UploadFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		http.Error(w, "Error parsing form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	if handler.Size > config.GetMaxFileSize() {
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	if !fm.checkQuota(handler.Size) {
		http.Error(w, "Quota exceeded", http.StatusInsufficientStorage)
		return
	}

	os.MkdirAll(fm.storagePath, os.ModePerm)
	dstPath := filepath.Join(fm.storagePath, handler.Filename)

	fileLock := fm.getFileLock(handler.Filename)
	fileLock.Lock()
	defer fileLock.Unlock()

	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Error creating file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Error saving file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ File uploaded: %s (size: %d bytes)", handler.Filename, handler.Size)

	go ReplicateToPeers(handler.Filename, dstPath)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"file":   handler.Filename,
		"size":   handler.Size,
	})
}

// DownloadFile handles file downloads with proper locking
func (fm *FileManager) DownloadFile(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("name")
	if filename == "" {
		http.Error(w, "Filename not provided", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(fm.storagePath, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	fileLock := fm.getFileLock(filename)
	fileLock.RLock()
	defer fileLock.RUnlock()

	http.ServeFile(w, r, filePath)
}

// ListFiles returns a list of files with metadata
func (fm *FileManager) ListFiles(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(fm.storagePath)
	if err != nil {
		http.Error(w, "Could not read storage directory", http.StatusInternalServerError)
		return
	}

	var files []map[string]interface{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(fm.storagePath, entry.Name())
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		files = append(files, map[string]interface{}{
			"name":    entry.Name(),
			"size":    info.Size(),
			"modTime": info.ModTime(),
			"isDir":   false,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// DeleteFile handles file deletion with proper locking
func (fm *FileManager) DeleteFile(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("name")
	if filename == "" {
		http.Error(w, "Filename not provided", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(fm.storagePath, filename)

	fileLock := fm.getFileLock(filename)
	fileLock.Lock()
	defer fileLock.Unlock()

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if err := os.Remove(filePath); err != nil {
		http.Error(w, "Error deleting file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ File deleted: %s", filename)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// GetFileInfo returns detailed information about a file
func (fm *FileManager) GetFileInfo(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("name")
	if filename == "" {
		http.Error(w, "Filename not provided", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(fm.storagePath, filename)
	info, err := os.Stat(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"name":    info.Name(),
		"size":    info.Size(),
		"modTime": info.ModTime().Unix(),
		"isDir":   info.IsDir(),
		"mode":    info.Mode().String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// checkQuota checks if there's enough quota for a new file
func (fm *FileManager) checkQuota(fileSize int64) bool {
	totalSize := fm.getTotalSize()
	return totalSize+fileSize <= fm.quotaLimit
}

// getTotalSize calculates the total size of all files
func (fm *FileManager) getTotalSize() int64 {
	entries, err := os.ReadDir(fm.storagePath)
	if err != nil {
		return 0
	}

	var totalSize int64
	for _, entry := range entries {
		if !entry.IsDir() {
			filePath := filepath.Join(fm.storagePath, entry.Name())
			if info, err := os.Stat(filePath); err == nil {
				totalSize += info.Size()
			}
		}
	}
	return totalSize
}

// GetStorageStats returns storage statistics
func (fm *FileManager) GetStorageStats() map[string]interface{} {
	entries, err := os.ReadDir(fm.storagePath)
	if err != nil {
		return map[string]interface{}{
			"error": "Could not read storage directory",
		}
	}

	var totalSize int64
	var fileCount int
	for _, entry := range entries {
		if !entry.IsDir() {
			filePath := filepath.Join(fm.storagePath, entry.Name())
			if info, err := os.Stat(filePath); err == nil {
				totalSize += info.Size()
				fileCount++
			}
		}
	}

	return map[string]interface{}{
		"totalFiles":   fileCount,
		"totalSize":    totalSize,
		"quotaLimit":   fm.quotaLimit,
		"quotaUsed":    totalSize,
		"quotaPercent": float64(totalSize) / float64(fm.quotaLimit) * 100,
		"storagePath":  fm.storagePath,
	}
}

// Cleanup removes old temporary files
func (fm *FileManager) Cleanup() {
	entries, err := os.ReadDir(fm.storagePath)
	if err != nil {
		return
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(fm.storagePath, entry.Name())
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		if info.ModTime().Before(now.Add(-24*time.Hour)) &&
			len(entry.Name()) > 5 && entry.Name()[:5] == "temp_" {
			if err := os.Remove(filePath); err == nil {
				log.Printf("🧹 Cleaned up old temp file: %s", entry.Name())
			}
		}
	}
}

// Global file manager instance
var globalFileManager *FileManager

// InitializeFileManager initializes the global file manager
func InitializeFileManager(storagePath string, quotaLimit int64) {
	globalFileManager = NewFileManager(storagePath, quotaLimit)
}

// GetGlobalFileManager returns the global file manager
func GetGlobalFileManager() *FileManager {
	return globalFileManager
}
