// Package open115 provides an interface to 115 Cloud Open API
package open115

// ------------------------------------------------------------
// NOTE
// ------------------------------------------------------------
// This is a backend for 115 Cloud Open API (115开放平台)
// It provides file management operations like list, upload, download, move, copy, delete
// Uses access_token and refresh_token for authentication
// Callback URL: https://api.oplist.org/115cloud/callback
//
// ------------------------------------------------------------

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/unicode/norm"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/rclone/rclone/backend/115_open/api"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/rclone/rclone/fs/config/obscure"
	"github.com/rclone/rclone/fs/fserrors"
	"github.com/rclone/rclone/fs/fshttp"
	"github.com/rclone/rclone/fs/hash"
	"github.com/rclone/rclone/lib/dircache"
	"github.com/rclone/rclone/lib/encoder"
	"github.com/rclone/rclone/lib/pacer"
	"github.com/rclone/rclone/lib/rest"
)

// Constants
const (
	rootURL           = "https://proapi.115.com"
	uploadURL         = "https://proapi.115.com"
	authURL           = "https://passportapi.115.com"
	tokenRefreshURL   = "https://passportapi.115.com"
	defaultClientID   = "c8156887490a6c9405c8565a3861f2b6"
	defaultAppKey     = "D43EF7C6D7C7E7D40A9E2B8A6F5D3C1E"
	defaultCallback   = "https://api.oplist.org/115cloud/callback"
	minSleep          = 500 * time.Millisecond // Increased from 100ms for rate limiting
	maxSleep          = 10 * time.Second       // Increased from 2s for rate limiting
	decayConstant     = 3                       // Increased from 2 for slower backoff
	defaultPageSize   = 200
	maxPageSize       = 1150
)

// Options defines the configuration for this backend
type Options struct {
	AccessToken    string               `config:"access_token"`
	RefreshToken   string               `config:"refresh_token"`
	RootFolderID   string               `config:"root_folder_id"`
	OrderBy        string               `config:"order_by"`
	OrderDirection string               `config:"order_direction"`
	PageSize       int64                `config:"page_size"`
	UseTrash       bool                 `config:"use_trash"`
	Enc            encoder.MultiEncoder `config:"encoding"`
	// Custom API parameters (optional, defaults to OpenList values)
	ClientID    string `config:"client_id"`
	AppKey      string `config:"app_key"`
	CallbackURL string `config:"callback_url"`
}

// Fs represents a remote 115 Open
type Fs struct {
	name         string
	root         string
	opt          Options
	features     *fs.Features
	client       *http.Client
	rest         *rest.Client
	dirCache     *dircache.DirCache
	pacer        *fs.Pacer
	rootFolderID string
	m            configmap.Mapper
	tokenMu      *sync.Mutex
}

// Object describes a 115 Open object
type Object struct {
	fs          *Fs
	remote      string
	hasMetaData bool
	id          string
	size        int64
	modTime     time.Time
	mimeType    string
	parent      string
	sha1        string
	pc          string // pick_code for download
	thumbnail   string
	isDir       bool
}

// ------------------------------------------------------------

// Name of the remote (as passed into NewFs)
func (f *Fs) Name() string {
	return f.name
}

// Root of the remote (as passed into NewFs)
func (f *Fs) Root() string {
	return f.root
}

// String converts this Fs to a string
func (f *Fs) String() string {
	return fmt.Sprintf("115 Open root '%s'", f.root)
}

// Features returns the optional features of this Fs
func (f *Fs) Features() *fs.Features {
	return f.features
}

// Precision returns the precision of this Fs
func (f *Fs) Precision() time.Duration {
	return fs.ModTimeNotSupported
}

// Hashes returns the supported hash sets.
func (f *Fs) Hashes() hash.Set {
	return hash.Set(hash.SHA1)
}

// getClientID returns the ClientID or default value
func (f *Fs) getClientID() string {
	if f.opt.ClientID != "" {
		return f.opt.ClientID
	}
	return defaultClientID
}

// getAppKey returns the AppKey or default value
func (f *Fs) getAppKey() string {
	if f.opt.AppKey != "" {
		return f.opt.AppKey
	}
	return defaultAppKey
}

// getCallbackURL returns the CallbackURL or default value
func (f *Fs) getCallbackURL() string {
	if f.opt.CallbackURL != "" {
		return f.opt.CallbackURL
	}
	return defaultCallback
}

// parsePath parses a remote path
func parsePath(path string) (root string) {
	root = strings.Trim(path, "/")
	return
}

// ------------------------------------------------------------

// retryErrorCodes is a slice of error codes that we will retry
var retryErrorCodes = []int{
	429, // Too Many Requests
	500, // Internal Server Error
	502, // Bad Gateway
	503, // Service Unavailable
	504, // Gateway Timeout
	509, // Bandwidth Limit Exceeded
}

// shouldRetry returns a boolean as to whether this resp and err deserve to be retried
func (f *Fs) shouldRetry(ctx context.Context, resp *http.Response, inErr error) (bool, error) {
	if fserrors.ContextError(ctx, &inErr) {
		return false, inErr
	}
	if inErr == nil {
		return false, nil
	}
	if fserrors.ShouldRetry(inErr) {
		return true, inErr
	}
	return fserrors.ShouldRetryHTTP(resp, retryErrorCodes), inErr
}

// errorHandler parses a non 2xx error response into an error
func errorHandler(resp *http.Response) error {
	errResponse := new(api.Error)
	err := rest.DecodeJSON(resp, &errResponse)
	if err != nil {
		fs.Debugf(nil, "Couldn't decode error response: %v", err)
	}
	if errResponse.Message == "" {
		errResponse.Message = resp.Status
	}
	if errResponse.Code == 0 {
		errResponse.Code = resp.StatusCode
	}
	// Check for rate limit error
	if errResponse.Code == -1 && strings.Contains(errResponse.Message, "已达到当前访问上限") {
		errResponse.Code = 429 // Treat as rate limit
	}
	return errResponse
}

