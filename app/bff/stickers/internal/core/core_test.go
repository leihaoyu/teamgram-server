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

		// [0] documentAttributeSticker
		if attrs[0].GetPredicateName() != mtproto.Predicate_documentAttributeSticker {
			t.Errorf("expected documentAttributeSticker, got %s", attrs[0].GetPredicateName())
		}

		// [1] documentAttributeVideo for video stickers
		videoAttr := attrs[1]
		if videoAttr.GetPredicateName() != mtproto.Predicate_documentAttributeVideo {
			t.Errorf("expected documentAttributeVideo, got %s", videoAttr.GetPredicateName())
		}
		if videoAttr.GetW() != 512 || videoAttr.GetH() != 512 {
			t.Errorf("video size: got %dx%d, want 512x512", videoAttr.GetW(), videoAttr.GetH())
		}

		// [2] documentAttributeFilename with .webm extension
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

// TestExtractStickerEmoji tests emoji extraction from Document attributes.
func TestExtractStickerEmoji(t *testing.T) {
	doc := mtproto.MakeTLDocument(&mtproto.Document{
		Id:         123,
		AccessHash: 456,
		MimeType:   "image/webp",
		Attributes: []*mtproto.DocumentAttribute{
			mtproto.MakeTLDocumentAttributeSticker(&mtproto.DocumentAttribute{
				Alt: "🦆",
				Stickerset: mtproto.MakeTLInputStickerSetID(&mtproto.InputStickerSet{
					Id: 1, AccessHash: 2,
				}).To_InputStickerSet(),
			}).To_DocumentAttribute(),
			mtproto.MakeTLDocumentAttributeImageSize(&mtproto.DocumentAttribute{
				W: 512, H: 512,
			}).To_DocumentAttribute(),
		},
	}).To_Document()

	emoji := extractStickerEmoji(doc)
	if emoji != "🦆" {
		t.Errorf("extractStickerEmoji: got %q, want %q", emoji, "🦆")
	}

	// Test with no sticker attribute
	docNoSticker := mtproto.MakeTLDocument(&mtproto.Document{
		Id:       789,
		MimeType: "image/webp",
		Attributes: []*mtproto.DocumentAttribute{
			mtproto.MakeTLDocumentAttributeImageSize(&mtproto.DocumentAttribute{
				W: 512, H: 512,
			}).To_DocumentAttribute(),
		},
	}).To_Document()

	emoji2 := extractStickerEmoji(docNoSticker)
	if emoji2 != "" {
		t.Errorf("extractStickerEmoji(no sticker attr): got %q, want empty", emoji2)
	}

	t.Log("extractStickerEmoji: PASS")
}

// TestComputeRecentStickersHash tests hash computation and consistency.
func TestComputeRecentStickersHash(t *testing.T) {
	// Empty rows → hash 0
	h0 := computeRecentStickersHash(nil)
	if h0 != 0 {
		t.Errorf("empty hash: got %d, want 0", h0)
	}

	rows := []dataobject.UserRecentStickersDO{
		{DocumentId: 100, Date2: 1000},
		{DocumentId: 200, Date2: 999},
		{DocumentId: 300, Date2: 998},
	}

	h1 := computeRecentStickersHash(rows)
	h2 := computeRecentStickersHash(rows)
	if h1 != h2 {
		t.Errorf("hash not stable: %d vs %d", h1, h2)
	}
	if h1 == 0 {
		t.Error("hash should not be 0 for non-empty rows")
	}

	// Different order → different hash
	rows2 := []dataobject.UserRecentStickersDO{
		{DocumentId: 200, Date2: 999},
		{DocumentId: 100, Date2: 1000},
		{DocumentId: 300, Date2: 998},
	}
	h3 := computeRecentStickersHash(rows2)
	if h3 == h1 {
		t.Error("different order should produce different hash")
	}

	// Verify Telegram hash algorithm manually for single element:
	// acc=0, combine(0,100): acc^=0>>21=0, acc^=0<<35=0, acc^=0>>4=0, acc+=100 → acc=100
	singleRow := []dataobject.UserRecentStickersDO{{DocumentId: 100}}
	hSingle := computeRecentStickersHash(singleRow)
	if hSingle != 100 {
		t.Errorf("single element hash: got %d, want 100", hSingle)
	}

	t.Logf("computeRecentStickersHash (telegramHash): PASS (h1=%d, h3=%d)", h1, h3)
}

