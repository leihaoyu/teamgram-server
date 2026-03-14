package dao

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dal/dao/mysql_dao"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dal/dataobject"
)

const testDBDSN = "root:my_root_secret@tcp(127.0.0.1:3306)/teamgram_stickers?charset=utf8mb4&parseTime=true"

func setupTestDB(t *testing.T) (*sqlx.DB, *mysql_dao.StickerSetsDAO, *mysql_dao.StickerSetDocumentsDAO) {
	t.Helper()
	db := sqlx.NewMySQL(&sqlx.Config{DSN: testDBDSN})
	return db, mysql_dao.NewStickerSetsDAO(db), mysql_dao.NewStickerSetDocumentsDAO(db)
}

func cleanupTestSet(t *testing.T, db *sqlx.DB, setId int64) {
	t.Helper()
	db.Exec(context.Background(), "DELETE FROM sticker_set_documents WHERE set_id = ?", setId)
	db.Exec(context.Background(), "DELETE FROM sticker_sets WHERE set_id = ?", setId)
}

// TestDBStickerSetsInsertAndSelect tests InsertIgnore + SelectByShortName + SelectBySetId
func TestDBStickerSetsInsertAndSelect(t *testing.T) {
	db, setsDAO, _ := setupTestDB(t)
	ctx := context.Background()

	setId := rand.Int63n(1e15) + 1e14
	defer cleanupTestSet(t, db, setId)

	do := &dataobject.StickerSetsDO{
		SetId:        setId,
		AccessHash:   rand.Int63(),
		ShortName:    "test_set_" + time.Now().Format("20060102150405"),
		Title:        "Test Sticker Set",
		StickerType:  "regular",
		IsAnimated:   true,
		IsVideo:      false,
		IsMasks:      false,
		IsEmojis:     false,
		IsOfficial:   false,
		StickerCount: 5,
		Hash:         0,
		ThumbDocId:   0,
		DataJson:     `{"test":true}`,
		FetchedAt:    time.Now().Unix(),
	}

	// Insert
	_, rows, err := setsDAO.InsertIgnore(ctx, do)
	if err != nil {
		t.Fatalf("InsertIgnore failed: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected rowsAffected=1, got %d", rows)
	}
	t.Logf("InsertIgnore: rowsAffected=%d", rows)

	// SelectByShortName
	got, err := setsDAO.SelectByShortName(ctx, do.ShortName)
	if err != nil {
		t.Fatalf("SelectByShortName failed: %v", err)
	}
	if got == nil {
		t.Fatal("SelectByShortName returned nil")
	}
	if got.SetId != do.SetId {
		t.Errorf("SetId: got %d, want %d", got.SetId, do.SetId)
	}
	if got.ShortName != do.ShortName {
		t.Errorf("ShortName: got %s, want %s", got.ShortName, do.ShortName)
	}
	if got.Title != do.Title {
		t.Errorf("Title: got %s, want %s", got.Title, do.Title)
	}
	if got.IsAnimated != do.IsAnimated {
		t.Errorf("IsAnimated: got %v, want %v", got.IsAnimated, do.IsAnimated)
	}
	if got.StickerCount != do.StickerCount {
		t.Errorf("StickerCount: got %d, want %d", got.StickerCount, do.StickerCount)
	}
	t.Logf("SelectByShortName: OK - title=%s, count=%d", got.Title, got.StickerCount)

	// SelectBySetId
	got2, err := setsDAO.SelectBySetId(ctx, setId)
	if err != nil {
		t.Fatalf("SelectBySetId failed: %v", err)
	}
	if got2 == nil {
		t.Fatal("SelectBySetId returned nil")
	}
	if got2.ShortName != do.ShortName {
		t.Errorf("SelectBySetId ShortName: got %s, want %s", got2.ShortName, do.ShortName)
	}
	t.Logf("SelectBySetId: OK - short_name=%s", got2.ShortName)

	// SelectByShortName not found
	notFound, err := setsDAO.SelectByShortName(ctx, "nonexistent_set_xyz_123")
	if err != nil {
		t.Fatalf("SelectByShortName(notfound) error: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for non-existent short_name")
	}
	t.Log("SelectByShortName(not found): OK - returned nil")
}