// getClient makes an http client according to the options
func getClient(ctx context.Context) *http.Client {
	return fshttp.NewClient(ctx)
}

// getAuthHeaders returns authorization headers
func (f *Fs) getAuthHeaders() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + f.opt.AccessToken,
	}
}

// refreshToken refreshes the access token using refresh token
func (f *Fs) refreshToken(ctx context.Context) error {
	f.tokenMu.Lock()
	defer f.tokenMu.Unlock()

	refreshToken, err := obscure.Reveal(f.opt.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to reveal refresh token: %w", err)
	}

	// Create a separate REST client for token refresh to avoid circular dependencies
	tokenClient := rest.NewClient(f.client).SetRoot(authURL)

	// Call token refresh endpoint
	opts := rest.Opts{
		Method: "POST",
		Path:   "/open/refreshToken",
	}

	req := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     f.getClientID(),
		"client_secret": f.getAppKey(),
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}

	var httpResp *http.Response
	err = f.pacer.Call(func() (bool, error) {
		httpResp, err = tokenClient.CallJSON(ctx, &opts, req, &tokenResp)
		return f.shouldRetry(ctx, httpResp, err)
	})
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update stored tokens
	f.opt.AccessToken = tokenResp.AccessToken
	f.opt.RefreshToken = tokenResp.RefreshToken

	// Save to config
	accessTokenEncoded, err := obscure.Reveal(f.opt.AccessToken)
	if err != nil {
		fs.Logf(f, "failed to encode access token: %v", err)
	}
	refreshTokenEncoded, err := obscure.Reveal(f.opt.RefreshToken)
	if err != nil {
		fs.Logf(f, "failed to encode refresh token: %v", err)
	}

	f.m.Set("access_token", accessTokenEncoded)
	f.m.Set("refresh_token", refreshTokenEncoded)

	return nil
}

// newFs partially constructs Fs from the path
func newFs(ctx context.Context, name, path string, m configmap.Mapper) (*Fs, error) {
	opt := new(Options)
	err := configstruct.Set(m, opt)
	if err != nil {
		return nil, err
	}

	root := parsePath(path)

	f := &Fs{
		name:    name,
		root:    root,
		opt:     *opt,
		m:       m,
		tokenMu: new(sync.Mutex),
	}

	f.features = (&fs.Features{
		ReadMimeType:            true,
		CanHaveEmptyDirectories: true,
		NoMultiThreading:        true,
	}).Fill(ctx, f)

	if opt.PageSize <= 0 {
		opt.PageSize = defaultPageSize
	} else if opt.PageSize > maxPageSize {
		opt.PageSize = maxPageSize
	}

	f.client = getClient(ctx)
	f.rest = rest.NewClient(f.client).SetRoot(rootURL).SetErrorHandler(errorHandler)

	// Create pacer with rate limiting
	f.pacer = fs.NewPacer(ctx, pacer.NewDefault(
		pacer.MinSleep(minSleep),
		pacer.MaxSleep(maxSleep),
		pacer.DecayConstant(decayConstant),
	))
	// Limit to 1 concurrent request (matches OpenList rate limiting)
	f.pacer.SetMaxConnections(1)

	// Override transfers to 1 for rate limiting (115 Open API has strict rate limits)
	ci := fs.GetConfig(ctx)
	if ci.Transfers == 4 { // Only override if user hasn't specified a custom value
		ci.Transfers = 1
	}

	return f, nil
}

// NewFs constructs an Fs from the path, container:path
func NewFs(ctx context.Context, name, path string, m configmap.Mapper) (fs.Fs, error) {
	f, err := newFs(ctx, name, path, m)
	if err != nil {
		return nil, err
	}

	// Set the root folder ID
	if f.opt.RootFolderID != "" {
		f.rootFolderID = f.opt.RootFolderID
	} else {
		f.rootFolderID = "0"
	}

	f.dirCache = dircache.New(f.root, f.rootFolderID, f)

	// Check if root looks like a file (contains extension)
	// This avoids extra API calls to detect files
	hasFileExtension := func(p string) bool {
		for i := len(p) - 1; i >= 0 && p[i] != '/'; i-- {
			if p[i] == '.' && i > 0 {
				return true
			}
		}
		return false
	}

	// If root looks like a file, try to treat it as file first
	if hasFileExtension(f.root) {
		newRoot, remote := dircache.SplitPath(f.root)
		if newRoot != f.root && newRoot != "" {
			tempF := *f
			tempF.dirCache = dircache.New(newRoot, f.rootFolderID, &tempF)
			tempF.root = newRoot
			err = tempF.dirCache.FindRoot(ctx, false)
			if err == nil {
				_, err := tempF.NewObject(ctx, remote)
				if err == nil {
					// Found the file!
					f.features.Fill(ctx, &tempF)
					f.dirCache = tempF.dirCache
					f.root = tempF.root
					return f, fs.ErrorIsFile
				}
			}
		}
	}

	// Normal directory handling
	err = f.dirCache.FindRoot(ctx, false)
	if err != nil {
		// Root doesn't exist, return as is
		return f, nil
	}
	return f, nil
}

// ------------------------------------------------------------

// List the objects and directories in dir into entries
func (f *Fs) List(ctx context.Context, dir string) (entries fs.DirEntries, err error) {
	dirID, err := f.dirCache.FindDir(ctx, dir, false)
	if err != nil {
		return nil, err
	}

	pageSize := f.opt.PageSize
	offset := int64(0)

	for {
		var files []api.GetFilesResp_File
		files, err = f.listFiles(ctx, dirID, pageSize, offset)
		if err != nil {
			return nil, err
		}

		for _, file := range files {
			entry, err := f.fileToDirEntry(ctx, dir, &file)
			if err != nil {
				return nil, err
			}
			if entry != nil {
				entries = append(entries, entry)
			}
		}

		if len(files) < int(pageSize) {
			break
		}
		offset += pageSize
	}

	return entries, nil
}

