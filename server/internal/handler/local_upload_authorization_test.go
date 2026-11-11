package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/multica-ai/multica/server/internal/storage"
	"github.com/multica-ai/multica/server/internal/testutil"
)

func TestServeLocalUpload_AuthorizedLegacyAttachment(t *testing.T) {
	t.Setenv("LOCAL_UPLOAD_DIR", t.TempDir())
	t.Setenv("LOCAL_UPLOAD_BASE_URL", "")
	local := storage.NewLocalStorageFromEnv()
	original := testHandler.Storage
	testHandler.Storage = local
	t.Cleanup(func() { testHandler.Storage = original })
	id := uuid.NewString()
	key := "workspaces/" + testWorkspaceID + "/" + id + ".png"
	const body = "private image bytes"
	if _, err := local.Upload(context.Background(), key, []byte(body), "image/png", "image.png"); err != nil {
		t.Fatal(err)
	}
	dbfx.Insert(t, "attachment", testutil.Cols{
		"id": id, "workspace_id": testWorkspaceID, "uploader_type": "member", "uploader_id": testUserID,
		"filename": "image.png", "url": "/uploads/" + key, "content_type": "image/png", "size_bytes": len(body),
	})
	other := dbfx.User(t, "Legacy outsider", "legacy-outsider@example.invalid")
	for _, tc := range []struct {
		name, path, user, rangeHeader, actor, boundWorkspace string
		want                                                 int
	}{
		{name: "member", path: "/uploads/" + key, user: testUserID, want: 200},
		{name: "anonymous handler", path: "/uploads/" + key, want: 401},
		{name: "non-member", path: "/uploads/" + key, user: other, want: 404},
		{name: "range", path: "/uploads/" + key, user: testUserID, rangeHeader: "bytes=0-6", want: 206},
		{name: "wrong extension", path: "/uploads/" + strings.TrimSuffix(key, ".png") + ".jpg", user: testUserID, want: 404},
		{name: "wrong workspace", path: "/uploads/workspaces/" + uuid.NewString() + "/" + id + ".png", user: testUserID, want: 404},
		{name: "sidecar", path: "/uploads/" + key + ".meta.json", user: testUserID, want: 404},
		{name: "directory", path: "/uploads/workspaces/" + testWorkspaceID + "/", user: testUserID, want: 404},
		{name: "task binding", path: "/uploads/" + key, user: testUserID, actor: "task_token", boundWorkspace: uuid.NewString(), want: 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("X-User-ID", tc.user)
			req.Header.Set("Range", tc.rangeHeader)
			req.Header.Set("X-Actor-Source", tc.actor)
			req.Header.Set("X-Workspace-ID", tc.boundWorkspace)
			w := httptest.NewRecorder()
			testHandler.ServeLocalUpload(w, req)
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d; %s", w.Code, tc.want, w.Body.String())
			}
			if tc.want == 200 || tc.want == 206 {
				wantBody := body
				if tc.want == 206 {
					wantBody = body[:7]
				}
				if w.Body.String() != wantBody || w.Header().Get("Content-Type") != "image/png" ||
					w.Header().Get("Cache-Control") != "no-store" || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "inline") {
					t.Fatalf("incorrect media body or headers: %s %v", w.Body.String(), w.Header())
				}
			} else if strings.Contains(w.Body.String(), body) {
				t.Fatal("unauthorized bytes returned")
			}
		})
	}
}