// TestDBStickerSetsInsertIgnoreDuplicate tests duplicate InsertIgnore returns rowsAffected=0
func TestDBStickerSetsInsertIgnoreDuplicate(t *testing.T) {
	db, setsDAO, _ := setupTestDB(t)
	ctx := context.Background()

	setId := rand.Int63n(1e15) + 1e14
	defer cleanupTestSet(t, db, setId)

	do := &dataobject.StickerSetsDO{
		SetId:       setId,
		AccessHash:  rand.Int63(),
		ShortName:   "dup_test_" + time.Now().Format("20060102150405"),
		Title:       "Dup Test",
		StickerType: "regular",
		DataJson:    "{}",
		FetchedAt:   time.Now().Unix(),
	}

	// First insert
	_, rows1, err := setsDAO.InsertIgnore(ctx, do)
	if err != nil {
		t.Fatalf("First InsertIgnore failed: %v", err)
	}
	if rows1 != 1 {
		t.Fatalf("First insert: expected rows=1, got %d", rows1)
	}

	// Second insert (same short_name) — should be ignored
	do2 := &dataobject.StickerSetsDO{
		SetId:       setId + 1,
		AccessHash:  rand.Int63(),
		ShortName:   do.ShortName,
		Title:       "Should Be Ignored",
		StickerType: "regular",
		DataJson:    "{}",
		FetchedAt:   time.Now().Unix(),
	}
	_, rows2, err := setsDAO.InsertIgnore(ctx, do2)
	if err != nil {
		t.Fatalf("Second InsertIgnore failed: %v", err)
	}
	if rows2 != 0 {
		t.Errorf("Second insert: expected rows=0 (ignored), got %d", rows2)
	}
	t.Logf("Duplicate InsertIgnore: rowsAffected=%d (correctly ignored)", rows2)

	// Verify original data is untouched
	got, err := setsDAO.SelectByShortName(ctx, do.ShortName)
	if err != nil {
		t.Fatalf("SelectByShortName after dup: %v", err)
	}
	if got.Title != "Dup Test" {
		t.Errorf("Title changed to %s, should still be 'Dup Test'", got.Title)
	}
	t.Logf("Original data preserved: title=%s", got.Title)

	cleanupTestSet(t, db, setId+1)
}

// TestDBDocumentsInsertAndSelect tests document InsertIgnore + SelectBySetId
func TestDBDocumentsInsertAndSelect(t *testing.T) {
	db, _, docsDAO := setupTestDB(t)
	ctx := context.Background()

	setId := rand.Int63n(1e15) + 1e14
	defer cleanupTestSet(t, db, setId)

	docs := []*dataobject.StickerSetDocumentsDO{
		{SetId: setId, DocumentId: setId*10 + 1, StickerIndex: 0, Emoji: "😂", BotFileId: "file1", BotFileUniqueId: "u1", DocumentData: "data1"},
		{SetId: setId, DocumentId: setId*10 + 2, StickerIndex: 1, Emoji: "🦆", BotFileId: "file2", BotFileUniqueId: "u2", DocumentData: "data2"},
		{SetId: setId, DocumentId: setId*10 + 3, StickerIndex: 2, Emoji: "😂", BotFileId: "file3", BotFileUniqueId: "u3", DocumentData: "data3"},
	}

	for _, d := range docs {
		_, rows, err := docsDAO.InsertIgnore(ctx, d)
		if err != nil {
			t.Fatalf("InsertIgnore doc %d failed: %v", d.DocumentId, err)
		}
		if rows != 1 {
			t.Fatalf("InsertIgnore doc %d: expected rows=1, got %d", d.DocumentId, rows)
		}
	}
	t.Logf("Inserted %d documents", len(docs))

	// SelectBySetId
	results, err := docsDAO.SelectBySetId(ctx, setId)
	if err != nil {
		t.Fatalf("SelectBySetId failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 docs, got %d", len(results))
	}

	for i, r := range results {
		if r.StickerIndex != int32(i) {
			t.Errorf("doc[%d] sticker_index: got %d, want %d", i, r.StickerIndex, i)
		}
	}
	if results[0].Emoji != "😂" || results[1].Emoji != "🦆" {
		t.Errorf("emoji mismatch: [0]=%s, [1]=%s", results[0].Emoji, results[1].Emoji)
	}
	t.Logf("SelectBySetId: OK - %d docs, ordered by sticker_index", len(results))
}