// listFiles lists files in a directory
func (f *Fs) listFiles(ctx context.Context, dirID string, limit, offset int64) ([]api.GetFilesResp_File, error) {
	opts := rest.Opts{
		Method:        "GET",
		Path:          "/open/ufile/files",
		ExtraHeaders: f.getAuthHeaders(),
	}

	params := url.Values{}
	params.Set("cid", dirID)
	params.Set("limit", strconv.FormatInt(limit, 10))
	params.Set("offset", strconv.FormatInt(offset, 10))
	params.Set("show_dir", "1")
	// asc should be "1" or "0" string (matching OpenList/SDK)
	if f.opt.OrderDirection == "asc" {
		params.Set("asc", "1")
	} else {
		params.Set("asc", "0")
	}

	if f.opt.OrderBy != "" {
		params.Set("o", f.opt.OrderBy)
	}

	opts.Parameters = params

	var resp api.GetFilesResp
	var httpResp *http.Response
	var callErr error
	callErr = f.pacer.Call(func() (bool, error) {
		var innerErr error
		httpResp, innerErr = f.rest.CallJSON(ctx, &opts, nil, &resp)
		// Check if token expired
		if httpResp != nil && httpResp.StatusCode == 401 {
			// Try to refresh token
			if refreshErr := f.refreshToken(ctx); refreshErr == nil {
				opts.ExtraHeaders = f.getAuthHeaders()
				httpResp, innerErr = f.rest.CallJSON(ctx, &opts, nil, &resp)
				return f.shouldRetry(ctx, httpResp, innerErr)
			}
		}
		return f.shouldRetry(ctx, httpResp, innerErr)
	})
	if callErr != nil {
		return nil, fmt.Errorf("failed to list files: %w", callErr)
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("API error: %s", resp.Message)
	}

	return resp.Data, nil
}

// fileToDirEntry converts an API file to an fs.DirEntry
func (f *Fs) fileToDirEntry(ctx context.Context, parent string, file *api.GetFilesResp_File) (fs.DirEntry, error) {
	// Handle empty parent (root directory)
	var remote string
	if parent == "" {
		remote = file.Fn
	} else {
		remote = parent + "/" + file.Fn
	}

	if file.Fc == "0" {
		// It's a directory
		f.dirCache.Put(remote, file.Fid)
		dir := fs.NewDir(remote, time.Unix(file.Upt, 0)).SetID(file.Fid)
		if file.Pid == "" {
			dir.SetParentID("0")
		} else {
			dir.SetParentID(file.Pid)
		}
		return dir, nil
	}

	// It's a file
	o := &Object{
		fs:          f,
		remote:      remote,
		id:          file.Fid,
		size:        file.FS,
		modTime:     time.Unix(file.Upt, 0),
		parent:      file.Pid,
		sha1:        file.Sha1,
		pc:          file.Pc,
		thumbnail:   file.Thumb,
		hasMetaData: true,
		isDir:       false,
	}
	return o, nil
}

// CreateDir makes a directory with pathID as parent and name leaf
func (f *Fs) CreateDir(ctx context.Context, pathID, leaf string) (newID string, err error) {
	opts := rest.Opts{
		Method:        "POST",
		Path:          "/open/folder/add",
		ExtraHeaders: f.getAuthHeaders(),
	}

	req := map[string]interface{}{
		"folder_name": f.opt.Enc.FromStandardName(leaf),
		"parent_id":   pathID,
	}

	var resp api.MkdirResp
	var httpResp *http.Response
	var callErr error
	_ = callErr
	callErr = f.pacer.Call(func() (bool, error) {
		var innerErr error
		httpResp, innerErr = f.rest.CallJSON(ctx, &opts, req, &resp)
		return f.shouldRetry(ctx, httpResp, innerErr)
	})
	if callErr != nil {
		return "", fmt.Errorf("failed to create directory: %w", callErr)
	}

	if resp.Code != 0 {
		return "", fmt.Errorf("API error: %s", resp.Message)
	}

	return resp.FileID, nil
}

// Mkdir creates the container if it doesn't exist
func (f *Fs) Mkdir(ctx context.Context, dir string) error {
	_, err := f.dirCache.FindDir(ctx, dir, true)
	return err
}

// About gets quota information
func (f *Fs) About(ctx context.Context) (usage *fs.Usage, err error) {
	opts := rest.Opts{
		Method:        "GET",
		Path:          "/open/user/info",
		ExtraHeaders: f.getAuthHeaders(),
	}

	var resp api.UserInfoResp
	var httpResp *http.Response
	var callErr error
	_ = callErr
	callErr = f.pacer.Call(func() (bool, error) {
		var innerErr error
		httpResp, innerErr = f.rest.CallJSON(ctx, &opts, nil, &resp)
		return f.shouldRetry(ctx, httpResp, innerErr)
	})
	if callErr != nil {
		return nil, fmt.Errorf("failed to get user info: %w", callErr)
	}

	// Parse space info from new API format
	usedSize, _ := resp.RtSpaceInfo.AllUse.Size.Int64()
	totalSize, _ := resp.RtSpaceInfo.AllTotal.Size.Int64()

	usage = &fs.Usage{
		Used: fs.NewUsageValue(usedSize),
	}
	if totalSize > 0 {
		usage.Total = fs.NewUsageValue(totalSize)
		usage.Free = fs.NewUsageValue(totalSize - usedSize)
	}

	return usage, nil
}

// ------------------------------------------------------------

