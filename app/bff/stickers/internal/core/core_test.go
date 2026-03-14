package core

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dal/dataobject"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dao"
)

// TestDocumentSerializationRoundtrip tests that Document protobuf survives serialize/deserialize
func TestDocumentSerializationRoundtrip(t *testing.T) {
	now := time.Now().Unix()

	sticker := dao.BotAPISticker{
		FileId:       "CAACAgIAAxkBAAIBdGF...",
		FileUniqueId: "AgADYQAD",
		FileSize:     5272,
		Width:        512,
		Height:       512,
		Emoji:        "😂",
		SetName:      "UtyaDuck",
		IsAnimated:   true,
		IsVideo:      false,
		Type:         "regular",
	}

	docId := int64(7654321)
	setId := int64(1234567)
	setAccessHash := int64(9999)
	mimeType := stickerMimeType(sticker)
	attributes := buildDocumentAttributes(sticker, setId, setAccessHash)

	original := mtproto.MakeTLDocument(&mtproto.Document{
		Id:            docId,
		AccessHash:    12345,
		FileReference: []byte{},
		Date:          int32(now),
		MimeType:      mimeType,
		Size2_INT32:   int32(sticker.FileSize),
		Size2_INT64:   sticker.FileSize,
		Thumbs:        nil,
		VideoThumbs:   nil,
		DcId:          1,
		Attributes:    attributes,
	}).To_Document()

	// Serialize
	serialized, err := dao.SerializeStickerDoc(original)
	if err != nil {
		t.Fatalf("SerializeStickerDoc failed: %v", err)
	}
	if serialized == "" {
		t.Fatal("serialized document is empty string")
	}
	t.Logf("Serialized document: %d chars (base64)", len(serialized))

	// Deserialize
	restored, err := dao.DeserializeStickerDoc(serialized)
	if err != nil {
		t.Fatalf("DeserializeStickerDoc failed: %v", err)
	}

	// Verify key fields
	if restored.Id != original.Id {
		t.Errorf("Id: got %d, want %d", restored.Id, original.Id)
	}
	if restored.AccessHash != original.AccessHash {
		t.Errorf("AccessHash: got %d, want %d", restored.AccessHash, original.AccessHash)
	}
	if restored.MimeType != original.MimeType {
		t.Errorf("MimeType: got %s, want %s", restored.MimeType, original.MimeType)
	}
	if restored.DcId != original.DcId {
		t.Errorf("DcId: got %d, want %d", restored.DcId, original.DcId)
	}
	if restored.Date != original.Date {
		t.Errorf("Date: got %d, want %d", restored.Date, original.Date)
	}
	if restored.Size2_INT64 != original.Size2_INT64 {
		t.Errorf("Size2_INT64: got %d, want %d", restored.Size2_INT64, original.Size2_INT64)
	}
	if len(restored.Attributes) != len(original.Attributes) {
		t.Fatalf("Attributes count: got %d, want %d", len(restored.Attributes), len(original.Attributes))
	}

	if !proto.Equal(original, restored) {
		t.Error("proto.Equal returned false: original and restored documents differ")
	}

	t.Log("Document serialization roundtrip: PASS")
}

// TestDocumentSerializationNoThumbs tests Document without thumbnails
func TestDocumentSerializationNoThumbs(t *testing.T) {
	doc := mtproto.MakeTLDocument(&mtproto.Document{
		Id:            999,
		AccessHash:    888,
		FileReference: []byte{},
		Date:          int32(time.Now().Unix()),
		MimeType:      "video/webm",
		Size2_INT32:   1024,
		Size2_INT64:   1024,
		Thumbs:        nil,
		VideoThumbs:   nil,
		DcId:          1,
		Attributes:    []*mtproto.DocumentAttribute{},
	}).To_Document()

	serialized, err := dao.SerializeStickerDoc(doc)
	if err != nil {
		t.Fatalf("SerializeStickerDoc failed: %v", err)
	}

	restored, err := dao.DeserializeStickerDoc(serialized)
	if err != nil {
		t.Fatalf("DeserializeStickerDoc failed: %v", err)
	}

	if restored.Id != doc.Id {
		t.Errorf("Id: got %d, want %d", restored.Id, doc.Id)
	}
	if restored.MimeType != "video/webm" {
		t.Errorf("MimeType: got %s, want video/webm", restored.MimeType)
	}

	t.Log("Document without thumbs: PASS")
}