// TestComputeFavedStickersHashReversed verifies faved stickers hash uses reversed order.
func TestComputeFavedStickersHashReversed(t *testing.T) {
	rows := []dataobject.UserFavedStickersDO{
		{DocumentId: 100},
		{DocumentId: 200},
	}
	hNormal := computeFavedStickersHash(rows)

	rowsReversed := []dataobject.UserFavedStickersDO{
		{DocumentId: 200},
		{DocumentId: 100},
	}
	hReversed := computeFavedStickersHash(rowsReversed)

	// faved hash reverses internally, so rows=[100,200] hashes as [200,100]
	// and rowsReversed=[200,100] hashes as [100,200]
	// They should NOT be equal since both get reversed differently
	if hNormal == hReversed {
		t.Error("faved hashes should differ for different input order")
	}

	// Verify: hash of [A] reversed = hash of [A] (single element)
	single := []dataobject.UserFavedStickersDO{{DocumentId: 42}}
	if computeFavedStickersHash(single) != 42 {
		t.Errorf("single faved element hash: got %d, want 42", computeFavedStickersHash(single))
	}

	t.Logf("computeFavedStickersHash (reversed): PASS (hNormal=%d, hReversed=%d)", hNormal, hReversed)
}

// TestBuildUserStickerPacks tests emoji→documentId grouping for recent stickers.
func TestBuildUserStickerPacks(t *testing.T) {
	rows := []dataobject.UserRecentStickersDO{
		{DocumentId: 1, Emoji: "😂"},
		{DocumentId: 2, Emoji: "🦆"},
		{DocumentId: 3, Emoji: "😂"},
		{DocumentId: 4, Emoji: ""},
	}

	packs := buildUserStickerPacks(rows)

	emojiMap := make(map[string][]int64)
	for _, p := range packs {
		emojiMap[p.GetEmoticon()] = p.GetDocuments()
	}

	if len(packs) != 2 {
		t.Errorf("expected 2 packs, got %d", len(packs))
	}
	if docs, ok := emojiMap["😂"]; !ok || len(docs) != 2 {
		t.Errorf("😂 pack: got %v", docs)
	}
	if docs, ok := emojiMap["🦆"]; !ok || len(docs) != 1 {
		t.Errorf("🦆 pack: got %v", docs)
	}

	t.Log("buildUserStickerPacks: PASS")
}

// TestBuildFavedStickerPacks tests emoji→documentId grouping for faved stickers.
func TestBuildFavedStickerPacks(t *testing.T) {
	rows := []dataobject.UserFavedStickersDO{
		{DocumentId: 10, Emoji: "❤️"},
		{DocumentId: 20, Emoji: "❤️"},
		{DocumentId: 30, Emoji: "😎"},
	}

	packs := buildFavedStickerPacks(rows)

	emojiMap := make(map[string][]int64)
	for _, p := range packs {
		emojiMap[p.GetEmoticon()] = p.GetDocuments()
	}

	if len(packs) != 2 {
		t.Errorf("expected 2 packs, got %d", len(packs))
	}
	if docs, ok := emojiMap["❤️"]; !ok || len(docs) != 2 {
		t.Errorf("❤️ pack: got %v", docs)
	}

	t.Log("buildFavedStickerPacks: PASS")
}

func TestSetTypeFromFlags(t *testing.T) {
	if st := setTypeFromFlags(false, false); st != 0 {
		t.Errorf("expected 0 (regular), got %d", st)
	}
	if st := setTypeFromFlags(true, false); st != 1 {
		t.Errorf("expected 1 (masks), got %d", st)
	}
	if st := setTypeFromFlags(false, true); st != 2 {
		t.Errorf("expected 2 (emojis), got %d", st)
	}
	// masks takes priority
	if st := setTypeFromFlags(true, true); st != 1 {
		t.Errorf("expected 1 (masks priority), got %d", st)
	}
	t.Log("setTypeFromFlags: PASS")
}

func TestComputeInstalledSetsHash(t *testing.T) {
	// empty → 0
	h0 := computeInstalledSetsHash(nil)
	if h0 != 0 {
		t.Errorf("expected 0, got %d", h0)
	}

	rows := []dataobject.UserInstalledStickerSetsDO{
		{SetId: 111},
		{SetId: 222},
		{SetId: 333},
	}
	h1 := computeInstalledSetsHash(rows)
	if h1 == 0 {
		t.Error("expected non-zero hash")
	}

	// Same input → same hash (deterministic)
	h2 := computeInstalledSetsHash(rows)
	if h1 != h2 {
		t.Errorf("hashes differ: %d vs %d", h1, h2)
	}

	// Different order → different hash
	rows2 := []dataobject.UserInstalledStickerSetsDO{
		{SetId: 222},
		{SetId: 111},
		{SetId: 333},
	}
	h3 := computeInstalledSetsHash(rows2)
	if h1 == h3 {
		t.Error("different order should produce different hash")
	}

	// Verify single element matches Telegram algorithm
	single := []dataobject.UserInstalledStickerSetsDO{{SetId: 777}}
	if computeInstalledSetsHash(single) != 777 {
		t.Errorf("single element hash: got %d, want 777", computeInstalledSetsHash(single))
	}

	t.Logf("computeInstalledSetsHash (telegramHash): PASS (h1=%d, h3=%d)", h1, h3)
}