// deleteObjects deletes files or directories
func (f *Fs) deleteObjects(ctx context.Context, IDs []string, useTrash bool) error {
	if len(IDs) == 0 {
		return nil
	}

	opts := rest.Opts{
		Method:        "POST",
		Path:          "/open/ufile/delete",
		ExtraHeaders: f.getAuthHeaders(),
	}

	req := map[string]interface{}{
		"file_ids":   strings.Join(IDs, ","),
		"delete_src": "1", // 1 = move to trash, 0 = permanent delete
	}

	if !useTrash {
		req["delete_src"] = "0"
	}

	var resp api.DelFileResp
	var httpResp *http.Response
	var callErr error
	_ = callErr
	callErr = f.pacer.Call(func() (bool, error) {
		var innerErr error
		httpResp, innerErr = f.rest.CallJSON(ctx, &opts, req, &resp)
		return f.shouldRetry(ctx, httpResp, innerErr)
	})
	if callErr != nil {
		return fmt.Errorf("failed to delete: %w", callErr)
	}

	if resp.Code != 0 {
		return fmt.Errorf("API error: %s", resp.Message)
	}

	return nil
}

// Move the object to a new parent folder
func (f *Fs) Move(ctx context.Context, src fs.Object, remote string) (fs.Object, error) {
	srcObj, ok := src.(*Object)
	if !ok {
		return nil, fs.ErrorCantMove
	}

	dstDir, dstLeaf := dircache.SplitPath(remote)
	dstDirID, err := f.dirCache.FindDir(ctx, dstDir, false)
	if err != nil {
		return nil, err
	}

	// Perform the move
	err = f.moveObjects(ctx, []string{srcObj.id}, dstDirID)
	if err != nil {
		return nil, fmt.Errorf("move failed: %w", err)
	}

	// Rename if name changed
	if dstLeaf != srcObj.remote {
		_, err = f.renameObject(ctx, srcObj.id, dstLeaf)
		if err != nil {
			return nil, fmt.Errorf("rename failed: %w", err)
		}
	}

	return f.NewObject(ctx, remote)
}

// moveObjects moves files to a new directory
func (f *Fs) moveObjects(ctx context.Context, IDs []string, dirID string) error {
	opts := rest.Opts{
		Method:        "POST",
		Path:          "/open/ufile/move",
		ExtraHeaders: f.getAuthHeaders(),
	}

	req := map[string]interface{}{
		"file_ids": strings.Join(IDs, ","),
		"to_cid":   dirID,
	}

	var resp api.MoveResp
	var httpResp *http.Response
	var callErr error
	_ = callErr
	callErr = f.pacer.Call(func() (bool, error) {
		var innerErr error
		httpResp, innerErr = f.rest.CallJSON(ctx, &opts, req, &resp)
		return f.shouldRetry(ctx, httpResp, innerErr)
	})
	if callErr != nil {
		return fmt.Errorf("failed to move: %w", callErr)
	}

	if resp.Code != 0 {
		return fmt.Errorf("API error: %s", resp.Message)
	}

	return nil
}

// renameObject renames a file or directory
func (f *Fs) renameObject(ctx context.Context, ID, newName string) (string, error) {
	opts := rest.Opts{
		Method:        "POST",
		Path:          "/open/ufile/update",
		ExtraHeaders: f.getAuthHeaders(),
	}

	req := map[string]interface{}{
		"file_id":   ID,
		"file_name": f.opt.Enc.FromStandardName(newName),
	}

	var resp api.UpdateFileResp
	var httpResp *http.Response
	var callErr error
	_ = callErr
	callErr = f.pacer.Call(func() (bool, error) {
		var innerErr error
		httpResp, innerErr = f.rest.CallJSON(ctx, &opts, req, &resp)
		return f.shouldRetry(ctx, httpResp, innerErr)
	})
	if callErr != nil {
		return "", fmt.Errorf("failed to rename: %w", callErr)
	}

	if resp.Code != 0 {
		return "", fmt.Errorf("API error: %s", resp.Message)
	}

	return ID, nil
}

// DirMove moves src, srcRemote to this remote at dstRemote
func (f *Fs) DirMove(ctx context.Context, src fs.Fs, srcRemote, dstRemote string) error {
	srcFs, ok := src.(*Fs)
	if !ok {
		fs.Debugf(srcFs, "Can't move directory - not same remote type")
		return fs.ErrorCantDirMove
	}

	srcID, srcParentID, srcLeaf, dstParentID, dstLeaf, err := f.dirCache.DirMove(ctx, srcFs.dirCache, srcFs.root, srcRemote, f.root, dstRemote)
	if err != nil {
		return err
	}

	if srcParentID != dstParentID {
		err = f.moveObjects(ctx, []string{srcID}, dstParentID)
		if err != nil {
			return fmt.Errorf("couldn't dir move: %w", err)
		}
	}

	if srcLeaf != dstLeaf {
		_, err = f.renameObject(ctx, srcID, dstLeaf)
		if err != nil {
			return fmt.Errorf("dirmove: couldn't rename moved dir: %w", err)
		}
	}

	srcFs.dirCache.FlushDir(srcRemote)
	return nil
}

// Copy src to this remote using server side copy operations
func (f *Fs) Copy(ctx context.Context, src fs.Object, remote string) (fs.Object, error) {
	srcObj, ok := src.(*Object)
	if !ok {
		return nil, fs.ErrorCantCopy
	}

	dstDir, _ := dircache.SplitPath(remote)
	dstDirID, err := f.dirCache.FindDir(ctx, dstDir, false)
	if err != nil {
		return nil, err
	}

	// Perform the copy
	err = f.copyObjects(ctx, []string{srcObj.id}, dstDirID)
	if err != nil {
		return nil, fmt.Errorf("copy failed: %w", err)
	}

	return f.NewObject(ctx, remote)
}