// TestStickerMimeType verifies MIME type determination
func TestStickerMimeType(t *testing.T) {
	tests := []struct {
		name     string
		sticker  dao.BotAPISticker
		wantMime string
		wantExt  string
	}{
		{
			name:     "animated TGS",
			sticker:  dao.BotAPISticker{IsAnimated: true, IsVideo: false},
			wantMime: "application/x-tgsticker",
			wantExt:  ".tgs",
		},
		{
			name:     "video WebM",
			sticker:  dao.BotAPISticker{IsAnimated: false, IsVideo: true},
			wantMime: "video/webm",
			wantExt:  ".webm",
		},
		{
			name:     "static WebP",
			sticker:  dao.BotAPISticker{IsAnimated: false, IsVideo: false},
			wantMime: "image/webp",
			wantExt:  ".webp",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotMime := stickerMimeType(tc.sticker)
			gotExt := stickerExt(tc.sticker)
			if gotMime != tc.wantMime {
				t.Errorf("stickerMimeType: got %s, want %s", gotMime, tc.wantMime)
			}
			if gotExt != tc.wantExt {
				t.Errorf("stickerExt: got %s, want %s", gotExt, tc.wantExt)
			}
		})
	}
}

// TestBuildDocumentAttributes verifies document attributes are correctly built
func TestBuildDocumentAttributes(t *testing.T) {
	t.Run("animated TGS", func(t *testing.T) {
		sticker := dao.BotAPISticker{
			Emoji:        "🦆",
			Width:        512,
			Height:       512,
			FileUniqueId: "AgADtest",
			IsAnimated:   true,
		}

		setId := int64(100)
		setAccessHash := int64(200)

		attrs := buildDocumentAttributes(sticker, setId, setAccessHash)

		if len(attrs) != 3 {
			t.Fatalf("expected 3 attributes, got %d", len(attrs))
		}

		// Verify documentAttributeSticker
		stickerAttr := attrs[0]
		if stickerAttr.GetAlt() != "🦆" {
			t.Errorf("sticker alt: got '%s', want '🦆'", stickerAttr.GetAlt())
		}
		stickerSetRef := stickerAttr.GetStickerset()
		if stickerSetRef == nil {
			t.Fatal("stickerset reference is nil")
		}
		if stickerSetRef.GetId() != setId {
			t.Errorf("stickerset id: got %d, want %d", stickerSetRef.GetId(), setId)
		}
		if stickerSetRef.GetAccessHash() != setAccessHash {
			t.Errorf("stickerset access_hash: got %d, want %d", stickerSetRef.GetAccessHash(), setAccessHash)
		}

		// Verify documentAttributeImageSize (for non-video stickers)
		imgAttr := attrs[1]
		if imgAttr.GetPredicateName() != mtproto.Predicate_documentAttributeImageSize {
			t.Errorf("expected documentAttributeImageSize, got %s", imgAttr.GetPredicateName())
		}
		if imgAttr.GetW() != 512 || imgAttr.GetH() != 512 {
			t.Errorf("image size: got %dx%d, want 512x512", imgAttr.GetW(), imgAttr.GetH())
		}

		// Verify documentAttributeFilename
		fileAttr := attrs[2]
		expectedName := "AgADtest.tgs"
		if fileAttr.GetFileName() != expectedName {
			t.Errorf("filename: got '%s', want '%s'", fileAttr.GetFileName(), expectedName)
		}
	})

	t.Run("video WebM", func(t *testing.T) {
		sticker := dao.BotAPISticker{
			Emoji:        "🐶",
			Width:        512,
			Height:       512,
			FileUniqueId: "AgADvideo",
			IsVideo:      true,
		}

		setId := int64(100)
		setAccessHash := int64(200)

		attrs := buildDocumentAttributes(sticker, setId, setAccessHash)

		if len(attrs) != 3 {
			t.Fatalf("expected 3 attributes, got %d", len(attrs))
		}

		// Verify documentAttributeVideo (for video stickers)
		videoAttr := attrs[1]
		if videoAttr.GetPredicateName() != mtproto.Predicate_documentAttributeVideo {
			t.Errorf("expected documentAttributeVideo, got %s", videoAttr.GetPredicateName())
		}
		if videoAttr.GetW() != 512 || videoAttr.GetH() != 512 {
			t.Errorf("video size: got %dx%d, want 512x512", videoAttr.GetW(), videoAttr.GetH())
		}
		if !videoAttr.GetNosound() {
			t.Error("video sticker should have Nosound=true")
		}

		// Verify documentAttributeFilename
		fileAttr := attrs[2]
		expectedName := "AgADvideo.webm"
		if fileAttr.GetFileName() != expectedName {
			t.Errorf("filename: got '%s', want '%s'", fileAttr.GetFileName(), expectedName)
		}
	})
}