func TestStickerSetType(t *testing.T) {
	regular := &dataobject.StickerSetsDO{IsMasks: false, IsEmojis: false}
	if stickerSetType(regular) != 0 {
		t.Error("expected 0 for regular")
	}

	masks := &dataobject.StickerSetsDO{IsMasks: true}
	if stickerSetType(masks) != 1 {
		t.Error("expected 1 for masks")
	}

	emojis := &dataobject.StickerSetsDO{IsEmojis: true}
	if stickerSetType(emojis) != 2 {
		t.Error("expected 2 for emojis")
	}

	t.Log("stickerSetType: PASS")
}

// ==============================================================================
// Tests for auto-save recent sticker logic (mirrors messages_plugin.go behavior)
// ==============================================================================

// TestAutoSaveStickerDetection verifies sticker detection from MessageMedia.
func TestAutoSaveStickerDetection(t *testing.T) {
	tests := []struct {
		name    string
		media   *mtproto.MessageMedia
		isStick bool
	}{
		{
			name: "webp sticker",
			media: mtproto.MakeTLMessageMediaDocument(&mtproto.MessageMedia{
				Document: mtproto.MakeTLDocument(&mtproto.Document{
					Id: 3001, MimeType: "image/webp",
					Attributes: []*mtproto.DocumentAttribute{
						mtproto.MakeTLDocumentAttributeSticker(&mtproto.DocumentAttribute{Alt: "😎"}).To_DocumentAttribute(),
					},
				}).To_Document(),
			}).To_MessageMedia(),
			isStick: true,
		},
		{
			name: "video sticker",
			media: mtproto.MakeTLMessageMediaDocument(&mtproto.MessageMedia{
				Document: mtproto.MakeTLDocument(&mtproto.Document{
					Id: 3002, MimeType: "video/webm",
					Attributes: []*mtproto.DocumentAttribute{
						mtproto.MakeTLDocumentAttributeVideo(&mtproto.DocumentAttribute{W: 512, H: 512}).To_DocumentAttribute(),
						mtproto.MakeTLDocumentAttributeSticker(&mtproto.DocumentAttribute{Alt: "🔥"}).To_DocumentAttribute(),
					},
				}).To_Document(),
			}).To_MessageMedia(),
			isStick: true,
		},
		{
			name:    "photo media - not sticker",
			media:   mtproto.MakeTLMessageMediaPhoto(&mtproto.MessageMedia{}).To_MessageMedia(),
			isStick: false,
		},
		{
			name: "pdf document - not sticker",
			media: mtproto.MakeTLMessageMediaDocument(&mtproto.MessageMedia{
				Document: mtproto.MakeTLDocument(&mtproto.Document{
					Id: 3003, MimeType: "application/pdf",
					Attributes: []*mtproto.DocumentAttribute{
						mtproto.MakeTLDocumentAttributeFilename(&mtproto.DocumentAttribute{FileName: "report.pdf"}).To_DocumentAttribute(),
					},
				}).To_Document(),
			}).To_MessageMedia(),
			isStick: false,
		},
		{
			name:    "nil media",
			media:   nil,
			isStick: false,
		},
		{
			name: "document media with nil document",
			media: mtproto.MakeTLMessageMediaDocument(&mtproto.MessageMedia{
				Document: nil,
			}).To_MessageMedia(),
			isStick: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectStickerInMedia(tt.media)
			if got != tt.isStick {
				t.Errorf("detectStickerInMedia() = %v, want %v", got, tt.isStick)
			}
		})
	}
	t.Log("autoSaveStickerDetection: PASS")
}

