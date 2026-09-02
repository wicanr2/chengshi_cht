// Package settings 保存 remake 自己的玩家設定；不碰原版城市存檔。
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/chengshi_cht/internal/i18n"
)

const Version = 1

type File struct {
	Version  int       `json:"version"`
	Language i18n.Lang `json:"language"`
	// SaveFormat 是存檔要寫哪一種版面："dos"（128 位元組檔頭 ＋ 27120，
	// 城市名存得住）或 "bare"（27120 裸檔身，餵得進 Micropolis）。
	//
	// ⚠ 這個欄位是後加的，所以**不升版本號**：舊的設定檔沒有這一欄，
	// 解出來是空字串，由呼叫端當成預設值。升版本會讓既有設定檔整份被判為
	// 不支援，玩家的語言選擇跟著一起掉。
	SaveFormat string `json:"save_format,omitempty"`
}

// DefaultPath 使用作業系統的使用者設定目錄，不與存檔或原版資料混在一起。
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return "", errors.New("無法取得使用者設定目錄")
	}
	return filepath.Join(dir, "chengshi", "settings.json"), nil
}

func validLang(l i18n.Lang) bool {
	for _, candidate := range i18n.Langs {
		if l == candidate {
			return true
		}
	}
	return false
}

// Load 嚴格讀取已知版本；損壞或未知值交由呼叫端退回預設值。
func Load(path string) (File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(raw, &f); err != nil {
		return File{}, fmt.Errorf("設定檔格式錯誤：%w", err)
	}
	if f.Version != Version {
		return File{}, fmt.Errorf("不支援的設定版本 %d", f.Version)
	}
	if !validLang(f.Language) {
		return File{}, fmt.Errorf("不支援的語言 %q", f.Language)
	}
	if !validSaveFormat(f.SaveFormat) {
		return File{}, fmt.Errorf("不支援的存檔格式 %q", f.SaveFormat)
	}
	return f, nil
}

// validSaveFormat 空字串代表舊設定檔沒有這一欄，交由呼叫端用預設值。
func validSaveFormat(s string) bool {
	switch s {
	case "", "dos", "bare":
		return true
	}
	return false
}

// Save 以同目錄暫存檔加 rename 原子替換，避免中途結束留下半份 JSON。
func Save(path string, lang i18n.Lang, saveFormat string) error {
	if !validLang(lang) {
		return fmt.Errorf("不支援的語言 %q", lang)
	}
	if !validSaveFormat(saveFormat) {
		return fmt.Errorf("不支援的存檔格式 %q", saveFormat)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(
		File{Version: Version, Language: lang, SaveFormat: saveFormat}, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(name)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	ok = true
	return nil
}