// copyObjects copies files to a new directory
func (f *Fs) copyObjects(ctx context.Context, IDs []string, dirID string) error {
	opts := rest.Opts{
		Method:        "POST",
		Path:          "/open/ufile/copy",
		ExtraHeaders: f.getAuthHeaders(),
	}

	req := map[string]interface{}{
		"file_ids": strings.Join(IDs, ","),
		"to_cid":   dirID,
		"no_dupli": "1",
	}

	var resp api.CopyResp
	var httpResp *http.Response
	var callErr error
	_ = callErr
	callErr = f.pacer.Call(func() (bool, error) {
		var innerErr error
		httpResp, innerErr = f.rest.CallJSON(ctx, &opts, req, &resp)
		return f.shouldRetry(ctx, httpResp, innerErr)
	})
	if callErr != nil {
		return fmt.Errorf("failed to copy: %w", callErr)
	}

	if resp.Code != 0 {
		return fmt.Errorf("API error: %s", resp.Message)
	}

	return nil
}

// ------------------------------------------------------------

// Rmdir deletes the root folder
func (f *Fs) Rmdir(ctx context.Context, dir string) error {
	rootID, err := f.dirCache.FindDir(ctx, dir, false)
	if err != nil {
		return err
	}

	entries, err := f.List(ctx, dir)
	if err != nil {
		return err
	}

	if len(entries) > 0 {
		return fs.ErrorDirectoryNotEmpty
	}

	return f.deleteObjects(ctx, []string{rootID}, f.opt.UseTrash)
}

// Purge deletes all the files and the container
func (f *Fs) Purge(ctx context.Context, dir string) error {
	rootID, err := f.dirCache.FindDir(ctx, dir, false)
	if err != nil {
		return err
	}

	// Collect all file IDs
	var IDs []string
	var processDir func(dirID string) error
	processDir = func(dirID string) error {
		pageSize := f.opt.PageSize
		offset := int64(0)
		for {
			files, err := f.listFiles(ctx, dirID, pageSize, offset)
			if err != nil {
				return err
			}
			for _, file := range files {
				IDs = append(IDs, file.Fid)
				if file.Fc == "0" {
					err = processDir(file.Fid)
					if err != nil {
						return err
					}
				}
			}
			if len(files) < int(pageSize) {
				break
			}
			offset += pageSize
		}
		return nil
	}

	err = processDir(rootID)
	if err != nil {
		return err
	}

	return f.deleteObjects(ctx, IDs, f.opt.UseTrash)
}

// ------------------------------------------------------------

// Put the object
func (f *Fs) Put(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	existingObj, err := f.NewObject(ctx, src.Remote())
	switch err {
	case nil:
		return existingObj, existingObj.Update(ctx, in, src, options...)
	case fs.ErrorObjectNotFound:
		newObj := &Object{
			fs:     f,
			remote: src.Remote(),
		}
		err = newObj.upload(ctx, in, src, options...)
		if err != nil {
			return nil, err
		}
		// After successful upload, directly set metadata from src instead of
		// querying the API which may have delays returning new file info.
		// This avoids "corrupted on transfer: sizes differ" errors due to
		// 115 API's eventual consistency.
		newObj.hasMetaData = true
		newObj.size = src.Size()
		newObj.modTime = src.ModTime(ctx)
		return newObj, nil
	default:
		return nil, err
	}
}

// ------------------------------------------------------------

// newObjectWithInfo creates an Object from a remote path and info
func (f *Fs) newObjectWithInfo(ctx context.Context, remote string, info *api.GetFilesResp_File) (fs.Object, error) {
	o := &Object{
		fs:     f,
		remote: remote,
	}
	if info != nil {
		err := o.setMetaData(info)
		if err != nil {
			return nil, err
		}
	} else {
		err := o.readMetaData(ctx)
		if err != nil {
			return nil, err
		}
	}
	return o, nil
}

// NewObject finds the Object at remote
func (f *Fs) NewObject(ctx context.Context, remote string) (fs.Object, error) {
	return f.newObjectWithInfo(ctx, remote, nil)
}

// normalize applies Unicode normalization to ensure consistent comparison
func normalize(s string) string {
	return norm.NFC.String(s)
}

// FindLeaf finds a directory of name leaf in the folder with ID pathID
func (f *Fs) FindLeaf(ctx context.Context, pathID, leaf string) (pathIDOut string, found bool, err error) {
	leaf = normalize(leaf)
	files, err := f.listFiles(ctx, pathID, f.opt.PageSize, 0)
	if err != nil {
		return "", false, err
	}
	for _, file := range files {
		if normalize(file.Fn) == leaf {
			return file.Fid, true, nil
		}
	}
	return "", false, nil
}

// ------------------------------------------------------------

// setMetaData sets the metadata from info
func (o *Object) setMetaData(info *api.GetFilesResp_File) error {
	o.hasMetaData = true
	o.id = info.Fid
	o.size = info.FS
	o.modTime = time.Unix(info.Upt, 0)
	o.parent = info.Pid
	o.sha1 = info.Sha1
	o.pc = info.Pc
	o.thumbnail = info.Thumb
	o.isDir = info.Fc == "0"
	return nil
}

// readMetaData gets the metadata if it hasn't already been fetched
func (o *Object) readMetaData(ctx context.Context) error {
	if o.hasMetaData {
		return nil
	}

	leaf, dirID, err := o.fs.dirCache.FindPath(ctx, o.remote, false)
	if err != nil {
		return fs.ErrorObjectNotFound
	}

	files, err := o.fs.listFiles(ctx, dirID, o.fs.opt.PageSize, 0)
	if err != nil {
		return err
	}

	leaf = normalize(leaf)
	for _, file := range files {
		if normalize(file.Fn) == leaf {
			return o.setMetaData(&file)
		}
	}

	return fs.ErrorObjectNotFound
}

// Fs returns the parent Fs
func (o *Object) Fs() fs.Info {
	return o.fs
}