// TestBuildStickerPacks verifies emoji -> document_id grouping
func TestBuildStickerPacks(t *testing.T) {
	docDOs := []dataobject.StickerSetDocumentsDO{
		{DocumentId: 1, Emoji: "😂"},
		{DocumentId: 2, Emoji: "😭"},
		{DocumentId: 3, Emoji: "😂"}, // duplicate emoji
		{DocumentId: 4, Emoji: "🦆"},
		{DocumentId: 5, Emoji: ""}, // empty emoji
	}

	packs := buildStickerPacks(docDOs)

	if len(packs) != 3 {
		t.Fatalf("expected 3 packs, got %d", len(packs))
	}

	packMap := make(map[string][]int64)
	for _, p := range packs {
		packMap[p.Emoticon] = p.Documents
	}

	if docs, ok := packMap["😂"]; !ok {
		t.Error("missing pack for 😂")
	} else if len(docs) != 2 || docs[0] != 1 || docs[1] != 3 {
		t.Errorf("😂 pack: got %v, want [1, 3]", docs)
	}

	if docs, ok := packMap["😭"]; !ok {
		t.Error("missing pack for 😭")
	} else if len(docs) != 1 || docs[0] != 2 {
		t.Errorf("😭 pack: got %v, want [2]", docs)
	}

	if docs, ok := packMap["🦆"]; !ok {
		t.Error("missing pack for 🦆")
	} else if len(docs) != 1 || docs[0] != 4 {
		t.Errorf("🦆 pack: got %v, want [4]", docs)
	}

	t.Log("Sticker packs: PASS")
}

// TestMakeStickerSetFromDO verifies StickerSet protobuf construction from DO
func TestMakeStickerSetFromDO(t *testing.T) {
	setDO := &dataobject.StickerSetsDO{
		SetId:        12345,
		AccessHash:   67890,
		ShortName:    "TestSet",
		Title:        "Test Title",
		StickerCount: 10,
		Hash:         0,
		IsAnimated:   true,
		IsVideo:      false,
		IsMasks:      false,
		IsEmojis:     false,
		IsOfficial:   false,
		ThumbDocId:   0,
	}

	ss := makeStickerSetFromDO(setDO)

	if ss.Id != 12345 {
		t.Errorf("Id: got %d, want 12345", ss.Id)
	}
	if ss.AccessHash != 67890 {
		t.Errorf("AccessHash: got %d, want 67890", ss.AccessHash)
	}
	if ss.ShortName != "TestSet" {
		t.Errorf("ShortName: got %s, want TestSet", ss.ShortName)
	}
	if ss.Title != "Test Title" {
		t.Errorf("Title: got %s, want Test Title", ss.Title)
	}
	if ss.Count != 10 {
		t.Errorf("Count: got %d, want 10", ss.Count)
	}
	if !ss.Animated {
		t.Error("Animated should be true")
	}
	if ss.Videos {
		t.Error("Videos should be false")
	}
	if ss.ThumbDocumentId != nil {
		t.Error("ThumbDocumentId should be nil when ThumbDocId is 0")
	}

	// Test with thumb
	setDO.ThumbDocId = 999
	ss2 := makeStickerSetFromDO(setDO)
	if ss2.ThumbDocumentId == nil || ss2.ThumbDocumentId.Value != 999 {
		t.Errorf("ThumbDocumentId: got %v, want 999", ss2.ThumbDocumentId)
	}

	t.Log("StickerSet from DO: PASS")
}
