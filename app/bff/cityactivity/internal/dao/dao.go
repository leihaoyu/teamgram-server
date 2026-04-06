package dao

import (
	"context"
	"flag"
	"fmt"
	"net"
	"sort"
	"time"

	"github.com/oschwald/geoip2-golang"
	"github.com/teamgram/marmota/pkg/net/rpcx"
	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/teamgram-server/app/bff/cityactivity/internal/config"
	chat_client "github.com/teamgram/teamgram-server/app/service/biz/chat/client"
	user_client "github.com/teamgram/teamgram-server/app/service/biz/user/client"
	media_client "github.com/teamgram/teamgram-server/app/service/media/client"
	"github.com/zeromicro/go-zero/core/logx"
)

var mmdb2 string

func init() {
	flag.StringVar(&mmdb2, "mmdb2", "./GeoLite2-City.mmdb", "mmdb path for cityactivity")
}

type Activity struct {
	Id               int64  `db:"id"`
	UserId           int64  `db:"user_id"`
	Title            string `db:"title"`
	Description      string `db:"description"`
	PhotoId          int64  `db:"photo_id"`
	City             string `db:"city"`
	StartTime        int64  `db:"start_time"`
	EndTime          int64  `db:"end_time"`
	MaxParticipants  int32  `db:"max_participants"`
	Status           int32  `db:"status"`
	IsGlobal         int32  `db:"is_global"`
	ChatId           int64  `db:"chat_id"`
	CreatedAt        int64  `db:"created_at"`
	UpdatedAt        int64  `db:"updated_at"`
	ParticipantCount int32  `db:"-"`
	IsJoined         bool   `db:"-"`
	CreatorName      string `db:"-"`
}

type ActivityParticipant struct {
	Id         int64  `db:"id"`
	ActivityId int64  `db:"activity_id"`
	UserId     int64  `db:"user_id"`
	City       string `db:"city"`
	JoinedAt   int64  `db:"joined_at"`
}

type Dao struct {
	db           *sqlx.DB
	MMDB         *geoip2.Reader
	TestCityName string
	media_client.MediaClient
	chat_client.ChatClient
	user_client.UserClient
}

func New(c config.Config) *Dao {
	d := &Dao{
		db:           sqlx.NewMySQL(c.Mysql),
		TestCityName: c.TestCityName,
		MediaClient:  media_client.NewMediaClient(rpcx.GetCachedRpcClient(c.MediaClient)),
		ChatClient:   chat_client.NewChatClient(rpcx.GetCachedRpcClient(c.ChatClient)),
		UserClient:   user_client.NewUserClient(rpcx.GetCachedRpcClient(c.UserClient)),
	}

	MMDB, err := geoip2.Open(mmdb2)
	if err != nil {
		logx.Errorf("cityactivity open mmdb(%s) error: %v", mmdb2, err)
	} else {
		d.MMDB = MMDB
	}

	return d
}

// countryToLocale maps ISO 3166-1 country codes to GeoIP2 locale keys.
var countryToLocale = map[string]string{
	"CN": "zh-CN", "TW": "zh-CN", "HK": "zh-CN", "MO": "zh-CN", "SG": "zh-CN",
	"JP": "ja",
	"DE": "de", "AT": "de", "CH": "de", "LI": "de",
	"ES": "es", "MX": "es", "AR": "es", "CO": "es", "CL": "es",
	"PE": "es", "VE": "es", "EC": "es", "GT": "es", "CU": "es",
	"BO": "es", "DO": "es", "HN": "es", "PY": "es", "SV": "es",
	"NI": "es", "CR": "es", "PA": "es", "UY": "es",
	"FR": "fr", "BE": "fr", "LU": "fr",
	"BR": "pt-BR", "PT": "pt-BR",
	"RU": "ru", "BY": "ru", "KZ": "ru", "KG": "ru",
}

