package service

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// UploadDir 头像等上传文件的存放目录（相对后端工作目录），通过 /uploads/ 对外访问。
const UploadDir = "uploads"

// PUT /api/me { nickname?, avatarUrl? } 修改昵称或头像地址
func (a *App) UpdateMe(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var in struct {
		Nickname  *string `json:"nickname"`
		AvatarURL *string `json:"avatarUrl"`
		Role      *string `json:"role"`
	}
	if ReadJSON(r, &in) != nil {
		Fail(w, 400, "invalid json")
		return
	}
	set := bson.M{}
	if in.Nickname != nil {
		n := strings.TrimSpace(*in.Nickname)
		if n == "" || utf8.RuneCountInString(n) > 20 {
			Fail(w, 400, "nickname must be 1-20 chars")
			return
		}
		set["nickname"] = n
	}
	if in.AvatarURL != nil {
		set["avatarUrl"] = strings.TrimSpace(*in.AvatarURL)
	}
	if in.Role != nil {
		if *in.Role != "patient" && *in.Role != "family" {
			Fail(w, 400, "role must be patient or family")
			return
		}
		set["role"] = *in.Role
	}
	if len(set) == 0 {
		Fail(w, 400, "nothing to update")
		return
	}
	if _, err := a.C("users").UpdateOne(r.Context(), bson.M{"_id": uid}, bson.M{"$set": set}); err != nil {
		slog.Error("update profile failed", "err", err)
		Fail(w, 500, "db error")
		return
	}
	a.Me(w, r, uid)
}

var avatarExt = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}

// POST /api/me/avatar  multipart 字段名 file，最大 2MB，支持 jpg/png/webp
func (a *App) UploadAvatar(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	f, _, err := r.FormFile("file")
	if err != nil {
		Fail(w, 400, "file required (max 2MB)")
		return
	}
	defer f.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	ext, ok := avatarExt[http.DetectContentType(head[:n])]
	if !ok {
		Fail(w, 400, "only jpg/png/webp")
		return
	}
	dir := filepath.Join(UploadDir, "avatars")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Error("mkdir avatars failed", "err", err)
		Fail(w, 500, "storage error")
		return
	}
	name := uid.Hex() + ext
	out, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		slog.Error("save avatar failed", "err", err)
		Fail(w, 500, "storage error")
		return
	}
	_, err = io.Copy(out, io.MultiReader(strings.NewReader(string(head[:n])), f))
	out.Close()
	if err != nil {
		Fail(w, 400, "upload failed")
		return
	}
	url := "/uploads/avatars/" + name + "?v=" + strconv.FormatInt(time.Now().Unix(), 10)
	_, _ = a.C("users").UpdateOne(r.Context(), bson.M{"_id": uid}, bson.M{"$set": bson.M{"avatarUrl": url}})
	OK(w, bson.M{"avatarUrl": url})
}