// String returns a string version
func (o *Object) String() string {
	if o == nil {
		return "<nil>"
	}
	return o.remote
}

// Remote returns the remote path
func (o *Object) Remote() string {
	return o.remote
}

// Hash returns the SHA1 of an object returning a lowercase hex string
func (o *Object) Hash(ctx context.Context, t hash.Type) (string, error) {
	if t != hash.SHA1 {
		return "", hash.ErrUnsupported
	}
	return strings.ToLower(o.sha1), nil
}

// Size returns the size of an object in bytes
func (o *Object) Size() int64 {
	return o.size
}

// MimeType of an Object if known, "" otherwise
func (o *Object) MimeType(ctx context.Context) string {
	return o.mimeType
}

// ID returns the ID of the Object if known, or "" if not
func (o *Object) ID() string {
	return o.id
}

// ParentID returns the ID of the Object parent if known, or "" if not
func (o *Object) ParentID() string {
	return o.parent
}

// ModTime returns the modification time of the object
func (o *Object) ModTime(ctx context.Context) time.Time {
	return o.modTime
}

// SetModTime sets the modification time of the local fs object
func (o *Object) SetModTime(ctx context.Context, modTime time.Time) error {
	return fs.ErrorCantSetModTime
}

// Storable returns a boolean showing whether this object storable
func (o *Object) Storable() bool {
	return true
}

// ------------------------------------------------------------

// Remove an object
func (o *Object) Remove(ctx context.Context) error {
	return o.fs.deleteObjects(ctx, []string{o.id}, o.fs.opt.UseTrash)
}

// Open an object for read
func (o *Object) Open(ctx context.Context, options ...fs.OpenOption) (io.ReadCloser, error) {
	if o.id == "" {
		return nil, fmt.Errorf("can't download: no id")
	}

	// Get download URL using POST method with form data (matching 115 SDK)
	// Build form data manually to match SDK's ReqWithForm
	formData := url.Values{}
	formData.Set("pick_code", o.pc)
	body := strings.NewReader(formData.Encode())

	opts := rest.Opts{
		Method:         "POST",
		Path:           "/open/ufile/downurl",
		ExtraHeaders:   o.fs.getAuthHeaders(),
		ContentType:    "application/x-www-form-urlencoded",
		Body:           body,
	}

	var resp struct {
		Code    int `json:"code"`
		Message string `json:"message"`
		Data    map[string]struct {
			FileName string `json:"file_name"`
			FileSize int64  `json:"file_size"`
			PickCode string `json:"pick_code"`
			Sha1     string `json:"sha1"`
			URL      struct {
				URL string `json:"url"`
			} `json:"url"`
		} `json:"data"`
	}

	var httpResp *http.Response
	var callErr error
	callErr = o.fs.pacer.Call(func() (bool, error) {
		var innerErr error
		httpResp, innerErr = o.fs.rest.CallJSON(ctx, &opts, nil, &resp)
		return o.fs.shouldRetry(ctx, httpResp, innerErr)
	})
	if callErr != nil {
		return nil, fmt.Errorf("failed to get download URL: %w", callErr)
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("API error %d: %s", resp.Code, resp.Message)
	}

	downloadInfo, ok := resp.Data[o.id]
	if !ok {
		return nil, fmt.Errorf("no download URL available")
	}

	// Download the file
	req, err := http.NewRequestWithContext(ctx, "GET", downloadInfo.URL.URL, nil)
	if err != nil {
		return nil, err
	}

	// Set User-Agent for download request (matching OpenList)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	fs.OpenOptionAddHTTPHeaders(req.Header, options)

	res, err := o.fs.client.Do(req)
	if err != nil {
		return nil, err
	}

	// Accept both 200 (full content) and 206 (partial content for streaming)
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("download failed with status: %d", res.StatusCode)
	}

	return res.Body, nil
}

// Update the object with the contents of the io.Reader, modTime and size
func (o *Object) Update(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) error {
	return o.upload(ctx, in, src, options...)
}