// TestDBPendingDownloadAndUpdate tests SelectPendingDownloadBySetId + UpdateFileDownloaded
func TestDBPendingDownloadAndUpdate(t *testing.T) {
	db, _, docsDAO := setupTestDB(t)
	ctx := context.Background()

	setId := rand.Int63n(1e15) + 1e14
	defer cleanupTestSet(t, db, setId)

	docId1 := setId*10 + 1
	docId2 := setId*10 + 2

	docs := []*dataobject.StickerSetDocumentsDO{
		{SetId: setId, DocumentId: docId1, StickerIndex: 0, Emoji: "😊", BotFileId: "f1", BotFileUniqueId: "u1", DocumentData: "d1", FileDownloaded: false},
		{SetId: setId, DocumentId: docId2, StickerIndex: 1, Emoji: "😎", BotFileId: "f2", BotFileUniqueId: "u2", DocumentData: "d2", FileDownloaded: false},
	}
	for _, d := range docs {
		docsDAO.InsertIgnore(ctx, d)
	}

	// All should be pending
	pending, err := docsDAO.SelectPendingDownloadBySetId(ctx, setId)
	if err != nil {
		t.Fatalf("SelectPendingDownloadBySetId failed: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(pending))
	}
	t.Logf("Pending before update: %d", len(pending))

	// Mark first as downloaded
	rows, err := docsDAO.UpdateFileDownloaded(ctx, docId1)
	if err != nil {
		t.Fatalf("UpdateFileDownloaded failed: %v", err)
	}
	if rows != 1 {
		t.Errorf("UpdateFileDownloaded rows: got %d, want 1", rows)
	}

	// Now only one pending
	pending2, err := docsDAO.SelectPendingDownloadBySetId(ctx, setId)
	if err != nil {
		t.Fatalf("SelectPendingDownloadBySetId(2) failed: %v", err)
	}
	if len(pending2) != 1 {
		t.Fatalf("expected 1 pending after update, got %d", len(pending2))
	}
	if pending2[0].DocumentId != docId2 {
		t.Errorf("remaining pending docId: got %d, want %d", pending2[0].DocumentId, docId2)
	}
	t.Logf("Pending after update: %d (doc %d)", len(pending2), pending2[0].DocumentId)
}

// TestDBFullFlowBotAPIToDB fetches a real sticker set via Bot API, inserts into DB, reads back and verifies.
func TestDBFullFlowBotAPIToDB(t *testing.T) {
	db, setsDAO, docsDAO := setupTestDB(t)
	botClient := NewBotAPIClient(testBotToken)
	ctx := context.Background()

	// 1. Fetch from Bot API
	botResult, err := botClient.GetStickerSet(ctx, "UtyaDuck")
	if err != nil {
		t.Fatalf("Bot API GetStickerSet failed: %v", err)
	}
	t.Logf("Fetched: name=%s, title=%s, count=%d", botResult.Name, botResult.Title, len(botResult.Stickers))

	// 2. Insert sticker set
	setId := rand.Int63n(1e15) + 1e14
	defer cleanupTestSet(t, db, setId)

	accessHash := rand.Int63()
	now := time.Now().Unix()
	dataJson, _ := json.Marshal(botResult)

	setDO := &dataobject.StickerSetsDO{
		SetId:        setId,
		AccessHash:   accessHash,
		ShortName:    botResult.Name + "_dbtest_" + time.Now().Format("150405"),
		Title:        botResult.Title,
		StickerType:  botResult.StickerType,
		IsAnimated:   len(botResult.Stickers) > 0 && botResult.Stickers[0].IsAnimated,
		IsVideo:      len(botResult.Stickers) > 0 && botResult.Stickers[0].IsVideo,
		StickerCount: int32(len(botResult.Stickers)),
		DataJson:     string(dataJson),
		FetchedAt:    now,
	}

	_, rows, err := setsDAO.InsertIgnore(ctx, setDO)
	if err != nil {
		t.Fatalf("InsertIgnore sticker_sets failed: %v", err)
	}
	if rows != 1 {
		t.Fatalf("InsertIgnore sticker_sets: expected rows=1, got %d", rows)
	}
	t.Logf("Inserted sticker set: set_id=%d, short_name=%s", setId, setDO.ShortName)

	// 3. Insert documents (first 5 to keep test fast)
	limit := 5
	if len(botResult.Stickers) < limit {
		limit = len(botResult.Stickers)
	}

	for idx := 0; idx < limit; idx++ {
		sticker := botResult.Stickers[idx]
		docId := setId*100 + int64(idx)

		doc := mtproto.MakeTLDocument(&mtproto.Document{
			Id:            docId,
			AccessHash:    rand.Int63(),
			FileReference: []byte{},
			Date:          int32(now),
			MimeType:      "application/x-tgsticker",
			Size2_INT32:   int32(sticker.FileSize),
			Size2_INT64:   sticker.FileSize,
			DcId:          1,
		}).To_Document()

		docData, err := proto.Marshal(doc)
		if err != nil {
			t.Fatalf("proto.Marshal doc %d: %v", idx, err)
		}

		thumbFileId := ""
		if sticker.Thumbnail != nil {
			thumbFileId = sticker.Thumbnail.FileId
		}

		docDO := &dataobject.StickerSetDocumentsDO{
			SetId:           setId,
			DocumentId:      docId,
			StickerIndex:    int32(idx),
			Emoji:           sticker.Emoji,
			BotFileId:       sticker.FileId,
			BotFileUniqueId: sticker.FileUniqueId,
			BotThumbFileId:  thumbFileId,
			DocumentData:    base64.StdEncoding.EncodeToString(docData),
			FileDownloaded:  false,
		}

		_, rows, err := docsDAO.InsertIgnore(ctx, docDO)
		if err != nil {
			t.Fatalf("InsertIgnore doc %d: %v", idx, err)
		}
		if rows != 1 {
			t.Fatalf("InsertIgnore doc %d: rows=%d", idx, rows)
		}
	}
	t.Logf("Inserted %d sticker documents", limit)

	// 4. Read back sticker set by ShortName
	readSet, err := setsDAO.SelectByShortName(ctx, setDO.ShortName)
	if err != nil || readSet == nil {
		t.Fatalf("SelectByShortName: err=%v, nil=%v", err, readSet == nil)
	}
	if readSet.Title != botResult.Title {
		t.Errorf("Title: got %s, want %s", readSet.Title, botResult.Title)
	}
	if readSet.StickerCount != int32(len(botResult.Stickers)) {
		t.Errorf("StickerCount: got %d, want %d", readSet.StickerCount, len(botResult.Stickers))
	}
	t.Logf("Read back set: title=%s, count=%d, animated=%v", readSet.Title, readSet.StickerCount, readSet.IsAnimated)

	// 5. Read back by SetId
	readSet2, err := setsDAO.SelectBySetId(ctx, setId)
	if err != nil || readSet2 == nil {
		t.Fatalf("SelectBySetId: err=%v, nil=%v", err, readSet2 == nil)
	}
	if readSet2.ShortName != setDO.ShortName {
		t.Errorf("SelectBySetId ShortName mismatch")
	}
	t.Log("SelectBySetId: OK")

	// 6. Read back documents
	readDocs, err := docsDAO.SelectBySetId(ctx, setId)
	if err != nil {
		t.Fatalf("SelectBySetId docs: %v", err)
	}
	if len(readDocs) != limit {
		t.Fatalf("expected %d docs, got %d", limit, len(readDocs))
	}

	// 7. Deserialize and verify protobuf roundtrip
	for i, rd := range readDocs {
		data, err := base64.StdEncoding.DecodeString(rd.DocumentData)
		if err != nil {
			t.Errorf("doc[%d] base64 decode: %v", i, err)
			continue
		}
		restored := &mtproto.Document{}
		if err := proto.Unmarshal(data, restored); err != nil {
			t.Errorf("doc[%d] proto.Unmarshal: %v", i, err)
			continue
		}
		expectedDocId := setId*100 + int64(i)
		if restored.Id != expectedDocId {
			t.Errorf("doc[%d] Id: got %d, want %d", i, restored.Id, expectedDocId)
		}
		if rd.Emoji == "" {
			t.Errorf("doc[%d] emoji is empty", i)
		}
		if rd.BotFileId == "" {
			t.Errorf("doc[%d] bot_file_id is empty", i)
		}
	}
	t.Logf("All %d documents deserialized and verified", len(readDocs))

	// 8. Verify pending downloads
	pending, err := docsDAO.SelectPendingDownloadBySetId(ctx, setId)
	if err != nil {
		t.Fatalf("SelectPendingDownloadBySetId: %v", err)
	}
	if len(pending) != limit {
		t.Errorf("pending: got %d, want %d", len(pending), limit)
	}
	t.Logf("Pending downloads: %d", len(pending))

	// 9. Mark one downloaded, verify count
	docsDAO.UpdateFileDownloaded(ctx, readDocs[0].DocumentId)
	pending2, _ := docsDAO.SelectPendingDownloadBySetId(ctx, setId)
	if len(pending2) != limit-1 {
		t.Errorf("pending after update: got %d, want %d", len(pending2), limit-1)
	}
	t.Logf("Pending after marking doc %d downloaded: %d", readDocs[0].DocumentId, len(pending2))

	t.Log("=== FULL FLOW BOT API -> DB -> READ BACK: PASS ===")
}
