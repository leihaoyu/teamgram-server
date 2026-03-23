package dao

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"
)

func findP8File() string {
	// Walk up from current file to find the p8 file in teamgramd/etc/
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	// dao -> internal -> sync -> messenger -> app -> teamgram-server
	root := filepath.Join(dir, "..", "..", "..", "..", "..")
	return filepath.Join(root, "teamgramd", "etc", "AuthKey_JH5C27A29G.p8")
}

func TestP8FileExists(t *testing.T) {
	p8Path := findP8File()
	if _, err := os.Stat(p8Path); os.IsNotExist(err) {
		t.Skipf("p8 file not found at %s, skipping", p8Path)
	}
	t.Logf("p8 file found at %s", p8Path)
}

func TestP8FileFormat(t *testing.T) {
	p8Path := findP8File()
	data, err := os.ReadFile(p8Path)
	if os.IsNotExist(err) {
		t.Skipf("p8 file not found, skipping")
	}
	if err != nil {
		t.Fatalf("failed to read p8 file: %v", err)
	}

	content := string(data)
	if len(content) < 100 {
		t.Fatalf("p8 file too small (%d bytes), likely invalid", len(data))
	}
	if content[:27] != "-----BEGIN PRIVATE KEY-----" {
		t.Fatalf("p8 file does not start with PEM header")
	}
	t.Logf("p8 file format valid, size: %d bytes", len(data))
}

func TestAuthKeyFromP8File(t *testing.T) {
	p8Path := findP8File()
	if _, err := os.Stat(p8Path); os.IsNotExist(err) {
		t.Skipf("p8 file not found, skipping")
	}

	authKey, err := token.AuthKeyFromFile(p8Path)
	if err != nil {
		t.Fatalf("failed to load auth key from p8: %v", err)
	}
	if authKey == nil {
		t.Fatal("auth key is nil")
	}
	t.Log("auth key loaded successfully from p8 file")
}

func TestAPNsTokenCreation(t *testing.T) {
	p8Path := findP8File()
	if _, err := os.Stat(p8Path); os.IsNotExist(err) {
		t.Skipf("p8 file not found, skipping")
	}

	authKey, err := token.AuthKeyFromFile(p8Path)
	if err != nil {
		t.Fatalf("failed to load auth key: %v", err)
	}

	tkn := &token.Token{
		AuthKey: authKey,
		KeyID:   "JH5C27A29G",
		TeamID:  "3WA4Q9D2GD",
	}

	// Create development client (won't actually connect)
	client := apns2.NewTokenClient(tkn).Development()
	if client == nil {
		t.Fatal("failed to create APNs client")
	}
	t.Log("APNs development client created successfully")

	// Create production client
	prodClient := apns2.NewTokenClient(tkn).Production()
	if prodClient == nil {
		t.Fatal("failed to create production APNs client")
	}
	t.Log("APNs production client created successfully")
}

func TestAPNsSendToInvalidToken(t *testing.T) {
	p8Path := findP8File()
	if _, err := os.Stat(p8Path); os.IsNotExist(err) {
		t.Skipf("p8 file not found, skipping")
	}

	authKey, err := token.AuthKeyFromFile(p8Path)
	if err != nil {
		t.Fatalf("failed to load auth key: %v", err)
	}

	tkn := &token.Token{
		AuthKey: authKey,
		KeyID:   "JH5C27A29G",
		TeamID:  "3WA4Q9D2GD",
	}

	client := apns2.NewTokenClient(tkn).Development()

	notification := &apns2.Notification{
		DeviceToken: "invalid_test_token_000000000000000000000000000000000000000000000000000000000000",
		Topic:       "org.delta.pchat",
		Payload:     []byte(`{"aps":{"alert":"test"}}`),
	}

	resp, err := client.Push(notification)
	if err != nil {
		t.Logf("push returned network error (expected if no internet): %v", err)
		return
	}

	// With invalid token, we expect BadDeviceToken response
	t.Logf("APNs response: StatusCode=%d Reason=%s ApnsID=%s", resp.StatusCode, resp.Reason, resp.ApnsID)
	if resp.StatusCode == 400 && resp.Reason == apns2.ReasonBadDeviceToken {
		t.Log("correctly received BadDeviceToken for invalid token - APNs connection works!")
	} else if resp.StatusCode == 200 {
		t.Log("unexpected success - token should be invalid")
	} else {
		t.Logf("received response: %d %s (APNs connection is working)", resp.StatusCode, resp.Reason)
	}
}

func TestPushPayloadStructure(t *testing.T) {
	p := &PushPayload{
		SenderName: "Alice",
		Message:    "Hello!",
		FromUserId: 123,
		MsgId:      1,
		PeerType:   "user",
		PeerId:     456,
	}

	if p.SenderName != "Alice" {
		t.Errorf("expected SenderName=Alice, got %s", p.SenderName)
	}
	if p.Silent {
		t.Error("expected Silent=false")
	}
	if p.ChatId != 0 {
		t.Error("expected ChatId=0 for user message")
	}
}

func TestDeviceInfoStructure(t *testing.T) {
	d := DeviceInfo{
		AuthKeyId:  12345,
		Token:      "abc123",
		AppSandbox: true,
		NoMuted:    true,
	}

	if d.AuthKeyId != 12345 {
		t.Error("wrong AuthKeyId")
	}
	if !d.AppSandbox {
		t.Error("expected AppSandbox=true")
	}
}