// upload uploads the object
func (o *Object) upload(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) error {
	size := src.Size()
	remote := o.Remote()

	// Create the directory for the object if it doesn't exist
	leaf, dirID, err := o.fs.dirCache.FindPath(ctx, remote, true)
	if err != nil {
		return err
	}

	// Read all data first to calculate SHA1
	data, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	// Calculate full file SHA1
	sha1Hash := sha1.Sum(data)
	fullSHA1 := fmt.Sprintf("%x", sha1Hash)

	// Calculate first 128KB SHA1 (PreHash)
	preHashSize := int64(128 * 1024)
	if size < preHashSize {
		preHashSize = size
	}
	preHashData := data[:preHashSize]
	preHashHash := sha1.Sum(preHashData)
	preSHA1 := strings.ToUpper(fmt.Sprintf("%x", preHashHash))

	// Convert to uppercase (matching OpenList)
	fullSHA1 = strings.ToUpper(fullSHA1)
	preSHA1 = strings.ToUpper(preSHA1)

	// Upload init with SHA1 and PreHash
	initData, err := o.fs.uploadInitWithHash(ctx, leaf, size, dirID, fullSHA1, preSHA1)
	if err != nil {
		return err
	}

	// If file already exists (status = 2), return
	if initData.Status == 2 {
		// Set metadata for existing file (quick upload/second-speed upload)
		o.hasMetaData = true
		o.size = size
		if initData.FileID != "" {
			o.id = initData.FileID
		}
		if initData.PickCode != "" {
			o.pc = initData.PickCode
		}
		if fullSHA1 != "" {
			o.sha1 = fullSHA1
		}
		return nil
	}

	// If status = 6,7,8, need sign check verification
	if initData.Status == 6 || initData.Status == 7 || initData.Status == 8 {
		// Parse sign_check: "2392148-2392298"
		signCheckParts := strings.Split(initData.SignCheck, "-")
		if len(signCheckParts) == 2 {
			start, err := strconv.ParseInt(signCheckParts[0], 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse sign_check start: %w", err)
			}
			end, err := strconv.ParseInt(signCheckParts[1], 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse sign_check end: %w", err)
			}

			// Read the byte range and calculate SHA1
			rangeStart := start
			rangeLength := end - start + 1
			if rangeStart+rangeLength > int64(len(data)) {
				rangeLength = int64(len(data)) - rangeStart
			}

			signData := data[rangeStart : rangeStart+rangeLength]
			signHash := sha1.Sum(signData)
			signVal := strings.ToUpper(fmt.Sprintf("%x", signHash))

			// Call upload init again with sign key and value
			initData, err = o.fs.uploadInitWithSign(ctx, leaf, size, dirID, fullSHA1, preSHA1, initData.SignKey, signVal)
			if err != nil {
				return err
			}

			// If file already exists after sign verification, return
			if initData.Status == 2 {
				// Set metadata for existing file
				o.hasMetaData = true
				o.size = size
				if initData.FileID != "" {
					o.id = initData.FileID
				}
				if initData.PickCode != "" {
					o.pc = initData.PickCode
				}
				if fullSHA1 != "" {
					o.sha1 = fullSHA1
				}
				return nil
			}
		}
	}

	// Get upload token
	tokenResp, err := o.fs.getUploadToken(ctx)
	if err != nil {
		return err
	}

	fs.Debugf(o.fs, "OSS Token - Endpoint: %s", tokenResp.Endpoint)
	fs.Debugf(o.fs, "Uploading to OSS - Bucket: %s, Object: %s", initData.Bucket, initData.Object)

	// Upload to OSS
	err = o.fs.uploadToOSS(ctx, data, size, tokenResp, initData)
	if err != nil {
		return err
	}

	// Set metadata directly from upload response to avoid API delay issues.
	// 115 API may not immediately reflect the new file in file listings,
	// so we set known values from the upload response.
	o.hasMetaData = true
	o.size = size
	if initData.FileID != "" {
		o.id = initData.FileID
	}
	if initData.PickCode != "" {
		o.pc = initData.PickCode
	}
	if fullSHA1 != "" {
		o.sha1 = fullSHA1
	}

	return nil
}

// uploadInitWithHash initializes the upload with SHA1 and PreHash
func (f *Fs) uploadInitWithHash(ctx context.Context, name string, size int64, dirID string, fileSHA1, preSHA1 string) (*api.UploadInitData, error) {
	// Use form data instead of JSON (matching 115 API format)
	// Note: SDK adds "U_1_" prefix automatically, so we use dirID directly
	formData := url.Values{}
	formData.Set("file_name", name)
	formData.Set("file_size", strconv.FormatInt(size, 10))
	formData.Set("target", "U_1_"+dirID) // Add U_1_ prefix (same as SDK does)
	formData.Set("fileid", fileSHA1)
	formData.Set("preid", preSHA1)

	fs.Debugf(f, "Upload init request: file_name=%s, file_size=%d, target=%s, fileid=%s, preid=%s", name, size, "U_1_"+dirID, fileSHA1, preSHA1)

	opts := rest.Opts{
		Method:         "POST",
		Path:           "/open/upload/init",
		RootURL:        uploadURL,
		ExtraHeaders:   f.getAuthHeaders(),
		ContentType:    "application/x-www-form-urlencoded",
		Body:           strings.NewReader(formData.Encode()),
	}

	var resp api.UploadInitResp
	var httpResp *http.Response
	var callErr error
	callErr = f.pacer.Call(func() (bool, error) {
		var innerErr error
		httpResp, innerErr = f.rest.CallJSON(ctx, &opts, nil, &resp)
		return f.shouldRetry(ctx, httpResp, innerErr)
	})
	if callErr != nil {
		return nil, fmt.Errorf("failed to init upload: %w", callErr)
	}

	fs.Debugf(f, "Upload init response: Code=%d, Message=%s, Data=%+v", resp.Code, resp.Message, resp.Data)

	if resp.Code != 0 && resp.Code != 2 { // Code 2 means file already exists
		return nil, fmt.Errorf("API error: %s", resp.Message)
	}

	// Extract data from response
	return &resp.Data, nil
}

// uploadInitWithSign initializes the upload with sign verification
func (f *Fs) uploadInitWithSign(ctx context.Context, name string, size int64, dirID string, fileSHA1, preSHA1, signKey, signVal string) (*api.UploadInitData, error) {
	// Use form data instead of JSON (matching 115 API format)
	// Note: SDK adds "U_1_" prefix automatically, so we use dirID directly
	formData := url.Values{}
	formData.Set("file_name", name)
	formData.Set("file_size", strconv.FormatInt(size, 10))
	formData.Set("target", "U_1_"+dirID) // SDK adds "U_1_" prefix internally
	formData.Set("fileid", fileSHA1)
	formData.Set("preid", preSHA1)
	formData.Set("sign_key", signKey)
	formData.Set("sign_val", signVal)

	opts := rest.Opts{
		Method:         "POST",
		Path:           "/open/upload/init",
		RootURL:        uploadURL,
		ExtraHeaders:   f.getAuthHeaders(),
		ContentType:    "application/x-www-form-urlencoded",
		Body:           strings.NewReader(formData.Encode()),
	}

	var resp api.UploadInitResp
	var httpResp *http.Response
	var callErr error
	callErr = f.pacer.Call(func() (bool, error) {
		var innerErr error
		httpResp, innerErr = f.rest.CallJSON(ctx, &opts, nil, &resp)
		return f.shouldRetry(ctx, httpResp, innerErr)
	})
	if callErr != nil {
		return nil, fmt.Errorf("failed to init upload with sign: %w", callErr)
	}

	fs.Debugf(f, "Upload init with sign response: Code=%d, Message=%s, Data=%+v", resp.Code, resp.Message, resp.Data)

	if resp.Code != 0 && resp.Code != 2 {
		return nil, fmt.Errorf("API error: %s", resp.Message)
	}

	// Extract data from response
	return &resp.Data, nil
}

