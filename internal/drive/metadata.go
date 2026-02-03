package drive

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"gdrive-audit/internal/logger"
	"gdrive-audit/pkg/gdrive"
)

// FolderInfo represents folder metadata
type FolderInfo struct {
	Name    string
	DriveID string
}

// MetadataResolver resolves drive names, folder paths, and owner names with caching
type MetadataResolver struct {
	client      gdrive.DriveClient
	logger      logger.Logger
	driveCache  map[string]string
	folderCache map[string]FolderInfo
	mu          sync.RWMutex
}

// NewMetadataResolver creates a new metadata resolver
func NewMetadataResolver(client gdrive.DriveClient, log logger.Logger) *MetadataResolver {
	return &MetadataResolver{
		client:      client,
		logger:      log,
		driveCache:  make(map[string]string),
		folderCache: make(map[string]FolderInfo),
	}
}

// PreloadDriveName adds a drive name to the cache
func (r *MetadataResolver) PreloadDriveName(driveID, driveName string) {
	if driveID == "" || driveName == "" {
		return
	}
	r.mu.Lock()
	r.driveCache[driveID] = driveName
	r.mu.Unlock()
	r.logger.Debug("Preloaded drive: %s -> %s", driveID, driveName)
}

// ResolveDriveName resolves a drive name by ID with caching
func (r *MetadataResolver) ResolveDriveName(driveID string) string {
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

// GetDriveName gets the drive name for a file
func (r *MetadataResolver) GetDriveName(file *gdrive.File) string {
	// 1. Check explicit DriveID on the file
	if file.DriveID != "" {
		return r.ResolveDriveName(file.DriveID)
	}

	// 2. Check if parent folder belongs to a Shared Drive
	if len(file.Parents) > 0 {
		parentID := file.Parents[0]
		folderInfo := r.GetFolderInfo(parentID)
		if folderInfo.DriveID != "" {
			return r.ResolveDriveName(folderInfo.DriveID)
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

// GetOwnerName gets the owner name for a file
func (r *MetadataResolver) GetOwnerName(file *gdrive.File) string {
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

// GetFolderName gets the folder name by ID
func (r *MetadataResolver) GetFolderName(parentID string) string {
	return r.GetFolderInfo(parentID).Name
}

// GetFolderPath builds the full folder path for a file by recursively traversing parents
func (r *MetadataResolver) GetFolderPath(file *gdrive.File) string {
	if len(file.Parents) == 0 {
		return "/"
	}

	// Build path by recursively traversing parent folders
	path := r.buildPathRecursive(file.Parents[0], 0, 20) // Max depth of 20 to prevent infinite loops

	if path == "" {
		return "/"
	}

	return path
}

// buildPathRecursive recursively builds the folder path
func (r *MetadataResolver) buildPathRecursive(folderID string, depth int, maxDepth int) string {
	// Safety check to prevent infinite recursion
	if depth >= maxDepth {
		r.logger.Debug("Max depth reached while building path for folder: %s", folderID)
		return ""
	}

	if folderID == "" {
		return ""
	}

	// Get folder info
	folderInfo := r.GetFolderInfo(folderID)

	// Check if we've reached root or an orphan
	if folderInfo.Name == "" || folderInfo.Name == "Root/Orphan" || strings.Contains(folderInfo.Name, "Unknown Folder") {
		return ""
	}

	// Get the parent folder using the Drive API
	ctx := context.Background()
	parentFile, err := r.client.GetFile(ctx, folderID)
	if err != nil {
		r.logger.Debug("Failed to get parent folder %s: %v", folderID, err)
		return "/" + folderInfo.Name
	}

	// If this folder has parents, recurse
	if len(parentFile.Parents) > 0 {
		parentPath := r.buildPathRecursive(parentFile.Parents[0], depth+1, maxDepth)
		if parentPath == "" {
			return "/" + folderInfo.Name
		}
		return parentPath + "/" + folderInfo.Name
	}

	// This is a top-level folder
	return "/" + folderInfo.Name
}

// GetFolderInfo gets folder information by ID with caching
func (r *MetadataResolver) GetFolderInfo(parentID string) FolderInfo {
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
	// Note: We need context for this, but for now we'll use background context
	// In production, this should accept context as a parameter
	ctx := context.Background()
	file, err := r.client.GetFile(ctx, parentID)
	if err != nil {
		r.mu.Lock()
		info := FolderInfo{Name: fmt.Sprintf("Unknown Folder (%s)", parentID), DriveID: ""}
		r.folderCache[parentID] = info
		r.mu.Unlock()
		return info
	}

	info := FolderInfo{Name: file.Name, DriveID: file.DriveID}
	r.mu.Lock()
	r.folderCache[parentID] = info
	r.mu.Unlock()

	return info
}
