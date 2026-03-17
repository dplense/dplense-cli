package google

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/dplense/dplense-cli/internal/logger"
)

// FolderInfo represents folder metadata
type FolderInfo struct {
	Name    string
	DriveID string
	Parents []string // cached parent IDs (avoids double GetFile call)
}

// metadataResolver resolves drive names, folder paths, and owner names with caching
type metadataResolver struct {
	client      DriveClient
	logger      logger.Logger
	driveCache  map[string]string
	folderCache map[string]FolderInfo
	pathCache   map[string]string // folderID -> resolved full path (avoids re-traversing)
	mu          sync.RWMutex
}

// newMetadataResolver creates a new metadata resolver
func newMetadataResolver(client DriveClient, log logger.Logger) *metadataResolver {
	return &metadataResolver{
		client:      client,
		logger:      log,
		driveCache:  make(map[string]string),
		folderCache: make(map[string]FolderInfo),
		pathCache:   make(map[string]string),
	}
}

// preloadDriveName adds a drive name to the cache
func (r *metadataResolver) preloadDriveName(driveID, driveName string) {
	if driveID == "" || driveName == "" {
		return
	}
	r.mu.Lock()
	r.driveCache[driveID] = driveName
	r.mu.Unlock()
	r.logger.Debug("Preloaded drive: %s -> %s", driveID, driveName)
}

// resolveDriveName resolves a drive name by ID with caching
func (r *metadataResolver) resolveDriveName(driveID string) string {
	if driveID == "" {
		return ""
	}

	r.mu.RLock()
	if name, ok := r.driveCache[driveID]; ok {
		r.mu.RUnlock()
		r.logger.Debug("Drive cache hit: %s -> %s", driveID, name)
		return name
	}
	r.mu.RUnlock()

	r.logger.Debug("Resolving Drive ID: %s", driveID)

	// Note: This requires access to the underlying drive.Service
	// For now, we'll return a placeholder. In a full implementation,
	// we'd need to add a GetDrive method to the DriveClient interface
	// or pass the service separately for metadata resolution.
	r.mu.Lock()
	name := fmt.Sprintf("Drive | ID: %s", driveID)
	r.driveCache[driveID] = name
	r.mu.Unlock()

	return name
}

// getDriveName gets the drive name for a file
func (r *metadataResolver) getDriveName(ctx context.Context, file *File) string {
	// 1. Check explicit DriveID on the file
	if file.DriveID != "" {
		return r.resolveDriveName(file.DriveID)
	}

	// 2. Check if parent folder belongs to a Shared Drive
	if len(file.Parents) > 0 {
		parentID := file.Parents[0]
		folderInfo := r.getFolderInfo(ctx, parentID)
		if folderInfo.DriveID != "" {
			return r.resolveDriveName(folderInfo.DriveID)
		}
	}

	// 3. Fallback to Owner's Personal Drive
	if len(file.Owners) > 0 {
		if file.Owners[0].Me {
			return "My Drive"
		}
		return fmt.Sprintf("%s's Personal Drive", file.Owners[0].DisplayName)
	}

	return "Unknown Drive"
}

// getOwnerName gets the owner name for a file
func (r *metadataResolver) getOwnerName(file *File) string {
	if len(file.Owners) > 0 {
		if file.Owners[0].Me {
			return "Me"
		}
		if file.Owners[0].DisplayName != "" {
			return file.Owners[0].DisplayName
		}
		return file.Owners[0].EmailAddress
	}
	return "Unknown"
}

// getFolderName gets the folder name by ID
func (r *metadataResolver) getFolderName(ctx context.Context, parentID string) string {
	return r.getFolderInfo(ctx, parentID).Name
}

// getFolderPath builds the full folder path for a file by recursively traversing parents.
// Uses path caching so files in the same folder resolve instantly after the first lookup.
func (r *metadataResolver) getFolderPath(ctx context.Context, file *File) string {
	if len(file.Parents) == 0 {
		return "/"
	}

	parentID := file.Parents[0]

	// Check path cache first (most files share parent folders)
	r.mu.RLock()
	if path, ok := r.pathCache[parentID]; ok {
		r.mu.RUnlock()
		return path
	}
	r.mu.RUnlock()

	// Build path by recursively traversing parent folders
	path := r.buildPathRecursive(ctx, parentID, 0, 5) // Max depth of 5 to limit API calls
	if path == "" {
		path = "/"
	}

	// Cache the resolved path
	r.mu.Lock()
	r.pathCache[parentID] = path
	r.mu.Unlock()

	return path
}

// buildPathRecursive recursively builds the folder path.
// Uses FolderInfo.Parents (cached) to avoid double GetFile calls per level.
// Caches every resolved path segment so sibling/ancestor lookups are instant.
func (r *metadataResolver) buildPathRecursive(ctx context.Context, folderID string, depth int, maxDepth int) string {
	if depth >= maxDepth || folderID == "" {
		return ""
	}

	// Check path cache — if this folder's path was already resolved, return immediately
	r.mu.RLock()
	if path, ok := r.pathCache[folderID]; ok {
		r.mu.RUnlock()
		return path
	}
	r.mu.RUnlock()

	// Get folder info (one GetFile call, caches name + driveID + parents)
	folderInfo := r.getFolderInfo(ctx, folderID)

	// Check if we've reached root or an orphan
	if folderInfo.Name == "" || folderInfo.Name == "Root/Orphan" || strings.Contains(folderInfo.Name, "Unknown Folder") {
		return ""
	}

	// Use cached parents from getFolderInfo — NO second GetFile call needed
	var path string
	if len(folderInfo.Parents) > 0 {
		parentPath := r.buildPathRecursive(ctx, folderInfo.Parents[0], depth+1, maxDepth)
		if parentPath != "" {
			path = parentPath + "/" + folderInfo.Name
		} else {
			path = "/" + folderInfo.Name
		}
	} else {
		path = "/" + folderInfo.Name
	}

	// Cache this path segment for future lookups
	r.mu.Lock()
	r.pathCache[folderID] = path
	r.mu.Unlock()

	return path
}

// getFolderInfo gets folder information by ID with caching
func (r *metadataResolver) getFolderInfo(ctx context.Context, parentID string) FolderInfo {
	if parentID == "" {
		return FolderInfo{Name: "Root/Orphan", DriveID: ""}
	}

	r.mu.RLock()
	if info, ok := r.folderCache[parentID]; ok {
		r.mu.RUnlock()
		r.logger.Debug("Folder cache hit: %s -> %s", parentID, info.Name)
		return info
	}
	r.mu.RUnlock()

	r.logger.Debug("Resolving Folder ID: %s", parentID)

	// Fetch folder using the client
	file, err := r.client.GetFile(ctx, parentID)
	if err != nil {
		r.mu.Lock()
		info := FolderInfo{Name: fmt.Sprintf("Unknown Folder (%s)", parentID), DriveID: ""}
		r.folderCache[parentID] = info
		r.mu.Unlock()
		return info
	}

	info := FolderInfo{Name: file.Name, DriveID: file.DriveID, Parents: file.Parents}
	r.mu.Lock()
	r.folderCache[parentID] = info
	r.mu.Unlock()

	return info
}
