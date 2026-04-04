package datadiff

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestFormatDataReport_NoDiff(t *testing.T) {
	results := []TableDiffResult{
		{Schema: "public", Name: "users"},
		{Schema: "public", Name: "orders"},
	}
	var buf bytes.Buffer
	formatDataReport(results, "/bkp/a", "/bkp/b", &buf)
	out := buf.String()

	if !strings.Contains(out, "差分なし") {
		t.Errorf("expected '差分なし' in output, got:\n%s", out)
	}
	if strings.Contains(out, "データ変更テーブル") {
		t.Errorf("did not expect '[データ変更テーブル]' section in no-diff output")
	}
}

func TestFormatDataReport_WithChanges(t *testing.T) {
	results := []TableDiffResult{
		{Schema: "public", Name: "users", Added: 3, Removed: 1, Changed: 2},
		{Schema: "public", Name: "orders", Added: 10},
		{Schema: "public", Name: "products"},
	}
	var buf bytes.Buffer
	formatDataReport(results, "/bkp/a", "/bkp/b", &buf)
	out := buf.String()

	if !strings.Contains(out, "データ変更テーブル") {
		t.Errorf("expected '[データ変更テーブル]' section")
	}
	if !strings.Contains(out, "追加 3行") {
		t.Errorf("expected '追加 3行' in output")
	}
	if !strings.Contains(out, "削除 1行") {
		t.Errorf("expected '削除 1行' in output")
	}
	if !strings.Contains(out, "変更 2行") {
		t.Errorf("expected '変更 2行' in output")
	}
	if !strings.Contains(out, "public.orders") {
		t.Errorf("expected 'public.orders' in output")
	}
}

func TestFormatDataReport_NoPK(t *testing.T) {
	results := []TableDiffResult{
		{Schema: "public", Name: "audit_logs", NoPK: true},
	}
	var buf bytes.Buffer
	formatDataReport(results, "/bkp/a", "/bkp/b", &buf)
	out := buf.String()

	if !strings.Contains(out, "主キーなし") {
		t.Errorf("expected '主キーなし' section in output, got:\n%s", out)
	}
	if !strings.Contains(out, "public.audit_logs") {
		t.Errorf("expected 'public.audit_logs' in output")
	}
}

func TestFormatDataReport_Error(t *testing.T) {
	results := []TableDiffResult{
		{Schema: "public", Name: "broken", Err: fmt.Errorf("disk read error")},
	}
	var buf bytes.Buffer
	formatDataReport(results, "/bkp/a", "/bkp/b", &buf)
	out := buf.String()

	if !strings.Contains(out, "エラー") {
		t.Errorf("expected '[エラー]' section in output, got:\n%s", out)
	}
}

func TestFormatDataReport_Header(t *testing.T) {
	var buf bytes.Buffer
	formatDataReport(nil, "/backups/2024-01", "/backups/2024-02", &buf)
	out := buf.String()

	if !strings.Contains(out, "2024-01") || !strings.Contains(out, "2024-02") {
		t.Errorf("expected backup dir names in header, got:\n%s", out)
	}
}
