// Package api has type definitions for 115 Open API
package api

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	// API endpoints
	APIURL        = "https://proapi.115.com"
	APIAuthURL    = "https://passportapi.115.com"
	UploadURL     = "https://upload.115.com"

	timeFormat = "2006-01-02 15:04:05"
)

// Error represents 115 API error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

// Token represents 115 API token
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
}

// UserInfo represents user information
type UserInfo struct {
	LoginID     string `json:"login_id"`
	NickName    string `json:"nick_name"`
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	IsVerify    int    `json:"is_verify"`
	Sex         int    `json:"sex"`
	Birthday    string `json:"birthday"`
	Avatar      string `json:"avatar"`
	Member      int    `json:"member"`
	SpaceInfo   SpaceInfo `json:"space_info"`
}

// UserInfoResp represents user info API response (new format)
type UserInfoResp struct {
	UserID      int64               `json:"user_id"`
	UserName    string              `json:"user_name"`
	UserFaceS   string              `json:"user_face_s"`
	UserFaceM   string              `json:"user_face_m"`
	UserFaceL   string              `json:"user_face_l"`
	RtSpaceInfo RtSpaceInfo        `json:"rt_space_info"`
	VipInfo     VipInfo            `json:"vip_info"`
}

// RtSpaceInfo represents space info in new format
type RtSpaceInfo struct {
	AllTotal UserInfoSize `json:"all_total"`
	AllRemain UserInfoSize `json:"all_remain"`
	AllUse UserInfoSize `json:"all_use"`
}

// UserInfoSize represents size info
type UserInfoSize struct {
	Size       json.Number `json:"size"`
	SizeFormat string     `json:"size_format"`
}

// VipInfo represents VIP info
type VipInfo struct {
	LevelName string `json:"level_name"`
	Expire    int64  `json:"expire"`
}

// SpaceInfo represents user's space information
type SpaceInfo struct {
	TotalSize int64 `json:"total_size,string"`
	UsedSize  int64 `json:"used_size,string"`
}

// File represents a file or folder
type File struct {
	// Common fields
	ID       string `json:"file_id"`
	PID      string `json:"parent_id"`
	Name     string `json:"file_name"`
	Size     int64  `json:"size,string"`
	SHA1    string `json:"sha1"`
	FileType string `json:"file_type"` // "1" = file, "2" = folder
	IsDir    bool   `json:"-"`

	// Time fields (Unix timestamp)
	CreatedTime int64 `json:"create_time,string"`
	ModifyTime  int64 `json:"modify_time,string"`

	// Additional fields
	Thumb     string `json:"thumb"`
	PC        string `json:"pc"` // pick_code for download
	MimeType  string `json:"mime_type"`
}

// FileListResponse represents file list API response
type FileListResponse struct {
	Code    int     `json:"code"`
	Message string  `json:"message"`
	Count   int     `json:"count"`
	Data    []File  `json:"data"`
}

// GetFilesReq represents request parameters for getting files
type GetFilesReq struct {
	CID      string `json:"cid"`
	Limit    int64  `json:"limit"`
	Offset   int64  `json:"offset"`
	ASC      bool   `json:"asc"`
	O        string `json:"o"` // order by: file_name, file_size, user_utime, file_type
	ShowDir  bool   `json:"show_dir"`
	Sign     string `json:"sign"`
	Timestamp int64 `json:"time"`
}

// GetFilesResp_File represents a file in GetFiles response
type GetFilesResp_File struct {
	Fid   string `json:"fid"`
	Pid   string `json:"pid"`
	Sha1  string `json:"sha1"`
	Fn    string `json:"fn"`
	Mime  string `json:"mtime"`
	FS    int64  `json:"fs"`
	Fc    string `json:"fc"` // "0" = folder, "1" = file
	Upt   int64  `json:"upt"`
	Uet   int64  `json:"uet"`
	Pt    int64  `json:"pt"`
	UpPt  int64  `json:"upPt"`
	Thumb string `json:"thumb"`
	Pc    string `json:"pc"`
}

// GetFilesResp represents GetFiles API response
type GetFilesResp struct {
	Code    int                `json:"code"`
	Message string             `json:"message"`
	Count   int                `json:"count"`
	Data    []GetFilesResp_File `json:"data"`
}

// MkdirReq represents mkdir request
type MkdirReq struct {
	FolderName string `json:"folder_name"`
	ParentID   string `json:"parent_id"`
}

// MkdirResp represents mkdir response
type MkdirResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	FileID  string `json:"file_id"`
}

// MoveReq represents move request
type MoveReq struct {
	FileIDs string `json:"file_ids"`
	ToCid   string `json:"to_cid"`
}

// MoveResp represents move response
type MoveResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CopyReq represents copy request
type CopyReq struct {
	PID     string `json:"pid"`
	FileID  string `json:"file_id"`
	NoDupli string `json:"no_dupli"` // "1" = don't duplicate
}

