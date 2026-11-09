// @vitest-environment node
import { describe, expect, it } from "vitest";
import { localUploadDownloadURL } from "./attachment-url";

describe("old local upload references", () => {
  const ws = "22222222-2222-4222-8222-222222222222";
  const id = "11111111-1111-4111-8111-111111111111";
  it("uses stable authorized download paths across client origins", () => {
    for (const prefix of ["", "https://selfhost.example"]) {
      expect(localUploadDownloadURL(`${prefix}/uploads/workspaces/${ws}/${id}.png?old=1`))
        .toBe(`/api/attachments/${id}/download`);
    }
  });
  it("does not treat directories, sidecars, arbitrary names or external paths as attachments", () => {
    for (const path of ["/uploads/", `/uploads/workspaces/${ws}/%2F`, `/uploads/workspaces/${ws}/${id}.png.meta.json`,
      "/uploads/avatar.png", `https://example.com/workspaces/${ws}/${id}.png`]) {
      expect(localUploadDownloadURL(path)).toBe(path);
    }
  });
});