// GetCityByIp resolves the client IP to a city name using GeoIP2.
// Returns the city name in the user's locale (zh-CN for Chinese IPs, etc.)
// Falls back to TestCityName if configured and MMDB lookup fails.
func (d *Dao) GetCityByIp(ip string) string {
	if d.MMDB != nil {
		r, err := d.MMDB.City(net.ParseIP(ip))
		if err != nil {
			logx.Errorf("GetCityByIp - ip: %s, error: %v", ip, err)
		} else {
			countryCode := r.Country.IsoCode
			locale := "en"
			if l, ok := countryToLocale[countryCode]; ok {
				locale = l
			}

			if name, ok := r.City.Names[locale]; ok && name != "" {
				logx.Infof("GetCityByIp - ip: %s, city: %s (locale: %s)", ip, name, locale)
				return name
			}
			if name, ok := r.City.Names["en"]; ok && name != "" {
				logx.Infof("GetCityByIp - ip: %s, city: %s (en)", ip, name)
				return name
			}
		}
	}

	// Fallback to test city (for dev environments)
	if d.TestCityName != "" {
		return d.TestCityName
	}

	return ""
}

func (d *Dao) GetActivitiesByCity(ctx context.Context, city string, offset, limit, filter int32, q string) ([]*Activity, int32, error) {
	var activities []*Activity
	var err error
	var count int32

	// Build search condition
	searchCond := ""
	searchArgs := []interface{}{}
	if q != "" {
		searchCond = " AND (title LIKE ? OR description LIKE ?)"
		likeQ := "%" + q + "%"
		searchArgs = append(searchArgs, likeQ, likeQ)
	}

	// Build query based on filter: 0=all, 1=city only, 2=global only
	switch filter {
	case 1: // city only
		if city == "" {
			query := "SELECT * FROM activities WHERE status = 1 AND is_global = 0" + searchCond + " ORDER BY created_at DESC LIMIT ?, ?"
			args := append(searchArgs, offset, limit)
			err = d.db.QueryRowsPartial(ctx, &activities, query, args...)
			if err != nil {
				return nil, 0, err
			}
			countQuery := "SELECT COUNT(*) FROM activities WHERE status = 1 AND is_global = 0" + searchCond
			_ = d.db.QueryRow(ctx, &count, countQuery, searchArgs...)
		} else {
			query := "SELECT * FROM activities WHERE city = ? AND is_global = 0 AND status = 1" + searchCond + " ORDER BY created_at DESC LIMIT ?, ?"
			args := append([]interface{}{city}, searchArgs...)
			args = append(args, offset, limit)
			err = d.db.QueryRowsPartial(ctx, &activities, query, args...)
			if err != nil {
				return nil, 0, err
			}
			countQuery := "SELECT COUNT(*) FROM activities WHERE city = ? AND is_global = 0 AND status = 1" + searchCond
			countArgs := append([]interface{}{city}, searchArgs...)
			_ = d.db.QueryRow(ctx, &count, countQuery, countArgs...)
		}
	case 2: // global only
		query := "SELECT * FROM activities WHERE is_global = 1 AND status = 1" + searchCond + " ORDER BY created_at DESC LIMIT ?, ?"
		args := append(searchArgs, offset, limit)
		err = d.db.QueryRowsPartial(ctx, &activities, query, args...)
		if err != nil {
			return nil, 0, err
		}
		countQuery := "SELECT COUNT(*) FROM activities WHERE is_global = 1 AND status = 1" + searchCond
		_ = d.db.QueryRow(ctx, &count, countQuery, searchArgs...)
	default: // 0 = all — global max 20, city max 10
		// Fetch global activities (max 20)
		var globalActivities []*Activity
		globalQuery := "SELECT * FROM activities WHERE is_global = 1 AND status = 1" + searchCond + " ORDER BY created_at DESC LIMIT 30"
		globalArgs := append([]interface{}{}, searchArgs...)
		err = d.db.QueryRowsPartial(ctx, &globalActivities, globalQuery, globalArgs...)
		if err != nil {
			return nil, 0, err
		}

		// Fetch city activities (max 10)
		var cityActivities []*Activity
		if city == "" {
			cityQuery := "SELECT * FROM activities WHERE status = 1 AND is_global = 0" + searchCond + " ORDER BY created_at DESC LIMIT 30"
			cityArgs := append([]interface{}{}, searchArgs...)
			err = d.db.QueryRowsPartial(ctx, &cityActivities, cityQuery, cityArgs...)
		} else {
			cityQuery := "SELECT * FROM activities WHERE city = ? AND is_global = 0 AND status = 1" + searchCond + " ORDER BY created_at DESC LIMIT 10"
			cityArgs := append([]interface{}{city}, searchArgs...)
			err = d.db.QueryRowsPartial(ctx, &cityActivities, cityQuery, cityArgs...)
		}
		if err != nil {
			return nil, 0, err
		}

		activities = append(globalActivities, cityActivities...)
		count = int32(len(activities))
	}

	if count == 0 {
		count = int32(len(activities))
	}

	for _, a := range activities {
		a.ParticipantCount = d.getParticipantCount(ctx, a.Id)
	}

	// Sort by participant count descending
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].ParticipantCount > activities[j].ParticipantCount
	})

	return activities, count, nil
}