// CopyResp represents copy response
type CopyResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// UpdateFileReq represents rename request
type UpdateFileReq struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
}

// UpdateFileResp represents rename response
type UpdateFileResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// DelFileReq represents delete request
type DelFileReq struct {
	FileIDs  string `json:"file_ids"`
	ParentID string `json:"parent_id"`
}

// DelFileResp represents delete response
type DelFileResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// UploadInitReq represents upload init request
type UploadInitReq struct {
	FileName   string `json:"file_name"`
	FileSize   int64  `json:"file_size"`
	Target     string `json:"target"`
	FileID     string `json:"file_id"`
	PreID      string `json:"preid"`
	SignKey    string `json:"sign_key,omitempty"`
	SignVal    string `json:"sign_val,omitempty"`
}

// UploadInitResp represents upload init response
type UploadInitResp struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Data    UploadInitData    `json:"data"`
}

// UploadInitData represents the data field in upload init response
type UploadInitData struct {
	PickCode    string         `json:"pick_code"`
	Status      int            `json:"status"` // 1 = 非秒传, 2 = 秒传, 6,7,8 = 需要二次认证
	SignKey     string         `json:"sign_key"`
	SignCheck   string         `json:"sign_check"`
	FileID      string         `json:"file_id"`
	Target      string         `json:"target"`
	Bucket      string         `json:"bucket"`
	Object      string         `json:"object"`
	Callback    UploadCallback `json:"callback"`
	CallbackVar string         `json:"callback_var"`
}

// UploadCallback represents the callback structure from 115 API
// It can be either an object or an array
type UploadCallback struct {
	Value    CallbackValue `json:"value"`
	Array    []CallbackValue
	IsArray  bool
}

func (uc *UploadCallback) UnmarshalJSON(data []byte) error {
	// Try to parse as object first
	var obj CallbackValue
	if err := json.Unmarshal(data, &obj); err == nil {
		uc.Value = obj
		uc.IsArray = false
		return nil
	}

	// Try to parse as array
	var arr []CallbackValue
	if err := json.Unmarshal(data, &arr); err == nil {
		uc.Array = arr
		uc.IsArray = true
		if len(arr) > 0 {
			uc.Value = arr[0]
		}
		return nil
	}

	return fmt.Errorf("callback is neither object nor array")
}

// CallbackValue represents the callback value
type CallbackValue struct {
	Callback    string `json:"callback"`
	CallbackVar string `json:"callback_var"`
}

// UploadGetTokenReq represents get upload token request
type UploadGetTokenReq struct {
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}

// UploadGetTokenResp represents get upload token response
type UploadGetTokenResp struct {
	Code            int              `json:"code"`
	Message         string           `json:"message"`
	Data            TokenData        `json:"data"`
}

// TokenData represents the token data
type TokenData struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyId     string `json:"AccessKeyId"`
	AccessKeySecret string `json:"AccessKeySecret"`
	SecurityToken   string `json:"SecurityToken"`
}

// DownloadURLResp represents download URL response
type DownloadURLResp struct {
	Code    int                `json:"code"`
	Message string             `json:"message"`
	Data    map[string]DownloadURL `json:"data"`
}

// DownloadURL represents download URL info
type DownloadURL struct {
	URL   string `json:"url"`
	Expire int64  `json:"expire"`
}

// OfflineTaskListResp represents offline task list response
type OfflineTaskListResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Count   int         `json:"count"`
	Data    []OfflineTask `json:"data"`
}

// OfflineTask represents an offline download task
type OfflineTask struct {
	InfoHash   string `json:"info_hash"`
	FileID     string `json:"file_id"`
	Name       string `json:"name"`
	Size       int64  `json:"size,string"`
	Status     int    `json:"status"`
	CreateTime int64  `json:"create_time,string"`
}

// AddOfflineTaskReq represents add offline task request
type AddOfflineTaskReq struct {
	URL      string `json:"url"`
	ParentID string `json:"parent_id"`
}

// AddOfflineTaskResp represents add offline task response
type AddOfflineTaskResp struct {
	Code      int      `json:"code"`
	Message   string   `json:"message"`
	InfoHash  string   `json:"info_hash"`
	FileID    string   `json:"file_id"`
}

// DeleteOfflineTaskReq represents delete offline task request
type DeleteOfflineTaskReq struct {
	InfoHash string `json:"info_hash"`
	DeleteFile bool `json:"delete_file"`
}

// ParseTime parses time string to time.Time
func ParseTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	t, err := time.Parse(timeFormat, timeStr)
	if err != nil {
		return time.Time{}
	}
	return t
}

// ParseInt64 parses JSON number to int64
func ParseInt64(v json.Number) (int64, error) {
	i, err := v.Int64()
	if err == nil {
		return i, nil
	}
	f, e1 := v.Float64()
	if e1 == nil {
		return int64(f), nil
	}
	return int64(0), err
}
