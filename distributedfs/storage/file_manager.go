package storage

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

var mu sync.Mutex

func UploadFile(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	os.MkdirAll("storage_data", os.ModePerm)
	dstPath := "storage_data/" + handler.Filename

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

	fmt.Println("✅ File uploaded:", handler.Filename)
	go ReplicateToPeers(handler.Filename, dstPath)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "File uploaded and replicated")
}

func DownloadFile(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("name")
	if filename == "" {
		http.Error(w, "Filename not provided", http.StatusBadRequest)
		return
	}

	filePath := "storage_data/" + filename
	http.ServeFile(w, r, filePath)
}

func ListFiles(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir("storage_data")
	if err != nil {
		http.Error(w, "Could not read storage directory", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, "[")
	for i, entry := range entries {
		fmt.Fprintf(w, "\"%s\"", entry.Name())
		if i < len(entries)-1 {
			fmt.Fprint(w, ",")
		}
	}
	fmt.Fprint(w, "]")
}
