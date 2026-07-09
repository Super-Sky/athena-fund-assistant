package conversation

import (
	"context"
	"strings"
	"testing"
)

func TestMemoryStoreConversationAttachmentAndTrace(t *testing.T) {
	store, err := NewMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store error = %v", err)
	}
	ctx := context.Background()
	detail, err := store.Create(ctx, CreateInput{UserID: "demo-user", SkillID: "document_intake"})
	if err != nil {
		t.Fatalf("create conversation error = %v", err)
	}
	attachment, err := store.SaveAttachment(ctx, detail.Session.ID, AttachmentInput{
		UserID:      "demo-user",
		FileName:    "statement.csv",
		ContentType: "text/csv",
		SizeBytes:   int64(len("symbol,amount\n510300,100\n")),
		Reader:      strings.NewReader("symbol,amount\n510300,100\n"),
	})
	if err != nil {
		t.Fatalf("save attachment error = %v", err)
	}
	if attachment.Status != "pending_parse" || attachment.Unsupported {
		t.Fatalf("attachment = %#v, want pending supported metadata", attachment)
	}
	updated, err := store.AddMessage(ctx, detail.Session.ID, MessageInput{
		Role:          "user",
		Content:       "请读取这份账单并给我一个组合复盘。",
		SkillID:       "document_intake",
		AttachmentIDs: []string{attachment.ID},
	})
	if err != nil {
		t.Fatalf("add message error = %v", err)
	}
	if len(updated.Messages) != 1 || len(updated.Attachments) != 1 {
		t.Fatalf("updated detail = %#v", updated)
	}
	foundPending := false
	for _, event := range updated.Trace {
		if event.Kind == "attachment_context" && event.Status == "pending" {
			foundPending = true
		}
	}
	if !foundPending {
		t.Fatalf("trace missing pending attachment context: %#v", updated.Trace)
	}
}

func TestMemoryStoreRejectsOversizedAttachment(t *testing.T) {
	store, err := NewMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store error = %v", err)
	}
	detail, err := store.Create(context.Background(), CreateInput{})
	if err != nil {
		t.Fatalf("create conversation error = %v", err)
	}
	_, err = store.SaveAttachment(context.Background(), detail.Session.ID, AttachmentInput{
		FileName:    "too-large.pdf",
		ContentType: "application/pdf",
		SizeBytes:   maxAttachmentSizeBytes + 1,
		Reader:      strings.NewReader("x"),
	})
	if err == nil {
		t.Fatal("expected oversized attachment error")
	}
}
