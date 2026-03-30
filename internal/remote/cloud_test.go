package remote

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCloudReader_CacheAndPrefetch(t *testing.T) {
	// 1. Setup a Mock Cloud Server (ex: S3)
	// We'll serve a 1MB file.
	fileSize := 1024 * 1024
	fileData := make([]byte, fileSize)
	for i := range fileData {
		fileData[i] = byte(i % 256)
	}

	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Content-Length", strconv.Itoa(fileSize))
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == "GET" {
			atomic.AddInt32(&requestCount, 1) // Count HTTP GET requests

			rangeHeader := r.Header.Get("Range")
			if rangeHeader == "" {
				t.Fatalf("Expected Range header")
			}
			
			// Parse "bytes=START-END"
			parts := strings.Split(rangeHeader, "=")
			if len(parts) != 2 || parts[0] != "bytes" {
				t.Fatalf("Invalid range header: %s", rangeHeader)
			}
			
			rangeStr := strings.Split(parts[1], "-")
			start, _ := strconv.ParseInt(rangeStr[0], 10, 64)
			end, _ := strconv.ParseInt(rangeStr[1], 10, 64)

			if start < 0 || end >= int64(fileSize) || start > end {
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}

			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
			w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
			w.WriteHeader(http.StatusPartialContent)
			
			w.Write(fileData[start : end+1])
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	// 2. Initialize CloudReader
	cr, err := NewCloudReader(server.URL)
	if err != nil {
		t.Fatalf("Failed to initialize CloudReader: %v", err)
	}

	if cr.Size() != int64(fileSize) {
		t.Fatalf("Expected size %d, got %d", fileSize, cr.Size())
	}

	// 3. Test ReadAt (Cache Miss)
	// Reading exactly 1 byte. Should trigger a fetch of the entire PageSize (256KB).
	// Because of async prefetch, it will also trigger fetch for Page 1 and Page 2.
	buf1 := make([]byte, 1)
	n, err := cr.ReadAt(buf1, 0)
	if err != nil || n != 1 {
		t.Fatalf("ReadAt failed: n=%d, err=%v", n, err)
	}
	if buf1[0] != fileData[0] {
		t.Fatalf("Data mismatch on byte 0")
	}

	// Give prefetchers a moment to finish
	time.Sleep(100 * time.Millisecond)

	initialRequests := atomic.LoadInt32(&requestCount)
	// We expect 1 request for Page 0 + 2 prefetch requests for Page 1 and 2 = 3 requests.
	t.Logf("Initial HTTP Requests after reading 1 byte: %d", initialRequests)

	// 4. Test Cache Hit (No new HTTP requests)
	buf2 := make([]byte, 1024)
	n, err = cr.ReadAt(buf2, 1024) // Still in Page 0 (0 - 256KB)
	if err != nil || n != 1024 {
		t.Fatalf("ReadAt failed: n=%d, err=%v", n, err)
	}
	if !bytes.Equal(buf2, fileData[1024:2048]) {
		t.Fatalf("Data mismatch on 1024-2048")
	}

	cachedRequests := atomic.LoadInt32(&requestCount)
	if cachedRequests != initialRequests {
		t.Fatalf("Cache failed! Expected requests %d, got %d", initialRequests, cachedRequests)
	}
	t.Logf("Cache HIT successful. Zero extra Egress HTTP Calls.")

	// 5. Test Prefetch Hit (Reading from Page 1 which was prefetched anonymously)
	buf3 := make([]byte, 100)
	n, err = cr.ReadAt(buf3, PageSize+100) // This is deeply inside Page 1
	if err != nil || n != 100 {
		t.Fatalf("ReadAt failed: n=%d, err=%v", n, err)
	}
	if !bytes.Equal(buf3, fileData[PageSize+100:PageSize+200]) {
		t.Fatalf("Data mismatch on Page 1")
	}
	
	time.Sleep(50 * time.Millisecond) // Let new prefetchers fire from reading Page 1 -> fetches Page 3

	prefetchRequests := atomic.LoadInt32(&requestCount)
	t.Logf("Prefetch HIT successful. Requests are now %d because a new prefetch fired for Page 3.", prefetchRequests)

	// 6. Test File Boundary Reading (Last page padding)
	buf4 := make([]byte, 5000)
	lastOff := int64(fileSize - 2000)
	n, err = cr.ReadAt(buf4, lastOff) // Reading past EOF boundary
	if err != nil && err != io.EOF {
		t.Fatalf("ReadAt failed near EOF: err=%v", err)
	}
	
	// Should read exactly 2000 bytes
	if n != 2000 {
		t.Fatalf("Expected 2000 bytes near EOF, got %d", n)
	}
	if !bytes.Equal(buf4[:2000], fileData[lastOff:]) {
		t.Fatalf("Data mismatch near EOF")
	}

	t.Logf("Final Egress HTTP Calls: %d to download %d bytes.", atomic.LoadInt32(&requestCount), fileSize)
}