func (d *Dao) GetActivityById(ctx context.Context, id int64) (*Activity, error) {
	var activity Activity
	query := "SELECT * FROM activities WHERE id = ?"
	err := d.db.QueryRowPartial(ctx, &activity, query, id)
	if err != nil {
		return nil, err
	}
	activity.ParticipantCount = d.getParticipantCount(ctx, id)
	return &activity, nil
}

func (d *Dao) GetMyActivities(ctx context.Context, userId int64, offset, limit int32) ([]*Activity, int32, error) {
	var activities []*Activity
	var count int32

	// Get activities created by user OR joined by user (including taken-down ones for management)
	query := `SELECT DISTINCT a.* FROM activities a
		LEFT JOIN activity_participants ap ON a.id = ap.activity_id AND ap.user_id = ?
		WHERE (a.user_id = ? OR ap.user_id = ?) AND a.status != 2
		ORDER BY a.created_at DESC LIMIT ?, ?`
	err := d.db.QueryRowsPartial(ctx, &activities, query, userId, userId, userId, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	countQuery := `SELECT COUNT(DISTINCT a.id) FROM activities a
		LEFT JOIN activity_participants ap ON a.id = ap.activity_id AND ap.user_id = ?
		WHERE (a.user_id = ? OR ap.user_id = ?) AND a.status != 2`
	_ = d.db.QueryRow(ctx, &count, countQuery, userId, userId, userId)

	if count == 0 {
		count = int32(len(activities))
	}

	for _, a := range activities {
		a.ParticipantCount = d.getParticipantCount(ctx, a.Id)
	}

	return activities, count, nil
}

func (d *Dao) CreateActivity(ctx context.Context, a *Activity) (int64, error) {
	now := time.Now().Unix()
	a.CreatedAt = now
	a.UpdatedAt = now
	a.Status = 1

	query := "INSERT INTO activities (user_id, title, description, photo_id, city, start_time, end_time, max_participants, status, is_global, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	result, err := d.db.Exec(ctx, query, a.UserId, a.Title, a.Description, a.PhotoId, a.City, a.StartTime, a.EndTime, a.MaxParticipants, a.Status, a.IsGlobal, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *Dao) EditActivity(ctx context.Context, id, userId int64, title, description string, photoId, startTime, endTime int64, status int32) error {
	now := time.Now().Unix()
	query := "UPDATE activities SET title=?, description=?, photo_id=?, start_time=?, end_time=?, status=?, updated_at=? WHERE id=? AND user_id=?"
	_, err := d.db.Exec(ctx, query, title, description, photoId, startTime, endTime, status, now, id, userId)
	return err
}

func (d *Dao) DeleteActivity(ctx context.Context, id, userId int64) error {
	now := time.Now().Unix()
	query := "UPDATE activities SET status=2, updated_at=? WHERE id=? AND user_id=?"
	_, err := d.db.Exec(ctx, query, now, id, userId)
	return err
}

func (d *Dao) JoinActivity(ctx context.Context, activityId, userId int64, city string) error {
	activity, err := d.GetActivityById(ctx, activityId)
	if err != nil {
		return err
	}

	if activity.IsGlobal == 0 && activity.City != "" && city != "" && activity.City != city {
		return fmt.Errorf("city mismatch")
	}

	if activity.MaxParticipants > 0 && activity.ParticipantCount >= activity.MaxParticipants {
		return fmt.Errorf("activity is full")
	}

	now := time.Now().Unix()
	query := "INSERT IGNORE INTO activity_participants (activity_id, user_id, city, joined_at) VALUES (?, ?, ?, ?)"
	_, err = d.db.Exec(ctx, query, activityId, userId, city, now)
	return err
}

func (d *Dao) LeaveActivity(ctx context.Context, activityId, userId int64) error {
	query := "DELETE FROM activity_participants WHERE activity_id = ? AND user_id = ?"
	_, err := d.db.Exec(ctx, query, activityId, userId)
	return err
}

func (d *Dao) UpdateActivityChatId(ctx context.Context, activityId, chatId int64) error {
	_, err := d.db.Exec(ctx, "UPDATE activities SET chat_id = ? WHERE id = ?", chatId, activityId)
	return err
}

func (d *Dao) UpdateActivityStatus(ctx context.Context, id, userId int64, status int32) error {
	now := time.Now().Unix()
	_, err := d.db.Exec(ctx, "UPDATE activities SET status=?, updated_at=? WHERE id=? AND user_id=?", status, now, id, userId)
	return err
}

func (d *Dao) IsUserJoined(ctx context.Context, activityId, userId int64) bool {
	var count int32
	query := "SELECT COUNT(*) FROM activity_participants WHERE activity_id = ? AND user_id = ?"
	err := d.db.QueryRow(ctx, &count, query, activityId, userId)
	if err != nil {
		return false
	}
	return count > 0
}

func (d *Dao) getParticipantCount(ctx context.Context, activityId int64) int32 {
	var count int32
	query := "SELECT COUNT(*) FROM activity_participants WHERE activity_id = ?"
	_ = d.db.QueryRow(ctx, &count, query, activityId)
	return count
}

func (d *Dao) SaveActivityMedia(ctx context.Context, activityId int64, photoIds []int64) error {
	now := time.Now().Unix()
	for i, photoId := range photoIds {
		query := "INSERT IGNORE INTO activity_media (activity_id, photo_id, sort_order, created_at) VALUES (?, ?, ?, ?)"
		_, err := d.db.Exec(ctx, query, activityId, photoId, i, now)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Dao) GetActivityPhotoIds(ctx context.Context, activityId int64) ([]int64, error) {
	var photoIds []int64
	query := "SELECT photo_id FROM activity_media WHERE activity_id = ? ORDER BY sort_order ASC"
	err := d.db.QueryRowsPartial(ctx, &photoIds, query, activityId)
	if err != nil {
		return nil, err
	}
	return photoIds, nil
}

func (d *Dao) GetActivitiesFirstPhotoIds(ctx context.Context, activityIds []int64) (map[int64]int64, error) {
	result := make(map[int64]int64)
	if len(activityIds) == 0 {
		return result, nil
	}
	type row struct {
		ActivityId int64 `db:"activity_id"`
		PhotoId    int64 `db:"photo_id"`
	}
	var rows []row
	// sqlx doesn't support IN() directly, query one by one
	for _, aid := range activityIds {
		var r row
		err := d.db.QueryRowPartial(ctx, &r, "SELECT activity_id, photo_id FROM activity_media WHERE activity_id = ? ORDER BY sort_order ASC LIMIT 1", aid)
		if err == nil {
			rows = append(rows, r)
		}
	}
	for _, r := range rows {
		result[r.ActivityId] = r.PhotoId
	}
	return result, nil
}