// TestAutoSaveEmojiExtraction verifies emoji extraction from sticker Documents.
func TestAutoSaveEmojiExtraction(t *testing.T) {
	tests := []struct {
		name      string
		doc       *mtproto.Document
		wantEmoji string
	}{
		{
			name: "normal sticker",
			doc: mtproto.MakeTLDocument(&mtproto.Document{
				Id: 4001, MimeType: "image/webp",
				Attributes: []*mtproto.DocumentAttribute{
					mtproto.MakeTLDocumentAttributeSticker(&mtproto.DocumentAttribute{Alt: "😂"}).To_DocumentAttribute(),
				},
			}).To_Document(),
			wantEmoji: "😂",
		},
		{
			name: "multi-attribute sticker",
			doc: mtproto.MakeTLDocument(&mtproto.Document{
				Id: 4002, MimeType: "video/webm",
				Attributes: []*mtproto.DocumentAttribute{
					mtproto.MakeTLDocumentAttributeVideo(&mtproto.DocumentAttribute{W: 512, H: 512}).To_DocumentAttribute(),
					mtproto.MakeTLDocumentAttributeSticker(&mtproto.DocumentAttribute{Alt: "👍"}).To_DocumentAttribute(),
				},
			}).To_Document(),
			wantEmoji: "👍",
		},
		{
			name: "non-sticker document",
			doc: mtproto.MakeTLDocument(&mtproto.Document{
				Id: 4003, MimeType: "image/jpeg",
				Attributes: []*mtproto.DocumentAttribute{
					mtproto.MakeTLDocumentAttributeImageSize(&mtproto.DocumentAttribute{W: 800, H: 600}).To_DocumentAttribute(),
				},
			}).To_Document(),
			wantEmoji: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractStickerEmoji(tt.doc)
			if got != tt.wantEmoji {
				t.Errorf("extractStickerEmoji() = %q, want %q", got, tt.wantEmoji)
			}
		})
	}
	t.Log("autoSaveEmojiExtraction: PASS")
}

// TestAutoSaveSerializeRoundtrip verifies document serialization works for auto-save use case.
func TestAutoSaveSerializeRoundtrip(t *testing.T) {
	doc := mtproto.MakeTLDocument(&mtproto.Document{
		Id:          5001,
		AccessHash:  99999,
		MimeType:    "image/webp",
		Size2_INT32: 5000,
		Attributes: []*mtproto.DocumentAttribute{
			mtproto.MakeTLDocumentAttributeSticker(&mtproto.DocumentAttribute{Alt: "🎉"}).To_DocumentAttribute(),
			mtproto.MakeTLDocumentAttributeImageSize(&mtproto.DocumentAttribute{W: 512, H: 512}).To_DocumentAttribute(),
		},
	}).To_Document()

	// Serialize
	docData, err := dao.SerializeStickerDoc(doc)
	if err != nil {
		t.Fatalf("SerializeStickerDoc error: %v", err)
	}
	if docData == "" {
		t.Fatal("serialized data is empty")
	}

	// Deserialize
	doc2, err := dao.DeserializeStickerDoc(docData)
	if err != nil {
		t.Fatalf("DeserializeStickerDoc error: %v", err)
	}
	if doc2.GetId() != 5001 {
		t.Errorf("roundtrip Id = %d, want 5001", doc2.GetId())
	}
	if doc2.GetAccessHash() != 99999 {
		t.Errorf("roundtrip AccessHash = %d, want 99999", doc2.GetAccessHash())
	}
	// Verify emoji survives roundtrip
	emoji := extractStickerEmoji(doc2)
	if emoji != "🎉" {
		t.Errorf("roundtrip emoji = %q, want 🎉", emoji)
	}

	t.Logf("autoSaveSerializeRoundtrip: PASS (%d chars base64)", len(docData))
}

// TestAutoSaveNoDuplicateDetection verifies that re-sending the same sticker should still be detected.
func TestAutoSaveNoDuplicateDetection(t *testing.T) {
	doc := mtproto.MakeTLDocument(&mtproto.Document{
		Id: 6001, MimeType: "image/webp",
		Attributes: []*mtproto.DocumentAttribute{
			mtproto.MakeTLDocumentAttributeSticker(&mtproto.DocumentAttribute{Alt: "🐱"}).To_DocumentAttribute(),
		},
	}).To_Document()

	media := mtproto.MakeTLMessageMediaDocument(&mtproto.MessageMedia{
		Document: doc,
	}).To_MessageMedia()

	// Detect 100 times (simulating sending same sticker repeatedly)
	for i := 0; i < 100; i++ {
		if !detectStickerInMedia(media) {
			t.Fatalf("iteration %d: failed to detect sticker", i)
		}
	}
	t.Log("autoSaveNoDuplicateDetection: PASS (detection is idempotent)")
}

// detectStickerInMedia mirrors the logic of maybeRecordRecentSticker in messages module.
func detectStickerInMedia(media *mtproto.MessageMedia) bool {
	if media == nil || media.GetPredicateName() != mtproto.Predicate_messageMediaDocument {
		return false
	}
	doc := media.GetDocument()
	if doc == nil {
		return false
	}
	for _, attr := range doc.GetAttributes() {
		if attr.GetPredicateName() == mtproto.Predicate_documentAttributeSticker {
			return true
		}
	}
	return false
}