// getUploadToken gets the upload token
func (f *Fs) getUploadToken(ctx context.Context) (*api.TokenData, error) {
	// Use GET request without body (matching 115 API format)
	opts := rest.Opts{
		Method:        "GET",
		Path:          "/open/upload/get_token",
		RootURL:       uploadURL,
		ExtraHeaders: f.getAuthHeaders(),
	}

	var resp api.UploadGetTokenResp
	var httpResp *http.Response
	var callErr error
	callErr = f.pacer.Call(func() (bool, error) {
		var innerErr error
		httpResp, innerErr = f.rest.CallJSON(ctx, &opts, nil, &resp)
		return f.shouldRetry(ctx, httpResp, innerErr)
	})
	if callErr != nil {
		return nil, fmt.Errorf("failed to get upload token: %w", callErr)
	}

	fs.Debugf(f, "Upload token response: Code=%d, Message=%s, Data=%+v", resp.Code, resp.Message, resp.Data)
	fs.Debugf(f, "OSS Token - Endpoint: %s", resp.Data.Endpoint)

	if resp.Code != 0 {
		return nil, fmt.Errorf("API error: %s", resp.Message)
	}

	return &resp.Data, nil
}

// uploadToOSS uploads the file to OSS with callback using Aliyun OSS SDK
func (f *Fs) uploadToOSS(ctx context.Context, data []byte, size int64, tokenResp *api.TokenData, initResp *api.UploadInitData) error {
	// Create OSS client with STS token
	ossClient, err := oss.New(tokenResp.Endpoint, tokenResp.AccessKeyId, tokenResp.AccessKeySecret, oss.SecurityToken(tokenResp.SecurityToken))
	if err != nil {
		return fmt.Errorf("failed to create OSS client: %w", err)
	}

	// Get bucket
	bucket, err := ossClient.Bucket(initResp.Bucket)
	if err != nil {
		return fmt.Errorf("failed to get bucket: %w", err)
	}

	// Upload with callback
	// Callback needs to be base64 encoded
	callbackBase64 := base64.StdEncoding.EncodeToString([]byte(initResp.Callback.Value.Callback))
	callbackVarBase64 := base64.StdEncoding.EncodeToString([]byte(initResp.Callback.Value.CallbackVar))

	fs.Debugf(f, "OSS Upload - Bucket: %s, Object: %s, Callback: %s, CallbackVar: %s",
		initResp.Bucket, initResp.Object, initResp.Callback.Value.Callback, initResp.Callback.Value.CallbackVar)

	err = bucket.PutObject(initResp.Object, strings.NewReader(string(data)),
		oss.Callback(callbackBase64),
		oss.CallbackVar(callbackVarBase64),
	)

	if err != nil {
		return fmt.Errorf("failed to upload to OSS: %w", err)
	}

	return nil
}

// ------------------------------------------------------------

// Register with Fs
func init() {
	fs.Register(&fs.RegInfo{
		Name:        "open115",
		Description: "115 Open Cloud Storage",
		NewFs:       NewFs,
		Options: []fs.Option{{
			Name: "access_token",
			Help: "115 Open Access Token",
			Required: true,
			Sensitive: true,
		}, {
			Name: "refresh_token",
			Help: "115 Open Refresh Token",
			Required: true,
			Sensitive: true,
		}, {
			Name: "root_folder_id",
			Help: "ID of the root folder",
			Default: "0",
			Advanced: true,
		}, {
			Name: "order_by",
			Help: "Order by file_name, file_size, user_utime, file_type",
			Default: "",
			Advanced: true,
		}, {
			Name: "order_direction",
			Help: "Order direction: asc or desc",
			Default: "asc",
			Advanced: true,
		}, {
			Name: "page_size",
			Help: "Page size for listing",
			Default: 200,
			Advanced: true,
		}, {
			Name: "use_trash",
			Help: "Send files to the trash instead of deleting permanently",
			Default: true,
			Advanced: true,
		}, {
			Name: "client_id",
			Help: "115 Open Client ID (leave empty to use OpenList default)",
			Default: "",
			Advanced: true,
		}, {
			Name: "app_key",
			Help: "115 Open App Key (leave empty to use OpenList default)",
			Default: "",
			Advanced: true,
		}, {
			Name: "callback_url",
			Help: "115 Open Callback URL (leave empty to use OpenList default)",
			Default: "",
			Advanced: true,
		}, {
			Name: config.ConfigEncoding,
			Help: config.ConfigEncodingHelp,
			Advanced: true,
			Default: (encoder.EncodeCtl |
				encoder.EncodeDot |
				encoder.EncodeBackSlash |
				encoder.EncodeDoubleQuote |
				encoder.EncodeAsterisk |
				encoder.EncodeColon |
				encoder.EncodeLtGt |
				encoder.EncodeQuestion |
				encoder.EncodePipe |
				encoder.EncodeLeftSpace |
				encoder.EncodeRightSpace |
				encoder.EncodeRightPeriod |
				encoder.EncodeInvalidUtf8),
		}},
	})
}

// Check the interfaces are satisfied
var (
	_ fs.Fs        = (*Fs)(nil)
	_ fs.Purger    = (*Fs)(nil)
	_ fs.Copier    = (*Fs)(nil)
	_ fs.Mover     = (*Fs)(nil)
	_ fs.DirMover  = (*Fs)(nil)
	_ fs.Abouter   = (*Fs)(nil)
	_ fs.Object    = (*Object)(nil)
	_ fs.MimeTyper = (*Object)(nil)
	_ fs.IDer      = (*Object)(nil)
	_ fs.ParentIDer = (*Object)(nil)
)
